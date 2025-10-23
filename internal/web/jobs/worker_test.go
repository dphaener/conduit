package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewWorkerPool(t *testing.T) {
	db, _, _ := setupMockDB(t)
	defer db.Close()

	queue := NewQueue(db)
	pool := NewWorkerPool(queue, "default", 5)

	assert.NotNil(t, pool)
	assert.Equal(t, 5, pool.numWorkers)
	assert.Equal(t, "default", pool.queueName)
	assert.NotNil(t, pool.handlers)
	assert.NotNil(t, pool.metrics)
}

func TestWorkerPoolRegisterHandler(t *testing.T) {
	db, _, _ := setupMockDB(t)
	defer db.Close()

	queue := NewQueue(db)
	pool := NewWorkerPool(queue, "default", 1)

	called := false
	handler := func(ctx context.Context, payload map[string]interface{}) error {
		called = true
		return nil
	}

	pool.RegisterHandler("test.job", handler)

	// Verify handler is registered
	h, err := pool.handlers.Get("test.job")
	assert.NoError(t, err)
	assert.NotNil(t, h)

	// Test handler execution
	err = h(context.Background(), nil)
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestWorkerPoolStartStop(t *testing.T) {
	db, _, _ := setupMockDB(t)
	defer db.Close()

	queue := NewQueue(db)
	pool := NewWorkerPool(queue, "default", 2)

	ctx := context.Background()
	pool.Start(ctx)

	// Give workers time to start
	time.Sleep(50 * time.Millisecond)

	pool.Stop()

	// Verify all workers stopped
	assert.Len(t, pool.workers, 2)
}

func TestWorkerProcessJobSuccess(t *testing.T) {
	db, mock, _ := setupMockDB(t)
	defer db.Close()

	queue := NewQueue(db)
	metrics := NewMetrics()

	worker := &Worker{
		ID:        "worker-1",
		queue:     queue,
		handlers:  NewHandlerRegistry(),
		queueName: "default",
	}

	// Register handler
	handlerCalled := false
	worker.handlers.Register("test.job", func(ctx context.Context, payload map[string]interface{}) error {
		handlerCalled = true
		assert.Equal(t, "test-value", payload["test-key"])
		return nil
	})

	// Create job
	job := NewJob("default", "test.job", map[string]interface{}{"test-key": "test-value"})
	job.ID = uuid.New()

	// Mock Complete call
	mock.ExpectExec(`UPDATE jobs`).
		WithArgs(JobStatusCompleted, sqlmock.AnyArg(), job.ID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Process job
	ctx := context.Background()
	worker.processJob(ctx, job, metrics)

	assert.True(t, handlerCalled)
	assert.NoError(t, mock.ExpectationsWereMet())

	// Verify metrics
	stats := metrics.GetStats("test.job")
	assert.Equal(t, int64(1), stats.Processed)
	assert.Equal(t, int64(1), stats.Succeeded)
	assert.Equal(t, int64(0), stats.Failed)
}

func TestWorkerProcessJobFailure(t *testing.T) {
	db, mock, _ := setupMockDB(t)
	defer db.Close()

	queue := NewQueue(db)
	metrics := NewMetrics()

	worker := &Worker{
		ID:        "worker-1",
		queue:     queue,
		handlers:  NewHandlerRegistry(),
		queueName: "default",
	}

	// Register handler that fails
	expectedErr := errors.New("handler error")
	worker.handlers.Register("test.job", func(ctx context.Context, payload map[string]interface{}) error {
		return expectedErr
	})

	// Create job
	job := NewJob("default", "test.job", map[string]interface{}{})
	job.ID = uuid.New()
	job.Attempts = 1
	job.MaxAttempts = 3

	// Mock atomic UPDATE query for retry
	rows := sqlmock.NewRows([]string{"attempts", "max_attempts"}).AddRow(1, 3)
	mock.ExpectQuery(`WITH job_data AS`).
		WithArgs(job.ID, JobStatusPending, sqlmock.AnyArg()).
		WillReturnRows(rows)

	// Process job
	ctx := context.Background()
	worker.processJob(ctx, job, metrics)

	assert.NoError(t, mock.ExpectationsWereMet())

	// Verify metrics - job was retried, not marked as processed yet
	stats := metrics.GetStats("test.job")
	assert.Equal(t, int64(0), stats.Processed) // Not completed yet, will retry
	assert.Equal(t, int64(0), stats.Succeeded)
	assert.Equal(t, int64(1), stats.Retried)
}

func TestWorkerProcessJobExceedsMaxAttempts(t *testing.T) {
	db, mock, _ := setupMockDB(t)
	defer db.Close()

	queue := NewQueue(db)
	metrics := NewMetrics()

	worker := &Worker{
		ID:        "worker-1",
		queue:     queue,
		handlers:  NewHandlerRegistry(),
		queueName: "default",
	}

	// Register handler that fails
	expectedErr := errors.New("handler error")
	worker.handlers.Register("test.job", func(ctx context.Context, payload map[string]interface{}) error {
		return expectedErr
	})

	// Create job that has exhausted retries
	job := NewJob("default", "test.job", map[string]interface{}{})
	job.ID = uuid.New()
	job.Attempts = 3
	job.MaxAttempts = 3

	// Mock Fail call
	mock.ExpectExec(`UPDATE jobs`).
		WithArgs(JobStatusFailed, expectedErr.Error(), sqlmock.AnyArg(), job.ID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Process job
	ctx := context.Background()
	worker.processJob(ctx, job, metrics)

	assert.NoError(t, mock.ExpectationsWereMet())

	// Verify metrics
	stats := metrics.GetStats("test.job")
	assert.Equal(t, int64(1), stats.Processed)
	assert.Equal(t, int64(0), stats.Succeeded)
	assert.Equal(t, int64(1), stats.Failed)
}

func TestWorkerProcessJobNoHandler(t *testing.T) {
	db, mock, _ := setupMockDB(t)
	defer db.Close()

	queue := NewQueue(db)
	metrics := NewMetrics()

	worker := &Worker{
		ID:        "worker-1",
		queue:     queue,
		handlers:  NewHandlerRegistry(),
		queueName: "default",
	}

	// Create job with unregistered type
	job := NewJob("default", "unknown.job", map[string]interface{}{})
	job.ID = uuid.New()

	// Mock Fail call
	mock.ExpectExec(`UPDATE jobs`).
		WithArgs(JobStatusFailed, sqlmock.AnyArg(), sqlmock.AnyArg(), job.ID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Process job
	ctx := context.Background()
	worker.processJob(ctx, job, metrics)

	assert.NoError(t, mock.ExpectationsWereMet())

	// Verify metrics
	stats := metrics.GetStats("unknown.job")
	assert.Equal(t, int64(1), stats.Failed)
}

func TestHandlerRegistry(t *testing.T) {
	registry := NewHandlerRegistry()

	// Test registration
	handler := func(ctx context.Context, payload map[string]interface{}) error {
		return nil
	}
	registry.Register("test.job", handler)

	// Test retrieval
	h, err := registry.Get("test.job")
	assert.NoError(t, err)
	assert.NotNil(t, h)

	// Test unknown type
	_, err = registry.Get("unknown.job")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no handler registered")

	// Test list types
	types := registry.ListTypes()
	assert.Len(t, types, 1)
	assert.Contains(t, types, "test.job")
}

func TestMetrics(t *testing.T) {
	metrics := NewMetrics()

	// Record some stats
	metrics.RecordSuccess("job1", 100*time.Millisecond)
	metrics.RecordSuccess("job1", 200*time.Millisecond)
	metrics.RecordFailure("job1", 150*time.Millisecond)
	metrics.RecordRetry("job1")

	metrics.RecordSuccess("job2", 50*time.Millisecond)

	// Test GetStats
	stats1 := metrics.GetStats("job1")
	assert.Equal(t, int64(3), stats1.Processed)
	assert.Equal(t, int64(2), stats1.Succeeded)
	assert.Equal(t, int64(1), stats1.Failed)
	assert.Equal(t, int64(1), stats1.Retried)
	assert.Equal(t, 100*time.Millisecond, stats1.MinDuration)
	assert.Equal(t, 200*time.Millisecond, stats1.MaxDuration)
	assert.Equal(t, 150*time.Millisecond, stats1.AvgDuration)

	stats2 := metrics.GetStats("job2")
	assert.Equal(t, int64(1), stats2.Processed)
	assert.Equal(t, int64(1), stats2.Succeeded)
	assert.Equal(t, 50*time.Millisecond, stats2.AvgDuration)

	// Test GetAllStats
	allStats := metrics.GetAllStats()
	assert.Len(t, allStats, 2)
	assert.Contains(t, allStats, "job1")
	assert.Contains(t, allStats, "job2")

	// Test SuccessRate
	assert.Equal(t, 66.66666666666666, stats1.SuccessRate())
	assert.Equal(t, 100.0, stats2.SuccessRate())
}

func TestJobStats(t *testing.T) {
	// Test success rate with no jobs
	stats := JobStats{
		JobType:   "test",
		Processed: 0,
		Succeeded: 0,
	}
	assert.Equal(t, 0.0, stats.SuccessRate())

	// Test success rate with jobs
	stats = JobStats{
		JobType:   "test",
		Processed: 10,
		Succeeded: 8,
	}
	assert.Equal(t, 80.0, stats.SuccessRate())
}

func TestWorkerPoolGetMetrics(t *testing.T) {
	db, _, _ := setupMockDB(t)
	defer db.Close()

	queue := NewQueue(db)
	pool := NewWorkerPool(queue, "default", 1)

	metrics := pool.GetMetrics()
	assert.NotNil(t, metrics)
	assert.NotNil(t, metrics.processed)
	assert.NotNil(t, metrics.succeeded)
	assert.NotNil(t, metrics.failed)
	assert.NotNil(t, metrics.retried)
}
