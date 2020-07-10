package hlr

//https://github.com/fiorix/go-eventsocket

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
)

//ConfigRoot 定义配置文件结构
type ConfigRoot struct {
	HTTP HTTPConfig        `xml:"http"`
	Db   DatabaseConfig    `xml:"database"`
	ESL  EventsocketConfig `xml:"eventsocket"`
}

//HTTPConfig 定义http配置数据
type HTTPConfig struct {
	Port    string `xml:"port"`
	Threads string `xml:"threads"`
}

//DatabaseConfig 定义db配置数据
type DatabaseConfig struct {
	Host     string `xml:"host"`
	Port     string `xml:"port"`
	Name     string `xml:"name"`
	User     string `xml:"user"`
	Password string `xml:"password"`
}

//EventsocketConfig 定义esl配置数据
type EventsocketConfig struct {
	Host     string `xml:"host"`
	Port     string `xml:"port"`
	User     string `xml:"user"`
	Password string `xml:"password"`
}

var hlrConfig ConfigRoot

// HTTPConfigGet 获取配置数据
func HTTPConfigGet() *HTTPConfig {
	return &hlrConfig.HTTP
}

// DatabaseConfigGet 获取配置数据
func DatabaseConfigGet() *DatabaseConfig {
	return &hlrConfig.Db
}

//EventsocketConfigGet 获取配置数据
func EventsocketConfigGet() *EventsocketConfig {
	return &hlrConfig.ESL
}

// LoadConfig 加载配置数据
func LoadConfig() {
	content, err := ioutil.ReadFile("./hlr.xml")
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(9)
	}

	xml.Unmarshal(content, &hlrConfig)
	fmt.Println("--------------------")
	fmt.Println(hlrConfig)
	fmt.Println("--------------------")
	//日志初始化
	loggerInit()
}
