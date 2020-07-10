package hlr

import (
	"encoding/json"
	"fmt"
)

type reason int

type httpRespDesc struct {
	Code    reason `json:"code"`
	Message string `json:"message"`
}

const (
	codeSuccess reason = 0
	/*********HLR内部定义错误状态码****************************/
	//http请求问题
	codeBodyEmpty         reason = 101
	codeLackOfParams      reason = 102
	codeBadRequestForm    reason = 103
	codeBadRequestMethod  reason = 104
	codeBodyParsingFailed reason = 105
	//数据不符
	codeDomainNotFound reason = 201
	codeGroupNotFound  reason = 202
	codeUserNotFound   reason = 203
	codeDomainExists   reason = 204
	codeGroupExists    reason = 205
	codeUserExists     reason = 206
	codeDomainDisabled reason = 207
	//服务端内部问题
	codeDatabaseConnectFailed reason = 300
	codeSQLExecutionFailed    reason = 301
	codeBodyReadFailed        reason = 302
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
	//http请求问题
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
	//数据不符
	case codeDomainNotFound:
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
	//服务端内部问题
	case codeDatabaseConnectFailed:
		return "can't connect Database"
	case codeSQLExecutionFailed:
		return "SQL execution failed"
	case codeBodyReadFailed:
		return "http body read failed"
	case codePageNotFound:
		return "page not Found"
	case codeBadRequest:
		return "Bad request"
	default:
		return "Unknown error"
	}
}

func errorMessage(r reason) string {
	//私有状态码
	if r < 400 {
		var desc httpRespDesc
		desc.Code = r
		desc.Message = r.String()
		jsonStr, _ := json.Marshal(&desc)
		return string(jsonStr)
	}
	//http状态码
	return fmt.Sprintf("%d %s", r, r.String())
}
