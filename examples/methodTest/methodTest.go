package main

import (
	"fmt"
	"image/color"
	"math"
	"sync"
)

//Point 00
type Point struct {
	x, y float64
}

//ColoredPoint cc
type ColoredPoint struct {
	Point
	Color color.RGBA
}

/*
var struct {
	mu 	sync.Mutex
	mapping = make(map[string]string)
}

//Lookup ll
func Lookup(key string) string {
	mu.Lock()
	v := mapping[key]
	mu.Unlock()
	return v
}

*/

var cache = struct {
	sync.Mutex
	mapping map[string]string
}{
	mapping: make(map[string]string),
}

//Lookup ll
func Lookup(key string) string {
	cache.Lock()
	v := cache.mapping[key]
	cache.Unlock()
	return v
}

//Distance 11
func Distance(p Point, q Point) float64 {
	return math.Hypot(q.x-p.x, q.y-p.y)
}

//Distance 22
func (p Point) Distance(q Point) float64 {
	return math.Hypot(q.x-p.x, q.y-p.y)
}

func main() {
	p := Point{2, 2}
	q := Point{5, 6}
	fmt.Println(Distance(p, q))
	fmt.Println(p.Distance(q))
}
