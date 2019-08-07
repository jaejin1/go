package jobqueue

import (
	_ "expvar"
	"fmt"
	_ "net/http/pprof"
)

type Jobfunc func() (string)
// Job holds the attributes needed to perform unit of work.

type Job struct {
	Name  string
	Do Jobfunc
}

// NewWorker creates takes a numeric id and a channel w/ worker pool.
func NewWorker(id int, workerPool chan chan *Job) Worker {
	return Worker{
		id:         id,
		jobQueue:   make(chan *Job),
		workerPool: workerPool,
		quitChan:   make(chan bool),
	}
}

type Worker struct {
	id         int
	jobQueue   chan *Job
	workerPool chan chan *Job
	quitChan   chan bool
}

func (w Worker) start() {
	go func() {
		for {
			// Add my jobQueue to the worker pool.
			w.workerPool <- w.jobQueue

			select {
			case job := <-w.jobQueue:
				// Dispatcher has added a job to my jobQueue.
				fmt.Printf("worker%d: started %s\n", w.id, job.Name)
				fmt.Println("====")
				result := job.Do()
				fmt.Println(result)
				fmt.Println("====")
				fmt.Printf("worker%d: completed %s!\n", w.id, job.Name)
			case <-w.quitChan:
				// We have been asked to stop.
				fmt.Printf("worker%d stopping\n", w.id)
				return
			}
		}
	}()
}

func (w Worker) stop() {
	go func() {
		w.quitChan <- true
	}()
}
