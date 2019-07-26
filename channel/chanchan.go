package main

import (
	"fmt"
	"time"
)

func main() {
	User := make(chan chan string)
	User2 := make(chan chan string)

	go makenode(User)
	go input(User, "user1")

	go makenode(User2)
	go input(User2, "user2")
	go input(User2, "user2")

	time.Sleep(time.Second)
}

func makenode(user chan chan string) {
	makechannel := make(chan string)
	for {
		select {
		case user <- makechannel:
		case channelname := <-makechannel:
			fmt.Printf("channelname: %v\n", channelname)
		}
	}

}

func input(user chan chan string, username string) {
	responsechan := <-user

	responsechan <- "data " + username
}
