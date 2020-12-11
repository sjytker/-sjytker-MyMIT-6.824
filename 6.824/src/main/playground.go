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


	vis := make(map[int]map[int]int)

	if _, ok := vis[0][1]; ok {
		fmt.Printf("got vis[0][1] : %v\n", vis[0][1])
	}

	if _, ok := vis[0][1]; !ok {
		vis[0] = make(map[int]int)
		vis[0][1] = 100
	}
	if _, ok := vis[0][1]; ok {
		fmt.Printf("got vis[0][1] : %v\n", vis[0][1])
	}
}
