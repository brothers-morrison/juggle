// Package daemon provides infrastructure for running the agent loop as a background daemon
// with file-based control and state communication.
package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

const (
	pidFileName   = "agent.pid"
	ctrlFileName  = "agent.ctrl"
	stateFileName = "agent.state"
)

// Info contains information about a running daemon
type Info struct {
	PID           int       `json:"pid"`
	SessionID     string    `json:"session_id"`
	ProjectDir    string    `json:"project_dir"`
	StartedAt     time.Time `json:"started_at"`
	MaxIterations int       `json:"max_iterations"`
	Model         string    `json:"model"`
	Provider      string    `json:"provider"`
}

// State represents the current state of the daemon, updated each iteration
type State struct {
	Running          bool      `json:"running"`
	Paused           bool      `json:"paused"`
	CurrentBallID    string    `json:"current_ball_id"`
	CurrentBallTitle string    `json:"current_ball_title"`
	Iteration        int       `json:"iteration"`
	MaxIterations    int       `json:"max_iterations"`
	FilesChanged     int       `json:"files_changed"`
	ACsComplete      int       `json:"acs_complete"`
	ACsTotal         int       `json:"acs_total"`
	Model            string    `json:"model"`
	Provider         string    `json:"provider"`
	LastUpdated      time.Time `json:"last_updated"`
	StartedAt        time.Time `json:"started_at"`
}

// Control represents a command sent to the daemon via the control file
type Control struct {
	Command   string    `json:"command"`   // pause, resume, cancel, skip_ball, change_model
	Args      string    `json:"args"`      // e.g., model name for change_model
	Timestamp time.Time `json:"timestamp"`
}

// Command constants
const (
	CmdPause       = "pause"
	CmdResume      = "resume"
	CmdCancel      = "cancel"
	CmdSkipBall    = "skip_ball"
	CmdChangeModel = "change_model"
)

// sessionDir returns the session directory path
func sessionDir(projectDir, sessionID string) string {
	return filepath.Join(projectDir, ".juggle", "sessions", sessionID)
}

// GetPIDFilePath returns the path to the PID file for a session
func GetPIDFilePath(projectDir, sessionID string) string {
	return filepath.Join(sessionDir(projectDir, sessionID), pidFileName)
}

// GetControlFilePath returns the path to the control file for a session
func GetControlFilePath(projectDir, sessionID string) string {
	return filepath.Join(sessionDir(projectDir, sessionID), ctrlFileName)
}

// GetStateFilePath returns the path to the state file for a session
func GetStateFilePath(projectDir, sessionID string) string {
	return filepath.Join(sessionDir(projectDir, sessionID), stateFileName)
}

// WritePIDFile creates a PID file for the running daemon
func WritePIDFile(projectDir, sessionID string, info *Info) error {
	// Ensure session directory exists
	dir := sessionDir(projectDir, sessionID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	path := GetPIDFilePath(projectDir, sessionID)
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal daemon info: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// ReadPIDFile reads the PID file for a session
func ReadPIDFile(projectDir, sessionID string) (*Info, error) {
	path := GetPIDFilePath(projectDir, sessionID)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var info Info
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to parse PID file: %w", err)
	}
	return &info, nil
}

// RemovePIDFile removes the PID file for a session
func RemovePIDFile(projectDir, sessionID string) error {
	path := GetPIDFilePath(projectDir, sessionID)
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil // Already gone
	}
	return err
}

// WriteStateFile writes the daemon state to disk
func WriteStateFile(projectDir, sessionID string, state *State) error {
	path := GetStateFilePath(projectDir, sessionID)
	state.LastUpdated = time.Now()
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal daemon state: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// ReadStateFile reads the daemon state from disk
func ReadStateFile(projectDir, sessionID string) (*State, error) {
	path := GetStateFilePath(projectDir, sessionID)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}
	return &state, nil
}

// RemoveStateFile removes the state file for a session
func RemoveStateFile(projectDir, sessionID string) error {
	path := GetStateFilePath(projectDir, sessionID)
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// SendControlCommand writes a control command to the control file
func SendControlCommand(projectDir, sessionID, command, args string) error {
	// Ensure session directory exists
	dir := sessionDir(projectDir, sessionID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	path := GetControlFilePath(projectDir, sessionID)
	ctrl := Control{
		Command:   command,
		Args:      args,
		Timestamp: time.Now(),
	}
	data, err := json.MarshalIndent(ctrl, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal control command: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// ReadControlCommand atomically reads and removes the control command file.
// Uses rename-based atomic consumption to prevent race conditions.
// Returns nil, nil if no command is pending.
func ReadControlCommand(projectDir, sessionID string) (*Control, error) {
	path := GetControlFilePath(projectDir, sessionID)

	// Atomically rename the file to claim ownership before reading
	// This prevents race conditions where multiple readers could read the same command
	consumedPath := path + ".consumed"
	if err := os.Rename(path, consumedPath); err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No command pending
		}
		return nil, err
	}

	// Now we own the file - read and delete it
	data, err := os.ReadFile(consumedPath)
	if err != nil {
		// Clean up the consumed file even on read error
		os.Remove(consumedPath)
		return nil, err
	}

	// Remove the consumed file
	os.Remove(consumedPath)

	var ctrl Control
	if err := json.Unmarshal(data, &ctrl); err != nil {
		return nil, fmt.Errorf("failed to parse control file: %w", err)
	}
	return &ctrl, nil
}

// isProcessRunning checks if a process with the given PID is running
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds, so we need to send signal 0
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// IsRunning checks if a daemon is running for a session
// Returns (running, info, error)
func IsRunning(projectDir, sessionID string) (bool, *Info, error) {
	info, err := ReadPIDFile(projectDir, sessionID)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil, nil
		}
		return false, nil, err
	}

	// Check if process is still running
	if isProcessRunning(info.PID) {
		return true, info, nil
	}

	// Stale PID file - clean up
	RemovePIDFile(projectDir, sessionID)
	RemoveStateFile(projectDir, sessionID)
	return false, nil, nil
}

// Cleanup removes all daemon-related files for a session
func Cleanup(projectDir, sessionID string) error {
	var lastErr error
	if err := RemovePIDFile(projectDir, sessionID); err != nil {
		lastErr = err
	}
	if err := RemoveStateFile(projectDir, sessionID); err != nil {
		lastErr = err
	}
	// Remove control file if it exists
	ctrlPath := GetControlFilePath(projectDir, sessionID)
	if err := os.Remove(ctrlPath); err != nil && !os.IsNotExist(err) {
		lastErr = err
	}
	return lastErr
}
