package hlr

//启动ws服务，多线程并发:
//	1.客户端向HLR发送建立连接请求，参数包含组ID
//	2.向业务层发送坐席状态变更的通知
import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
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
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		Error.Println("domainGet:", err)
	}
	Error.Println(r.RequestURI, r.Method, string(buf))
	w.Write(buf)
}

func domainModify(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, errorMessage(codeBodyReadFailed))
		return
	}
	if len(buf) == 0 {
		fmt.Fprintf(w, errorMessage(codeBodyEmpty))
		return
	}
	req := domainJSONRequest{}
	err = json.Unmarshal(buf, &req)
	if err != nil {
		fmt.Fprintf(w, errorMessage(codeBodyParsingFailed))
		return
	}
	// debug start
	Error.Println(string(buf), "---json Unmarshal---", req)
	//debug end
	//域不存在
	if req.Name == "" {
		fmt.Fprintf(w, errorMessage(codeDomainNotFound))
		return
	}
	db, err := GetDBConnector()
	if err != nil {
		fmt.Fprintf(w, errorMessage(codeDatabaseConnectFailed))
		Error.Println("domainModify getDBDriver fail", err)
		return
	}
	domainInfo, err := db.ReadDomain(id)
	if err != nil {
		fmt.Fprintf(w, errorMessage(codeDomainNotFound))
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
	//检查租户ID是否重复
	//主要在于是否允许一个租户拥有多个域

	err = db.UpdateDomain(domainInfo)
	if err != nil {
		fmt.Fprintf(w, errorMessage(codeSQLExecutionFailed))
		return
	}
}

func domainDelete(w http.ResponseWriter, r *http.Request) {
}

// domain 处理POST请求，添加域
func domainAdd(w http.ResponseWriter, r *http.Request) {
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		Error.Println("domainGet:", err)
	}
	time.Sleep(3 * time.Second)
	Error.Println(r.RequestURI, r.Method, string(buf))
	w.Write(buf)
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
<variable name="accountcode" value="%s"/>
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
		username,
		groupID)
}

//鉴权
func numberAuth(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		Error.Println("number auth fail", err)
		w.WriteHeader(400)
		fmt.Fprintf(w, errorMessage(codeBadRequest))
		return
	}
	userArr := r.PostForm["user"]
	domainArr := r.PostForm["domain"]
	if len(userArr) != 1 || len(domainArr) != 1 {
		Error.Println("number auth fail: bad request ", userArr, domainArr)
		w.WriteHeader(400)
		fmt.Fprintf(w, errorMessage(codeBadRequest))
		return
	}

	domainStr := domainArr[0]
	userStr := userArr[0]

	// Error.Println(r.RequestURI, r.Method)
	// w.Write(buf)
	//res := authInfoMarshal("ai-ym.com", "dev", "3000051001", "1001")

	thisDomain := hlrDataManage[domainStr]
	Debug.Println("-----debug--------->", userStr, domainStr, thisDomain)
	thisDomain.RLock()
	thisUser := thisDomain.mapping[userStr]
	thisUser.Lock()
	res := authInfoMarshal(thisDomain.Name, thisDomain.id, thisUser.Username, thisUser.Password)
	thisUser.Unlock()
	thisDomain.RUnlock()
	fmt.Println(res)
	w.Write([]byte(res))
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
		fmt.Fprintf(w, errorMessage(codeBadRequestMethod))
		return
	}
	vars := mux.Vars(r)
	v := vars["category"]
	handler := srv.httpHandlerMap[v][r.Method]
	if handler == nil {
		fmt.Fprintf(w, errorMessage(codeBadRequestForm))
		return
	}
	handler.(func(http.ResponseWriter, *http.Request))(w, r)
}

func (srv *WebServer) hlrHTTPFunc(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	v := vars["category"]
	handler := srv.httpHandlerMap[v][r.Method]
	if handler == nil {
		fmt.Fprintf(w, v, errorMessage(codeBadRequestForm))
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
	//获取空闲坐席
	handlers["agent"] = make(map[string]interface{})
	handlers["agent"]["GET"] = agentHandler
	//生成web服务实例
	server.httpHandlerMap = handlers
	return &server
}

//Serve web服务启动入口
func (srv *WebServer) Serve(addr string) error {
	if srv == nil {
		return errors.New("wbserver is null")
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
