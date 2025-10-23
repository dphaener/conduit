package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Queue provides PostgreSQL-backed job queue operations
type Queue struct {
	db *sql.DB
}

// NewQueue creates a new job queue with PostgreSQL backing
func NewQueue(db *sql.DB) *Queue {
	return &Queue{db: db}
}

// Enqueue adds a new job to the queue
func (q *Queue) Enqueue(ctx context.Context, job *Job) error {
	payloadJSON, err := json.Marshal(job.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	query := `
		INSERT INTO jobs (
			id, queue, type, payload, status, priority,
			attempts, max_attempts, created_at, run_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = q.db.ExecContext(ctx, query,
		job.ID, job.Queue, job.Type, payloadJSON, job.Status, job.Priority,
		job.Attempts, job.MaxAttempts, job.CreatedAt, job.RunAt,
	)

	if err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	return nil
}

// EnqueueWithPriority adds a job with specific priority
func (q *Queue) EnqueueWithPriority(ctx context.Context, job *Job, priority JobPriority) error {
	job.Priority = priority
	return q.Enqueue(ctx, job)
}

// Schedule adds a job to be executed at a specific time
func (q *Queue) Schedule(ctx context.Context, job *Job, runAt time.Time) error {
	job.RunAt = runAt
	return q.Enqueue(ctx, job)
}

// Dequeue retrieves and locks the next available job for processing
func (q *Queue) Dequeue(ctx context.Context, workerID string, queueName string) (*Job, error) {
	// Use PostgreSQL's SKIP LOCKED for efficient concurrent dequeuing
	query := `
		UPDATE jobs
		SET status = $1, locked_by = $2, locked_at = $3, started_at = $4, attempts = attempts + 1
		WHERE id = (
			SELECT id FROM jobs
			WHERE status = $5
				AND queue = $6
				AND run_at <= $7
			ORDER BY priority DESC, created_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING id, queue, type, payload, status, priority, attempts, max_attempts,
		          error, created_at, run_at, started_at, completed_at, locked_by, locked_at
	`

	now := time.Now()
	var job Job
	var payloadJSON []byte

	err := q.db.QueryRowContext(ctx, query,
		JobStatusRunning, workerID, now, now, // SET values
		JobStatusPending, queueName, now, // WHERE conditions
	).Scan(
		&job.ID, &job.Queue, &job.Type, &payloadJSON, &job.Status, &job.Priority,
		&job.Attempts, &job.MaxAttempts, &job.Error, &job.CreatedAt, &job.RunAt,
		&job.StartedAt, &job.CompletedAt, &job.LockedBy, &job.LockedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no jobs available")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to dequeue job: %w", err)
	}

	// Unmarshal payload
	if err := json.Unmarshal(payloadJSON, &job.Payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return &job, nil
}

// Complete marks a job as successfully completed
func (q *Queue) Complete(ctx context.Context, jobID uuid.UUID) error {
	now := time.Now()
	query := `
		UPDATE jobs
		SET status = $1, completed_at = $2, locked_by = NULL, locked_at = NULL
		WHERE id = $3
	`

	result, err := q.db.ExecContext(ctx, query, JobStatusCompleted, now, jobID)
	if err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("job not found: %s", jobID)
	}

	return nil
}

// Fail marks a job as failed with an error message
func (q *Queue) Fail(ctx context.Context, jobID uuid.UUID, errMsg string) error {
	now := time.Now()
	query := `
		UPDATE jobs
		SET status = $1, error = $2, completed_at = $3, locked_by = NULL, locked_at = NULL
		WHERE id = $4
	`

	result, err := q.db.ExecContext(ctx, query, JobStatusFailed, errMsg, now, jobID)
	if err != nil {
		return fmt.Errorf("failed to fail job: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("job not found: %s", jobID)
	}

	return nil
}

// Retry reschedules a failed job with exponential backoff
// Uses atomic UPDATE to prevent race conditions between checking attempts and updating job
func (q *Queue) Retry(ctx context.Context, jobID uuid.UUID) error {
	// Calculate exponential backoff based on current attempts
	// Uses atomic UPDATE with attempts check to prevent race conditions
	query := `
		WITH job_data AS (
			SELECT attempts, max_attempts FROM jobs WHERE id = $1
		)
		UPDATE jobs
		SET status = $2,
			run_at = $3 + (INTERVAL '1 minute' * (1 << LEAST(attempts - 1, 10))),
			locked_by = NULL,
			locked_at = NULL,
			error = NULL
		FROM job_data
		WHERE jobs.id = $1
		  AND job_data.attempts < job_data.max_attempts
		RETURNING attempts, max_attempts
	`

	var attempts, maxAttempts int
	err := q.db.QueryRowContext(ctx, query, jobID, JobStatusPending, time.Now()).Scan(&attempts, &maxAttempts)
	if err == sql.ErrNoRows {
		return fmt.Errorf("job not found or exceeded max attempts")
	}
	return err
}

// Cancel marks a job as cancelled
// Note: Status constants are passed as parameters to prevent SQL injection
func (q *Queue) Cancel(ctx context.Context, jobID uuid.UUID) error {
	now := time.Now()
	query := `
		UPDATE jobs
		SET status = $1, completed_at = $2, locked_by = NULL, locked_at = NULL
		WHERE id = $3 AND status IN ($4, $5)
	`

	result, err := q.db.ExecContext(ctx, query,
		JobStatusCancelled, now, jobID, JobStatusPending, JobStatusRunning,
	)
	if err != nil {
		return fmt.Errorf("failed to cancel job: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("job not found or already completed: %s", jobID)
	}

	return nil
}

// GetJob retrieves a job by ID
func (q *Queue) GetJob(ctx context.Context, jobID uuid.UUID) (*Job, error) {
	query := `
		SELECT id, queue, type, payload, status, priority, attempts, max_attempts,
		       error, created_at, run_at, started_at, completed_at, locked_by, locked_at
		FROM jobs
		WHERE id = $1
	`

	var job Job
	var payloadJSON []byte

	err := q.db.QueryRowContext(ctx, query, jobID).Scan(
		&job.ID, &job.Queue, &job.Type, &payloadJSON, &job.Status, &job.Priority,
		&job.Attempts, &job.MaxAttempts, &job.Error, &job.CreatedAt, &job.RunAt,
		&job.StartedAt, &job.CompletedAt, &job.LockedBy, &job.LockedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	// Unmarshal payload
	if err := json.Unmarshal(payloadJSON, &job.Payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return &job, nil
}

// ListJobs retrieves jobs with optional filtering
func (q *Queue) ListJobs(ctx context.Context, queueName string, status JobStatus, limit int) ([]*Job, error) {
	query := `
		SELECT id, queue, type, payload, status, priority, attempts, max_attempts,
		       error, created_at, run_at, started_at, completed_at, locked_by, locked_at
		FROM jobs
		WHERE ($1 = '' OR queue = $1)
		  AND ($2 = '' OR status = $2)
		ORDER BY priority DESC, created_at ASC
		LIMIT $3
	`

	rows, err := q.db.QueryContext(ctx, query, queueName, status, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		var job Job
		var payloadJSON []byte

		err := rows.Scan(
			&job.ID, &job.Queue, &job.Type, &payloadJSON, &job.Status, &job.Priority,
			&job.Attempts, &job.MaxAttempts, &job.Error, &job.CreatedAt, &job.RunAt,
			&job.StartedAt, &job.CompletedAt, &job.LockedBy, &job.LockedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}

		// Unmarshal payload
		if err := json.Unmarshal(payloadJSON, &job.Payload); err != nil {
			return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
		}

		jobs = append(jobs, &job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating jobs: %w", err)
	}

	return jobs, nil
}

// PurgeCompleted removes completed jobs older than the specified duration
func (q *Queue) PurgeCompleted(ctx context.Context, olderThan time.Duration) (int64, error) {
	query := `
		DELETE FROM jobs
		WHERE status = $1 AND completed_at < $2
	`

	cutoff := time.Now().Add(-olderThan)
	result, err := q.db.ExecContext(ctx, query, JobStatusCompleted, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to purge jobs: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rows, nil
}

// GetQueueStats returns statistics for a queue
func (q *Queue) GetQueueStats(ctx context.Context, queueName string) (*QueueStats, error) {
	query := `
		SELECT
			COUNT(*) FILTER (WHERE status = 'pending') as pending,
			COUNT(*) FILTER (WHERE status = 'running') as running,
			COUNT(*) FILTER (WHERE status = 'completed') as completed,
			COUNT(*) FILTER (WHERE status = 'failed') as failed,
			COUNT(*) FILTER (WHERE status = 'cancelled') as cancelled
		FROM jobs
		WHERE queue = $1
	`

	var stats QueueStats
	err := q.db.QueryRowContext(ctx, query, queueName).Scan(
		&stats.Pending, &stats.Running, &stats.Completed, &stats.Failed, &stats.Cancelled,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue stats: %w", err)
	}

	stats.Queue = queueName
	return &stats, nil
}

// QueueStats holds statistics for a job queue
type QueueStats struct {
	Queue     string `json:"queue"`
	Pending   int    `json:"pending"`
	Running   int    `json:"running"`
	Completed int    `json:"completed"`
	Failed    int    `json:"failed"`
	Cancelled int    `json:"cancelled"`
}
