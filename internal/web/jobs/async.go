package jobs

import (
	"context"
	"fmt"
	"log"
)

// AsyncExecutor provides integration with @async lifecycle hooks
// This allows Conduit's lifecycle hooks to enqueue background jobs
type AsyncExecutor struct {
	queue *Queue
}

// NewAsyncExecutor creates a new async executor
func NewAsyncExecutor(queue *Queue) *AsyncExecutor {
	return &AsyncExecutor{
		queue: queue,
	}
}

// Execute enqueues a job to be executed asynchronously
// This is called by the generated code from @async blocks in lifecycle hooks
func (e *AsyncExecutor) Execute(ctx context.Context, queueName, jobType string, payload map[string]interface{}) error {
	job := NewJob(queueName, jobType, payload)

	if err := e.queue.Enqueue(ctx, job); err != nil {
		return fmt.Errorf("failed to enqueue async job: %w", err)
	}

	log.Printf("Enqueued async job %s (type: %s) to queue: %s", job.ID, job.Type, queueName)
	return nil
}

// ExecuteWithPriority enqueues a job with a specific priority
func (e *AsyncExecutor) ExecuteWithPriority(ctx context.Context, queueName, jobType string, payload map[string]interface{}, priority JobPriority) error {
	job := NewJob(queueName, jobType, payload)
	job.Priority = priority

	if err := e.queue.Enqueue(ctx, job); err != nil {
		return fmt.Errorf("failed to enqueue async job: %w", err)
	}

	log.Printf("Enqueued async job %s (type: %s, priority: %d) to queue: %s",
		job.ID, job.Type, priority, queueName)
	return nil
}

// AsyncContext provides context for async job execution
// This is passed to job handlers to provide access to the job metadata
type AsyncContext struct {
	JobID    string
	JobType  string
	Queue    string
	Attempts int
}

// Example usage in generated code:
//
// // From Conduit lifecycle hook:
// // @after create @transaction {
// //   @async {
// //     Email.send(user, "welcome")
// //   }
// // }
//
// // Generated Go code:
// func (r *UserResource) AfterCreate(ctx context.Context, user *User) error {
//     // ... transaction logic ...
//
//     // Enqueue async job
//     asyncExec := jobs.NewAsyncExecutor(jobQueue)
//     payload := map[string]interface{}{
//         "template": "welcome",
//         "user_id": user.ID,
//         "user_email": user.Email,
//     }
//
//     return asyncExec.Execute(ctx, "default", "email.send", payload)
// }
