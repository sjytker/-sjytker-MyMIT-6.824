package main

import (
	"fmt"
	"sync"
)

type classA struct {
	mu sync.Mutex
	c chan int
}

func main(){

	//obj := classA{
	//	mu: sync.Mutex{},
	//	c:  make(chan int),
	//}
	//go func() {
	//	for i := 0; i < 5; i ++ {
	//		obj.c <- i
	//	}
	//}()
	//
	//var n int
	//fmt.Printf("requesting lock\n")
	//for true {
	//	obj.mu.Lock()
	//	fmt.Printf("acquired lock\n")
	//	select {
	//	case n =<- obj.c:
	//		fmt.Printf("got value %v\n", n)
	//	}
	//	obj.mu.Unlock()
	//	fmt.Printf("released lock\n")
	//}

	//lst := make([]int, 0)
	//M := 10000
	//wg := sync.WaitGroup{}
	//wg.Add(4)
	//go func() {
	//	for i := 0; i < M; i ++ {
	//		lst = append(lst, 1)
	//	}
	//	wg.Done()
	//}()
	//
	//go func() {
	//	for i := 0; i < M; i ++ {
	//		lst = append(lst, 2)
	//	}
	//	wg.Done()
	//}()
	//
	//go func() {
	//	for i := 0; i < M; i ++ {
	//		lst = append(lst, 3)
	//	}
	//	wg.Done()
	//}()
	//
	//go func() {
	//	for i := 0; i < M; i ++ {
	//		lst = append(lst, 4)
	//	}
	//	wg.Done()
	//}()
	//
	//wg.Wait()
	//fmt.Println(len(lst))


	a := make(map[int]int)
	a[5] = 10
	a[2] = 1
	a[3] = 2323
	fmt.Println(len(a))
}
