package session

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// Standard error types for the session package.
// These errors can be checked using errors.Is() and errors.As().

var (
	// ErrBallNotFound is returned when a ball cannot be found by ID.
	ErrBallNotFound = errors.New("ball not found")

	// ErrInvalidState is returned when an invalid state or state transition is attempted.
	ErrInvalidState = errors.New("invalid state")

	// ErrSessionLocked is returned when a session is already locked by another process.
	ErrSessionLocked = errors.New("session locked")

	// ErrBallLocked is returned when a ball is already locked by another process.
	ErrBallLocked = errors.New("ball locked")
)

// BallNotFoundError provides detailed information about a ball lookup failure.
type BallNotFoundError struct {
	ID       string // The ball ID that was not found
	IsShort  bool   // True if the ID was a short ID
	IsPrefix bool   // True if the ID was a prefix match
}

func (e *BallNotFoundError) Error() string {
	if e.IsShort {
		return fmt.Sprintf("ball with short ID %s not found", e.ID)
	}
	if e.IsPrefix {
		return fmt.Sprintf("ball matching prefix %s not found", e.ID)
	}
	return fmt.Sprintf("ball %s not found", e.ID)
}

func (e *BallNotFoundError) Is(target error) bool {
	return target == ErrBallNotFound
}

// NewBallNotFoundError creates a new BallNotFoundError.
func NewBallNotFoundError(id string) *BallNotFoundError {
	return &BallNotFoundError{ID: id}
}

// NewBallNotFoundShortError creates a BallNotFoundError for a short ID.
func NewBallNotFoundShortError(shortID string) *BallNotFoundError {
	return &BallNotFoundError{ID: shortID, IsShort: true}
}

// NewBallNotFoundPrefixError creates a BallNotFoundError for a prefix match.
func NewBallNotFoundPrefixError(prefix string) *BallNotFoundError {
	return &BallNotFoundError{ID: prefix, IsPrefix: true}
}

// InvalidStateError provides detailed information about an invalid state error.
type InvalidStateError struct {
	State    string // The invalid state value
	From     string // The source state (for transition errors)
	To       string // The target state (for transition errors)
	Reason   string // Additional context about why it's invalid
	ValidSet []string // List of valid states (if applicable)
}

func (e *InvalidStateError) Error() string {
	if e.From != "" && e.To != "" {
		return fmt.Sprintf("invalid state transition from %s to %s", e.From, e.To)
	}
	if e.Reason != "" {
		return fmt.Sprintf("invalid state %s: %s", e.State, e.Reason)
	}
	if len(e.ValidSet) > 0 {
		return fmt.Sprintf("invalid state %s (valid: %v)", e.State, e.ValidSet)
	}
	return fmt.Sprintf("invalid state: %s", e.State)
}

func (e *InvalidStateError) Is(target error) bool {
	return target == ErrInvalidState
}

// NewInvalidStateError creates a new InvalidStateError for an invalid state value.
func NewInvalidStateError(state string, validStates []string) *InvalidStateError {
	return &InvalidStateError{State: state, ValidSet: validStates}
}

// NewInvalidStateTransitionError creates an InvalidStateError for an invalid state transition.
func NewInvalidStateTransitionError(from, to string) *InvalidStateError {
	return &InvalidStateError{From: from, To: to}
}

// SessionLockedError provides detailed information about a lock conflict.
type SessionLockedError struct {
	SessionID  string    // The session that is locked
	PID        int       // The process ID holding the lock (0 if unknown)
	Hostname   string    // The hostname where the lock was acquired
	ProcessRunning *bool // True if PID is still running, false if dead, nil if unknown
}

func (e *SessionLockedError) Error() string {
	var msg string
	if e.PID > 0 {
		status := ""
		if e.ProcessRunning != nil {
			if *e.ProcessRunning {
				status = " (process running)"
			} else {
				status = " (process not running - stale lock?)"
			}
		}
		if e.Hostname != "" {
			msg = fmt.Sprintf("session %s is already locked by PID %d on %s%s",
				e.SessionID, e.PID, e.Hostname, status)
		} else {
			msg = fmt.Sprintf("session %s is already locked by PID %d%s", e.SessionID, e.PID, status)
		}
	} else {
		msg = fmt.Sprintf("session %s is already locked by another agent", e.SessionID)
	}
	return msg + "\nUse --ignore-lock to bypass (use with caution)"
}

func (e *SessionLockedError) Is(target error) bool {
	return target == ErrSessionLocked
}

// NewSessionLockedError creates a new SessionLockedError.
func NewSessionLockedError(sessionID string, info *LockInfo) *SessionLockedError {
	err := &SessionLockedError{SessionID: sessionID}
	if info != nil {
		err.PID = info.PID
		err.Hostname = info.Hostname
		// Check if the process is still running (local host only)
		currentHostname, _ := os.Hostname()
		if info.Hostname == currentHostname && info.PID > 0 {
			running := isProcessRunning(info.PID)
			err.ProcessRunning = &running
		}
	}
	return err
}

// BallLockedError provides detailed information about a ball lock conflict.
type BallLockedError struct {
	BallID         string // The ball that is locked
	PID            int    // The process ID holding the lock (0 if unknown)
	Hostname       string // The hostname where the lock was acquired
	ProcessRunning *bool  // True if PID is still running, false if dead, nil if unknown
}

func (e *BallLockedError) Error() string {
	var msg string
	if e.PID > 0 {
		status := ""
		if e.ProcessRunning != nil {
			if *e.ProcessRunning {
				status = " (process running)"
			} else {
				status = " (process not running - stale lock?)"
			}
		}
		if e.Hostname != "" {
			msg = fmt.Sprintf("ball %s is already locked by PID %d on %s%s",
				e.BallID, e.PID, e.Hostname, status)
		} else {
			msg = fmt.Sprintf("ball %s is already locked by PID %d%s", e.BallID, e.PID, status)
		}
	} else {
		msg = fmt.Sprintf("ball %s is already locked by another agent", e.BallID)
	}
	return msg + "\nUse --ignore-lock to bypass (use with caution)"
}

func (e *BallLockedError) Is(target error) bool {
	return target == ErrBallLocked
}

// NewBallLockedError creates a new BallLockedError.
func NewBallLockedError(ballID string, info *LockInfo) *BallLockedError {
	err := &BallLockedError{BallID: ballID}
	if info != nil {
		err.PID = info.PID
		err.Hostname = info.Hostname
		// Check if the process is still running (local host only)
		currentHostname, _ := os.Hostname()
		if info.Hostname == currentHostname && info.PID > 0 {
			running := isProcessRunning(info.PID)
			err.ProcessRunning = &running
		}
	}
	return err
}

// isProcessRunning checks if a process with the given PID is still running.
// This works by sending signal 0 to the process - if the process exists,
// the call succeeds; if not, it returns an error.
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 doesn't actually send a signal, but checks if the process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// AmbiguousIDError is returned when a ball ID prefix matches multiple balls.
type AmbiguousIDError struct {
	Prefix     string   // The ambiguous prefix
	MatchCount int      // Number of balls matched
	MatchIDs   []string // IDs of matching balls
}

func (e *AmbiguousIDError) Error() string {
	if len(e.MatchIDs) > 0 {
		return fmt.Sprintf("ambiguous ID '%s' matches %d balls: %v", e.Prefix, e.MatchCount, e.MatchIDs)
	}
	return fmt.Sprintf("ambiguous ID '%s' matches %d balls", e.Prefix, e.MatchCount)
}

// NewAmbiguousIDError creates a new AmbiguousIDError.
func NewAmbiguousIDError(prefix string, matchingIDs []string) *AmbiguousIDError {
	return &AmbiguousIDError{
		Prefix:     prefix,
		MatchCount: len(matchingIDs),
		MatchIDs:   matchingIDs,
	}
}
