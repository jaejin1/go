package main

import "fmt"

type Jobfunc func() error //jobfunc

type Function struct { // Job
	Do Jobfunc
}

type jaejin struct { // queue
	job chan *Function
}

func main() {
	jaejin := &jaejin{
		job: make(chan *Function, 1),
	}

	jaejin.Enqueue(&Function{
		Do: func() error {
			fmt.Println("test")
			return nil
		},
	})

	test := <-jaejin.job

	test.Do()
}

func test() string {
	fmt.Println("test")
	return "ang"
}

func (q *jaejin) Enqueue(j *Function){
	q.job <- j
}