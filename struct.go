package hlr

import (
	"bytes"
	"runtime"
	"strconv"
	"sync"
)

/******************************************************* 常量 ******************************************************************/
type userstatusType string
type agentStateType string

//注册状态和坐席状态
const (
	//StatusLoggedOut 注册状态：未登录
	StatusLoggedOut userstatusType = "Logged Out"
	//StatusAvailable 注册状态：已登录
	StatusAvailable userstatusType = "Available"
)

const (
	//StateIdle 坐席状态：休息/整理中
	StateIdle agentStateType = "Idle"
	//StateWaiting 坐席状态：休息/整理中
	StateWaiting agentStateType = "Waiting"
	//StateMakingCall 坐席状态：拨打中
	StateMakingCall agentStateType = "Making Call"
	//StateInCall 坐席状态：正在通话
	StateInCall agentStateType = "In Call"
)

/*********************************************** hlr服务各类数据的结构体 *********************************************************/

//DomainInfo 域信息
type DomainInfo struct {
	id       int
	Name     string
	TenantID int
	Company  string
	Enable   string
}

//GroupInfo 组信息
type GroupInfo struct {
	id        int
	Name      string
	GroupDesc string
	ParentID  int
	DomainID  int
}

//UserInfo 号码基本信息
type UserInfo struct {
	id       int
	Username string
	Password string
	update   string
	GroupID  int
}

//UserCompleteInfo 用户号码完整信息
type UserCompleteInfo struct {
	*sync.Mutex                  //互斥锁
	*UserInfo                    //用户号码基本信息
	Status        userstatusType // 注册状态
	State         agentStateType // 坐席状态
	Talking       bool           // 是否通话中
	TalkedTime    int            //通话总时长,单位是秒，自注册成功开始算
	AnsweredCalls int            // 接听通话数，自注册成功开始算
}

//DomainManage 管理域内的所有用户信息
//key: 域ID
//value: 属于该域的用户信息
type DomainManage struct {
	*sync.RWMutex
	*DomainInfo
	agentCount int
	mapping    map[string]*UserCompleteInfo
}

/*********************************************** 处理HTTP请求的相关JSON结构 *********************************************************/
//domain的http请求消息
type domainJSONRequest struct {
	Name     string `json:"name"`
	TenantID int    `json:"tenant_id"`
	Company  string `json:"company"`
	Enable   string `json:"enable"`
}

//group的http请求消息
type groupJSONRequest struct {
	Name      string `json:"name"`
	GroupDesc string `json:"group_desc"`
	ParentID  int    `json:"parent_id"`
	DomainID  int    `json:"domain_id"`
}

//user的http请求消息
type userJSONRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	GroupID  int    `json:"group_id"`
}

type agentStateRequest struct {
	Realm    string         `json:"realm"`
	Username string         `json:"username"`
	State    agentStateType `json:"state"`
}

//agent的http请求消息
type agentJSONRequest struct {
	GroupID int `json:"group_id"`
}

//agent的http响应消息
type agentJSONResponse struct {
	Username   string `json:"username"`
	DomainID   int    `json:"domain_id"`
	GroupDesc  string `json:"group_desc"`
	SwitchName string `json:"switch_name"`
}

// //auth的http请求信息（不完整）
// type authXMLRequest struct {
// 	User   string `xml:"user"`
// 	Domain string `xml:"domain"`
// }

/*********************************************** 通用管理函数、调试函数 *********************************************************/

func getGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}
