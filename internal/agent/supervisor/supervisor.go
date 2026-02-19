// Package supervisor provides a polling daemon that monitors juggle sessions,
// detects stalled agents, and optionally auto-restarts them.
package supervisor

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/ohare93/juggle/internal/agent/daemon"
	"github.com/ohare93/juggle/internal/session"
)

// Supervisor monitors running juggle daemon sessions, detects stalls,
// recovers missed signals, and optionally auto-launches/restarts daemons.
type Supervisor struct {
	config     *session.SupervisorConfig
	configOpts session.ConfigOptions
	bridge     *OpenCodeBridge
	stopCh     chan struct{}
	wg         sync.WaitGroup
	mu         sync.Mutex
	running    bool
	pidFile    string
}

// Status represents the supervisor's view of a session
type Status struct {
	ProjectDir  string    `json:"project_dir"`
	SessionID   string    `json:"session_id"`
	DaemonPID   int       `json:"daemon_pid,omitempty"`
	Running     bool      `json:"running"`
	Stalled     bool      `json:"stalled"`
	LastUpdated time.Time `json:"last_updated,omitempty"`
	Status      string    `json:"status"`
	Pending     int       `json:"pending_balls"`
	InProgress  int       `json:"in_progress_balls"`
	Complete    int       `json:"complete_balls"`
	Blocked     int       `json:"blocked_balls"`
}

// New creates a new Supervisor with the given config
func New(config *session.SupervisorConfig, configOpts session.ConfigOptions) *Supervisor {
	if config == nil {
		config = session.DefaultSupervisorConfig()
	}
	return &Supervisor{
		config:     config,
		configOpts: configOpts,
		bridge:     NewOpenCodeBridge(),
		stopCh:     make(chan struct{}),
	}
}

// Start begins the supervisor polling loop in the background
func (s *Supervisor) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("supervisor is already running")
	}

	// Write PID file
	s.pidFile = supervisorPIDPath()
	if err := os.MkdirAll(filepath.Dir(s.pidFile), 0755); err != nil {
		return fmt.Errorf("failed to create supervisor directory: %w", err)
	}

	pidData := map[string]interface{}{
		"pid":        os.Getpid(),
		"started_at": time.Now(),
	}
	data, _ := json.MarshalIndent(pidData, "", "  ")
	if err := os.WriteFile(s.pidFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write supervisor PID file: %w", err)
	}

	s.running = true
	s.stopCh = make(chan struct{})

	s.wg.Add(1)
	go s.pollLoop()

	return nil
}

// Stop signals the supervisor to stop and waits for it to finish
func (s *Supervisor) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.stopCh)
	s.mu.Unlock()

	s.wg.Wait()

	// Clean up PID file
	if s.pidFile != "" {
		os.Remove(s.pidFile)
	}
}

// IsRunning returns whether the supervisor is currently running
func (s *Supervisor) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// RunOnce performs a single poll cycle (useful for testing or one-shot mode)
func (s *Supervisor) RunOnce() ([]Status, error) {
	return s.pollSessions()
}

// pollLoop runs the main polling loop
func (s *Supervisor) pollLoop() {
	defer s.wg.Done()

	// Run immediately on start
	statuses, err := s.pollSessions()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[supervisor] poll error: %v\n", err)
	} else {
		s.handlePollResults(statuses)
	}

	interval := time.Duration(s.config.GetPollInterval()) * time.Minute
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			statuses, err := s.pollSessions()
			if err != nil {
				fmt.Fprintf(os.Stderr, "[supervisor] poll error: %v\n", err)
				continue
			}
			s.handlePollResults(statuses)
		}
	}
}

// pollSessions scans all projects and returns session statuses
func (s *Supervisor) pollSessions() ([]Status, error) {
	config, err := session.LoadConfigWithOptions(s.configOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	projects, err := session.DiscoverProjects(config)
	if err != nil {
		return nil, fmt.Errorf("failed to discover projects: %w", err)
	}

	var statuses []Status
	stallTimeout := time.Duration(s.config.GetStallTimeout()) * time.Minute

	for _, projectDir := range projects {
		sessionsDir := filepath.Join(projectDir, ".juggle", "sessions")
		entries, err := os.ReadDir(sessionsDir)
		if err != nil {
			continue // No sessions in this project
		}

		for _, entry := range entries {
			if !entry.IsDir() || entry.Name() == "_all" {
				continue
			}

			sessionID := entry.Name()
			status := s.checkSession(projectDir, sessionID, stallTimeout)
			statuses = append(statuses, status)
		}
	}

	return statuses, nil
}

// checkSession inspects a single session's state
func (s *Supervisor) checkSession(projectDir, sessionID string, stallTimeout time.Duration) Status {
	status := Status{
		ProjectDir: projectDir,
		SessionID:  sessionID,
	}

	// Count balls by state
	store, err := session.NewStore(projectDir)
	if err == nil {
		balls, err := store.LoadBalls()
		if err == nil {
			for _, ball := range balls {
				// Only count balls for this session (balls linked via Tags)
				if sessionID != "_all" {
					found := false
					for _, tag := range ball.Tags {
						if tag == sessionID {
							found = true
							break
						}
					}
					if !found {
						continue
					}
				}
				switch ball.State {
				case session.StatePending:
					status.Pending++
				case session.StateInProgress:
					status.InProgress++
				case session.StateComplete:
					status.Complete++
				case session.StateBlocked:
					status.Blocked++
				}
			}
		}
	}

	// Check daemon status
	running, info, err := daemon.IsRunning(projectDir, sessionID)
	if err == nil && running && info != nil {
		status.Running = true
		status.DaemonPID = info.PID

		// Check for stall
		state, err := daemon.ReadStateFile(projectDir, sessionID)
		if err == nil && state != nil {
			status.LastUpdated = state.LastUpdated
			status.Status = state.Status

			if !state.LastUpdated.IsZero() && time.Since(state.LastUpdated) > stallTimeout {
				status.Stalled = true
				status.Status = fmt.Sprintf("STALLED (no update for %v)", time.Since(state.LastUpdated).Round(time.Minute))
			}
		}
	} else {
		// Not running - check if there's a stale state file
		state, err := daemon.ReadStateFile(projectDir, sessionID)
		if err == nil && state != nil && state.Running {
			// State says running but process is dead - fix it
			state.Running = false
			state.Status = "Daemon exited unexpectedly"
			daemon.WriteStateFile(projectDir, sessionID, state)
			status.Status = "Dead (state corrected)"
		} else if status.Pending > 0 || status.InProgress > 0 {
			status.Status = "Not running (has pending work)"
		} else {
			status.Status = "Idle"
		}
	}

	return status
}

// handlePollResults processes poll results - restarts stalled daemons, launches new ones
func (s *Supervisor) handlePollResults(statuses []Status) {
	launched := 0
	maxConcurrent := s.config.GetMaxConcurrent()

	// Count currently running daemons
	runningCount := 0
	for _, st := range statuses {
		if st.Running && !st.Stalled {
			runningCount++
		}
	}

	for _, st := range statuses {
		// Handle stalled daemons
		if st.Stalled && s.config.AutoRestart {
			fmt.Fprintf(os.Stderr, "[supervisor] Restarting stalled daemon: %s/%s (PID %d)\n",
				st.ProjectDir, st.SessionID, st.DaemonPID)

			// Try to recover signals from OpenCode first
			s.bridge.RecoverSession(st.ProjectDir, st.SessionID)

			// Kill the stalled process and restart
			if st.DaemonPID > 0 {
				if proc, err := os.FindProcess(st.DaemonPID); err == nil {
					proc.Kill()
				}
			}
			daemon.Cleanup(st.ProjectDir, st.SessionID)

			if runningCount+launched < maxConcurrent {
				if err := s.launchDaemon(st.ProjectDir, st.SessionID); err != nil {
					fmt.Fprintf(os.Stderr, "[supervisor] Failed to restart %s: %v\n", st.SessionID, err)
				} else {
					launched++
				}
			}
		}

		// Auto-launch for sessions with pending work and no daemon
		if s.config.AutoLaunch && !st.Running && (st.Pending > 0 || st.InProgress > 0) {
			if runningCount+launched < maxConcurrent {
				fmt.Fprintf(os.Stderr, "[supervisor] Auto-launching daemon for %s/%s (%d pending, %d in progress)\n",
					st.ProjectDir, st.SessionID, st.Pending, st.InProgress)

				if err := s.launchDaemon(st.ProjectDir, st.SessionID); err != nil {
					fmt.Fprintf(os.Stderr, "[supervisor] Failed to launch %s: %v\n", st.SessionID, err)
				} else {
					launched++
				}
			}
		}
	}

	if launched > 0 {
		fmt.Fprintf(os.Stderr, "[supervisor] Launched %d daemon(s) this cycle\n", launched)
	}
}

// launchDaemon starts a juggle agent daemon for a session
func (s *Supervisor) launchDaemon(projectDir, sessionID string) error {
	juggleBin, err := exec.LookPath("juggle")
	if err != nil {
		return fmt.Errorf("juggle binary not found: %w", err)
	}

	logPath := filepath.Join(projectDir, ".juggle", "sessions", sessionID, "agent.log")

	cmd := exec.Command(juggleBin, "agent", "run", "--daemon", sessionID)
	cmd.Dir = projectDir

	// Redirect output to log file
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Detach - don't wait for the child
	go func() {
		cmd.Wait()
		logFile.Close()
	}()

	// Give it a moment to initialize
	time.Sleep(time.Second)

	return nil
}

// supervisorPIDPath returns the path to the supervisor's own PID file
func supervisorPIDPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".juggle", "supervisor.pid")
}

// IsSupervisorRunning checks if a supervisor process is already running
func IsSupervisorRunning() (bool, int) {
	pidFile := supervisorPIDPath()
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return false, 0
	}

	var pidData map[string]interface{}
	if err := json.Unmarshal(data, &pidData); err != nil {
		return false, 0
	}

	pid, ok := pidData["pid"].(float64)
	if !ok {
		return false, 0
	}

	pidInt := int(pid)
	proc, err := os.FindProcess(pidInt)
	if err != nil {
		return false, 0
	}

	// Signal 0 checks if process exists
	if err := proc.Signal(os.Signal(nil)); err != nil {
		// Process doesn't exist - clean up stale PID
		os.Remove(pidFile)
		return false, 0
	}

	return true, pidInt
}

// StopSupervisor sends SIGTERM to a running supervisor
func StopSupervisor() error {
	running, pid := IsSupervisorRunning()
	if !running {
		return fmt.Errorf("supervisor is not running")
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find supervisor process %d: %w", pid, err)
	}

	if err := proc.Signal(os.Interrupt); err != nil {
		return fmt.Errorf("failed to stop supervisor (PID %d): %w", pid, err)
	}

	// Wait briefly for cleanup
	time.Sleep(500 * time.Millisecond)

	// Check if it stopped
	if stillRunning, _ := IsSupervisorRunning(); stillRunning {
		// Force kill
		proc.Kill()
		os.Remove(supervisorPIDPath())
	}

	return nil
}
