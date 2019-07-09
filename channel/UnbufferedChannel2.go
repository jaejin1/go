package main

import (
	"fmt"
	"sync"
	"time"
)

var wg sync.WaitGroup

func main() {
	baton := make(chan int)

	wg.Add(1)

	go Runner(baton)

	baton <- 1

	wg.Wait()
}

func Runner(baton chan int) {
	var newRunner int

	runner := <-baton

	fmt.Printf("%d번째 주자가 바통을 받아 달리기 시작했습니다.\n", runner)

	if runner != 4 {
		newRunner = runner + 1
		fmt.Printf("%d번째 주자가 대기합니다.\n", newRunner)
		go Runner(baton)
	}

	time.Sleep(100 * time.Millisecond)

	if runner == 4 {
		fmt.Printf("%d번째 주자가 도착했습니다. 끝났습니다. \n", runner)
		wg.Done()
		return
	}

	fmt.Printf("%d 번쨰 주자가 %d번째 에게 바통을 넘겼습니다.\n", runner, newRunner)

	baton <- newRunner
}
