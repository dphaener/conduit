package jobs

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Handler is a function that processes a job's payload
type Handler func(ctx context.Context, payload map[string]interface{}) error

// Worker represents a single worker goroutine that processes jobs
type Worker struct {
	ID       string
	queue    *Queue
	handlers *HandlerRegistry
	queueName string
	stopChan chan struct{}
	wg       *sync.WaitGroup
}

// WorkerPool manages multiple worker goroutines for concurrent job processing
type WorkerPool struct {
	queue      *Queue
	handlers   *HandlerRegistry
	queueName  string
	numWorkers int
	workers    []*Worker
	stopChan   chan struct{}
	wg         sync.WaitGroup
	mu         sync.RWMutex
	metrics    *Metrics
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(queue *Queue, queueName string, numWorkers int) *WorkerPool {
	return &WorkerPool{
		queue:      queue,
		handlers:   NewHandlerRegistry(),
		queueName:  queueName,
		numWorkers: numWorkers,
		workers:    make([]*Worker, 0, numWorkers),
		stopChan:   make(chan struct{}),
		metrics:    NewMetrics(),
	}
}

// RegisterHandler registers a job handler for a specific job type
func (p *WorkerPool) RegisterHandler(jobType string, handler Handler) {
	p.handlers.Register(jobType, handler)
}

// Start starts all workers in the pool
func (p *WorkerPool) Start(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()

	log.Printf("Starting worker pool with %d workers for queue '%s'", p.numWorkers, p.queueName)

	for i := 0; i < p.numWorkers; i++ {
		worker := &Worker{
			ID:        fmt.Sprintf("worker-%s-%d", p.queueName, i),
			queue:     p.queue,
			handlers:  p.handlers,
			queueName: p.queueName,
			stopChan:  p.stopChan,
			wg:        &p.wg,
		}

		p.workers = append(p.workers, worker)
		p.wg.Add(1)
		go worker.run(ctx, p.metrics)
	}
}

// Stop gracefully stops all workers
func (p *WorkerPool) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	log.Printf("Stopping worker pool for queue '%s'", p.queueName)
	close(p.stopChan)
	p.wg.Wait()
	log.Printf("Worker pool stopped for queue '%s'", p.queueName)
}

// GetMetrics returns the current metrics
func (p *WorkerPool) GetMetrics() *Metrics {
	return p.metrics
}

// run is the main worker loop
func (w *Worker) run(ctx context.Context, metrics *Metrics) {
	defer w.wg.Done()

	log.Printf("Worker %s started", w.ID)

	for {
		select {
		case <-w.stopChan:
			log.Printf("Worker %s stopped", w.ID)
			return
		default:
			// Try to dequeue a job
			job, err := w.queue.Dequeue(ctx, w.ID, w.queueName)
			if err != nil {
				// No jobs available, sleep briefly
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// Process the job
			w.processJob(ctx, job, metrics)
		}
	}
}

// processJob handles a single job execution
func (w *Worker) processJob(ctx context.Context, job *Job, metrics *Metrics) {
	startTime := time.Now()
	log.Printf("Worker %s processing job %s (type: %s, attempt: %d/%d)",
		w.ID, job.ID, job.Type, job.Attempts, job.MaxAttempts)

	// Get handler for this job type
	handler, err := w.handlers.Get(job.Type)
	if err != nil {
		errMsg := fmt.Sprintf("no handler registered for job type: %s", job.Type)
		log.Printf("Worker %s: %s", w.ID, errMsg)
		w.queue.Fail(ctx, job.ID, errMsg)
		metrics.RecordFailure(job.Type, time.Since(startTime))
		return
	}

	// Execute the handler
	err = handler(ctx, job.Payload)

	duration := time.Since(startTime)

	if err != nil {
		w.handleJobError(ctx, job, err, metrics, duration)
		return
	}

	// Job succeeded
	if err := w.queue.Complete(ctx, job.ID); err != nil {
		log.Printf("Worker %s: failed to mark job %s as complete: %v", w.ID, job.ID, err)
		return
	}

	log.Printf("Worker %s: job %s completed successfully in %v", w.ID, job.ID, duration)
	metrics.RecordSuccess(job.Type, duration)
}

// handleJobError processes a job failure and determines retry logic
func (w *Worker) handleJobError(ctx context.Context, job *Job, err error, metrics *Metrics, duration time.Duration) {
	errMsg := err.Error()
	log.Printf("Worker %s: job %s failed: %v (attempt %d/%d)",
		w.ID, job.ID, err, job.Attempts, job.MaxAttempts)

	// Check if job should be retried
	if job.IsRetryable() {
		if retryErr := w.queue.Retry(ctx, job.ID); retryErr != nil {
			log.Printf("Worker %s: failed to retry job %s: %v", w.ID, job.ID, retryErr)
			w.queue.Fail(ctx, job.ID, errMsg)
			metrics.RecordFailure(job.Type, duration)
			return
		}

		// Calculate next retry time for logging
		backoffMinutes := 1 << (job.Attempts - 1)
		nextRunAt := time.Now().Add(time.Duration(backoffMinutes) * time.Minute)
		log.Printf("Worker %s: job %s scheduled for retry at %v", w.ID, job.ID, nextRunAt)
		metrics.RecordRetry(job.Type)
		return
	}

	// Job exceeded max attempts, mark as failed
	if failErr := w.queue.Fail(ctx, job.ID, errMsg); failErr != nil {
		log.Printf("Worker %s: failed to mark job %s as failed: %v", w.ID, job.ID, failErr)
	}

	log.Printf("Worker %s: job %s exceeded max attempts, moved to dead letter queue", w.ID, job.ID)
	metrics.RecordFailure(job.Type, duration)
}

// HandlerRegistry manages job type handlers
type HandlerRegistry struct {
	handlers map[string]Handler
	mu       sync.RWMutex
}

// NewHandlerRegistry creates a new handler registry
func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		handlers: make(map[string]Handler),
	}
}

// Register adds a handler for a job type
func (r *HandlerRegistry) Register(jobType string, handler Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[jobType] = handler
	log.Printf("Registered handler for job type: %s", jobType)
}

// Get retrieves a handler for a job type
func (r *HandlerRegistry) Get(jobType string) (Handler, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handler, ok := r.handlers[jobType]
	if !ok {
		return nil, fmt.Errorf("no handler registered for job type: %s", jobType)
	}

	return handler, nil
}

// ListTypes returns all registered job types
func (r *HandlerRegistry) ListTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.handlers))
	for t := range r.handlers {
		types = append(types, t)
	}
	return types
}

// Metrics tracks job processing statistics
type Metrics struct {
	mu sync.RWMutex

	// Per job type metrics
	processed map[string]int64
	succeeded map[string]int64
	failed    map[string]int64
	retried   map[string]int64

	// Timing metrics
	totalDuration   map[string]time.Duration
	minDuration     map[string]time.Duration
	maxDuration     map[string]time.Duration
	avgDuration     map[string]time.Duration
}

// NewMetrics creates a new metrics tracker
func NewMetrics() *Metrics {
	return &Metrics{
		processed:     make(map[string]int64),
		succeeded:     make(map[string]int64),
		failed:        make(map[string]int64),
		retried:       make(map[string]int64),
		totalDuration: make(map[string]time.Duration),
		minDuration:   make(map[string]time.Duration),
		maxDuration:   make(map[string]time.Duration),
		avgDuration:   make(map[string]time.Duration),
	}
}

// RecordSuccess records a successful job execution
func (m *Metrics) RecordSuccess(jobType string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.processed[jobType]++
	m.succeeded[jobType]++
	m.updateDuration(jobType, duration)
}

// RecordFailure records a failed job execution
func (m *Metrics) RecordFailure(jobType string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.processed[jobType]++
	m.failed[jobType]++
	m.updateDuration(jobType, duration)
}

// RecordRetry records a job retry
func (m *Metrics) RecordRetry(jobType string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.retried[jobType]++
}

// updateDuration updates timing metrics
func (m *Metrics) updateDuration(jobType string, duration time.Duration) {
	// Update total
	m.totalDuration[jobType] += duration

	// Update min
	if min, ok := m.minDuration[jobType]; !ok || duration < min {
		m.minDuration[jobType] = duration
	}

	// Update max
	if max, ok := m.maxDuration[jobType]; !ok || duration > max {
		m.maxDuration[jobType] = duration
	}

	// Update average
	count := m.processed[jobType]
	if count > 0 {
		m.avgDuration[jobType] = m.totalDuration[jobType] / time.Duration(count)
	}
}

// GetStats returns statistics for a job type
func (m *Metrics) GetStats(jobType string) JobStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return JobStats{
		JobType:     jobType,
		Processed:   m.processed[jobType],
		Succeeded:   m.succeeded[jobType],
		Failed:      m.failed[jobType],
		Retried:     m.retried[jobType],
		MinDuration: m.minDuration[jobType],
		MaxDuration: m.maxDuration[jobType],
		AvgDuration: m.avgDuration[jobType],
	}
}

// GetAllStats returns statistics for all job types
func (m *Metrics) GetAllStats() map[string]JobStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]JobStats)
	for jobType := range m.processed {
		stats[jobType] = JobStats{
			JobType:     jobType,
			Processed:   m.processed[jobType],
			Succeeded:   m.succeeded[jobType],
			Failed:      m.failed[jobType],
			Retried:     m.retried[jobType],
			MinDuration: m.minDuration[jobType],
			MaxDuration: m.maxDuration[jobType],
			AvgDuration: m.avgDuration[jobType],
		}
	}
	return stats
}

// JobStats holds statistics for a specific job type
type JobStats struct {
	JobType     string        `json:"job_type"`
	Processed   int64         `json:"processed"`
	Succeeded   int64         `json:"succeeded"`
	Failed      int64         `json:"failed"`
	Retried     int64         `json:"retried"`
	MinDuration time.Duration `json:"min_duration"`
	MaxDuration time.Duration `json:"max_duration"`
	AvgDuration time.Duration `json:"avg_duration"`
}

// SuccessRate returns the success rate as a percentage
func (s JobStats) SuccessRate() float64 {
	if s.Processed == 0 {
		return 0
	}
	return float64(s.Succeeded) / float64(s.Processed) * 100
}
