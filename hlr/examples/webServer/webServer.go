package main

import (
	"bytes"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{
		//允许跨域访问
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func getCurrentThreadID() int {
	var user32 *syscall.DLL
	var GetCurrentThreadID *syscall.Proc
	var err error

	user32, err = syscall.LoadDLL("Kernel32.dll")
	if err != nil {
		fmt.Printf("syscall.LoadDLL fail: %v\n", err.Error())
		return 0
	}
	GetCurrentThreadID, err = user32.FindProc("GetCurrentThreadId")
	if err != nil {
		fmt.Printf("user32.FindProc fail: %v\n", err.Error())
		return 0
	}

	var pid uintptr
	pid, _, err = GetCurrentThreadID.Call()

	return int(pid)
}

func getGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

func domainHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("RequestURI:", r.RequestURI)
	w.Write([]byte("Hello World!"))
}

func groupHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("RequestURI:", r.RequestURI)
	w.Write([]byte("Hello World!"))
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("RequestURI:", r.RequestURI)
	w.Write([]byte("Hello World!"))
}

func agentHandler(w http.ResponseWriter, r *http.Request) {
	//w.Write([]byte("hello"))
	//收到http请求(upgrade),完成websocket协议转换
	//在应答的header中放上upgrade:websoket
	var (
		conn *websocket.Conn
		err  error
		//msgType int
		data []byte
	)
	if conn, err = upgrader.Upgrade(w, r, nil); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("get ws connection")
	data = []byte("test ws")
	//得到了websocket.Conn长连接的对象，实现数据的收发
	go func() {
		var recvData []byte
		var err error

		for {
			if _, recvData, err = conn.ReadMessage(); err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println("read messsage gid: ", getGID(), "thread ID:", getCurrentThreadID(), string(recvData))
		}
	}()
	for {
		/* 		if _, data, err = conn.ReadMessage(); err != nil {
			//报错关闭websocket
			fmt.Println(err)
			goto ERR
		} */
		//发送数据，判断返回值是否报错
		if err = conn.WriteMessage(websocket.TextMessage, data); err != nil {
			fmt.Println(err)
			goto ERR
		}
		fmt.Println("send messsage gid: ", getGID(), "thread ID:", getCurrentThreadID())
		//fmt.Println("send msg:", string(data))
		time.Sleep(5 * time.Second)
	}

ERR:
	conn.Close()
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("RequestURI:", r.RequestURI)
	w.Write([]byte("Hello World!"))
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("httpHandler: RawPath", r.URL.RawPath)
	w.Write([]byte("http handler!"))
}
func main() {

	/* 	http.HandleFunc("/ws", agentHandler)
	   	http.HandleFunc("/hello", helloHandler) */
	http.HandleFunc("/v1/voip/hlr/domain", domainHandler)
	http.HandleFunc("/v1/voip/hlr/group", groupHandler)
	http.HandleFunc("/v1/voip/hlr/user", userHandler)
	http.HandleFunc("/v1/voip/hlr/agent", agentHandler)

	fmt.Println("func main gid: ", getGID(), "thread ID:", getCurrentThreadID())
	//服务端启动
	http.ListenAndServe("0.0.0.0:7777", nil)
}
