package ams

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type reason int

//http请求正确返回
type httpOKRespDesc struct {
	ID int `json:"id"`
}

//http请求错误返回
type httpErrRespDesc struct {
	Code    reason `json:"code"`
	Message string `json:"message"`
}

const (
	codeSuccess reason = 0
	/*********ams内部定义错误状态码****************************/
	//http请求原因
	codeBodyEmpty             reason = 101
	codeLackOfParams          reason = 102
	codeBadRequestForm        reason = 103
	codeBadRequestMethod      reason = 104
	codeBodyParsingFailed     reason = 105
	codeMissingRequiredParams reason = 106
	codeRequestRefused        reason = 107
	codeRequestIDInvalid      reason = 108

	//数据原因
	codeGroupNotFound     reason = 202
	codeDomainNotFound    reason = 201
	codeUserNotFound      reason = 203
	codeDomainExists      reason = 204
	codeGroupExists       reason = 205
	codeUserExists        reason = 206
	codeStateChangeFailed reason = 207
	codeDomainDisabled    reason = 208
	codeParamValueInvalid reason = 209
	codeDomainInUse       reason = 210
	codeGroupInUse        reason = 211
	codeUserInUse         reason = 212
	//服务端原因
	codeDatabaseConnectFailed reason = 300
	codeSQLExecutionFailed    reason = 301
	codeBodyReadFailed        reason = 302
	codeServerInternalError   reason = 303
	//http协议头状态码
	/***************http协议***********************************/
	codeBadRequest   reason = 400
	codePageNotFound reason = 404
)

func (r reason) Intger() int {
	var e interface{}
	e = r
	return e.(int)
}

func (r reason) String() string {
	switch r {
	case codeSuccess:
		return "success"
	//http请求原因
	case codeBodyEmpty:
		return "http body data is empty"
	case codeLackOfParams:
		return "Lack of params"
	case codeBadRequestForm:
		return "Bad request form"
	case codeBadRequestMethod:
		return "Bad request method"
	case codeBodyParsingFailed:
		return "Body parsing failed"
	case codeMissingRequiredParams:
		return "Missing required parameter"
	case codeRequestRefused:
		return "The request is refused"
	case codeRequestIDInvalid:
		return "The id is invalid"

	//数据原因
	case codeDomainNotFound: //201
		return "Domain not found"
	case codeGroupNotFound:
		return "Group not found"
	case codeUserNotFound:
		return "User not found"
	case codeDomainExists:
		return "Domain exists"
	case codeGroupExists:
		return "Group exists"
	case codeUserExists:
		return "User exists"
	case codeStateChangeFailed: //207
		return "Change agent state failed"
	case codeDomainDisabled: //208
		return "Domain disabled"
	case codeParamValueInvalid: //209
		return "The parameter value is invalid"
	case codeDomainInUse:
		return "The domain in use"
	case codeGroupInUse:
		return "The group in use"
	case codeUserInUse:
		return "The user in use"
	//服务端原因
	case codeDatabaseConnectFailed: //300
		return "can't connect Database"
	case codeSQLExecutionFailed:
		return "SQL execution failed"
	case codeBodyReadFailed:
		return "http body read failed"
	case codePageNotFound:
		return "page not Found"
	case codeBadRequest:
		return "Bad request"
	case codeServerInternalError:
		return "Server internal error"
	default:
		return "Unknown error"
	}
}

func respErrorMessage(w http.ResponseWriter, r reason) {
	if r < 400 {
		//私有状态码,统一回复400
		w.WriteHeader(400)
	} else {
		w.WriteHeader(int(r))
	}
	var desc httpErrRespDesc
	desc.Code = r
	desc.Message = r.String()
	jsonStr, _ := json.Marshal(&desc)
	fmt.Fprintf(w, string(jsonStr))
}

// func respErrorMessage(r reason) string {
// 	//私有状态码
// 	if r < 400 {
// 		var desc httpErrRespDesc
// 		desc.Code = r
// 		desc.Message = r.String()
// 		jsonStr, _ := json.Marshal(&desc)
// 		return string(jsonStr)
// 	}
// 	//http状态码
// 	return fmt.Sprintf("%d %s", r, r.String())
// }

func respOKMessage(w http.ResponseWriter, id int) {
	w.WriteHeader(200)
	if id > 0 {
		var desc httpOKRespDesc
		desc.ID = id
		jsonStr, _ := json.Marshal(&desc)
		fmt.Fprintf(w, string(jsonStr))
	}
}
