package main

import "fmt"

type shape interface {
	Area() float64
	Perimeter() float64
}

//Rect ww
type Rect struct {
	width  float64
	height float64
}

// Area bb
func (r Rect) Area() float64 {
	return r.width * r.height
}

//Perimeter cc
func (r Rect) Perimeter() float64 {
	return 2 * (r.width + r.height)
}

type mapTest map[string]interface{}

func main() {
	var s shape
	s = Rect{5.0, 4.0}
	r := Rect{5.0, 4.0}
	fmt.Printf("%T", s)
	fmt.Println("\n-------")
	fmt.Printf("%v", s)
	fmt.Println("\n-------")
	fmt.Println(r)
	fmt.Println("\n-------")
	var mt mapTest
	mt = make(map[string]interface{})
	mt["jin"] = "jinzw"
	mt["jin2"] = 2
	fmt.Printf("%#v", mt)
	fmt.Println(mt)
	fmt.Println("\n-------")
	var a interface{}
	var b interface{}
	a = "123"
	b = a.(string)

	fmt.Printf("%#v", b)
}
