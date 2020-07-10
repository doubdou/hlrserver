// 这个示例程序展示如何创建定制的日志记录器
package main

import (
    "io"
    "log"
	"os"
	"fmt"
)

// 日志
var (
    Debug   *log.Logger // 追踪调试的日志
    Info    *log.Logger // 比较重要的信息
    Warning *log.Logger // 需要注意的内容
    Error   *log.Logger // 非常严重的问题
)

func init() {
    file, err := os.OpenFile("hlr.log",
        os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        log.Fatalln("Failed to open error log file:", err)
    }

    Debug = log.New(io.MultiWriter(file, os.Stdout),
        "[DEBUG] ",
        log.Ldate|log.Ltime|log.Lshortfile|log.Lmsgprefix)

    Info = log.New(io.MultiWriter(file, os.Stdout),
        "[INFO] ",
        log.Ldate|log.Ltime|log.Lshortfile|log.Lmsgprefix)

    Warning = log.New(io.MultiWriter(file, os.Stdout),
        "[WARNING] ",
        log.Ldate|log.Ltime|log.Lshortfile|log.Lmsgprefix)

    Error = log.New(io.MultiWriter(file, os.Stdout),
        "[ERROR] ",
        log.Ldate|log.Ltime|log.Lshortfile|log.Lmsgprefix)
}

// DomainData 域信息
type domainData struct {
	domain string
	tenantID string
	company string
	enable bool
}

// hashNode 域信息的哈希节点
type hashNode struct {
	key string
	value domainData
}

func testHash(){
	fmt.Printf("hello world!\n")
	var da1 domainData
	var da2 domainData
	 
	da1.company = "da1"
	da1.domain  = "da1"
	da1.enable  = true
	da1.tenantID = "da1"

	da2.company = "da2"
	da2.domain  = "da2"
	da2.enable  = true
	da2.tenantID = "da2"

	var m = make(map[string]domainData)//创建一个空白的hash
	m["str1"] = da1
	m["str2"] = da2
	fmt.Println(m["str1"])
	// 遍历 hash
	for key,value := range m { 
		fmt.Printf("---%s %+v---\n",key,value) 
	} 
}


func main() {
    Debug.Println("I have something standard to say")
    Info.Println("Special Information")
    Warning.Println("There is something you need to know about")
	Error.Println("Something has failed")
	testHash()
}