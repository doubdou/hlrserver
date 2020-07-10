package main

import (
	"fmt"
	"unsafe"
)

type reason int

func main() {
	var i1 int = 1
	var i2 int32 = 2
	var i3 int64 = 3

	fmt.Println(unsafe.Sizeof(i1))
	fmt.Println(unsafe.Sizeof(i2))
	fmt.Println(unsafe.Sizeof(i3))

	var r reason
	r = 10
	//fmt.Printf("r %v", r)
	i := &r

	fmt.Printf("r %d", i)
}
