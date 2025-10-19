package hooks

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// AsyncTask represents a task to be executed asynchronously
type AsyncTask struct {
	Name string
	Fn   func(ctx context.Context) error
}

// AsyncQueue manages asynchronous task execution with a worker pool
type AsyncQueue struct {
	tasks       chan AsyncTask
	workerCount int
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	started     bool
	shutdown    bool
	mu          sync.Mutex
}

// NewAsyncQueue creates a new async task queue with the specified worker count
func NewAsyncQueue(workerCount int) *AsyncQueue {
	if workerCount <= 0 {
		workerCount = 4 // Default worker count
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &AsyncQueue{
		tasks:       make(chan AsyncTask, 100), // Buffer 100 tasks
		workerCount: workerCount,
		ctx:         ctx,
		cancel:      cancel,
		started:     false,
	}
}

// Start starts the worker pool
func (q *AsyncQueue) Start() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.started {
		return
	}

	for i := 0; i < q.workerCount; i++ {
		q.wg.Add(1)
		go q.worker(i)
	}

	q.started = true
}

// worker is a worker goroutine that processes tasks
func (q *AsyncQueue) worker(id int) {
	defer q.wg.Done()

	for {
		select {
		case <-q.ctx.Done():
			return
		case task, ok := <-q.tasks:
			if !ok {
				return
			}

			// Execute task with panic recovery
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("AsyncQueue worker %d: panic in task %s: %v", id, task.Name, r)
					}
				}()

				if err := task.Fn(q.ctx); err != nil {
					log.Printf("AsyncQueue worker %d: task %s failed: %v", id, task.Name, err)
				}
			}()
		}
	}
}

// Enqueue adds a task to the queue
func (q *AsyncQueue) Enqueue(task AsyncTask) error {
	q.mu.Lock()
	if !q.started {
		q.mu.Unlock()
		return fmt.Errorf("queue not started")
	}
	if q.shutdown {
		q.mu.Unlock()
		return fmt.Errorf("queue shutdown")
	}
	q.mu.Unlock()

	select {
	case q.tasks <- task:
		return nil
	case <-q.ctx.Done():
		return fmt.Errorf("queue closed")
	}
}

// Shutdown gracefully shuts down the queue
// It stops accepting new tasks and waits for existing tasks to complete
func (q *AsyncQueue) Shutdown() {
	q.mu.Lock()
	if !q.started || q.shutdown {
		q.mu.Unlock()
		return
	}
	q.shutdown = true
	q.mu.Unlock()

	close(q.tasks)
	q.wg.Wait()
}

// Stop immediately stops the queue without waiting for tasks to complete
func (q *AsyncQueue) Stop() {
	q.cancel()
	q.wg.Wait()
}
