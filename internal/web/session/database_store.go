package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// DatabaseStore is a database-backed session store
type DatabaseStore struct {
	db        *sql.DB
	tableName string
	stopChan  chan struct{}
	wg        sync.WaitGroup
}

// DatabaseConfig holds database session store configuration
type DatabaseConfig struct {
	// DB is the database connection
	DB *sql.DB

	// TableName is the name of the sessions table
	TableName string

	// CleanupInterval is how often to run cleanup (0 = no auto cleanup)
	CleanupInterval time.Duration
}

// DefaultDatabaseConfig returns default database configuration
func DefaultDatabaseConfig(db *sql.DB) *DatabaseConfig {
	return &DatabaseConfig{
		DB:              db,
		TableName:       "sessions",
		CleanupInterval: 5 * time.Minute,
	}
}

// NewDatabaseStore creates a new database session store
func NewDatabaseStore(config *DatabaseConfig) (*DatabaseStore, error) {
	store := &DatabaseStore{
		db:        config.DB,
		tableName: config.TableName,
		stopChan:  make(chan struct{}),
	}

	// Create table if it doesn't exist
	if err := store.createTable(); err != nil {
		return nil, fmt.Errorf("failed to create sessions table: %w", err)
	}

	// Start cleanup goroutine if enabled
	if config.CleanupInterval > 0 {
		store.wg.Add(1)
		go store.cleanup(config.CleanupInterval)
	}

	return store, nil
}

// createTable creates the sessions table if it doesn't exist
func (s *DatabaseStore) createTable() error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id VARCHAR(255) PRIMARY KEY,
			user_id VARCHAR(255),
			data JSONB NOT NULL,
			flash_messages JSONB,
			csrf_token VARCHAR(255),
			created_at TIMESTAMP NOT NULL,
			expires_at TIMESTAMP NOT NULL
		)
	`, s.tableName)

	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}

	// Create index on expires_at for efficient cleanup
	indexQuery := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS idx_%s_expires_at ON %s (expires_at)
	`, s.tableName, s.tableName)

	_, err = s.db.Exec(indexQuery)
	return err
}

// Get retrieves a session from the database
func (s *DatabaseStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	query := fmt.Sprintf(`
		SELECT id, user_id, data, flash_messages, csrf_token, created_at, expires_at
		FROM %s
		WHERE id = $1 AND expires_at > $2
	`, s.tableName)

	var session Session
	var userID sql.NullString
	var dataJSON, flashJSON, csrfToken sql.NullString

	err := s.db.QueryRowContext(ctx, query, sessionID, time.Now()).Scan(
		&session.ID,
		&userID,
		&dataJSON,
		&flashJSON,
		&csrfToken,
		&session.CreatedAt,
		&session.ExpiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("database query error: %w", err)
	}

	// Set user ID
	if userID.Valid {
		session.UserID = userID.String
	}

	// Set CSRF token
	if csrfToken.Valid {
		session.CSRFToken = csrfToken.String
	}

	// Unmarshal data
	if dataJSON.Valid {
		if err := json.Unmarshal([]byte(dataJSON.String), &session.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
		}
	} else {
		session.Data = make(map[string]interface{})
	}

	// Unmarshal flash messages
	if flashJSON.Valid {
		if err := json.Unmarshal([]byte(flashJSON.String), &session.FlashMessages); err != nil {
			return nil, fmt.Errorf("failed to unmarshal flash messages: %w", err)
		}
	} else {
		session.FlashMessages = []FlashMessage{}
	}

	return &session, nil
}

// Set stores a session in the database
func (s *DatabaseStore) Set(ctx context.Context, sessionID string, session *Session, ttl time.Duration) error {
	// Marshal data
	dataJSON, err := json.Marshal(session.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	// Marshal flash messages
	flashJSON, err := json.Marshal(session.FlashMessages)
	if err != nil {
		return fmt.Errorf("failed to marshal flash messages: %w", err)
	}

	// Update expires_at based on TTL
	session.ExpiresAt = time.Now().Add(ttl)

	query := fmt.Sprintf(`
		INSERT INTO %s (id, user_id, data, flash_messages, csrf_token, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			data = EXCLUDED.data,
			flash_messages = EXCLUDED.flash_messages,
			csrf_token = EXCLUDED.csrf_token,
			expires_at = EXCLUDED.expires_at
	`, s.tableName)

	var userID interface{}
	if session.UserID != "" {
		userID = session.UserID
	}

	var csrfToken interface{}
	if session.CSRFToken != "" {
		csrfToken = session.CSRFToken
	}

	_, err = s.db.ExecContext(ctx, query,
		session.ID,
		userID,
		dataJSON,
		flashJSON,
		csrfToken,
		session.CreatedAt,
		session.ExpiresAt,
	)

	if err != nil {
		return fmt.Errorf("database insert error: %w", err)
	}

	return nil
}

// Delete removes a session from the database
func (s *DatabaseStore) Delete(ctx context.Context, sessionID string) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, s.tableName)

	_, err := s.db.ExecContext(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("database delete error: %w", err)
	}

	return nil
}

// Refresh updates the expiration time of a session
func (s *DatabaseStore) Refresh(ctx context.Context, sessionID string, ttl time.Duration) error {
	expiresAt := time.Now().Add(ttl)
	query := fmt.Sprintf(`
		UPDATE %s SET expires_at = $1 WHERE id = $2 AND expires_at > $3
	`, s.tableName)

	result, err := s.db.ExecContext(ctx, query, expiresAt, sessionID, time.Now())
	if err != nil {
		return fmt.Errorf("database update error: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrSessionNotFound
	}

	return nil
}

// Close stops the cleanup goroutine and waits for it to finish
func (s *DatabaseStore) Close() error {
	if s.stopChan != nil {
		close(s.stopChan)
		s.wg.Wait()
	}
	// We don't close the DB as it's managed externally
	return nil
}

// cleanup periodically removes expired sessions
func (s *DatabaseStore) cleanup(interval time.Duration) {
	defer s.wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			query := fmt.Sprintf(`DELETE FROM %s WHERE expires_at <= $1`, s.tableName)
			_, err := s.db.Exec(query, time.Now())
			if err != nil {
				// Log error but continue
				fmt.Printf("Session cleanup error: %v\n", err)
			}
		}
	}
}
