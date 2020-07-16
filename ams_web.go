package ams

//启动ws服务，多线程并发:
//	1.客户端向ams发送建立连接请求，参数包含组ID
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

type amsHTTPHandlers map[string]map[string]interface{}

/*
WebServer web服务存储结构，二维map表以如下方式：
	【路由：http请求方法：处理函数】
*/
type WebServer struct {
	httpHandlerMap amsHTTPHandlers
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
	return
}

func domainModify(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
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
	domainInfo, err := db.ReadDomain(id)
	if err != nil {
		respErrorMessage(w, codeDomainNotFound)
		Error.Println("domainModify ReadDomain ", id, err)
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
}

func domainDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respErrorMessage(w, codeRequestIDInvalid)
		Error.Println("domainDelete fail", err)
		return
	}
	db, err := GetDBConnector()
	if err != nil {
		respErrorMessage(w, codeDatabaseConnectFailed)
		Error.Println("domainDelete getDBDriver fail", err)
		return
	}
	realm := db.GetDomainByID(id)
	//从内存中释放
	domain := locateDomainAndLock(realm)
	if domain == nil {
		respErrorMessage(w, codeDomainNotFound)
		Error.Println("domainDelete fail: domain not found")
		return
	}
	if domain.agentCount > 0 {
		respErrorMessage(w, codeDomainInUse)
		Error.Println("domainDelete fail: domain in use")
		domain.Unlock()
		return
	}
	deleteDomainData(realm)
	//删除数据库记录
	err = db.DeleteDomain(id)
	if err != nil {
		respErrorMessage(w, codeSQLExecutionFailed)
		Error.Println("domainDelete fail: SQL execute failed:", err)
		domain.Unlock()
		return
	}
}

// domain 处理POST请求，添加域
func domainAdd(w http.ResponseWriter, r *http.Request) {
	//1) 必选项检查
	//2) 检查domain名字的合法性(暂无)
	//3) 检查域是否已存在
	//4) 如果添加的域默认是无效的(enable=0)则无需在内存中创建.
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

	if req.Name == "" || req.TenantID == 0 || req.Enable == "" {
		respErrorMessage(w, codeMissingRequiredParams)
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
		Error.Println("domainAdd getDBDriver fail", err)
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

	fmt.Fprintf(w, respOKMessage(newID))
	if req.Enable == "true" {
		//加载到内存
		domain := locateDomainAndLock(req.Name)
		reloadDomain(domain)
		domain.Unlock()
	}
	return
}

func groupGet(w http.ResponseWriter, r *http.Request) {
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		Error.Println("groupGet:", err)
	}
	Error.Println(r.RequestURI, r.Method, string(buf))
}

func groupModify(w http.ResponseWriter, r *http.Request) {
}

func groupDelete(w http.ResponseWriter, r *http.Request) {
}

func groupAdd(w http.ResponseWriter, r *http.Request) {
}

func userGet(w http.ResponseWriter, r *http.Request) {
}

func userModify(w http.ResponseWriter, r *http.Request) {
}

func userDelete(w http.ResponseWriter, r *http.Request) {
}

func userAdd(w http.ResponseWriter, r *http.Request) {
}

//<?xml version=\"1.0\" standalone=\"no\"?>
func authInfoMarshal(domain string, groupID int, username string, password string) string {
	return fmt.Sprintf(`
	<?xml version=\"1.0\" standalone=\"no\"?>
	<document type="freeswitch/xml">
	<section name="directory">
	<domain name="%s">
	<params>
	<param name="dial-string" value="{^^:sip_invite_domain=%s:presence_id=%s@%s}${sofia_contact(*/%s@%s)},${verto_contact(%s@%s)}"/>
	<param name="jsonrpc-allowed-methods" value="verto"/>
	<param name="jsonrpc-allowed-event-channels" value="demo,conference,presence"/>
	</params>
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
	<params>
	<param name="password" value="%s"/>
	<param name="vm-password" value="%s"/>
	</params>
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
	} else if eventNameArr[0] == "REQUEST_PARAMS" {
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
	Debug.Println(r.PostForm)
	userArr := r.PostForm["user"]
	domainArr := r.PostForm["domain"]
	if len(userArr) != 1 || len(domainArr) != 1 {
		Error.Println("number auth fail: bad request ", userArr, domainArr)
		respErrorMessage(w, codeBadRequest)
		return
	}

	domainStr := domainArr[0]
	userStr := userArr[0]
	amsDataManage.RLock()
	defer amsDataManage.RUnlock()
	thisDomain := amsDataManage.mapping[domainStr]
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

	if req.Realm == "" || req.username == "" || req.State == "" {
		respErrorMessage(w, codeMissingRequiredParams)
		Error.Println("agentState handle fail: missing required parameter")
		return
	}
	//坐席自己能切换的只有idle状态和waiting状态
	if req.State != StateIdle && req.State != StateWaiting {
		respErrorMessage(w, codeRequestRefused)
		Error.Println("agentState handle fail: request is refused")
		return
	}
	err = userStateSet(req.username, req.Realm, req.State)
	if err != nil {
		respErrorMessage(w, codeRequestRefused)
		Error.Println("agentState handle fail: request is refused")
		return
	}
}

/*
在浏览器的console中可以调试websocket服务，举例：
	var ws = new WebSocket("ws://localhost:8083/v1/voip/ams/agent")
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

func (srv *WebServer) amsHTTPSubFunc(w http.ResponseWriter, r *http.Request) {
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

func (srv *WebServer) amsHTTPFunc(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	v := vars["category"]
	handler := srv.httpHandlerMap[v][r.Method]
	if handler == nil {
		respErrorMessage(w, codeBadRequestForm)
		return
	}
	handler.(func(http.ResponseWriter, *http.Request))(w, r)
}

//NewWebServer 注册http路由，生成一个新的web服务实例
func NewWebServer() *WebServer {
	var server WebServer
	handlers := make(amsHTTPHandlers)
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
	r.HandleFunc("/v1/voip/ams/{category}", srv.amsHTTPFunc)
	r.HandleFunc("/v1/voip/ams/{category}/{id:[0-9]+}", srv.amsHTTPSubFunc)

	Info.Println("http serve gid: ", getGID())
	//服务端启动

	http.ListenAndServe(addr, r)
	return nil
}
