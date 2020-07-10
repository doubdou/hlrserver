package main

import (
	"fmt"
	"time"
)

func helloPrint() {
	time.Sleep(5 * time.Second)
	fmt.Println("helloPrint Routinue...")
}

func main() {
	go helloPrint()
	fmt.Println("main func working...")
	time.Sleep(10 * time.Second)
	fmt.Println("main func stop work...")
}
