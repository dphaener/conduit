package jobs_test

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/conduit-lang/conduit/internal/web/jobs"
)

// Example demonstrates basic job enqueueing and processing
func Example() {
	// Setup database connection
	db, _ := sql.Open("postgres", "postgres://localhost/conduit")
	defer db.Close()

	// Create queue and worker pool
	queue := jobs.NewQueue(db)
	pool := jobs.NewWorkerPool(queue, "default", 5)

	// Register a job handler
	pool.RegisterHandler("email.send", func(ctx context.Context, payload map[string]interface{}) error {
		recipient := payload["to"].(string)
		template := payload["template"].(string)
		fmt.Printf("Sending %s email to %s\n", template, recipient)
		return nil
	})

	// Start workers
	ctx := context.Background()
	pool.Start(ctx)
	defer pool.Stop()

	// Enqueue a job
	job := jobs.NewJob("default", "email.send", map[string]interface{}{
		"to":       "user@example.com",
		"template": "welcome",
	})

	queue.Enqueue(ctx, job)
	fmt.Printf("Enqueued job %s\n", job.ID)
}

// ExampleScheduler demonstrates scheduled jobs
func ExampleCronScheduler() {
	db, _ := sql.Open("postgres", "postgres://localhost/conduit")
	defer db.Close()

	queue := jobs.NewQueue(db)
	scheduler := jobs.NewCronScheduler(queue)

	// Schedule a job to run every hour
	schedule := jobs.ScheduleEveryHours(1, "default", "cleanup.old_sessions", map[string]interface{}{
		"max_age_hours": 24,
	})

	scheduler.AddSchedule(schedule)

	// Start the scheduler
	ctx := context.Background()
	scheduler.Start(ctx)
	defer scheduler.Stop()

	fmt.Printf("Scheduled job %s to run every hour\n", schedule.ID)
}

// ExampleAsyncExecutor demonstrates integration with lifecycle hooks
func ExampleAsyncExecutor() {
	db, _ := sql.Open("postgres", "postgres://localhost/conduit")
	defer db.Close()

	queue := jobs.NewQueue(db)
	executor := jobs.NewAsyncExecutor(queue)

	// This would be called from a lifecycle hook
	ctx := context.Background()
	err := executor.Execute(ctx, "default", "email.send", map[string]interface{}{
		"to":       "user@example.com",
		"template": "welcome",
	})

	if err != nil {
		log.Printf("Failed to enqueue async job: %v", err)
	}
}

// ExamplePriority demonstrates job prioritization
func ExampleQueue_EnqueueWithPriority() {
	db, _ := sql.Open("postgres", "postgres://localhost/conduit")
	defer db.Close()

	queue := jobs.NewQueue(db)
	ctx := context.Background()

	// Enqueue urgent job (will be processed first)
	urgentJob := jobs.NewJob("default", "fraud.check", map[string]interface{}{
		"transaction_id": "txn_123",
	})
	queue.EnqueueWithPriority(ctx, urgentJob, jobs.PriorityUrgent)

	// Enqueue normal job
	normalJob := jobs.NewJob("default", "report.generate", map[string]interface{}{
		"report_type": "monthly",
	})
	queue.EnqueueWithPriority(ctx, normalJob, jobs.PriorityNormal)

	fmt.Println("Enqueued jobs with different priorities")
}

// ExampleSchedule demonstrates scheduled job execution
func ExampleQueue_Schedule() {
	db, _ := sql.Open("postgres", "postgres://localhost/conduit")
	defer db.Close()

	queue := jobs.NewQueue(db)
	ctx := context.Background()

	// Schedule a job to run in the future
	job := jobs.NewJob("default", "reminder.send", map[string]interface{}{
		"user_id": "user_123",
		"message": "Your trial expires in 3 days",
	})

	runAt := time.Now().Add(72 * time.Hour)
	queue.Schedule(ctx, job, runAt)

	fmt.Printf("Scheduled job to run at %v\n", runAt)
}

// ExampleMetrics demonstrates job metrics tracking
func ExampleMetrics() {
	db, _ := sql.Open("postgres", "postgres://localhost/conduit")
	defer db.Close()

	queue := jobs.NewQueue(db)
	pool := jobs.NewWorkerPool(queue, "default", 5)

	// Register handler
	pool.RegisterHandler("test.job", func(ctx context.Context, payload map[string]interface{}) error {
		return nil
	})

	// Process some jobs
	ctx := context.Background()
	pool.Start(ctx)
	defer pool.Stop()

	// Get metrics
	time.Sleep(1 * time.Second)
	metrics := pool.GetMetrics()
	stats := metrics.GetStats("test.job")

	fmt.Printf("Job Type: %s\n", stats.JobType)
	fmt.Printf("Processed: %d\n", stats.Processed)
	fmt.Printf("Success Rate: %.2f%%\n", stats.SuccessRate())
	fmt.Printf("Avg Duration: %v\n", stats.AvgDuration)
}

// ExampleRetry demonstrates retry logic with exponential backoff
func ExampleQueue_Retry() {
	db, _ := sql.Open("postgres", "postgres://localhost/conduit")
	defer db.Close()

	queue := jobs.NewQueue(db)
	pool := jobs.NewWorkerPool(queue, "default", 1)

	// Register handler that may fail
	attempts := 0
	pool.RegisterHandler("flaky.job", func(ctx context.Context, payload map[string]interface{}) error {
		attempts++
		if attempts < 3 {
			return fmt.Errorf("temporary failure")
		}
		return nil // Success on third attempt
	})

	ctx := context.Background()
	pool.Start(ctx)
	defer pool.Stop()

	// Enqueue job that will be retried
	job := jobs.NewJob("default", "flaky.job", map[string]interface{}{})
	job.MaxAttempts = 5
	queue.Enqueue(ctx, job)

	fmt.Println("Job will retry with exponential backoff: 1min, 2min, 4min, 8min, ...")
}

// ExampleCancel demonstrates job cancellation
func ExampleQueue_Cancel() {
	db, _ := sql.Open("postgres", "postgres://localhost/conduit")
	defer db.Close()

	queue := jobs.NewQueue(db)
	ctx := context.Background()

	// Enqueue a job
	job := jobs.NewJob("default", "report.generate", map[string]interface{}{
		"report_id": "report_123",
	})
	queue.Enqueue(ctx, job)

	// Cancel it before it processes
	err := queue.Cancel(ctx, job.ID)
	if err != nil {
		log.Printf("Failed to cancel job: %v", err)
	}

	fmt.Printf("Cancelled job %s\n", job.ID)
}

// ExampleQueueStats demonstrates queue statistics
func ExampleQueue_GetQueueStats() {
	db, _ := sql.Open("postgres", "postgres://localhost/conduit")
	defer db.Close()

	queue := jobs.NewQueue(db)
	ctx := context.Background()

	stats, err := queue.GetQueueStats(ctx, "default")
	if err != nil {
		log.Printf("Failed to get stats: %v", err)
		return
	}

	fmt.Printf("Queue: %s\n", stats.Queue)
	fmt.Printf("Pending: %d\n", stats.Pending)
	fmt.Printf("Running: %d\n", stats.Running)
	fmt.Printf("Completed: %d\n", stats.Completed)
	fmt.Printf("Failed: %d\n", stats.Failed)
}

// ExamplePurgeCompleted demonstrates cleaning up old jobs
func ExampleQueue_PurgeCompleted() {
	db, _ := sql.Open("postgres", "postgres://localhost/conduit")
	defer db.Close()

	queue := jobs.NewQueue(db)
	ctx := context.Background()

	// Delete completed jobs older than 7 days
	count, err := queue.PurgeCompleted(ctx, 7*24*time.Hour)
	if err != nil {
		log.Printf("Failed to purge: %v", err)
		return
	}

	fmt.Printf("Purged %d old jobs\n", count)
}

// ExampleMultipleQueues demonstrates using multiple queues
func ExampleMultipleQueues() {
	db, _ := sql.Open("postgres", "postgres://localhost/conduit")
	defer db.Close()

	queue := jobs.NewQueue(db)

	// Create separate worker pools for different queues
	defaultPool := jobs.NewWorkerPool(queue, "default", 5)
	highPriorityPool := jobs.NewWorkerPool(queue, "high-priority", 10)
	lowPriorityPool := jobs.NewWorkerPool(queue, "low-priority", 2)

	// Register handlers for each pool
	defaultPool.RegisterHandler("email.send", func(ctx context.Context, payload map[string]interface{}) error {
		return nil
	})

	highPriorityPool.RegisterHandler("fraud.check", func(ctx context.Context, payload map[string]interface{}) error {
		return nil
	})

	lowPriorityPool.RegisterHandler("report.generate", func(ctx context.Context, payload map[string]interface{}) error {
		return nil
	})

	// Start all pools
	ctx := context.Background()
	defaultPool.Start(ctx)
	highPriorityPool.Start(ctx)
	lowPriorityPool.Start(ctx)

	defer func() {
		defaultPool.Stop()
		highPriorityPool.Stop()
		lowPriorityPool.Stop()
	}()

	fmt.Println("Running multiple job queues with different worker counts")
}
