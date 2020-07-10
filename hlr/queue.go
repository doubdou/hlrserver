package hlr

import (
	"sync"
)

type queueNode struct {
	data interface{}
	next *queueNode
}

//Queue 队列
type Queue struct {
	*sync.Cond
	len     int
	Waiting chan int
	front   *queueNode
	rear    *queueNode
}

//QueueCreate 创建新的队列
func QueueCreate() *Queue {
	return &Queue{
		len:  0,
		Cond: sync.NewCond(new(sync.Mutex)),
		// Waiting: make(chan int, 3000),
		front: nil,
		rear:  nil,
	}
}

//IsEmpty 判断队列空
func (q *Queue) IsEmpty() bool {
	return q.len == 0
}

//Length 队列节点个数
func (q *Queue) Length() int {
	if q == nil {
		return -1
	}

	return q.len
}

//Enqueue 入队
func (q *Queue) Enqueue(data interface{}) {
	q.L.Lock()
	buf := &queueNode{
		data: data,
		next: nil,
	}
	if q.len == 0 {
		q.front = buf
		q.rear = buf

	} else {
		q.rear.next = buf
		q.rear = q.rear.next
	}
	q.len++
	q.Signal()
	q.L.Unlock()
}

//Dequeue 出队，如果队列空会阻塞
func (q *Queue) Dequeue() (data interface{}) {
	q.L.Lock()
	if q.len == 0 {
		/* return errors.New("failed to dequeue, queue is empty") */
		q.Wait()
	}

	data = q.front.data
	q.front = q.front.next

	// 当只有一个元素时，出列后front和rear都等于nil
	// 这时要将rear置为nil，不然rear还会指向第一个元素的位置
	// 比如唯一的元素原本为2，不做这步tail还会指向2
	if q.len == 1 {
		q.rear = nil
	}
	q.len--
	q.L.Unlock()
	return
}
