// +build integration

package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: These integration tests require a running PostgreSQL instance
// Run with: go test -tags=integration

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	// Use test database
	connStr := "postgres://postgres:postgres@localhost:5432/conduit_test?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Skipf("Skipping integration test: %v", err)
		return nil, func() {}
	}

	// Test connection
	if err := db.Ping(); err != nil {
		t.Skipf("Skipping integration test: %v", err)
		return nil, func() {}
	}

	// Create jobs table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS jobs (
			id UUID PRIMARY KEY,
			queue VARCHAR(255) NOT NULL,
			type VARCHAR(255) NOT NULL,
			payload JSONB NOT NULL,
			status VARCHAR(50) NOT NULL,
			priority INTEGER NOT NULL DEFAULT 50,
			attempts INTEGER NOT NULL DEFAULT 0,
			max_attempts INTEGER NOT NULL DEFAULT 3,
			error TEXT,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL,
			run_at TIMESTAMP WITH TIME ZONE NOT NULL,
			started_at TIMESTAMP WITH TIME ZONE,
			completed_at TIMESTAMP WITH TIME ZONE,
			locked_by VARCHAR(255),
			locked_at TIMESTAMP WITH TIME ZONE
		)
	`)
	require.NoError(t, err)

	cleanup := func() {
		db.Exec("DROP TABLE IF EXISTS jobs")
		db.Close()
	}

	return db, cleanup
}

func TestIntegrationEnqueueDequeue(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	queue := NewQueue(db)
	ctx := context.Background()

	// Enqueue a job
	job := NewJob("default", "test.job", map[string]interface{}{
		"message": "hello world",
	})

	err := queue.Enqueue(ctx, job)
	require.NoError(t, err)

	// Dequeue the job
	dequeuedJob, err := queue.Dequeue(ctx, "worker-1", "default")
	require.NoError(t, err)
	assert.Equal(t, job.ID, dequeuedJob.ID)
	assert.Equal(t, "hello world", dequeuedJob.Payload["message"])
	assert.Equal(t, JobStatusRunning, dequeuedJob.Status)
	assert.Equal(t, 1, dequeuedJob.Attempts)
}

func TestIntegrationWorkerPool(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	queue := NewQueue(db)
	pool := NewWorkerPool(queue, "default", 2)
	ctx := context.Background()

	// Track processed jobs
	var mu sync.Mutex
	processed := make(map[string]bool)

	// Register handler
	pool.RegisterHandler("test.job", func(ctx context.Context, payload map[string]interface{}) error {
		mu.Lock()
		defer mu.Unlock()
		jobID := payload["id"].(string)
		processed[jobID] = true
		time.Sleep(10 * time.Millisecond) // Simulate work
		return nil
	})

	// Start workers
	pool.Start(ctx)
	defer pool.Stop()

	// Enqueue multiple jobs
	jobCount := 10
	for i := 0; i < jobCount; i++ {
		job := NewJob("default", "test.job", map[string]interface{}{
			"id": fmt.Sprintf("job-%d", i),
		})
		err := queue.Enqueue(ctx, job)
		require.NoError(t, err)
	}

	// Wait for jobs to be processed
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for jobs to complete")
		case <-ticker.C:
			mu.Lock()
			count := len(processed)
			mu.Unlock()
			if count == jobCount {
				// All jobs processed
				return
			}
		}
	}
}

func TestIntegrationRetryLogic(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	queue := NewQueue(db)
	pool := NewWorkerPool(queue, "default", 1)
	ctx := context.Background()

	// Track attempts
	var mu sync.Mutex
	attempts := 0

	// Register handler that fails twice then succeeds
	pool.RegisterHandler("test.job", func(ctx context.Context, payload map[string]interface{}) error {
		mu.Lock()
		attempts++
		currentAttempts := attempts
		mu.Unlock()

		if currentAttempts < 3 {
			return fmt.Errorf("simulated failure %d", currentAttempts)
		}
		return nil
	})

	// Start worker
	pool.Start(ctx)
	defer pool.Stop()

	// Enqueue job
	job := NewJob("default", "test.job", map[string]interface{}{})
	job.MaxAttempts = 3
	err := queue.Enqueue(ctx, job)
	require.NoError(t, err)

	// Wait for job to complete
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for job to complete")
		case <-ticker.C:
			retrievedJob, err := queue.GetJob(ctx, job.ID)
			require.NoError(t, err)

			if retrievedJob.Status == JobStatusCompleted {
				mu.Lock()
				finalAttempts := attempts
				mu.Unlock()
				assert.Equal(t, 3, finalAttempts)
				return
			}
		}
	}
}

func TestIntegrationScheduledJobs(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	queue := NewQueue(db)
	scheduler := NewCronScheduler(queue)
	ctx := context.Background()

	// Track executions
	var mu sync.Mutex
	executions := 0

	// Add schedule that runs every second
	schedule := &Schedule{
		Queue:    "default",
		Type:     "test.scheduled",
		Interval: 1 * time.Second,
		Payload:  map[string]interface{}{"task": "cleanup"},
	}

	err := scheduler.AddSchedule(schedule)
	require.NoError(t, err)

	// Start scheduler
	scheduler.Start(ctx)
	defer scheduler.Stop()

	// Wait for a few executions
	time.Sleep(3 * time.Second)

	// Check that jobs were enqueued
	jobs, err := queue.ListJobs(ctx, "default", JobStatusPending, 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(jobs), 2) // Should have at least 2 jobs
}

func TestIntegrationPriorityOrdering(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	queue := NewQueue(db)
	ctx := context.Background()

	// Enqueue jobs with different priorities
	lowJob := NewJob("default", "test.job", map[string]interface{}{"priority": "low"})
	normalJob := NewJob("default", "test.job", map[string]interface{}{"priority": "normal"})
	highJob := NewJob("default", "test.job", map[string]interface{}{"priority": "high"})

	err := queue.EnqueueWithPriority(ctx, lowJob, PriorityLow)
	require.NoError(t, err)

	err = queue.EnqueueWithPriority(ctx, normalJob, PriorityNormal)
	require.NoError(t, err)

	err = queue.EnqueueWithPriority(ctx, highJob, PriorityHigh)
	require.NoError(t, err)

	// Dequeue should return high priority first
	job1, err := queue.Dequeue(ctx, "worker-1", "default")
	require.NoError(t, err)
	assert.Equal(t, "high", job1.Payload["priority"])

	job2, err := queue.Dequeue(ctx, "worker-2", "default")
	require.NoError(t, err)
	assert.Equal(t, "normal", job2.Payload["priority"])

	job3, err := queue.Dequeue(ctx, "worker-3", "default")
	require.NoError(t, err)
	assert.Equal(t, "low", job3.Payload["priority"])
}

func TestIntegrationQueueStats(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	queue := NewQueue(db)
	ctx := context.Background()

	// Create jobs with different statuses
	pendingJob := NewJob("default", "test.job", nil)
	err := queue.Enqueue(ctx, pendingJob)
	require.NoError(t, err)

	completedJob := NewJob("default", "test.job", nil)
	err = queue.Enqueue(ctx, completedJob)
	require.NoError(t, err)
	_, err = queue.Dequeue(ctx, "worker-1", "default")
	require.NoError(t, err)
	err = queue.Complete(ctx, completedJob.ID)
	require.NoError(t, err)

	failedJob := NewJob("default", "test.job", nil)
	err = queue.Enqueue(ctx, failedJob)
	require.NoError(t, err)
	_, err = queue.Dequeue(ctx, "worker-2", "default")
	require.NoError(t, err)
	err = queue.Fail(ctx, failedJob.ID, "test error")
	require.NoError(t, err)

	// Get stats
	stats, err := queue.GetQueueStats(ctx, "default")
	require.NoError(t, err)

	assert.Equal(t, 1, stats.Pending)
	assert.Equal(t, 1, stats.Completed)
	assert.Equal(t, 1, stats.Failed)
}

func TestIntegrationConcurrentWorkers(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	queue := NewQueue(db)
	pool := NewWorkerPool(queue, "default", 10) // 10 concurrent workers
	ctx := context.Background()

	// Track processed jobs
	var mu sync.Mutex
	processed := 0

	// Register handler
	pool.RegisterHandler("test.job", func(ctx context.Context, payload map[string]interface{}) error {
		time.Sleep(50 * time.Millisecond) // Simulate work
		mu.Lock()
		processed++
		mu.Unlock()
		return nil
	})

	// Start workers
	pool.Start(ctx)
	defer pool.Stop()

	// Enqueue many jobs
	jobCount := 100
	start := time.Now()

	for i := 0; i < jobCount; i++ {
		job := NewJob("default", "test.job", map[string]interface{}{
			"id": i,
		})
		err := queue.Enqueue(ctx, job)
		require.NoError(t, err)
	}

	// Wait for all jobs to complete
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for jobs to complete")
		case <-ticker.C:
			mu.Lock()
			count := processed
			mu.Unlock()
			if count == jobCount {
				duration := time.Since(start)
				t.Logf("Processed %d jobs in %v (%.2f jobs/sec)",
					jobCount, duration, float64(jobCount)/duration.Seconds())
				return
			}
		}
	}
}
