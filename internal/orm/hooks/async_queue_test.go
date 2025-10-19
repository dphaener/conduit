package hooks

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAsyncQueue_StartStop(t *testing.T) {
	queue := NewAsyncQueue(2)
	queue.Start()

	// Should be able to enqueue tasks
	executed := make(chan bool, 1)
	task := AsyncTask{
		Name: "test-task",
		Fn: func(ctx context.Context) error {
			executed <- true
			return nil
		},
	}

	err := queue.Enqueue(task)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	select {
	case <-executed:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("Task did not execute within timeout")
	}

	queue.Shutdown()
}

func TestAsyncQueue_MultipleWorkers(t *testing.T) {
	workerCount := 4
	queue := NewAsyncQueue(workerCount)
	queue.Start()
	defer queue.Shutdown()

	taskCount := 20
	var executed atomic.Int32
	var wg sync.WaitGroup

	wg.Add(taskCount)

	for i := 0; i < taskCount; i++ {
		task := AsyncTask{
			Name: "concurrent-task",
			Fn: func(ctx context.Context) error {
				executed.Add(1)
				time.Sleep(10 * time.Millisecond) // Simulate work
				wg.Done()
				return nil
			},
		}

		err := queue.Enqueue(task)
		if err != nil {
			t.Fatalf("Enqueue failed: %v", err)
		}
	}

	// Wait for all tasks to complete
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		if int(executed.Load()) != taskCount {
			t.Errorf("Expected %d tasks to execute, got %d", taskCount, executed.Load())
		}
	case <-time.After(5 * time.Second):
		t.Errorf("Tasks did not complete within timeout, executed: %d/%d", executed.Load(), taskCount)
	}
}

func TestAsyncQueue_TaskError(t *testing.T) {
	queue := NewAsyncQueue(2)
	queue.Start()
	defer queue.Shutdown()

	executed := make(chan bool, 1)
	expectedErr := errors.New("task failed")

	task := AsyncTask{
		Name: "failing-task",
		Fn: func(ctx context.Context) error {
			executed <- true
			return expectedErr
		},
	}

	err := queue.Enqueue(task)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	// Task should execute even if it returns an error
	select {
	case <-executed:
		// Success - error is logged but doesn't affect queue
	case <-time.After(2 * time.Second):
		t.Error("Task did not execute within timeout")
	}
}

func TestAsyncQueue_TaskPanic(t *testing.T) {
	queue := NewAsyncQueue(2)
	queue.Start()
	defer queue.Shutdown()

	panicTask := AsyncTask{
		Name: "panic-task",
		Fn: func(ctx context.Context) error {
			panic("intentional panic")
		},
	}

	err := queue.Enqueue(panicTask)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	// Give task time to panic and recover
	time.Sleep(100 * time.Millisecond)

	// Queue should still work after panic
	executed := make(chan bool, 1)
	normalTask := AsyncTask{
		Name: "normal-task",
		Fn: func(ctx context.Context) error {
			executed <- true
			return nil
		},
	}

	err = queue.Enqueue(normalTask)
	if err != nil {
		t.Fatalf("Enqueue after panic failed: %v", err)
	}

	select {
	case <-executed:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("Queue failed to recover from panic")
	}
}

func TestAsyncQueue_EnqueueBeforeStart(t *testing.T) {
	queue := NewAsyncQueue(2)

	task := AsyncTask{
		Name: "test-task",
		Fn: func(ctx context.Context) error {
			return nil
		},
	}

	err := queue.Enqueue(task)
	if err == nil {
		t.Error("Expected error when enqueueing before start")
	}

	queue.Start()
	defer queue.Shutdown()

	err = queue.Enqueue(task)
	if err != nil {
		t.Errorf("Enqueue after start should work: %v", err)
	}
}

func TestAsyncQueue_ShutdownGraceful(t *testing.T) {
	queue := NewAsyncQueue(2)
	queue.Start()

	taskCount := 10
	var executed atomic.Int32

	for i := 0; i < taskCount; i++ {
		task := AsyncTask{
			Name: "shutdown-task",
			Fn: func(ctx context.Context) error {
				executed.Add(1)
				time.Sleep(50 * time.Millisecond)
				return nil
			},
		}

		err := queue.Enqueue(task)
		if err != nil {
			t.Fatalf("Enqueue failed: %v", err)
		}
	}

	// Shutdown should wait for all tasks
	queue.Shutdown()

	if int(executed.Load()) != taskCount {
		t.Errorf("Not all tasks completed before shutdown: %d/%d", executed.Load(), taskCount)
	}

	// Should not be able to enqueue after shutdown
	task := AsyncTask{
		Name: "post-shutdown-task",
		Fn: func(ctx context.Context) error {
			return nil
		},
	}

	err := queue.Enqueue(task)
	if err == nil {
		t.Error("Should not be able to enqueue after shutdown")
	}
}

func TestAsyncQueue_Stop(t *testing.T) {
	queue := NewAsyncQueue(2)
	queue.Start()

	// Enqueue a long-running task
	started := make(chan bool, 1)
	task := AsyncTask{
		Name: "long-task",
		Fn: func(ctx context.Context) error {
			started <- true
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Second):
				return nil
			}
		},
	}

	err := queue.Enqueue(task)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	// Wait for task to start
	<-started

	// Stop should cancel context and return quickly
	stopStart := time.Now()
	queue.Stop()
	stopDuration := time.Since(stopStart)

	if stopDuration > 1*time.Second {
		t.Errorf("Stop took too long: %v", stopDuration)
	}
}

func TestAsyncQueue_BufferFull(t *testing.T) {
	queue := NewAsyncQueue(1)
	queue.Start()
	defer queue.Shutdown()

	// Fill the buffer (100 tasks)
	blockTask := make(chan bool)
	for i := 0; i < 101; i++ {
		task := AsyncTask{
			Name: "blocking-task",
			Fn: func(ctx context.Context) error {
				<-blockTask // Block until released
				return nil
			},
		}
		queue.Enqueue(task)
	}

	// Try to enqueue one more - should succeed quickly or block briefly
	start := time.Now()
	normalTask := AsyncTask{
		Name: "normal-task",
		Fn: func(ctx context.Context) error {
			return nil
		},
	}

	done := make(chan bool)
	go func() {
		queue.Enqueue(normalTask)
		done <- true
	}()

	// Release one blocking task to make room
	time.Sleep(50 * time.Millisecond)
	blockTask <- true

	select {
	case <-done:
		duration := time.Since(start)
		if duration > 2*time.Second {
			t.Logf("Enqueue took %v (expected delay due to buffer)", duration)
		}
	case <-time.After(3 * time.Second):
		t.Error("Enqueue blocked for too long")
	}

	// Unblock remaining tasks
	close(blockTask)
}

func TestAsyncQueue_DefaultWorkerCount(t *testing.T) {
	queue := NewAsyncQueue(0) // Invalid count
	queue.Start()
	defer queue.Shutdown()

	// Should still work with default worker count
	executed := make(chan bool, 1)
	task := AsyncTask{
		Name: "test-task",
		Fn: func(ctx context.Context) error {
			executed <- true
			return nil
		},
	}

	err := queue.Enqueue(task)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	select {
	case <-executed:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("Task did not execute within timeout")
	}
}

func TestAsyncQueue_MultipleStartCalls(t *testing.T) {
	queue := NewAsyncQueue(2)
	queue.Start()
	queue.Start() // Should be safe to call multiple times
	queue.Start()

	executed := make(chan bool, 1)
	task := AsyncTask{
		Name: "test-task",
		Fn: func(ctx context.Context) error {
			executed <- true
			return nil
		},
	}

	err := queue.Enqueue(task)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	select {
	case <-executed:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("Task did not execute")
	}

	queue.Shutdown()
}
