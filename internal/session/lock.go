package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

const lockFile = "agent.lock"

// LockInfo contains information about the current lock holder
type LockInfo struct {
	PID       int       `json:"pid"`
	Hostname  string    `json:"hostname"`
	StartedAt time.Time `json:"started_at"`
}

// SessionLock represents a lock on a session to prevent concurrent agent runs
type SessionLock struct {
	sessionID  string
	projectDir string
	config     StoreConfig
	lockPath   string
	fileLock   *flock.Flock
}

// AcquireSessionLock attempts to acquire an exclusive lock on the session.
// Returns a SessionLock on success, or an error if the session is already locked.
// Special case: "_all" is a virtual session for the "all" meta-session and skips
// session verification (used by "juggle agent run all").
func (s *SessionStore) AcquireSessionLock(sessionID string) (*SessionLock, error) {
	// Verify session exists (skip for "_all" virtual session)
	if sessionID != "_all" {
		if _, err := s.LoadSession(sessionID); err != nil {
			return nil, err
		}
	} else {
		// For "_all", ensure the directory exists
		sessionDir := s.sessionPath(sessionID)
		if err := os.MkdirAll(sessionDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create _all session directory: %w", err)
		}
	}

	lockPath := filepath.Join(s.sessionPath(sessionID), lockFile)

	// Create the flock
	fileLock := flock.New(lockPath)

	// Try to acquire exclusive lock (non-blocking)
	locked, err := fileLock.TryLock()
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !locked {
		// Lock is held by another process - read lock info
		info, _ := readLockInfo(lockPath)
		return nil, NewSessionLockedError(sessionID, info)
	}

	// Write lock info to a separate info file
	hostname, _ := os.Hostname()
	info := LockInfo{
		PID:       os.Getpid(),
		Hostname:  hostname,
		StartedAt: time.Now(),
	}

	data, err := json.Marshal(info)
	if err != nil {
		fileLock.Unlock()
		return nil, fmt.Errorf("failed to marshal lock info: %w", err)
	}

	// Write lock info to the lock file
	if err := os.WriteFile(lockPath, data, 0644); err != nil {
		fileLock.Unlock()
		return nil, fmt.Errorf("failed to write lock info: %w", err)
	}

	return &SessionLock{
		sessionID:  sessionID,
		projectDir: s.projectDir,
		config:     s.config,
		lockPath:   lockPath,
		fileLock:   fileLock,
	}, nil
}

// Release releases the session lock
func (l *SessionLock) Release() error {
	if l.fileLock == nil {
		return nil // Already released
	}

	// Release the flock
	if err := l.fileLock.Unlock(); err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	// Remove the lock file (best-effort cleanup - file may already be gone or
	// we may lack permissions, but the OS-level lock is already released)
	_ = os.Remove(l.lockPath)

	l.fileLock = nil
	return nil
}

// IsLocked checks if a session currently has an active lock
func (s *SessionStore) IsLocked(sessionID string) (bool, *LockInfo) {
	lockPath := filepath.Join(s.sessionPath(sessionID), lockFile)

	// Check if lock file exists
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		return false, nil
	}

	// Try to acquire lock (non-blocking)
	fileLock := flock.New(lockPath)
	locked, err := fileLock.TryLock()
	if err != nil {
		// Error acquiring lock - assume it's locked
		info, _ := readLockInfo(lockPath)
		return true, info
	}

	if !locked {
		// Lock is held by another process
		info, _ := readLockInfo(lockPath)
		return true, info
	}

	// We got the lock - release it immediately
	fileLock.Unlock()
	return false, nil
}

// readLockInfo reads the lock info from a lock file
func readLockInfo(lockPath string) (*LockInfo, error) {
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return nil, err
	}

	var info LockInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	return &info, nil
}
