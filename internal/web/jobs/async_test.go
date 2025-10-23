package jobs

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestNewAsyncExecutor(t *testing.T) {
	db, _, _ := setupMockDB(t)
	defer db.Close()

	queue := NewQueue(db)
	executor := NewAsyncExecutor(queue)

	assert.NotNil(t, executor)
	assert.NotNil(t, executor.queue)
}

func TestAsyncExecutorExecute(t *testing.T) {
	db, mock, _ := setupMockDB(t)
	defer db.Close()

	queue := NewQueue(db)
	executor := NewAsyncExecutor(queue)

	ctx := context.Background()
	queueName := "default"
	jobType := "email.send"
	payload := map[string]interface{}{
		"to":       "user@example.com",
		"template": "welcome",
	}

	// Mock Enqueue
	mock.ExpectExec(`INSERT INTO jobs`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := executor.Execute(ctx, queueName, jobType, payload)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAsyncExecutorExecuteWithPriority(t *testing.T) {
	db, mock, _ := setupMockDB(t)
	defer db.Close()

	queue := NewQueue(db)
	executor := NewAsyncExecutor(queue)

	ctx := context.Background()
	queueName := "default"
	jobType := "report.generate"
	payload := map[string]interface{}{
		"report_id": "123",
	}

	// Mock Enqueue
	mock.ExpectExec(`INSERT INTO jobs`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := executor.ExecuteWithPriority(ctx, queueName, jobType, payload, PriorityUrgent)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAsyncExecutorExecuteError(t *testing.T) {
	db, mock, _ := setupMockDB(t)
	defer db.Close()

	queue := NewQueue(db)
	executor := NewAsyncExecutor(queue)

	ctx := context.Background()
	queueName := "default"
	jobType := "email.send"
	payload := map[string]interface{}{"to": "user@example.com"}

	// Mock Enqueue failure
	mock.ExpectExec(`INSERT INTO jobs`).
		WillReturnError(sql.ErrConnDone)

	err := executor.Execute(ctx, queueName, jobType, payload)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to enqueue async job")
	assert.NoError(t, mock.ExpectationsWereMet())
}
