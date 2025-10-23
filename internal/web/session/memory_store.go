package session

import (
	"context"
	"sync"
	"time"
)

// MemoryStore is an in-memory session store suitable for development
type MemoryStore struct {
	sessions sync.Map
	stopChan chan struct{}
	wg       sync.WaitGroup
}

type sessionEntry struct {
	session   *Session
	expiresAt time.Time
}

// NewMemoryStore creates a new in-memory session store
func NewMemoryStore() *MemoryStore {
	store := &MemoryStore{
		stopChan: make(chan struct{}),
	}

	// Start cleanup goroutine
	store.wg.Add(1)
	go store.cleanup()

	return store
}

// Get retrieves a session from memory
func (s *MemoryStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	value, ok := s.sessions.Load(sessionID)
	if !ok {
		return nil, ErrSessionNotFound
	}

	entry := value.(*sessionEntry)
	if entry.expiresAt.Before(time.Now()) {
		s.sessions.Delete(sessionID)
		return nil, ErrSessionExpired
	}

	return entry.session, nil
}

// Set stores a session in memory
func (s *MemoryStore) Set(ctx context.Context, sessionID string, session *Session, ttl time.Duration) error {
	entry := &sessionEntry{
		session:   session,
		expiresAt: time.Now().Add(ttl),
	}
	s.sessions.Store(sessionID, entry)
	return nil
}

// Delete removes a session from memory
func (s *MemoryStore) Delete(ctx context.Context, sessionID string) error {
	s.sessions.Delete(sessionID)
	return nil
}

// Refresh updates the expiration time of a session
func (s *MemoryStore) Refresh(ctx context.Context, sessionID string, ttl time.Duration) error {
	value, ok := s.sessions.Load(sessionID)
	if !ok {
		return ErrSessionNotFound
	}

	entry := value.(*sessionEntry)
	entry.expiresAt = time.Now().Add(ttl)
	return nil
}

// Close stops the cleanup goroutine and clears all sessions
func (s *MemoryStore) Close() error {
	close(s.stopChan)
	s.wg.Wait()
	s.sessions.Range(func(key, value interface{}) bool {
		s.sessions.Delete(key)
		return true
	})
	return nil
}

// cleanup periodically removes expired sessions
func (s *MemoryStore) cleanup() {
	defer s.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			now := time.Now()
			s.sessions.Range(func(key, value interface{}) bool {
				entry := value.(*sessionEntry)
				if entry.expiresAt.Before(now) {
					s.sessions.Delete(key)
				}
				return true
			})
		}
	}
}

// Count returns the number of active sessions (for testing/monitoring)
func (s *MemoryStore) Count() int {
	count := 0
	s.sessions.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}
