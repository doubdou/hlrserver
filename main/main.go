package main

import (
	"fmt"
	"hlr"

	_ "github.com/lib/pq"
)

// 主函数概述：
//	1.读取配置文件
//	2.初始化数据库相关的全局变量（锁、句柄等）
//	3.连接数据库
//	4.初始化全局数据结构，从数据库加载数据到内存
//
//	5.连接FreeSWITCH的esl服务，如果连接失败，定时重连；使用独立的线程管理ESL连接
//	6.初始化HTTP Server、websocket Server等
// 		启动HTTP Server,提供服务：
// 			a.添加、修改、删除坐席
// 			b.FreeSWITCH鉴权
// 			c.修改坐席状态
//
// 		启动websocket Server，提供服务:
// 			获取空闲坐席。注意，客户端向HLR发送建立连接请求，参数包含域（初期会让所有租户使用同一个域）、租户ID、组ID

func main() {
	hlr.LoadConfig()

	dbConf := hlr.DatabaseConfigGet()
	_, err := hlr.OpenDBConnector(dbConf.Host, dbConf.Port, dbConf.User, dbConf.Password, dbConf.Name)
	if err != nil {
		hlr.Error.Println("db connect", err)
		return
	}

	hlr.LoadAllDataFromDB()

	httpConf := hlr.HTTPConfigGet()
	httpAddr := fmt.Sprintf("0.0.0.0:%s", httpConf.Port)

	srv := hlr.NewWebServer()
	srv.Serve(httpAddr)
}
