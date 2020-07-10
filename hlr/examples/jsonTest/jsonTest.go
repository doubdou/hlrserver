package main

import (
	"encoding/json"
	"fmt"
)

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

func main() {
	//json字符中的"引号，需用\进行转义，否则编译出错
	//json字符串沿用上面的结果，但对key进行了大小的修改，并添加了sex数据
	data := "{\"group_id\":18, \"company\":\"cbz1\",\"tenant_id\":300067 }"
	str := []byte(data)

	//1.Unmarshal的第一个参数是json字符串，第二个参数是接受json解析的数据结构。
	//第二个参数必须是指针，否则无法接收解析的数据，如stu仍为空对象StuRead{}
	//2.可以直接stu:=new(StuRead),此时的stu自身就是指针
	req := domainReq{}
	err := json.Unmarshal(str, &req)

	//解析失败会报错，如json字符串格式不对，缺"号，缺}等。
	if err != nil {
		fmt.Println(err)
	}

	if req.Name != "eedsf" {
		fmt.Println("you found the rule.", req.Name)
	}
	if req.Company != "" && req.Company != "cbz" {
		fmt.Println("company is exists and not cbz:", req.Company)
	}
	// fmt.Println("req, group_id :")

}
