package hlr

//启动ws服务，多线程并发:
//	1.客户端向hlr发送建立连接请求，参数包含组ID
//	2.向业务层发送坐席状态变更的通知
import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type hlrHTTPHandlers map[string]map[string]interface{}

/*
WebServer web服务存储结构，二维map表以如下方式：
	【路由：http请求方法：处理函数】
*/
type WebServer struct {
	httpHandlerMap hlrHTTPHandlers
}

/* 空闲坐席哈希表存储结构
key：groupID，组ID
value：Queue，空闲坐席号码队列
*/
var waitingAgentCache = struct {
	sync.Mutex
	mapping map[int]Queue
}{
	mapping: make(map[int]Queue),
}

var upgrader = websocket.Upgrader{
	//允许跨域访问
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func decodeAgentRequest(data []byte) (*agentJSONRequest, error) {
	req := agentJSONRequest{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func encodeAgentResponse(resp *agentJSONResponse) ([]byte, error) {
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}

	return jsonResp, nil
}

func joinInWaitingAgents(groupID int, userID int) {
	defer waitingAgentCache.Unlock()
	waitingAgentCache.Lock()
	q := waitingAgentCache.mapping[groupID]
	q.Enqueue(userID)
}

func exitFromWaitingAgents(groupID int) int {
	defer waitingAgentCache.Unlock()
	waitingAgentCache.Lock()
	q := waitingAgentCache.mapping[groupID]

	return q.Dequeue().(int)
}

func domainGet(w http.ResponseWriter, r *http.Request) {
	//支持模糊查询
	//使用id是精确查询，使用domain(对应db中name字段)则是模糊查询
	vars := r.URL.Query()
	var domainID string
	var name string
	var pageStr string
	var pageSizeStr string

	if len(vars["id"]) != 0 {
		domainID = vars["id"][0]
	}
	if len(vars["domain"]) != 0 {
		name = vars["domain"][0]
	}
	if len(vars["page"]) != 0 {
		pageStr = vars["page"][0]
	}
	if len(vars["pageSize"]) != 0 {
		pageSizeStr = vars["pageSize"][0]
	}
	id, _ := strconv.Atoi(domainID)
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	Debug.Printf("domainGet id:%d Name:%s page:%d pageSize:%d", id, name, page, pageSize)
	if page == 0 || pageSize == 0 {
		respErrorMessage(w, codeMissingRequiredParhlr)
		Error.Println("domainGet fail:", codeMissingRequiredParhlr.String())
		return
	}
	db, err := GetDBConnector()
	if err != nil {
		respErrorMessage(w, codeDatabaseConnectFailed)
		Error.Println("domainGet getDBDriver fail", err)
		return
	}

	resJSON := make(map[string]interface{})
	if id != 0 {
		//查询具体id的域
		data := make(map[string]interface{})
		domainInfo, _ := db.ReadDomain(id)
		Debug.Println(domainInfo)
		data["id"] = domainInfo.id
		data["domain"] = domainInfo.Name
		data["tenant_id"] = domainInfo.TenantID
		data["company"] = domainInfo.Company
		data["enable"] = domainInfo.Enable

		resJSON["page"] = page
		resJSON["pageSize"] = pageSize
		resJSON["total"] = 1
		resJSON["data"] = data

	} else {
		//模糊查询域信息
		cnt, err := db.domainCount(name)
		if err != nil {
			respErrorMessage(w, codeSQLExecutionFailed)
			Error.Println("domainGet count fail:", codeSQLExecutionFailed.String())
			return
		}
		data, err := db.domainInfoMapList(name, page, pageSize)
		if err != nil {
			respErrorMessage(w, codeSQLExecutionFailed)
			Error.Println("domainGet fail:", codeSQLExecutionFailed.String())
			return
		}
		resJSON["page"] = page
		resJSON["pageSize"] = pageSize
		resJSON["total"] = cnt
		resJSON["data"] = data
	}

	binData, err := json.Marshal(resJSON) //json化结果集

	if err != nil {
		respErrorMessage(w, codeServerInternalError)
		Error.Println("domainGet fail:", codeServerInternalError.String())
	} else {
		fmt.Fprintf(w, string(binData))
	}
	return
}

func domainModify(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	domainID, err := strconv.Atoi(idStr)
	if err != nil {
		respErrorMessage(w, codeRequestIDInvalid)
		Error.Println("domainModify fail", "request id invalid")
		return
	}
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		respErrorMessage(w, codeBodyReadFailed)
		return
	}
	if len(buf) == 0 {
		respErrorMessage(w, codeBodyEmpty)
		return
	}
	req := domainJSONRequest{}
	err = json.Unmarshal(buf, &req)
	if err != nil {
		respErrorMessage(w, codeBodyParsingFailed)
		return
	}
	if req.Name == "" {
		respErrorMessage(w, codeDomainNotFound)
		return
	}
	db, err := GetDBConnector()
	if err != nil {
		respErrorMessage(w, codeDatabaseConnectFailed)
		Error.Println("domainModify getDBDriver fail", err)
		return
	}
	domainInfo, err := db.ReadDomain(domainID)
	if err != nil {
		respErrorMessage(w, codeDomainNotFound)
		Error.Println("domainModify ReadDomain fail:", err)
		return
	}
	//数据检查
	if req.Name != domainInfo.Name {
		domainInfo.Name = req.Name
	}
	if req.Company != domainInfo.Company {
		domainInfo.Company = req.Company
	}
	if req.TenantID != domainInfo.TenantID {
		domainInfo.TenantID = req.TenantID
	}
	if req.Enable != domainInfo.Enable {
		domainInfo.Enable = req.Enable
	}
	//检查域名重复
	thisID := db.GetDomainIDByName(req.Name)
	if thisID != 0 {
		respErrorMessage(w, codeDomainExists)
		Error.Println("domainModify fail: domain exists:", req.Name)
		return
	}
	err = db.UpdateDomain(domainInfo)
	if err != nil {
		respErrorMessage(w, codeSQLExecutionFailed)
		return
	}
	//设为true,则重载内存数据
	if domainInfo.Enable == "false" && req.Enable == "true" {
		ReloadAllData()
	}
}

func domainDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	domainID, err := strconv.Atoi(idStr)
	if err != nil {
		respErrorMessage(w, codeRequestIDInvalid)
		Error.Println("groupDelete fail:", "request id invalid")
		return
	}
	db, err := GetDBConnector()
	if err != nil {
		respErrorMessage(w, codeDatabaseConnectFailed)
		Error.Println("domainDelete fail:", err)
		return
	}

	domainInfo, _ := db.ReadDomain(domainID)
	//从内存中释放
	if domainInfo.id == 0 {
		respErrorMessage(w, codeDomainNotFound)
		Error.Println("domainDelete fail: domain not found")
		return
	}

	groupID := db.GetOneGroupIDByDomainID(domainInfo.id)
	if groupID != 0 {
		respErrorMessage(w, codeDomainInUse)
		Error.Println("domainDelete fail: domain in use")
		return
	}

	//删除数据库记录
	err = db.DeleteDomain(domainInfo.id)
	if err != nil {
		respErrorMessage(w, codeSQLExecutionFailed)
		Error.Println("domainDelete failed:", err)
		return
	}
	respOKMessage(w, domainInfo.id)
}

// domain 处理POST请求，添加域
func domainAdd(w http.ResponseWriter, r *http.Request) {
	time.Sleep(3 * time.Second)
	//1) 必选项检查
	//2) 检查domain名字的合法性(暂无)
	//3) 检查域是否已存在
	//4) 如果添加的域默认是无效的(enable=0)则无需在内存中创建.
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		respErrorMessage(w, codeBodyReadFailed)
		Error.Println("domainAdd fail:", codeBodyReadFailed.String(), err)
		return
	}
	if len(buf) == 0 {
		respErrorMessage(w, codeBodyEmpty)
		Error.Println("domainAdd fail:", codeBodyEmpty.String())
		return
	}
	req := domainJSONRequest{}
	err = json.Unmarshal(buf, &req)
	if err != nil {
		respErrorMessage(w, codeBodyParsingFailed)
		return
	}

	if req.Name == "" || req.TenantID == 0 || req.Enable == "" {
		respErrorMessage(w, codeMissingRequiredParhlr)
		Error.Println("domainAdd fail: missing required parameter")
		return
	}
	if req.Enable != "true" && req.Enable != "false" {
		respErrorMessage(w, codeParamValueInvalid)
		Error.Println("domainAdd fail: enable value can only be true or false")
		return
	}

	db, err := GetDBConnector()
	if err != nil {
		respErrorMessage(w, codeDatabaseConnectFailed)
		Error.Println("domainAdd fail", err)
		return
	}
	id := db.GetDomainIDByName(req.Name)
	if id != 0 {
		respErrorMessage(w, codeDomainExists)
		Error.Println("domainAdd fail: domain exists")
		return
	}
	newID, err := db.InsertDomain(req.Name, req.TenantID, req.Company, req.Enable)
	if err != nil {
		respErrorMessage(w, codeSQLExecutionFailed)
		Error.Println("domainAdd fail:", err)
		return
	}

	respOKMessage(w, newID)
	if req.Enable == "true" {
		//加载到内存
		p := new(DomainInfo)
		p.id = newID
		p.Name = req.Name
		p.Company = req.Company
		p.TenantID = req.TenantID
		p.Enable = req.Enable
		err := addDomainData(p)
		if err != nil {
			Error.Println("domainAdd fail: insert db success but not load into memory data")
		}
	}
	return
}

func groupGet(w http.ResponseWriter, r *http.Request) {
	// $domain_id 域id, 必填, 所要查询的域.
	// $group_id 实际需要查询的组id，如果无值，则查询域内所有组
	// $page 页码，从1开始
	// $pageSize 每页显示数量

	vars := r.URL.Query()
	var domainIDStr string
	var groupIDStr string
	var pageStr string
	var pageSizeStr string

	//domain_id 必填参数
	if len(vars["domain_id"]) == 0 {
		respErrorMessage(w, codeMissingRequiredParhlr)
		Error.Println("groupGet fail:", codeMissingRequiredParhlr.String())
		return
	}
	if len(vars["domain_id"]) != 0 {
		domainIDStr = vars["domain_id"][0]
	}
	if len(vars["group_id"]) != 0 {
		groupIDStr = vars["group_id"][0]
	}
	if len(vars["page"]) == 0 || len(vars["pageSize"]) == 0 {
		respErrorMessage(w, codeMissingRequiredParhlr)
		Error.Println("groupGet fail:", codeMissingRequiredParhlr.String())
		return
	}
	pageStr = vars["page"][0]
	pageSizeStr = vars["pageSize"][0]

	domainID, _ := strconv.Atoi(domainIDStr)
	if domainID == 0 {
		respErrorMessage(w, codeDomainNotFound)
		Error.Println("groupGet fail:", codeDomainNotFound.String())
		return
	}
	groupID, _ := strconv.Atoi(groupIDStr)
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	Debug.Printf("groupGet domain id:%d group id；%d page:%d pageSize:%d", domainID, groupID, page, pageSize)
	if page == 0 || pageSize == 0 {
		respErrorMessage(w, codeMissingRequiredParhlr)
		Error.Println("groupGet fail:", codeMissingRequiredParhlr.String())
		return
	}
	db, err := GetDBConnector()
	if err != nil {
		respErrorMessage(w, codeDatabaseConnectFailed)
		Error.Println("groupGet fail", err)
		return
	}

	resJSON := make(map[string]interface{})
	if groupID != 0 {
		// 精确查询
		data := make(map[string]interface{})
		groupInfo, err := db.ReadGroup(groupID)
		if err != nil {
			respErrorMessage(w, codeSQLExecutionFailed)
			Error.Println("groupGet fail:", codeSQLExecutionFailed.String())
			return
		}
		Debug.Println(groupInfo)
		data["id"] = groupInfo.id
		data["name"] = groupInfo.Name
		data["group_desc"] = groupInfo.GroupDesc
		data["parent_id"] = groupInfo.ParentID
		data["domain_id"] = groupInfo.DomainID

		resJSON["page"] = page
		resJSON["pageSize"] = pageSize
		resJSON["total"] = 1
		resJSON["data"] = data

	} else {
		// 查询域内所有组
		cnt, err := db.groupCount(domainID)
		if err != nil {
			respErrorMessage(w, codeSQLExecutionFailed)
			Error.Println("domainGet count fail:", codeSQLExecutionFailed.String())
			return
		}
		data, err := db.groupInfoMapList(domainID, page, pageSize)
		if err != nil {
			respErrorMessage(w, codeSQLExecutionFailed)
			Error.Println("domainGet fail:", codeSQLExecutionFailed.String())
			return
		}
		resJSON["page"] = page
		resJSON["pageSize"] = pageSize
		resJSON["total"] = cnt
		resJSON["data"] = data
	}

	binData, err := json.Marshal(resJSON) //json化结果集

	if err != nil {
		respErrorMessage(w, codeServerInternalError)
		Error.Println("groupGet fail:", codeServerInternalError.String())
	} else {
		fmt.Fprintf(w, string(binData))
	}
	return
}

func groupModify(w http.ResponseWriter, r *http.Request) {
	// 1) 检查请求内容是否合法
	// 可修改的信息:
	//		组的名称;
	//		上级组;
	//		组的描述;
	// 不可修改的:
	// 		域id;
	// 2)检查组是否存在
	// 3) 若需要改变上级组信息. 则判断:
	// 		上级组是否存在, 且在同一个域中;
	// 		检查是否互为上级组;
	// 4) 修改数据库
	vars := mux.Vars(r)
	idStr := vars["id"]
	groupID, err := strconv.Atoi(idStr)
	if err != nil {
		respErrorMessage(w, codeRequestIDInvalid)
		Error.Println("groupModify fail", "request id invalid")
		return
	}
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		respErrorMessage(w, codeBodyReadFailed)
		return
	}
	if len(buf) == 0 {
		respErrorMessage(w, codeBodyEmpty)
		return
	}
	req := groupJSONRequest{}
	err = json.Unmarshal(buf, &req)
	if err != nil {
		respErrorMessage(w, codeBodyParsingFailed)
		Error.Println("groupModify fail:", codeBodyParsingFailed.String())
		return
	}
	if req.DomainID != 0 {
		respErrorMessage(w, codeRequestRefused)
		Error.Println("groupModify fail:", codeRequestRefused.String())
		return
	}

	db, err := GetDBConnector()
	if err != nil {
		respErrorMessage(w, codeDatabaseConnectFailed)
		Error.Println("groupModify fail", err)
		return
	}
	groupInfo, _ := db.ReadGroup(groupID)
	if groupInfo.id == 0 {
		respErrorMessage(w, codeGroupNotFound)
		Error.Println("groupModify fail:", codeGroupNotFound.String())
		return
	}

	if req.ParentID > 0 {
		groupParentInfo, _ := db.ReadGroup(req.ParentID)
		if groupParentInfo.id == 0 {
			//上级组不存在
			respErrorMessage(w, codeParamValueInvalid)
			Error.Println("groupModify fail:", codeParamValueInvalid.String())
			return
		}
		if groupParentInfo.DomainID != groupInfo.DomainID {
			//和上级组不在同一个域
			respErrorMessage(w, codeParamValueInvalid)
			Error.Println("groupModify fail:", codeParamValueInvalid.String())
			return
		}
		if groupInfo.id == groupParentInfo.ParentID {
			//互为上级组
			respErrorMessage(w, codeParamValueInvalid)
			Error.Println("groupModify fail:", codeParamValueInvalid.String())
			return
		}
		groupInfo.ParentID = req.ParentID
	}
	groupInfo.GroupDesc = req.GroupDesc
	groupInfo.Name = req.Name

	err = db.UpdateGroup(groupInfo)
	if err != nil {
		respErrorMessage(w, codeSQLExecutionFailed)
		Error.Println("groupModify fail:", codeSQLExecutionFailed.String())
		return
	}
	respOKMessage(w, groupInfo.id)
	return
}

func groupDelete(w http.ResponseWriter, r *http.Request) {
	// 1) 找到组, 如果找不到, 返回部分或小组不存在
	// 2) 查找下级组, 如果有, 则提示有下级小组存在, 不能删除
	// 3) 查询用户, 如果有, 则提示有用户存在, 不能删除.
	// 4) 删除操作.

	vars := mux.Vars(r)
	idStr := vars["id"]
	groupID, err := strconv.Atoi(idStr)
	if err != nil {
		respErrorMessage(w, codeRequestIDInvalid)
		Error.Println("groupDelete fail", "request id invalid")
		return
	}
	db, err := GetDBConnector()
	if err != nil {
		respErrorMessage(w, codeDatabaseConnectFailed)
		Error.Println("groupDelete fail", err)
		return
	}
	groupInfo, err := db.ReadGroup(groupID)
	if err != nil {
		respErrorMessage(w, codeDatabaseConnectFailed)
		Error.Println("groupDelete fail", err)
		return
	}

	if groupInfo.id == 0 {
		respErrorMessage(w, codeGroupNotFound)
		Error.Println("groupDelete fail: group not found")
		return
	}

	childID := db.GetChildGroupIDbyParentID(groupInfo.id)
	if childID != 0 {
		respErrorMessage(w, codeGroupInUse)
		Error.Println("groupDelete fail: group in use by child group ", childID)
		return
	}
	userID := db.GetOneUserByGroupID(groupInfo.id)
	if userID != 0 {
		respErrorMessage(w, codeGroupInUse)
		Error.Println("groupDelete fail: group in use because still users in group")
		return
	}
	//无需从内存中删除
	//删除数据库记录
	err = db.DeleteGroup(groupInfo.id)
	if err != nil {
		respErrorMessage(w, codeSQLExecutionFailed)
		Error.Println("domainDelete fail: SQL execute failed:", err)
		return
	}
	respOKMessage(w, groupInfo.id)
}

func groupAdd(w http.ResponseWriter, r *http.Request) {
	//1) 必选项检查
	//2) 检查group名字是否已经存在
	//3) 如果存在上级组parentID参数，检查是否存在，并且该上级组所属domainID与本请求的domainID一致
	//4) 如果不存在上级组，检查domainID是否存在
	//4) 在内存中加载

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		respErrorMessage(w, codeBodyReadFailed)
		return
	}
	if len(buf) == 0 {
		respErrorMessage(w, codeBodyEmpty)
		return
	}
	req := groupJSONRequest{}
	err = json.Unmarshal(buf, &req)
	if err != nil {
		respErrorMessage(w, codeBodyParsingFailed)
		return
	}

	if req.Name == "" || req.DomainID == 0 {
		respErrorMessage(w, codeMissingRequiredParhlr)
		Error.Println("groupAdd fail: missing required parameter")
		return
	}

	db, err := GetDBConnector()
	if err != nil {
		respErrorMessage(w, codeDatabaseConnectFailed)
		Error.Println("groupAdd getDBDriver fail", err)
		return
	}
	id := db.GetGroupIDByName(req.Name)
	if id != 0 {
		respErrorMessage(w, codeGroupExists)
		Error.Println("groupAdd fail: group exists")
		return
	}
	if req.ParentID != 0 {
		parentGroup, _ := db.ReadGroup(req.ParentID)
		if parentGroup.id == 0 {
			respErrorMessage(w, codeRequestRefused)
			Error.Println("groupAdd fail: parent id not exists")
			return
		}
		if parentGroup.DomainID != req.DomainID {
			respErrorMessage(w, codeRequestRefused)
			Error.Println("groupAdd fail: request domain is different from parent's domain")
			return
		}
	}
	domainInfo, _ := db.ReadDomain(req.DomainID)
	if domainInfo.Name == "" {
		respErrorMessage(w, codeDomainNotFound)
		Error.Println("groupAdd fail: request domain not exists")
		return
	}

	newID, err := db.InsertGroup(req.Name, req.GroupDesc, req.ParentID, req.DomainID)
	if err != nil {
		respErrorMessage(w, codeSQLExecutionFailed)
		Error.Println("groupAdd fail:", err)
		return
	}

	if domainInfo.Enable == "true" {
		//加载到内存
		domain := locateDomainAndLock(domainInfo.Name)
		reloadDomain(domain)
		domain.Unlock()
	}
	respOKMessage(w, newID)
	return
}

func userGet(w http.ResponseWriter, r *http.Request) {
	// $id为实际需要查询的号码id
	// $domain需要查询的域
	// $group 需要查询的组
	// 注意: $id, $domain, $group这三个参数只能出现一个,不能同时出现.
	vars := r.URL.Query()
	var userIDStr string
	var domainIDStr string
	var groupIDStr string
	var pageStr string
	var pageSizeStr string
	resJSON := make(map[string]interface{})

	//domain_id 必填参数
	if len(vars["user_id"]) == 0 && len(vars["group_id"]) == 0 && len(vars["domain_id"]) == 0 {
		respErrorMessage(w, codeMissingRequiredParhlr)
		Error.Println("userGet fail:", codeMissingRequiredParhlr.String())
		return
	}
	if len(vars["page"]) == 0 || len(vars["pageSize"]) == 0 {
		respErrorMessage(w, codeMissingRequiredParhlr)
		Error.Println("userGet fail:", codeMissingRequiredParhlr.String())
		return
	}
	pageStr = vars["page"][0]
	pageSizeStr = vars["pageSize"][0]
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	if page == 0 || pageSize == 0 {
		respErrorMessage(w, codeMissingRequiredParhlr)
		Error.Println("userGet fail:", codeMissingRequiredParhlr.String())
		return
	}

	db, err := GetDBConnector()
	if err != nil {
		respErrorMessage(w, codeDatabaseConnectFailed)
		Error.Println("userGet fail", err)
		return
	}

	if len(vars["user_id"]) != 0 {
		userIDStr = vars["user_id"][0]
		userID, _ := strconv.Atoi(userIDStr)
		if userID == 0 {
			respErrorMessage(w, codeGroupNotFound)
			Error.Println("userGet fail:", codeGroupNotFound.String())
			return
		}
		data, err := db.userInfoMapListByUserID(userID, page, pageSize)
		if err != nil {
			respErrorMessage(w, codeSQLExecutionFailed)
			Error.Println("userGet fail:", codeSQLExecutionFailed.String())
			return
		}

		resJSON["page"] = page
		resJSON["pageSize"] = pageSize
		resJSON["total"] = 1
		resJSON["data"] = data
	} else if len(vars["group_id"]) != 0 {
		// 精确查询组内user
		groupIDStr = vars["group_id"][0]
		groupID, _ := strconv.Atoi(groupIDStr)
		if groupID == 0 {
			respErrorMessage(w, codeGroupNotFound)
			Error.Println("userGet fail:", codeGroupNotFound.String())
			return
		}
		cnt, err := db.userCountByGroupID(groupID)
		if err != nil {
			respErrorMessage(w, codeSQLExecutionFailed)
			Error.Println("userGet count fail:", codeSQLExecutionFailed.String())
			return
		}
		data, err := db.userInfoMapListByGroupID(groupID, page, pageSize)
		if err != nil {
			respErrorMessage(w, codeSQLExecutionFailed)
			Error.Println("userGet fail:", codeSQLExecutionFailed.String())
			return
		}

		resJSON["page"] = page
		resJSON["pageSize"] = pageSize
		resJSON["total"] = cnt
		resJSON["data"] = data
	} else if len(vars["domain_id"]) != 0 {
		// 精确查询域内user
		domainIDStr = vars["domain_id"][0]
		domainID, _ := strconv.Atoi(domainIDStr)
		if domainID == 0 {
			respErrorMessage(w, codeDomainNotFound)
			Error.Println("userGet fail:", codeDomainNotFound.String())
			return
		}
		cnt, err := db.userCountByDomainID(domainID)
		if err != nil {
			respErrorMessage(w, codeSQLExecutionFailed)
			Error.Println("userGet count fail:", codeSQLExecutionFailed.String())
			return
		}
		data, err := db.userInfoMapListByDomainID(domainID, page, pageSize)
		if err != nil {
			respErrorMessage(w, codeSQLExecutionFailed)
			Error.Println("userGet fail:", codeSQLExecutionFailed.String())
			return
		}

		resJSON["page"] = page
		resJSON["pageSize"] = pageSize
		resJSON["total"] = cnt
		resJSON["data"] = data
	} else {
		// 无能为力
	}

	binData, err := json.Marshal(resJSON) //json化结果集

	if err != nil {
		respErrorMessage(w, codeServerInternalError)
		Error.Println("userGet fail:", codeServerInternalError.String())
	} else {
		fmt.Fprintf(w, string(binData))
	}
	return
}

func userModify(w http.ResponseWriter, r *http.Request) {
	// 1) 检查用户是否存在，获取用户信息、组ID 、域名等
	// 2) 检查允许修改的内容以及合法性：
	//   	账号名;
	//   	密码;
	//   	组ID(检查请求的group是否与当前的group在同一个domain);
	// 3) 检查用户注册状态是否登出
	// 4) 修改数据库
	// 5) 更新内存

	vars := mux.Vars(r)
	idStr := vars["id"]
	userID, err := strconv.Atoi(idStr)
	if err != nil {
		respErrorMessage(w, codeRequestIDInvalid)
		Error.Println("userModify fail:", "request id invalid")
		return
	}
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		respErrorMessage(w, codeBodyReadFailed)
		return
	}
	if len(buf) == 0 {
		respErrorMessage(w, codeBodyEmpty)
		Error.Println("userModify fail:", codeBodyEmpty.String())
		return
	}
	db, err := GetDBConnector()
	if err != nil {
		respErrorMessage(w, codeDatabaseConnectFailed)
		Error.Println("userModify fail", err)
		return
	}
	req := userJSONRequest{}
	err = json.Unmarshal(buf, &req)
	if err != nil {
		respErrorMessage(w, codeBodyParsingFailed)
		Error.Println("userModify fail:", codeBodyParsingFailed.String())
		return
	}
	if req.Password == "" || req.GroupID == 0 {
		respErrorMessage(w, codeMissingRequiredParhlr)
		Error.Println("userModify fail:", codeMissingRequiredParhlr.String())
		return
	}
	userInfo, err := db.ReadUser(userID)
	if err != nil {
		respErrorMessage(w, codeDatabaseConnectFailed)
		Error.Println("userModify fail", err)
		return
	}
	if userInfo.id == 0 {
		respErrorMessage(w, codeUserNotFound)
		Error.Println("userModify fail", codeUserNotFound.String())
		return
	}
	groupInfo, _ := db.ReadGroup(userInfo.GroupID)
	if groupInfo.id == 0 {
		respErrorMessage(w, codeServerInternalError)
		Error.Println("userModify fail:", codeServerInternalError.String())
		return
	}
	realm := db.GetRealmByID(groupInfo.DomainID)
	if realm == "" {
		respErrorMessage(w, codeServerInternalError)
		Error.Println("userModify fail:", codeServerInternalError.String())
		return
	}
	//检查请求的group是否与当前的group在同一个domain
	if userInfo.GroupID != req.GroupID {
		realmRequest := db.GetRealmByID(req.GroupID)
		if realmRequest != realm {
			respErrorMessage(w, codeRequestRefused)
			Error.Println("userModify fail:", codeRequestRefused.String())
		}
	}
	domainManage := findDomainAndRLock(realm)
	user := domainManage.mapping[userInfo.Username]
	user.Lock()
	domainManage.RUnlock()
	if user.Status == StatusAvailable {
		respErrorMessage(w, codeRequestRefused)
		Error.Println("userModify fail:", codeRequestRefused.String())
		user.Unlock()
		return
	}
	//更新数据库
	userInfo.Password = req.Password
	userInfo.GroupID = req.GroupID
	err = db.UpdateUser(userInfo)
	if err != nil {
		//数据库更新失败，回滚，内存数据不动
		respErrorMessage(w, codeSQLExecutionFailed)
		Error.Println("userModify fail:", codeSQLExecutionFailed.String())
		user.Unlock()
		return
	}

	user.UserInfo = userInfo
	user.Unlock()

	respOKMessage(w, userInfo.id)
}

func userDelete(w http.ResponseWriter, r *http.Request) {
	// 1) 检查用户是否存在，获取用户信息、组ID 、域名等
	// 2) 检查用户注册状态是否登出
	// 3) 从数据库删除
	// 4) 更新内存

	vars := mux.Vars(r)
	idStr := vars["id"]
	userID, err := strconv.Atoi(idStr)
	if err != nil {
		respErrorMessage(w, codeRequestIDInvalid)
		Error.Println("userDelete fail:", "request id invalid")
		return
	}

	db, err := GetDBConnector()
	if err != nil {
		respErrorMessage(w, codeDatabaseConnectFailed)
		Error.Println("userDelete fail", err)
		return
	}

	userInfo, err := db.ReadUser(userID)
	if err != nil {
		respErrorMessage(w, codeSQLExecutionFailed)
		Error.Println("userDelete fail ", codeSQLExecutionFailed.String(), err)
		return
	}
	if userInfo.id == 0 {
		respErrorMessage(w, codeUserNotFound)
		Error.Println("userDelete fail", codeUserNotFound.String())
		return
	}
	groupInfo, _ := db.ReadGroup(userInfo.GroupID)
	if groupInfo.id == 0 {
		respErrorMessage(w, codeServerInternalError)
		Error.Println("userDelete fail:", codeServerInternalError.String())
		return
	}
	realm := db.GetRealmByID(groupInfo.DomainID)
	if realm == "" {
		respErrorMessage(w, codeServerInternalError)
		Error.Println("userDelete fail:", codeServerInternalError.String())
		return
	}

	domainManage := findDomainAndRLock(realm)
	user := domainManage.mapping[userInfo.Username]
	user.Lock()
	domainManage.RUnlock()

	if user.Status == StatusAvailable {
		respErrorMessage(w, codeRequestRefused)
		Error.Println("userDelete fail:", codeRequestRefused.String())
		user.Unlock()
		return
	}
	//从数据库删除
	user.Unlock()
	err = db.DeleteUser(userInfo.id)
	if err != nil {
		//数据库更新失败，回滚，内存数据不动
		respErrorMessage(w, codeSQLExecutionFailed)
		Error.Println("userDelete fail:", codeSQLExecutionFailed.String())
		return
	}

	domainManage = locateDomainAndLock(realm)
	domainManage.mapping[userInfo.Username] = nil
	domainManage.Unlock()

	respOKMessage(w, userInfo.id)
}

func userAdd(w http.ResponseWriter, r *http.Request) {
	// 1) 检查需要添加的组是否存在
	// 2) 检查username 在对应的域中是否重复
	// 3) 添加到数据库中
	// 4) 如果内存中有对应的域, 则添加到内存中.
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		respErrorMessage(w, codeBodyReadFailed)
		return
	}
	if len(buf) == 0 {
		respErrorMessage(w, codeBodyEmpty)
		return
	}
	req := userJSONRequest{}
	err = json.Unmarshal(buf, &req)
	if err != nil {
		respErrorMessage(w, codeBodyParsingFailed)
		Error.Println("useradd fail", codeBodyParsingFailed.String())
		return
	}
	if req.Username == "" || req.Password == "" || req.GroupID == 0 {
		respErrorMessage(w, codeMissingRequiredParhlr)
		Error.Println("useradd fail", codeMissingRequiredParhlr.String())
		return
	}
	Debug.Println("-----", req.Username, req.Password, req.GroupID)
	db, err := GetDBConnector()
	if err != nil {
		respErrorMessage(w, codeDatabaseConnectFailed)
		Error.Println("useradd fail", err)
		return
	}
	groupInfo, err := db.ReadGroup(req.GroupID)
	if err != nil {
		respErrorMessage(w, codeServerInternalError)
		Error.Println("useradd fail", codeServerInternalError.String())
		return
	}
	if groupInfo.id == 0 {
		respErrorMessage(w, codeGroupNotFound)
		Error.Println("useradd fail", codeGroupNotFound.String())
		return
	}

	// 检查username 在对应的域中是否重复
	if db.CheckUsernameInDomainExist(groupInfo.DomainID, req.Username) {
		respErrorMessage(w, codeUserExists)
		Error.Println("useradd fail:", codeUserExists.String())
		return
	}
	//数据库操作
	userID, err := db.InsertUser(req.Username, req.Password, req.GroupID)
	if err != nil {
		respErrorMessage(w, codeSQLExecutionFailed)
		Error.Println("useradd fail:", codeSQLExecutionFailed.String())
		return
	}
	respOKMessage(w, userID)
	return
}

//<?xml version=\"1.0\" standalone=\"no\"?>
func authInfoMarshal(domain string, groupID int, username string, password string) string {
	return fmt.Sprintf(`
	<?xml version=\"1.0\" standalone=\"no\"?>
	<document type="freeswitch/xml">
	<section name="directory">
	<domain name="%s">
	<parhlr>
	<param name="dial-string" value="{^^:sip_invite_domain=%s:presence_id=%s@%s}${sofia_contact(*/%s@%s)},${verto_contact(%s@%s)}"/>
	<param name="jsonrpc-allowed-methods" value="verto"/>
	<param name="jsonrpc-allowed-event-channels" value="demo,conference,presence"/>
	</parhlr>
	<variables>
	<variable name="record_stereo" value="true"/>
	<variable name="default_gateway" value="%s"/>
	<variable name="default_areacode" value="%s"/>
	<variable name="transfer_fallback_usernamesion" value="operator"/>
	</variables>
	<groups>
	<group name="g%d">
	<users>
	<user id="%s">
	<parhlr>
	<param name="password" value="%s"/>
	<param name="vm-password" value="%s"/>
	</parhlr>
	<variables>
	<variable name="toll_allow" value="domestic,international,local"/>
	<variable name="accountcode" value="86"/>
	<variable name="user_context" value="default"/>
	<variable name="effective_caller_id_name" value="Extension %s"/>
	<variable name="effective_caller_id_number" value="%s"/>
	<variable name="outbound_caller_id_name" value="FS callcenter"/>
	<variable name="outbound_caller_id_number" value="8888"/>
	<variable name="callgroup" value="g%d"/>
	</variables>
	</user>  
	</users>
	</group>
	</groups>
	</domain>
	</section> 
	</document>
	`,
		domain,
		domain, username, domain, username, domain, username, domain,
		domain,
		domain,
		groupID,
		username,
		password,
		username,
		username,
		username,
		groupID)
}

//鉴权
func numberAuth(w http.ResponseWriter, r *http.Request) {
	Debug.Println("recv auth request", r.RequestURI)
	err := r.ParseForm()
	if err != nil {
		Error.Println("number auth, pase form fail", err)
		respErrorMessage(w, codeBadRequest)
		return
	}
	eventNameArr := r.PostForm["Event-Name"]
	if len(eventNameArr) != 1 {
		Error.Println("number auth bad request:event-name not found")
		respErrorMessage(w, codeBadRequest)
		return
	}

	if eventNameArr[0] == "GENERAL" {
		w.WriteHeader(200)
		return
	} else if eventNameArr[0] == "REQUEST_PARhlr" {
		actionArr := r.PostForm["action"]
		if len(actionArr) != 1 {
			w.WriteHeader(200)
			return
		}
		if actionArr[0] != "sip_auth" && actionArr[0] != "jsonrpc-authenticate" && actionArr[0] != "user_call" {
			Error.Printf("number auth bad request:event-name:%s action:%s", eventNameArr[0], actionArr[0])
			respErrorMessage(w, codeBadRequest)
			return
		}
	}
	//如果鉴权请求来自陌生主机，拒绝
	pos := strings.Index(r.RemoteAddr, ":")
	if pos < 0 {
		Error.Println("remote address parsing error:", r.RemoteAddr)
		respErrorMessage(w, codeRequestRefused)
		return
	}
	fsAddr := r.RemoteAddr[:pos]
	if fsAddr != EventsocketConfigGet().Host {
		Error.Println("The authentication request comes from unconfigured host: ", fsAddr)
		respErrorMessage(w, codeRequestRefused)
		return
	}
	// authtype := r.PostForm["action"]
	// if len(authtype) != 1 {
	// 	Error.Println("auth request refused because authtype not found")
	// 	w.WriteHeader(400)
	// 	respErrorMessage(codeBadRequest))
	// 	return
	// }
	// Debug.Println(r.PostForm)
	userArr := r.PostForm["user"]
	domainArr := r.PostForm["domain"]
	if len(userArr) != 1 || len(domainArr) != 1 {
		Error.Println("number auth fail: bad request ", userArr, domainArr)
		respErrorMessage(w, codeBadRequest)
		return
	}

	domainStr := domainArr[0]
	userStr := userArr[0]
	hlrDataManage.RLock()
	defer hlrDataManage.RUnlock()
	thisDomain := hlrDataManage.mapping[domainStr]
	if thisDomain == nil {
		Error.Printf("auth request refused because domain[%s] not exists:", domainStr)
		respErrorMessage(w, codeBadRequest)
		return
	}

	thisDomain.RLock()
	thisUser := thisDomain.mapping[userStr]
	if thisUser == nil {
		Error.Printf("auth request refused because user[%s] not exists:", userStr)
		respErrorMessage(w, codeBadRequest)
		thisDomain.RUnlock()
		return
	}
	thisUser.Lock()
	res := authInfoMarshal(thisDomain.Name, thisUser.GroupID, thisUser.Username, thisUser.Password)
	thisUser.Unlock()
	thisDomain.RUnlock()
	Debug.Printf("user [%s][%s] auth request accepted", thisUser.Username, thisUser.Password)
	// Debug.Println(res)
	w.Write([]byte(res))
}

func agentStateHandler(w http.ResponseWriter, r *http.Request) {
	// vars := mux.Vars(r)
	// userID := vars["id"]
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		respErrorMessage(w, codeBodyReadFailed)
		return
	}
	if len(buf) == 0 {
		respErrorMessage(w, codeBodyEmpty)
		return
	}
	req := agentStateRequest{}
	err = json.Unmarshal(buf, &req)
	if err != nil {
		respErrorMessage(w, codeBodyParsingFailed)
		return
	}

	if req.Realm == "" || req.Username == "" || req.State == "" {
		respErrorMessage(w, codeMissingRequiredParhlr)
		Error.Println("agentState handle fail: missing required parameter")
		return
	}
	//坐席自己能切换的只有idle状态和waiting状态
	if req.State != StateIdle && req.State != StateWaiting {
		respErrorMessage(w, codeRequestRefused)
		Error.Println("agentState handle fail: request is refused")
		return
	}
	err = userStateSet(req.Username, req.Realm, req.State)
	if err != nil {
		respErrorMessage(w, codeRequestRefused)
		Error.Println("agentState handle fail: request is refused")
		return
	}
}

/*
在浏览器的console中可以调试websocket服务，举例：
	var ws = new WebSocket("ws://localhost:8083/v1/voip/hlr/agent")
	ws.addEventListener("message", function(e){console.log(e);});
	ws.send("hello, this is ws client")
	ws.close()
*/
func agentHandler(w http.ResponseWriter, r *http.Request) {
	//收到http请求(upgrade),完成websocket协议转换
	//在应答的header中放上upgrade:websoket
	var (
		conn *websocket.Conn
		err  error
		//msgType int
		data []byte
	)
	if conn, err = upgrader.Upgrade(w, r, nil); err != nil {
		Error.Println(err)
		return
	}
	Error.Println("get ws connection")
	data = []byte("test ws")
	//得到了websocket.Conn长连接的对象，实现数据的收发
	go func() {
		var recvData []byte
		var err error

		for {
			if _, recvData, err = conn.ReadMessage(); err != nil {
				Error.Println(err)
				return
			}
			//读取数据，请求格式 {"group_id": 1}
			Error.Println("read messsage gid: ", getGID(), string(recvData))
		}
	}()
	for {
		//发送数据，判断返回值是否报错
		if err = conn.WriteMessage(websocket.TextMessage, data); err != nil {
			Error.Println(err)
			goto ERR
		}
		Error.Println("send messsage gid: ", getGID())
		//Error.Println("send msg:", string(data))
		time.Sleep(5 * time.Second)
	}

ERR:
	conn.Close()
}

func (srv *WebServer) hlrHTTPSubFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		Error.Println("sub directory not support POST method")
		respErrorMessage(w, codeBadRequestMethod)
		return
	}
	vars := mux.Vars(r)
	v := vars["category"]
	handler := srv.httpHandlerMap[v][r.Method]
	if handler == nil {
		respErrorMessage(w, codeBadRequestForm)
		return
	}
	handler.(func(http.ResponseWriter, *http.Request))(w, r)
}

func (srv *WebServer) hlrHTTPFunc(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	v := vars["category"]
	handler := srv.httpHandlerMap[v][r.Method]
	Debug.Printf("---- v:%s method:%s", v, r.Method)
	if handler == nil {
		respErrorMessage(w, codeBadRequestForm)
		return
	}
	handler.(func(http.ResponseWriter, *http.Request))(w, r)
}

//NewWebServer 注册http路由，生成一个新的web服务实例
func NewWebServer() *WebServer {
	var server WebServer
	handlers := make(hlrHTTPHandlers)
	//域路由
	handlers["domain"] = make(map[string]interface{})
	handlers["domain"]["GET"] = domainGet
	handlers["domain"]["PUT"] = domainModify
	handlers["domain"]["DELETE"] = domainDelete
	handlers["domain"]["POST"] = domainAdd
	//组路由
	handlers["group"] = make(map[string]interface{})
	handlers["group"]["GET"] = groupGet
	handlers["group"]["PUT"] = groupModify
	handlers["group"]["DELETE"] = groupDelete
	handlers["group"]["POST"] = groupAdd
	//用户路由
	handlers["user"] = make(map[string]interface{})
	handlers["user"]["GET"] = userGet
	handlers["user"]["PUT"] = userModify
	handlers["user"]["DELETE"] = userDelete
	handlers["user"]["POST"] = userAdd
	//号码鉴权
	handlers["auth"] = make(map[string]interface{})
	handlers["auth"]["POST"] = numberAuth
	//坐席状态管理 (获取空闲坐席、坐席状态切换)
	handlers["agent"] = make(map[string]interface{})
	handlers["agent"]["GET"] = agentHandler
	handlers["agent"]["PUT"] = agentStateHandler
	//生成web服务实例
	server.httpHandlerMap = handlers
	return &server
}

//Serve web服务启动入口
func (srv *WebServer) Serve(addr string) error {
	if srv == nil {
		return errors.New("webserver is null")
	}
	//路由绑定
	r := mux.NewRouter()
	r.HandleFunc("/v1/voip/hlr/{category}", srv.hlrHTTPFunc)
	r.HandleFunc("/v1/voip/hlr/{category}/{id:[0-9]+}", srv.hlrHTTPSubFunc)

	Info.Println("http serve gid: ", getGID())
	//服务端启动

	http.ListenAndServe(addr, r)
	return nil
}
