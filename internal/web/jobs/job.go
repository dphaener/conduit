package jobs

import (
	"time"

	"github.com/google/uuid"
)

// JobStatus represents the current state of a job
type JobStatus string

const (
	// JobStatusPending indicates the job is waiting to be processed
	JobStatusPending JobStatus = "pending"
	// JobStatusRunning indicates the job is currently being processed
	JobStatusRunning JobStatus = "running"
	// JobStatusCompleted indicates the job finished successfully
	JobStatusCompleted JobStatus = "completed"
	// JobStatusFailed indicates the job failed after all retries
	JobStatusFailed JobStatus = "failed"
	// JobStatusCancelled indicates the job was cancelled
	JobStatusCancelled JobStatus = "cancelled"
)

// JobPriority represents the priority level of a job
type JobPriority int

const (
	// PriorityLow is for non-urgent background tasks
	PriorityLow JobPriority = 0
	// PriorityNormal is the default priority
	PriorityNormal JobPriority = 50
	// PriorityHigh is for important tasks
	PriorityHigh JobPriority = 75
	// PriorityUrgent is for critical tasks that should run ASAP
	PriorityUrgent JobPriority = 100
)

// Job represents a background job with all its metadata
type Job struct {
	// ID is the unique identifier for the job
	ID uuid.UUID `db:"id" json:"id"`
	// Queue is the name of the queue this job belongs to
	Queue string `db:"queue" json:"queue"`
	// Type is the job handler type (e.g., "email.send", "report.generate")
	Type string `db:"type" json:"type"`
	// Payload is the JSON-encoded job data
	Payload map[string]interface{} `db:"payload" json:"payload"`
	// Status is the current state of the job
	Status JobStatus `db:"status" json:"status"`
	// Priority determines execution order (higher = sooner)
	Priority JobPriority `db:"priority" json:"priority"`
	// Attempts is the number of times this job has been attempted
	Attempts int `db:"attempts" json:"attempts"`
	// MaxAttempts is the maximum number of retry attempts
	MaxAttempts int `db:"max_attempts" json:"max_attempts"`
	// Error stores the last error message if the job failed
	Error *string `db:"error" json:"error,omitempty"`
	// CreatedAt is when the job was first created
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	// RunAt is when the job should be executed (for scheduled jobs)
	RunAt time.Time `db:"run_at" json:"run_at"`
	// StartedAt is when the job started processing
	StartedAt *time.Time `db:"started_at" json:"started_at,omitempty"`
	// CompletedAt is when the job finished (success or failure)
	CompletedAt *time.Time `db:"completed_at" json:"completed_at,omitempty"`
	// LockedBy is the worker ID that's currently processing this job
	LockedBy *string `db:"locked_by" json:"locked_by,omitempty"`
	// LockedAt is when the job was locked for processing
	LockedAt *time.Time `db:"locked_at" json:"locked_at,omitempty"`
}

// NewJob creates a new job with default values
func NewJob(queue, jobType string, payload map[string]interface{}) *Job {
	now := time.Now()
	return &Job{
		ID:          uuid.New(),
		Queue:       queue,
		Type:        jobType,
		Payload:     payload,
		Status:      JobStatusPending,
		Priority:    PriorityNormal,
		Attempts:    0,
		MaxAttempts: 3,
		CreatedAt:   now,
		RunAt:       now,
	}
}

// IsProcessable returns true if the job can be picked up by a worker
func (j *Job) IsProcessable() bool {
	return j.Status == JobStatusPending && time.Now().After(j.RunAt)
}

// IsLocked returns true if the job is currently locked by a worker
func (j *Job) IsLocked() bool {
	return j.LockedBy != nil && j.LockedAt != nil
}

// IsRetryable returns true if the job can be retried
func (j *Job) IsRetryable() bool {
	return j.Attempts < j.MaxAttempts
}
