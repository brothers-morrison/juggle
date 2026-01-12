package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
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
	file       *os.File
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

	// Open/create the lock file
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}

	// Try to acquire exclusive lock (non-blocking)
	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		file.Close()
		// Lock is held by another process - read lock info
		info, readErr := readLockInfo(lockPath)
		if readErr == nil && info != nil {
			return nil, fmt.Errorf("session %s is already locked by PID %d (started %s ago on %s)",
				sessionID, info.PID, time.Since(info.StartedAt).Round(time.Second), info.Hostname)
		}
		return nil, fmt.Errorf("session %s is already locked by another agent", sessionID)
	}

	// Write lock info
	hostname, _ := os.Hostname()
	info := LockInfo{
		PID:       os.Getpid(),
		Hostname:  hostname,
		StartedAt: time.Now(),
	}

	data, err := json.Marshal(info)
	if err != nil {
		syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		file.Close()
		os.Remove(lockPath)
		return nil, fmt.Errorf("failed to marshal lock info: %w", err)
	}

	// Truncate and write
	if err := file.Truncate(0); err != nil {
		syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		file.Close()
		return nil, fmt.Errorf("failed to truncate lock file: %w", err)
	}

	if _, err := file.WriteAt(data, 0); err != nil {
		syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		file.Close()
		return nil, fmt.Errorf("failed to write lock info: %w", err)
	}

	return &SessionLock{
		sessionID:  sessionID,
		projectDir: s.projectDir,
		config:     s.config,
		lockPath:   lockPath,
		file:       file,
	}, nil
}

// Release releases the session lock
func (l *SessionLock) Release() error {
	if l.file == nil {
		return nil // Already released
	}

	// Release the flock
	err := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	if err != nil {
		l.file.Close()
		return fmt.Errorf("failed to release lock: %w", err)
	}

	// Close the file
	if err := l.file.Close(); err != nil {
		return fmt.Errorf("failed to close lock file: %w", err)
	}

	// Remove the lock file
	os.Remove(l.lockPath)

	l.file = nil
	return nil
}

// IsLocked checks if a session currently has an active lock
func (s *SessionStore) IsLocked(sessionID string) (bool, *LockInfo) {
	lockPath := filepath.Join(s.sessionPath(sessionID), lockFile)

	// Try to open and lock the file
	file, err := os.OpenFile(lockPath, os.O_RDWR, 0644)
	if err != nil {
		// File doesn't exist or can't be opened - not locked
		return false, nil
	}
	defer file.Close()

	// Try to acquire lock (non-blocking)
	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		// Lock is held by another process
		info, _ := readLockInfo(lockPath)
		return true, info
	}

	// We got the lock - release it immediately
	syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
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
