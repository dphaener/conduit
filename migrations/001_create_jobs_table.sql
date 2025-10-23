-- Migration: Create jobs table for background job processing
-- Description: PostgreSQL-backed job queue with priority, retry logic, and scheduling

-- Create jobs table
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
    locked_at TIMESTAMP WITH TIME ZONE,

    -- Constraints
    CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
    CHECK (priority >= 0 AND priority <= 100),
    CHECK (attempts >= 0),
    CHECK (max_attempts > 0)
);

-- Index for efficient job dequeuing (most important)
-- This supports: WHERE status = 'pending' AND queue = X AND run_at <= NOW() ORDER BY priority DESC, created_at ASC
CREATE INDEX idx_jobs_dequeue ON jobs (queue, status, run_at, priority DESC, created_at ASC)
WHERE status = 'pending';

-- Index for finding jobs by status and queue
CREATE INDEX idx_jobs_status_queue ON jobs (status, queue);

-- Index for finding jobs by type (for metrics and monitoring)
CREATE INDEX idx_jobs_type ON jobs (type);

-- Index for cleanup operations (purging old completed jobs)
CREATE INDEX idx_jobs_completed_at ON jobs (completed_at)
WHERE status = 'completed';

-- Index for finding locked jobs (for detecting stuck workers)
CREATE INDEX idx_jobs_locked ON jobs (locked_by, locked_at)
WHERE status = 'running';

-- Index for scheduled jobs (finding jobs to run in the future)
CREATE INDEX idx_jobs_scheduled ON jobs (run_at)
WHERE status = 'pending' AND run_at > NOW();

-- Comments for documentation
COMMENT ON TABLE jobs IS 'Background jobs queue with PostgreSQL backing';
COMMENT ON COLUMN jobs.id IS 'Unique job identifier (UUID)';
COMMENT ON COLUMN jobs.queue IS 'Queue name for job organization and routing';
COMMENT ON COLUMN jobs.type IS 'Job handler type (e.g., email.send, report.generate)';
COMMENT ON COLUMN jobs.payload IS 'Job data as JSONB for flexibility';
COMMENT ON COLUMN jobs.status IS 'Current job status: pending, running, completed, failed, cancelled';
COMMENT ON COLUMN jobs.priority IS 'Job priority (0-100, higher = sooner)';
COMMENT ON COLUMN jobs.attempts IS 'Number of execution attempts';
COMMENT ON COLUMN jobs.max_attempts IS 'Maximum number of retry attempts';
COMMENT ON COLUMN jobs.error IS 'Last error message if job failed';
COMMENT ON COLUMN jobs.created_at IS 'When the job was created';
COMMENT ON COLUMN jobs.run_at IS 'When the job should be executed (supports scheduling)';
COMMENT ON COLUMN jobs.started_at IS 'When the job started processing';
COMMENT ON COLUMN jobs.completed_at IS 'When the job finished (success or failure)';
COMMENT ON COLUMN jobs.locked_by IS 'Worker ID that locked this job';
COMMENT ON COLUMN jobs.locked_at IS 'When the job was locked';

-- Rollback migration
-- DROP TABLE IF EXISTS jobs;
