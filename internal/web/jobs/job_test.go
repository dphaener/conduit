package jobs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewJob(t *testing.T) {
	queue := "default"
	jobType := "test.job"
	payload := map[string]interface{}{"key": "value"}

	job := NewJob(queue, jobType, payload)

	assert.NotNil(t, job)
	assert.NotEqual(t, "", job.ID.String())
	assert.Equal(t, queue, job.Queue)
	assert.Equal(t, jobType, job.Type)
	assert.Equal(t, payload, job.Payload)
	assert.Equal(t, JobStatusPending, job.Status)
	assert.Equal(t, PriorityNormal, job.Priority)
	assert.Equal(t, 0, job.Attempts)
	assert.Equal(t, 3, job.MaxAttempts)
	assert.False(t, job.CreatedAt.IsZero())
	assert.False(t, job.RunAt.IsZero())
}

func TestJobStatusConstants(t *testing.T) {
	assert.Equal(t, JobStatus("pending"), JobStatusPending)
	assert.Equal(t, JobStatus("running"), JobStatusRunning)
	assert.Equal(t, JobStatus("completed"), JobStatusCompleted)
	assert.Equal(t, JobStatus("failed"), JobStatusFailed)
	assert.Equal(t, JobStatus("cancelled"), JobStatusCancelled)
}

func TestJobPriorityConstants(t *testing.T) {
	assert.Equal(t, JobPriority(0), PriorityLow)
	assert.Equal(t, JobPriority(50), PriorityNormal)
	assert.Equal(t, JobPriority(75), PriorityHigh)
	assert.Equal(t, JobPriority(100), PriorityUrgent)
}

func TestJobIsProcessableAllStatuses(t *testing.T) {
	// Pending job with past run_at
	job := NewJob("default", "test", nil)
	job.RunAt = time.Now().Add(-1 * time.Hour)
	assert.True(t, job.IsProcessable())

	// Pending job with future run_at
	job = NewJob("default", "test", nil)
	job.RunAt = time.Now().Add(1 * time.Hour)
	assert.False(t, job.IsProcessable())

	// Running job
	job = NewJob("default", "test", nil)
	job.Status = JobStatusRunning
	assert.False(t, job.IsProcessable())

	// Completed job
	job = NewJob("default", "test", nil)
	job.Status = JobStatusCompleted
	assert.False(t, job.IsProcessable())

	// Failed job
	job = NewJob("default", "test", nil)
	job.Status = JobStatusFailed
	assert.False(t, job.IsProcessable())

	// Cancelled job
	job = NewJob("default", "test", nil)
	job.Status = JobStatusCancelled
	assert.False(t, job.IsProcessable())
}

func TestJobIsLockedEdgeCases(t *testing.T) {
	job := NewJob("default", "test", nil)

	// Not locked
	assert.False(t, job.IsLocked())

	// Locked with worker ID and time
	workerID := "worker-1"
	lockTime := time.Now()
	job.LockedBy = &workerID
	job.LockedAt = &lockTime
	assert.True(t, job.IsLocked())

	// Only worker ID (should be false)
	job2 := NewJob("default", "test", nil)
	job2.LockedBy = &workerID
	assert.False(t, job2.IsLocked())

	// Only lock time (should be false)
	job3 := NewJob("default", "test", nil)
	job3.LockedAt = &lockTime
	assert.False(t, job3.IsLocked())
}

func TestJobWithDifferentMaxAttempts(t *testing.T) {
	job := NewJob("default", "test", nil)
	job.MaxAttempts = 5

	job.Attempts = 4
	assert.True(t, job.IsRetryable())

	job.Attempts = 5
	assert.False(t, job.IsRetryable())
}
