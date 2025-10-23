package jobs

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// CronScheduler manages scheduled jobs with cron-like functionality
type CronScheduler struct {
	queue     *Queue
	schedules map[string]*Schedule
	stopChan  chan struct{}
	wg        sync.WaitGroup
	mu        sync.RWMutex
}

// Schedule defines a recurring job schedule
type Schedule struct {
	ID       string
	Queue    string
	Type     string
	Payload  map[string]interface{}
	Interval time.Duration
	Enabled  bool
	LastRun  time.Time
	NextRun  time.Time
}

// NewCronScheduler creates a new cron scheduler
func NewCronScheduler(queue *Queue) *CronScheduler {
	return &CronScheduler{
		queue:     queue,
		schedules: make(map[string]*Schedule),
		stopChan:  make(chan struct{}),
	}
}

// AddSchedule adds a new recurring job schedule
func (s *CronScheduler) AddSchedule(schedule *Schedule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if schedule.ID == "" {
		schedule.ID = uuid.New().String()
	}

	if schedule.Interval <= 0 {
		return fmt.Errorf("interval must be positive")
	}

	if schedule.Queue == "" {
		return fmt.Errorf("queue name is required")
	}

	if schedule.Type == "" {
		return fmt.Errorf("job type is required")
	}

	// Set initial next run time
	schedule.NextRun = time.Now().Add(schedule.Interval)
	schedule.Enabled = true

	s.schedules[schedule.ID] = schedule
	log.Printf("Added schedule %s: %s every %v", schedule.ID, schedule.Type, schedule.Interval)

	return nil
}

// RemoveSchedule removes a recurring job schedule
func (s *CronScheduler) RemoveSchedule(scheduleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.schedules[scheduleID]; !ok {
		return fmt.Errorf("schedule not found: %s", scheduleID)
	}

	delete(s.schedules, scheduleID)
	log.Printf("Removed schedule %s", scheduleID)

	return nil
}

// EnableSchedule enables a schedule
func (s *CronScheduler) EnableSchedule(scheduleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	schedule, ok := s.schedules[scheduleID]
	if !ok {
		return fmt.Errorf("schedule not found: %s", scheduleID)
	}

	schedule.Enabled = true
	log.Printf("Enabled schedule %s", scheduleID)

	return nil
}

// DisableSchedule disables a schedule
func (s *CronScheduler) DisableSchedule(scheduleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	schedule, ok := s.schedules[scheduleID]
	if !ok {
		return fmt.Errorf("schedule not found: %s", scheduleID)
	}

	schedule.Enabled = false
	log.Printf("Disabled schedule %s", scheduleID)

	return nil
}

// ListSchedules returns all schedules
func (s *CronScheduler) ListSchedules() []*Schedule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	schedules := make([]*Schedule, 0, len(s.schedules))
	for _, schedule := range s.schedules {
		schedules = append(schedules, schedule)
	}

	return schedules
}

// GetSchedule retrieves a schedule by ID
func (s *CronScheduler) GetSchedule(scheduleID string) (*Schedule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	schedule, ok := s.schedules[scheduleID]
	if !ok {
		return nil, fmt.Errorf("schedule not found: %s", scheduleID)
	}

	return schedule, nil
}

// Start starts the scheduler
func (s *CronScheduler) Start(ctx context.Context) {
	s.wg.Add(1)
	go s.run(ctx)
	log.Printf("Cron scheduler started")
}

// Stop stops the scheduler
func (s *CronScheduler) Stop() {
	close(s.stopChan)
	s.wg.Wait()
	log.Printf("Cron scheduler stopped")
}

// run is the main scheduler loop
func (s *CronScheduler) run(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case now := <-ticker.C:
			s.checkSchedules(ctx, now)
		}
	}
}

// checkSchedules checks and enqueues due scheduled jobs
func (s *CronScheduler) checkSchedules(ctx context.Context, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, schedule := range s.schedules {
		if !schedule.Enabled {
			continue
		}

		if now.After(schedule.NextRun) || now.Equal(schedule.NextRun) {
			if err := s.enqueueScheduledJob(ctx, schedule); err != nil {
				log.Printf("Failed to enqueue scheduled job %s: %v", schedule.ID, err)
				continue
			}

			// Update schedule timing
			schedule.LastRun = now
			schedule.NextRun = now.Add(schedule.Interval)

			log.Printf("Enqueued scheduled job %s (type: %s), next run at %v",
				schedule.ID, schedule.Type, schedule.NextRun)
		}
	}
}

// enqueueScheduledJob creates and enqueues a job from a schedule
func (s *CronScheduler) enqueueScheduledJob(ctx context.Context, schedule *Schedule) error {
	job := NewJob(schedule.Queue, schedule.Type, schedule.Payload)
	job.Priority = PriorityNormal

	return s.queue.Enqueue(ctx, job)
}

// ScheduleEvery is a helper to create a schedule with a simple interval
func ScheduleEvery(interval time.Duration, queue, jobType string, payload map[string]interface{}) *Schedule {
	return &Schedule{
		ID:       uuid.New().String(),
		Queue:    queue,
		Type:     jobType,
		Payload:  payload,
		Interval: interval,
	}
}

// ScheduleEveryMinutes creates a schedule that runs every N minutes
func ScheduleEveryMinutes(minutes int, queue, jobType string, payload map[string]interface{}) *Schedule {
	return ScheduleEvery(time.Duration(minutes)*time.Minute, queue, jobType, payload)
}

// ScheduleEveryHours creates a schedule that runs every N hours
func ScheduleEveryHours(hours int, queue, jobType string, payload map[string]interface{}) *Schedule {
	return ScheduleEvery(time.Duration(hours)*time.Hour, queue, jobType, payload)
}

// ScheduleDaily creates a schedule that runs once per day
func ScheduleDaily(queue, jobType string, payload map[string]interface{}) *Schedule {
	return ScheduleEvery(24*time.Hour, queue, jobType, payload)
}

// ScheduleWeekly creates a schedule that runs once per week
func ScheduleWeekly(queue, jobType string, payload map[string]interface{}) *Schedule {
	return ScheduleEvery(7*24*time.Hour, queue, jobType, payload)
}
