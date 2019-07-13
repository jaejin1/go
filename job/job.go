package main

import (
	"fmt"
	"github.com/go-kit/kit/log"
	"strconv"
	"sync"
	"context"
	"time"
)

const (
	// This is set to be in sympathy with the request / RPC timeout (i.e., empirically)
	defaultHandlerTimeout = 10 * time.Second
	// A job can take an arbitrary amount of time but we want to have
	// a (generous) threshold for considering a job stuck and
	// abandoning it
	defaultJobTimeout = 60 * time.Second
)

type ID string
type JobFunc func(log.Logger) error

type Job struct {
	ID ID
	Do JobFunc
}

type StatusString string

type cacheEntry struct {
	ID     ID
	Status Status
}
type StatusCache struct {
	// Size is the number of statuses to store. When full, jobs are evicted in FIFO ordering.
	// oldest ones will be evicted to make room.
	Size int

	// Store cache entries in an array to make fifo eviction easier. Efficiency
	// doesn't matter because the cache is small and computers are fast.
	cache []cacheEntry
	sync.RWMutex
}

const (
	StatusQueued    StatusString = "queued"
	StatusRunning   StatusString = "running"
	StatusFailed    StatusString = "failed"
	StatusSucceeded StatusString = "succeeded"
)

type Result struct {
	Revision string `json:"revision,omitempty"`
}

type Status struct {
	Result       Result
	Err          string
	StatusString StatusString
}

func (s Status) Error() string {
	return s.Err
}

type Queue struct {
	ready       chan *Job
	incoming    chan *Job
	waiting     []*Job
	waitingLock sync.Mutex
	sync        chan struct{}
}

func NewQueue(stop <-chan struct{}, wg *sync.WaitGroup) *Queue {
	q := &Queue{
		ready:    make(chan *Job),
		incoming: make(chan *Job),
		waiting:  make([]*Job, 0),
		sync:     make(chan struct{}),
	}

	wg.Add(1)
	go q.loop(stop, wg)
	return q
}

func (q *Queue) test(stop <-chan struct{}, wg *sync.WaitGroup) {
	wg.Add(1)
	var testlog log.Logger
	go func() {
		defer wg.Done()
		for {
			select {
			case j := <-q.Ready():
				q.waitingLock.Lock()
				fmt.Println("Dequeued from empty queue: ========================================")
				fmt.Println(j.ID)
				j.Do(testlog)
				q.waitingLock.Unlock()
			}
		}
	}()
}

func (q *Queue) Len() int {
	q.waitingLock.Lock()
	defer q.waitingLock.Unlock()
	return len(q.waiting)
}

func (q *Queue) Enqueue(j *Job) {
	q.incoming <- j
}
func (q *Queue) Ready() <-chan *Job {
	return q.ready
}

func (q *Queue) ForEach(fn func(int, *Job) bool) {
	q.waitingLock.Lock()
	jobs := q.waiting
	q.waitingLock.Unlock()
	for i, job := range jobs {
		if !fn(i, job) {
			return
		}
	}
}

func (q *Queue) Sync() {
	q.sync <- struct{}{}
}

func (q *Queue) loop(stop <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		var out chan *Job = nil
		if len(q.waiting) > 0 {
			out = q.ready
		}

		select {
		case <-stop:
			return
		case <-q.sync:
			continue
		case in := <-q.incoming:
			q.waitingLock.Lock()
			q.waiting = append(q.waiting, in)
			q.waitingLock.Unlock()
		case out <- q.nextOrNil():
			q.waitingLock.Lock()
			q.waiting = q.waiting[1:]
			q.waitingLock.Unlock()
		}
	}
}

func (q *Queue) nextOrNil() *Job {
	q.waitingLock.Lock()
	defer q.waitingLock.Unlock()
	if len(q.waiting) > 0 {
		return q.waiting[0]
	}
	return nil
}

func main() {
	shutdown := make(chan struct{})

	wg := &sync.WaitGroup{}
	defer close(shutdown)

	q := NewQueue(shutdown, wg)
	q.test(shutdown, wg)

	JobStatusCache := &StatusCache{
		Size: 1000,
	}

	if q.Len() != 0 {
		fmt.Println("0")
	}

	var vari ID = "job 1"
	q.Enqueue(&Job{
		ID: "job 1",
		Do: func(logger log.Logger) error {
			JobStatusCache.ExecuteJob(vari, makejob(), logger)
			return nil

		},
	})
	q.Sync()


	for i := 3; i < 100; i++ {
		q.Enqueue(&Job{
			ID: ID("job " + strconv.Itoa(i)),
			Do: func(logger log.Logger) error {
				JobStatusCache.ExecuteJob(ID("job "+strconv.Itoa(i)), makejob(), logger)
				return nil

			},
		})
		q.Sync()
	}

	q.Enqueue(&Job{
		ID: "job 2",
		Do: func(logger log.Logger) error {
			testfunc()
			return nil

		},
	})
	q.Sync()


	if len(q.waiting) != 0 {
		fmt.Println("errororororor")
		fmt.Println(len(q.waiting))

		fmt.Println("zzz")
	}
	if q.Len() != 0 {
		fmt.Printf("Queue has length %d (!= 0) after dequeuing only item (and sync)\n", q.Len())
	}

}

func testfunc() {
	fmt.Println("ang")
}

type jobFunc func(ctx context.Context, jobID ID, logger log.Logger) (Result, error)

func makejob() jobFunc {
	return func(ctx context.Context, id ID, logger log.Logger) (Result, error) {
		result := Result{
			Revision: string(id),
		}
		return result, nil
	}
}

func (c *StatusCache) ExecuteJob(id ID, do jobFunc, logger log.Logger) (Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultJobTimeout)
	defer cancel()

	c.SetStatus(id, Status{StatusString: StatusRunning})
	result, err := do(ctx, id, logger)
	fmt.Println(result)

	if err != nil {
		c.SetStatus(id, Status{StatusString: StatusFailed, Err: err.Error(), Result: result})
		return result, err
	}
	c.SetStatus(id, Status{StatusString: StatusSucceeded, Result: result})

	return result, nil
}

func (c *StatusCache) SetStatus(id ID, status Status) {
	if c.Size <= 0 {
		fmt.Println("in stetstaus error")
		return
	}
	c.Lock()
	defer c.Unlock()
	if i := c.statusIndex(id); i >= 0 {

		// already exists, update
		c.cache[i].Status = status
	}

	// Evict, if we need to. Eviction is done first, so that append can only copy
	// the things we care about keeping. Micro-optimize to the max.
	if c.Size <= len(c.cache) {
		c.cache = c.cache[len(c.cache)-(c.Size-1):]
	}
	c.cache = append(c.cache, cacheEntry{
		ID:     id,
		Status: status,
	})
}

func (c *StatusCache) statusIndex(id ID) int {
	// entries are sorted by arrival time, not id, so we can't use binary search.
	for i := range c.cache {
		if c.cache[i].ID == id {
			return i
		}
	}
	return -1
}


func (c *StatusCache) Status(id ID) (Status, bool) {
	c.RLock()
	defer c.RUnlock()
	i := c.statusIndex(id)
	if i < 0 {
		return Status{}, false
	}
	return c.cache[i].Status, true
}


