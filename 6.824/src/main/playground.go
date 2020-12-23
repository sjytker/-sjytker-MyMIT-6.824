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


<<<<<<< HEAD
	a := make(map[int]int)
	a[5] = 10
	a[2] = 1
	a[3] = 2323
	fmt.Println(len(a))
=======
	//vis := make(map[int]map[int]int)
	//
	//if _, ok := vis[0][1]; ok {
	//	fmt.Printf("got vis[0][1] : %v\n", vis[0][1])
	//}
	//
	//if _, ok := vis[0][1]; !ok {
	//	vis[0] = make(map[int]int)
	//	vis[0][1] = 100
	//}
	//if _, ok := vis[0][1]; ok {
	//	fmt.Printf("got vis[0][1] : %v\n", vis[0][1])
	//}

	v := make(map[int]string)
	v[0] = "123"
	fmt.Printf("getting v[0] : %v\n", v[0])
	if v, ok := v[1]; ok {
		fmt.Printf("getting v[1] : %v\n", v[1])
	}
>>>>>>> 2694adff741d395474232a36415f966716f74bd9
}
