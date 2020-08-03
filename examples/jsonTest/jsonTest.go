package main

import (
	"encoding/json"
	"fmt"
)

type domainJSONResponseData struct {
	ID       int    `json:"id"`
	Domain   string `json:"domain"`
	TenantID int    `json:"tenant_id"`
	Company  string `json:"company"`
	Enable   string `json:"enable"`
}

type getJSONResponse struct {
	Page  int                       `json:"page"`
	Total int                       `json:"total"`
	Data  [2]domainJSONResponseData `json:"data"`
}

type myError struct {
	Code    int32  `json:"code"`
	Message string `json:"message"`
}

type domainReq struct {
	Name     string `json:"name"`
	TenantID int    `json:"tenant_id"`
	Company  string `json:"company"`
	Enable   string `json:"enable"`
}

type objData map[string]interface{}

func demo1() {
	var data []map[string]interface{}
	obj1 := make(map[string]interface{})
	obj1["id"] = 1
	obj1["domain"] = "ai-ym.com"
	obj1["tenant_id"] = 7896
	obj1["company"] = "chibizhun"
	obj1["enable"] = "true"

	obj2 := make(map[string]interface{})
	obj2["startTime"] = "ccccc"

	data = append(data, obj1)
	data = append(data, obj2)

	cnnJSON := make(map[string]interface{})

	cnnJSON["page"] = 1

	cnnJSON["total"] = 10

	cnnJSON["data"] = data

	b, err := json.Marshal(cnnJSON) //json化结果集

	if err != nil {

		fmt.Println("encoding faild")

	} else {

		fmt.Println(string(b))

	}
}

func demo() {
	data := [2]domainJSONResponseData{}
	data[0].Company = "co1"
	data[1].ID = 1
	data[1].TenantID = 40000
	req := getJSONResponse{
		Page:  1,
		Total: 11,
		Data:  data,
	}

	// req.Data = data
	reqStr, _ := json.Marshal(req)
	fmt.Println(string(reqStr))
}
func main() {
	demo1()
}

/*
func main() {
	// json字符中的"引号，需用\进行转义，否则编译出错
	// json字符串沿用上面的结果，但对key进行了大小的修改，并添加了sex数据
	data := "{\"group_id\":18, \"company\":\"cbz1\",\"tenant_id\":300067 }"
	str := []byte(data)

	// 1.Unmarshal的第一个参数是json字符串，第二个参数是接受json解析的数据结构。
	// 第二个参数必须是指针，否则无法接收解析的数据，如stu仍为空对象StuRead{}
	// 2.可以直接stu:=new(StuRead),此时的stu自身就是指针
	req := domainReq{}
	err := json.Unmarshal(str, &req)

	// 解析失败会报错，如json字符串格式不对，缺"号，缺}等。
	if err != nil {
		fmt.Println(err)
	}

	if req.Name != "eedsf" {
		fmt.Println("you found the rule.", req.Name)
	}
	if req.Company != "" && req.Company != "cbz" {
		fmt.Println("company is exists and not cbz:", req.Company)
	}
	fmt.Println("req, group_id :")
}
*/
