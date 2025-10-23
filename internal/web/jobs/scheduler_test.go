package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCronScheduler(t *testing.T) {
	db, _, _ := setupMockDB(t)
	defer db.Close()

	queue := NewQueue(db)
	scheduler := NewCronScheduler(queue)

	assert.NotNil(t, scheduler)
	assert.NotNil(t, scheduler.queue)
	assert.NotNil(t, scheduler.schedules)
}

func TestAddSchedule(t *testing.T) {
	db, _, _ := setupMockDB(t)
	defer db.Close()

	queue := NewQueue(db)
	scheduler := NewCronScheduler(queue)

	schedule := &Schedule{
		Queue:    "default",
		Type:     "test.job",
		Interval: 5 * time.Minute,
		Payload:  map[string]interface{}{"key": "value"},
	}

	err := scheduler.AddSchedule(schedule)
	assert.NoError(t, err)
	assert.NotEmpty(t, schedule.ID)
	assert.True(t, schedule.Enabled)
	assert.False(t, schedule.NextRun.IsZero())
}

func TestAddScheduleValidation(t *testing.T) {
	db, _, _ := setupMockDB(t)
	defer db.Close()

	queue := NewQueue(db)
	scheduler := NewCronScheduler(queue)

	// Test missing queue
	err := scheduler.AddSchedule(&Schedule{
		Type:     "test.job",
		Interval: 5 * time.Minute,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "queue name is required")

	// Test missing type
	err = scheduler.AddSchedule(&Schedule{
		Queue:    "default",
		Interval: 5 * time.Minute,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "job type is required")

	// Test invalid interval
	err = scheduler.AddSchedule(&Schedule{
		Queue:    "default",
		Type:     "test.job",
		Interval: 0,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "interval must be positive")
}

func TestRemoveSchedule(t *testing.T) {
	db, _, _ := setupMockDB(t)
	defer db.Close()

	queue := NewQueue(db)
	scheduler := NewCronScheduler(queue)

	schedule := &Schedule{
		ID:       "test-schedule",
		Queue:    "default",
		Type:     "test.job",
		Interval: 5 * time.Minute,
	}

	err := scheduler.AddSchedule(schedule)
	require.NoError(t, err)

	err = scheduler.RemoveSchedule(schedule.ID)
	assert.NoError(t, err)

	// Verify removed
	_, err = scheduler.GetSchedule(schedule.ID)
	assert.Error(t, err)
}

func TestEnableDisableSchedule(t *testing.T) {
	db, _, _ := setupMockDB(t)
	defer db.Close()

	queue := NewQueue(db)
	scheduler := NewCronScheduler(queue)

	schedule := &Schedule{
		ID:       "test-schedule",
		Queue:    "default",
		Type:     "test.job",
		Interval: 5 * time.Minute,
	}

	err := scheduler.AddSchedule(schedule)
	require.NoError(t, err)

	// Disable
	err = scheduler.DisableSchedule(schedule.ID)
	assert.NoError(t, err)

	retrieved, err := scheduler.GetSchedule(schedule.ID)
	assert.NoError(t, err)
	assert.False(t, retrieved.Enabled)

	// Enable
	err = scheduler.EnableSchedule(schedule.ID)
	assert.NoError(t, err)

	retrieved, err = scheduler.GetSchedule(schedule.ID)
	assert.NoError(t, err)
	assert.True(t, retrieved.Enabled)
}

func TestListSchedules(t *testing.T) {
	db, _, _ := setupMockDB(t)
	defer db.Close()

	queue := NewQueue(db)
	scheduler := NewCronScheduler(queue)

	// Add multiple schedules
	for i := 0; i < 3; i++ {
		err := scheduler.AddSchedule(&Schedule{
			Queue:    "default",
			Type:     "test.job",
			Interval: 5 * time.Minute,
		})
		require.NoError(t, err)
	}

	schedules := scheduler.ListSchedules()
	assert.Len(t, schedules, 3)
}

func TestGetSchedule(t *testing.T) {
	db, _, _ := setupMockDB(t)
	defer db.Close()

	queue := NewQueue(db)
	scheduler := NewCronScheduler(queue)

	schedule := &Schedule{
		ID:       "test-schedule",
		Queue:    "default",
		Type:     "test.job",
		Interval: 5 * time.Minute,
	}

	err := scheduler.AddSchedule(schedule)
	require.NoError(t, err)

	retrieved, err := scheduler.GetSchedule(schedule.ID)
	assert.NoError(t, err)
	assert.Equal(t, schedule.ID, retrieved.ID)
	assert.Equal(t, schedule.Queue, retrieved.Queue)
	assert.Equal(t, schedule.Type, retrieved.Type)
}

func TestGetScheduleNotFound(t *testing.T) {
	db, _, _ := setupMockDB(t)
	defer db.Close()

	queue := NewQueue(db)
	scheduler := NewCronScheduler(queue)

	_, err := scheduler.GetSchedule("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schedule not found")
}

func TestCheckSchedules(t *testing.T) {
	db, mock, _ := setupMockDB(t)
	defer db.Close()

	queue := NewQueue(db)
	scheduler := NewCronScheduler(queue)

	// Add schedule that should run now
	schedule := &Schedule{
		Queue:    "default",
		Type:     "test.job",
		Interval: 5 * time.Minute,
		Payload:  map[string]interface{}{"key": "value"},
	}
	schedule.NextRun = time.Now().Add(-1 * time.Second) // In the past
	schedule.Enabled = true
	scheduler.schedules["test"] = schedule

	// Mock Enqueue
	mock.ExpectExec(`INSERT INTO jobs`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Check schedules
	ctx := context.Background()
	scheduler.checkSchedules(ctx, time.Now())

	assert.NoError(t, mock.ExpectationsWereMet())

	// Verify schedule was updated
	assert.True(t, schedule.NextRun.After(time.Now()))
	assert.False(t, schedule.LastRun.IsZero())
}

func TestCheckSchedulesDisabled(t *testing.T) {
	db, mock, _ := setupMockDB(t)
	defer db.Close()

	queue := NewQueue(db)
	scheduler := NewCronScheduler(queue)

	// Add disabled schedule
	schedule := &Schedule{
		Queue:    "default",
		Type:     "test.job",
		Interval: 5 * time.Minute,
		Enabled:  false,
	}
	schedule.NextRun = time.Now().Add(-1 * time.Second)
	scheduler.schedules["test"] = schedule

	// Should not enqueue
	ctx := context.Background()
	scheduler.checkSchedules(ctx, time.Now())

	// Verify no DB calls were made
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSchedulerStartStop(t *testing.T) {
	db, _, _ := setupMockDB(t)
	defer db.Close()

	queue := NewQueue(db)
	scheduler := NewCronScheduler(queue)

	ctx := context.Background()
	scheduler.Start(ctx)

	// Let it run briefly
	time.Sleep(100 * time.Millisecond)

	scheduler.Stop()
}

func TestScheduleHelpers(t *testing.T) {
	// Test ScheduleEvery
	s1 := ScheduleEvery(5*time.Minute, "default", "test", nil)
	assert.Equal(t, 5*time.Minute, s1.Interval)
	assert.Equal(t, "default", s1.Queue)
	assert.NotEmpty(t, s1.ID)

	// Test ScheduleEveryMinutes
	s2 := ScheduleEveryMinutes(10, "default", "test", nil)
	assert.Equal(t, 10*time.Minute, s2.Interval)

	// Test ScheduleEveryHours
	s3 := ScheduleEveryHours(2, "default", "test", nil)
	assert.Equal(t, 2*time.Hour, s3.Interval)

	// Test ScheduleDaily
	s4 := ScheduleDaily("default", "test", nil)
	assert.Equal(t, 24*time.Hour, s4.Interval)

	// Test ScheduleWeekly
	s5 := ScheduleWeekly("default", "test", nil)
	assert.Equal(t, 7*24*time.Hour, s5.Interval)
}
