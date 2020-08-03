package main

import (
	"ams"
	"fmt"
	"time"
)

func main() {
	q := ams.QueueCreate()

	go func() {
		for {
			fmt.Println("Dequeue: ", q.Dequeue())
		}
	}()

	q.Enqueue(1)
	q.Enqueue(3)
	q.Enqueue(5)
	time.Sleep(5 * time.Second)
	q.Enqueue(7)
	q.Enqueue("hello")
	q.Enqueue(9)

	time.Sleep(5 * time.Second)
	q.Enqueue("hello2")
	q.Enqueue("hello3")
	fmt.Println("q length:", q.Length())
	time.Sleep(50 * time.Second)
}
