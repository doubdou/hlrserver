package main

import "fmt"

func aPutHandler(num int) {
	fmt.Println("post", num)
}

func aGetHandler(num int) {
	fmt.Println("post", num)
}

func bDeleteHandler(num int) {
	fmt.Println("post", num)
}

type hlrHTTPHandlers map[string]map[string]interface{}

/* handlerCache = hlrHTTPHandlers{
	handlerCache = make(map[string]map[string]interface{})
	handlerCache["domain"] =  make(map[string]interface{})
} */

func main() {
	var HTTPHandlerMap hlrHTTPHandlers
	HTTPHandlerMap = make(map[string]map[string]interface{})
	// myHTTPMap := make(map[string]map[string]interface{})

	HTTPHandlerMap["a"] = make(map[string]interface{})
	HTTPHandlerMap["a"]["PUT"] = aPutHandler
	HTTPHandlerMap["a"]["GET"] = aGetHandler
	HTTPHandlerMap["b"] = make(map[string]interface{})
	HTTPHandlerMap["b"]["DELETE"] = bDeleteHandler

	cateGeory := "v"
	reqStr := "GET"
	myf := HTTPHandlerMap[cateGeory][reqStr]
	if myf == nil {
		fmt.Println("func not found.")
		return
	}
	myf.(func(int))(5)
}
