package hlr

import (
	"io"
	"log"
	"os"
)

// 日志
var (
	Debug   *log.Logger // 追踪调试的日志
	Info    *log.Logger // 比较重要的信息
	Warning *log.Logger // 需要注意的内容
	Error   *log.Logger // 非常严重的问题
)

//日志初始化
func loggerInit() {
	file, err := os.OpenFile("hlr.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
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
