package main

import (
	"fmt"
	"net/url"
)

func main(){
	m:= url.Values{"lang":{"cn"}}
	m1 := m
	m.Add("item", "1")
	m.Add("item", "2")

	fmt.Println(m.Get("lang"))
	fmt.Println(m.Get("q"))
	fmt.Println(m.Get("item"))
	fmt.Println(m["item"])


	m = nil
	//m.Add("item", "3")
	fmt.Println(m1["item"])

}