package jobs

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *Queue) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	queue := NewQueue(db)
	return db, mock, queue
}

func TestNewQueue(t *testing.T) {
	db, _, queue := setupMockDB(t)
	defer db.Close()

	assert.NotNil(t, queue)
	assert.NotNil(t, queue.db)
}

func TestEnqueue(t *testing.T) {
	db, mock, queue := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()
	job := NewJob("default", "test.job", map[string]interface{}{"key": "value"})

	mock.ExpectExec(`INSERT INTO jobs`).
		WithArgs(
			job.ID, job.Queue, job.Type, sqlmock.AnyArg(), job.Status, job.Priority,
			job.Attempts, job.MaxAttempts, job.CreatedAt, job.RunAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := queue.Enqueue(ctx, job)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestEnqueueWithPriority(t *testing.T) {
	db, mock, queue := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()
	job := NewJob("default", "test.job", map[string]interface{}{"key": "value"})

	mock.ExpectExec(`INSERT INTO jobs`).
		WithArgs(
			job.ID, job.Queue, job.Type, sqlmock.AnyArg(), job.Status, PriorityUrgent,
			job.Attempts, job.MaxAttempts, job.CreatedAt, job.RunAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := queue.EnqueueWithPriority(ctx, job, PriorityUrgent)
	assert.NoError(t, err)
	assert.Equal(t, PriorityUrgent, job.Priority)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSchedule(t *testing.T) {
	db, mock, queue := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()
	job := NewJob("default", "test.job", map[string]interface{}{"key": "value"})
	runAt := time.Now().Add(1 * time.Hour)

	mock.ExpectExec(`INSERT INTO jobs`).
		WithArgs(
			job.ID, job.Queue, job.Type, sqlmock.AnyArg(), job.Status, job.Priority,
			job.Attempts, job.MaxAttempts, job.CreatedAt, runAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := queue.Schedule(ctx, job, runAt)
	assert.NoError(t, err)
	assert.Equal(t, runAt, job.RunAt)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDequeue(t *testing.T) {
	db, mock, queue := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()
	workerID := "worker-1"
	queueName := "default"
	jobID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "queue", "type", "payload", "status", "priority", "attempts", "max_attempts",
		"error", "created_at", "run_at", "started_at", "completed_at", "locked_by", "locked_at",
	}).AddRow(
		jobID, "default", "test.job", []byte(`{"key":"value"}`), JobStatusRunning, PriorityNormal,
		1, 3, nil, time.Now(), time.Now(), time.Now(), nil, workerID, time.Now(),
	)

	mock.ExpectQuery(`UPDATE jobs`).
		WithArgs(
			JobStatusRunning, workerID, sqlmock.AnyArg(), sqlmock.AnyArg(),
			JobStatusPending, queueName, sqlmock.AnyArg(),
		).
		WillReturnRows(rows)

	job, err := queue.Dequeue(ctx, workerID, queueName)
	assert.NoError(t, err)
	assert.NotNil(t, job)
	assert.Equal(t, jobID, job.ID)
	assert.Equal(t, "test.job", job.Type)
	assert.Equal(t, "value", job.Payload["key"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDequeueNoJobs(t *testing.T) {
	db, mock, queue := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()
	workerID := "worker-1"
	queueName := "default"

	mock.ExpectQuery(`UPDATE jobs`).
		WithArgs(
			JobStatusRunning, workerID, sqlmock.AnyArg(), sqlmock.AnyArg(),
			JobStatusPending, queueName, sqlmock.AnyArg(),
		).
		WillReturnError(sql.ErrNoRows)

	job, err := queue.Dequeue(ctx, workerID, queueName)
	assert.Error(t, err)
	assert.Nil(t, job)
	assert.Contains(t, err.Error(), "no jobs available")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestComplete(t *testing.T) {
	db, mock, queue := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()
	jobID := uuid.New()

	mock.ExpectExec(`UPDATE jobs`).
		WithArgs(JobStatusCompleted, sqlmock.AnyArg(), jobID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := queue.Complete(ctx, jobID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFail(t *testing.T) {
	db, mock, queue := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()
	jobID := uuid.New()
	errMsg := "job failed"

	mock.ExpectExec(`UPDATE jobs`).
		WithArgs(JobStatusFailed, errMsg, sqlmock.AnyArg(), jobID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := queue.Fail(ctx, jobID, errMsg)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRetry(t *testing.T) {
	db, mock, queue := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()
	jobID := uuid.New()

	// Mock atomic UPDATE query that returns attempts and max_attempts
	rows := sqlmock.NewRows([]string{"attempts", "max_attempts"}).AddRow(1, 3)
	mock.ExpectQuery(`WITH job_data AS`).
		WithArgs(jobID, JobStatusPending, sqlmock.AnyArg()).
		WillReturnRows(rows)

	err := queue.Retry(ctx, jobID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRetryExceedsMaxAttempts(t *testing.T) {
	db, mock, queue := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()
	jobID := uuid.New()

	// Mock atomic UPDATE query that returns no rows (job exceeded max attempts)
	mock.ExpectQuery(`WITH job_data AS`).
		WithArgs(jobID, JobStatusPending, sqlmock.AnyArg()).
		WillReturnError(sql.ErrNoRows)

	err := queue.Retry(ctx, jobID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "job not found or exceeded max attempts")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCancel(t *testing.T) {
	db, mock, queue := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()
	jobID := uuid.New()

	mock.ExpectExec(`UPDATE jobs`).
		WithArgs(JobStatusCancelled, sqlmock.AnyArg(), jobID, JobStatusPending, JobStatusRunning).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := queue.Cancel(ctx, jobID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetJob(t *testing.T) {
	db, mock, queue := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()
	jobID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "queue", "type", "payload", "status", "priority", "attempts", "max_attempts",
		"error", "created_at", "run_at", "started_at", "completed_at", "locked_by", "locked_at",
	}).AddRow(
		jobID, "default", "test.job", []byte(`{"key":"value"}`), JobStatusPending, PriorityNormal,
		0, 3, nil, time.Now(), time.Now(), nil, nil, nil, nil,
	)

	mock.ExpectQuery(`SELECT (.+) FROM jobs WHERE id`).
		WithArgs(jobID).
		WillReturnRows(rows)

	job, err := queue.GetJob(ctx, jobID)
	assert.NoError(t, err)
	assert.NotNil(t, job)
	assert.Equal(t, jobID, job.ID)
	assert.Equal(t, "test.job", job.Type)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListJobs(t *testing.T) {
	db, mock, queue := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()
	queueName := "default"

	rows := sqlmock.NewRows([]string{
		"id", "queue", "type", "payload", "status", "priority", "attempts", "max_attempts",
		"error", "created_at", "run_at", "started_at", "completed_at", "locked_by", "locked_at",
	}).
		AddRow(uuid.New(), "default", "test.job1", []byte(`{}`), JobStatusPending, PriorityNormal, 0, 3, nil, time.Now(), time.Now(), nil, nil, nil, nil).
		AddRow(uuid.New(), "default", "test.job2", []byte(`{}`), JobStatusPending, PriorityHigh, 0, 3, nil, time.Now(), time.Now(), nil, nil, nil, nil)

	mock.ExpectQuery(`SELECT (.+) FROM jobs`).
		WithArgs(queueName, "", 10).
		WillReturnRows(rows)

	jobs, err := queue.ListJobs(ctx, queueName, "", 10)
	assert.NoError(t, err)
	assert.Len(t, jobs, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPurgeCompleted(t *testing.T) {
	db, mock, queue := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()
	olderThan := 24 * time.Hour

	mock.ExpectExec(`DELETE FROM jobs`).
		WithArgs(JobStatusCompleted, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 42))

	count, err := queue.PurgeCompleted(ctx, olderThan)
	assert.NoError(t, err)
	assert.Equal(t, int64(42), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetQueueStats(t *testing.T) {
	db, mock, queue := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()
	queueName := "default"

	rows := sqlmock.NewRows([]string{"pending", "running", "completed", "failed", "cancelled"}).
		AddRow(10, 2, 100, 5, 3)

	mock.ExpectQuery(`SELECT (.+) FROM jobs`).
		WithArgs(queueName).
		WillReturnRows(rows)

	stats, err := queue.GetQueueStats(ctx, queueName)
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, queueName, stats.Queue)
	assert.Equal(t, 10, stats.Pending)
	assert.Equal(t, 2, stats.Running)
	assert.Equal(t, 100, stats.Completed)
	assert.Equal(t, 5, stats.Failed)
	assert.Equal(t, 3, stats.Cancelled)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestJobIsProcessable(t *testing.T) {
	// Job in the past
	job1 := NewJob("default", "test", nil)
	job1.RunAt = time.Now().Add(-1 * time.Hour)
	assert.True(t, job1.IsProcessable())

	// Job in the future
	job2 := NewJob("default", "test", nil)
	job2.RunAt = time.Now().Add(1 * time.Hour)
	assert.False(t, job2.IsProcessable())

	// Job not pending
	job3 := NewJob("default", "test", nil)
	job3.Status = JobStatusRunning
	assert.False(t, job3.IsProcessable())
}

func TestJobIsLocked(t *testing.T) {
	job := NewJob("default", "test", nil)
	assert.False(t, job.IsLocked())

	workerID := "worker-1"
	lockTime := time.Now()
	job.LockedBy = &workerID
	job.LockedAt = &lockTime
	assert.True(t, job.IsLocked())
}

func TestJobIsRetryable(t *testing.T) {
	job := NewJob("default", "test", nil)
	job.MaxAttempts = 3

	job.Attempts = 0
	assert.True(t, job.IsRetryable())

	job.Attempts = 2
	assert.True(t, job.IsRetryable())

	job.Attempts = 3
	assert.False(t, job.IsRetryable())

	job.Attempts = 5
	assert.False(t, job.IsRetryable())
}

func TestEnqueueMarshalError(t *testing.T) {
	db, _, queue := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()
	// Create job with unmarshalable payload (channels can't be marshaled to JSON)
	job := NewJob("default", "test.job", map[string]interface{}{
		"bad": make(chan int),
	})

	err := queue.Enqueue(ctx, job)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal payload")
}

func TestDequeueMarshalError(t *testing.T) {
	db, mock, queue := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()
	workerID := "worker-1"
	queueName := "default"
	jobID := uuid.New()

	// Return invalid JSON
	rows := sqlmock.NewRows([]string{
		"id", "queue", "type", "payload", "status", "priority", "attempts", "max_attempts",
		"error", "created_at", "run_at", "started_at", "completed_at", "locked_by", "locked_at",
	}).AddRow(
		jobID, "default", "test.job", []byte(`{invalid json`), JobStatusRunning, PriorityNormal,
		1, 3, nil, time.Now(), time.Now(), time.Now(), nil, workerID, time.Now(),
	)

	mock.ExpectQuery(`UPDATE jobs`).
		WithArgs(
			JobStatusRunning, workerID, sqlmock.AnyArg(), sqlmock.AnyArg(),
			JobStatusPending, queueName, sqlmock.AnyArg(),
		).
		WillReturnRows(rows)

	job, err := queue.Dequeue(ctx, workerID, queueName)
	assert.Error(t, err)
	assert.Nil(t, job)
	assert.Contains(t, err.Error(), "failed to unmarshal payload")
}

func TestCompleteJobNotFound(t *testing.T) {
	db, mock, queue := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()
	jobID := uuid.New()

	mock.ExpectExec(`UPDATE jobs`).
		WithArgs(JobStatusCompleted, sqlmock.AnyArg(), jobID).
		WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows affected

	err := queue.Complete(ctx, jobID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "job not found")
}

func TestFailJobNotFound(t *testing.T) {
	db, mock, queue := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()
	jobID := uuid.New()
	errMsg := "test error"

	mock.ExpectExec(`UPDATE jobs`).
		WithArgs(JobStatusFailed, errMsg, sqlmock.AnyArg(), jobID).
		WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows affected

	err := queue.Fail(ctx, jobID, errMsg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "job not found")
}

func TestRetryJobNotFound(t *testing.T) {
	db, mock, queue := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()
	jobID := uuid.New()

	// Mock atomic UPDATE query that returns no rows (job not found)
	mock.ExpectQuery(`WITH job_data AS`).
		WithArgs(jobID, JobStatusPending, sqlmock.AnyArg()).
		WillReturnError(sql.ErrNoRows)

	err := queue.Retry(ctx, jobID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "job not found or exceeded max attempts")
}

func TestCancelJobNotFound(t *testing.T) {
	db, mock, queue := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()
	jobID := uuid.New()

	mock.ExpectExec(`UPDATE jobs`).
		WithArgs(JobStatusCancelled, sqlmock.AnyArg(), jobID, JobStatusPending, JobStatusRunning).
		WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows affected

	err := queue.Cancel(ctx, jobID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "job not found or already completed")
}

func TestGetJobMarshalError(t *testing.T) {
	db, mock, queue := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()
	jobID := uuid.New()

	// Return invalid JSON
	rows := sqlmock.NewRows([]string{
		"id", "queue", "type", "payload", "status", "priority", "attempts", "max_attempts",
		"error", "created_at", "run_at", "started_at", "completed_at", "locked_by", "locked_at",
	}).AddRow(
		jobID, "default", "test.job", []byte(`{invalid`), JobStatusPending, PriorityNormal,
		0, 3, nil, time.Now(), time.Now(), nil, nil, nil, nil,
	)

	mock.ExpectQuery(`SELECT (.+) FROM jobs WHERE id`).
		WithArgs(jobID).
		WillReturnRows(rows)

	job, err := queue.GetJob(ctx, jobID)
	assert.Error(t, err)
	assert.Nil(t, job)
	assert.Contains(t, err.Error(), "failed to unmarshal payload")
}

func TestListJobsMarshalError(t *testing.T) {
	db, mock, queue := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()
	queueName := "default"

	// Return invalid JSON
	rows := sqlmock.NewRows([]string{
		"id", "queue", "type", "payload", "status", "priority", "attempts", "max_attempts",
		"error", "created_at", "run_at", "started_at", "completed_at", "locked_by", "locked_at",
	}).AddRow(
		uuid.New(), "default", "test.job", []byte(`{bad json}`), JobStatusPending, PriorityNormal,
		0, 3, nil, time.Now(), time.Now(), nil, nil, nil, nil,
	)

	mock.ExpectQuery(`SELECT (.+) FROM jobs`).
		WithArgs(queueName, "", 10).
		WillReturnRows(rows)

	jobs, err := queue.ListJobs(ctx, queueName, "", 10)
	assert.Error(t, err)
	assert.Nil(t, jobs)
	assert.Contains(t, err.Error(), "failed to unmarshal payload")
}
