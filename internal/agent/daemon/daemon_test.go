package daemon

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDaemonFiles(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "daemon-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sessionID := "test-session"

	// Test WritePIDFile and ReadPIDFile
	t.Run("PIDFile", func(t *testing.T) {
		info := &Info{
			PID:           12345,
			SessionID:     sessionID,
			ProjectDir:    tmpDir,
			StartedAt:     time.Now(),
			MaxIterations: 10,
			Model:         "opus",
			Provider:      "claude",
		}

		if err := WritePIDFile(tmpDir, sessionID, info); err != nil {
			t.Fatalf("WritePIDFile failed: %v", err)
		}

		readInfo, err := ReadPIDFile(tmpDir, sessionID)
		if err != nil {
			t.Fatalf("ReadPIDFile failed: %v", err)
		}

		if readInfo.PID != info.PID {
			t.Errorf("PID mismatch: got %d, want %d", readInfo.PID, info.PID)
		}
		if readInfo.SessionID != info.SessionID {
			t.Errorf("SessionID mismatch: got %s, want %s", readInfo.SessionID, info.SessionID)
		}
		if readInfo.Model != info.Model {
			t.Errorf("Model mismatch: got %s, want %s", readInfo.Model, info.Model)
		}
	})

	// Test WriteStateFile and ReadStateFile
	t.Run("StateFile", func(t *testing.T) {
		state := &State{
			Running:          true,
			Paused:           false,
			CurrentBallID:    "juggle-5",
			CurrentBallTitle: "Test Ball",
			Iteration:        3,
			MaxIterations:    10,
			FilesChanged:     7,
			ACsComplete:      2,
			ACsTotal:         5,
			Model:            "sonnet",
			Provider:         "claude",
			StartedAt:        time.Now(),
		}

		if err := WriteStateFile(tmpDir, sessionID, state); err != nil {
			t.Fatalf("WriteStateFile failed: %v", err)
		}

		readState, err := ReadStateFile(tmpDir, sessionID)
		if err != nil {
			t.Fatalf("ReadStateFile failed: %v", err)
		}

		if readState.Running != state.Running {
			t.Errorf("Running mismatch: got %v, want %v", readState.Running, state.Running)
		}
		if readState.Iteration != state.Iteration {
			t.Errorf("Iteration mismatch: got %d, want %d", readState.Iteration, state.Iteration)
		}
		if readState.CurrentBallID != state.CurrentBallID {
			t.Errorf("CurrentBallID mismatch: got %s, want %s", readState.CurrentBallID, state.CurrentBallID)
		}
	})

	// Test SendControlCommand and ReadControlCommand
	t.Run("ControlCommand", func(t *testing.T) {
		if err := SendControlCommand(tmpDir, sessionID, CmdPause, ""); err != nil {
			t.Fatalf("SendControlCommand failed: %v", err)
		}

		ctrl, err := ReadControlCommand(tmpDir, sessionID)
		if err != nil {
			t.Fatalf("ReadControlCommand failed: %v", err)
		}

		if ctrl == nil {
			t.Fatal("ReadControlCommand returned nil")
		}
		if ctrl.Command != CmdPause {
			t.Errorf("Command mismatch: got %s, want %s", ctrl.Command, CmdPause)
		}

		// Reading again should return nil (command consumed)
		ctrl2, err := ReadControlCommand(tmpDir, sessionID)
		if err != nil {
			t.Fatalf("Second ReadControlCommand failed: %v", err)
		}
		if ctrl2 != nil {
			t.Error("Expected nil after command was consumed")
		}
	})

	// Test Cleanup
	t.Run("Cleanup", func(t *testing.T) {
		// Write files
		WritePIDFile(tmpDir, sessionID, &Info{PID: 1})
		WriteStateFile(tmpDir, sessionID, &State{Running: true})
		SendControlCommand(tmpDir, sessionID, CmdCancel, "")

		// Cleanup
		if err := Cleanup(tmpDir, sessionID); err != nil {
			t.Fatalf("Cleanup failed: %v", err)
		}

		// Verify files are gone
		pidPath := GetPIDFilePath(tmpDir, sessionID)
		if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
			t.Error("PID file should be removed after cleanup")
		}

		statePath := GetStateFilePath(tmpDir, sessionID)
		if _, err := os.Stat(statePath); !os.IsNotExist(err) {
			t.Error("State file should be removed after cleanup")
		}
	})

	// Test GetFilePaths
	t.Run("FilePaths", func(t *testing.T) {
		pidPath := GetPIDFilePath(tmpDir, sessionID)
		expectedPID := filepath.Join(tmpDir, ".juggle", "sessions", sessionID, "agent.pid")
		if pidPath != expectedPID {
			t.Errorf("PID path mismatch: got %s, want %s", pidPath, expectedPID)
		}

		ctrlPath := GetControlFilePath(tmpDir, sessionID)
		expectedCtrl := filepath.Join(tmpDir, ".juggle", "sessions", sessionID, "agent.ctrl")
		if ctrlPath != expectedCtrl {
			t.Errorf("Control path mismatch: got %s, want %s", ctrlPath, expectedCtrl)
		}

		statePath := GetStateFilePath(tmpDir, sessionID)
		expectedState := filepath.Join(tmpDir, ".juggle", "sessions", sessionID, "agent.state")
		if statePath != expectedState {
			t.Errorf("State path mismatch: got %s, want %s", statePath, expectedState)
		}
	})
}

func TestIsRunning(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "daemon-isrunning-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sessionID := "test-session"

	// No PID file - should return false
	running, info, err := IsRunning(tmpDir, sessionID)
	if err != nil {
		t.Fatalf("IsRunning failed: %v", err)
	}
	if running {
		t.Error("Expected not running when no PID file exists")
	}
	if info != nil {
		t.Error("Expected nil info when no PID file exists")
	}

	// Write PID file with fake PID (not running)
	fakeInfo := &Info{
		PID:       999999999, // Very unlikely to be running
		SessionID: sessionID,
		ProjectDir: tmpDir,
		StartedAt: time.Now(),
	}
	if err := WritePIDFile(tmpDir, sessionID, fakeInfo); err != nil {
		t.Fatalf("WritePIDFile failed: %v", err)
	}

	// Should return false and clean up stale PID file
	running, _, err = IsRunning(tmpDir, sessionID)
	if err != nil {
		t.Fatalf("IsRunning failed: %v", err)
	}
	if running {
		t.Error("Expected not running for non-existent PID")
	}

	// PID file should be cleaned up
	pidPath := GetPIDFilePath(tmpDir, sessionID)
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Error("Stale PID file should be cleaned up")
	}
}

func TestControlCommandAtomicity(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "daemon-atomic-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sessionID := "test-session"

	// Send a command
	if err := SendControlCommand(tmpDir, sessionID, CmdPause, "test-args"); err != nil {
		t.Fatalf("SendControlCommand failed: %v", err)
	}

	// Verify control file exists
	ctrlPath := GetControlFilePath(tmpDir, sessionID)
	if _, err := os.Stat(ctrlPath); os.IsNotExist(err) {
		t.Fatal("Control file should exist after SendControlCommand")
	}

	// Read the command (this should atomically consume it)
	ctrl, err := ReadControlCommand(tmpDir, sessionID)
	if err != nil {
		t.Fatalf("ReadControlCommand failed: %v", err)
	}
	if ctrl == nil {
		t.Fatal("Expected control command, got nil")
	}
	if ctrl.Command != CmdPause {
		t.Errorf("Command mismatch: got %s, want %s", ctrl.Command, CmdPause)
	}
	if ctrl.Args != "test-args" {
		t.Errorf("Args mismatch: got %s, want %s", ctrl.Args, "test-args")
	}

	// Verify both the original and consumed files are gone
	if _, err := os.Stat(ctrlPath); !os.IsNotExist(err) {
		t.Error("Original control file should be removed after reading")
	}
	consumedPath := ctrlPath + ".consumed"
	if _, err := os.Stat(consumedPath); !os.IsNotExist(err) {
		t.Error("Consumed control file should be removed after reading")
	}

	// Reading again should return nil
	ctrl2, err := ReadControlCommand(tmpDir, sessionID)
	if err != nil {
		t.Fatalf("Second ReadControlCommand failed: %v", err)
	}
	if ctrl2 != nil {
		t.Error("Expected nil after command was consumed")
	}
}
