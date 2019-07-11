package main

import "fmt"

func main() {
	numbers := make(chan int, 5)
	numbers2 := make(chan int, 5)

	counter := 10

	go read(numbers, "channel1")

	for i := 0; i < counter; i++ {
		select {
		case numbers <- i:
		default:
			fmt.Println("channel1 Not enough space for ", i)
		}
	}

	go read(numbers2, "channel2")

	for i := 0; i < counter; i++ {
		select {
		case numbers2 <- i:
		default:
			fmt.Println("channel2 Not enough space for ", i)
		}
	}

	for i := 0; i < counter+5; i++ {
		select {
		case num := <-numbers:
			fmt.Println(num)
		default:
			fmt.Println("channel1 Nothing more to be done!")
			break
		}
	}

}
func Job() {
	fmt.Println("asdf")
}

func read(in <-chan int, channel string) {
	//var str int
	//str = 0
	for x2 := range in {
		fmt.Printf("%s 으아악 %d\n", channel, x2)
	}

	fmt.Printf("End %s\n", "sadfasdf")
}
