package integration_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/ohare93/juggle/internal/session"
)

// TestPlanJSONOutput tests that 'juggle plan ... --json' outputs created ball as JSON
func TestPlanJSONOutput(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleBinary := GetJuggleBinaryPath(t)

	// Build binary if needed
	if _, err := os.Stat(juggleBinary); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", GetJuggleBinaryName(), "./cmd/juggle")
		buildCmd.Dir = GetRepoRoot(t)
		if output, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build juggle: %v\nOutput: %s", err, output)
		}
	}

	// Test basic plan with --json
	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "plan", "Test task for JSON output", "--json")
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'juggle plan --json' failed: %v\nOutput: %s", err, output)
	}

	// Verify it's valid JSON
	var ball session.Ball
	if err := json.Unmarshal(output, &ball); err != nil {
		t.Fatalf("Failed to parse ball JSON: %v\nOutput: %s", err, output)
	}

	// Verify ball properties
	if ball.Title != "Test task for JSON output" {
		t.Errorf("Expected title 'Test task for JSON output', got %q", ball.Title)
	}
	if ball.State != session.StatePending {
		t.Errorf("Expected state 'pending', got %q", ball.State)
	}
	if ball.ID == "" {
		t.Error("Expected ball ID to be set")
	}
}

// TestPlanJSONOutputWithOptions tests plan --json with various options
func TestPlanJSONOutputWithOptions(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleBinary := GetJuggleBinaryPath(t)

	// Build binary if needed
	if _, err := os.Stat(juggleBinary); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", GetJuggleBinaryName(), "./cmd/juggle")
		buildCmd.Dir = GetRepoRoot(t)
		if output, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build juggle: %v\nOutput: %s", err, output)
		}
	}

	// Test plan with priority, tags, and acceptance criteria
	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "plan",
		"Complex task", "--json",
		"-p", "high",
		"-t", "backend,api",
		"-c", "AC1: First criterion",
		"-c", "AC2: Second criterion",
		"--context", "Background context for agents",
	)
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'juggle plan --json' with options failed: %v\nOutput: %s", err, output)
	}

	var ball session.Ball
	if err := json.Unmarshal(output, &ball); err != nil {
		t.Fatalf("Failed to parse ball JSON: %v\nOutput: %s", err, output)
	}

	// Verify all properties
	if ball.Title != "Complex task" {
		t.Errorf("Expected title 'Complex task', got %q", ball.Title)
	}
	if ball.Priority != session.PriorityHigh {
		t.Errorf("Expected priority 'high', got %q", ball.Priority)
	}
	if ball.Context != "Background context for agents" {
		t.Errorf("Expected context to be set, got %q", ball.Context)
	}
	if len(ball.AcceptanceCriteria) != 2 {
		t.Errorf("Expected 2 acceptance criteria, got %d", len(ball.AcceptanceCriteria))
	}
	if len(ball.Tags) < 2 {
		t.Errorf("Expected at least 2 tags, got %d: %v", len(ball.Tags), ball.Tags)
	}
}

// TestPlanJSONOutputError tests that plan --json outputs errors as JSON
func TestPlanJSONOutputError(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleBinary := GetJuggleBinaryPath(t)

	// Build binary if needed
	if _, err := os.Stat(juggleBinary); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", GetJuggleBinaryName(), "./cmd/juggle")
		buildCmd.Dir = GetRepoRoot(t)
		if output, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build juggle: %v\nOutput: %s", err, output)
		}
	}

	// Test plan without intent (should error)
	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "plan", "--json")
	cmd.Dir = env.ProjectDir
	output, _ := cmd.CombinedOutput()

	// Should be valid JSON error
	var errResp map[string]string
	if err := json.Unmarshal(output, &errResp); err != nil {
		t.Fatalf("Failed to parse error JSON: %v\nOutput: %s", err, output)
	}

	if _, ok := errResp["error"]; !ok {
		t.Errorf("Expected 'error' field in JSON response, got: %s", output)
	}
}

// TestUpdateJSONOutput tests that 'juggle update ... --json' outputs updated ball as JSON
func TestUpdateJSONOutput(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a ball first
	ball := env.CreateBall(t, "Update test ball", session.PriorityMedium)

	juggleBinary := GetJuggleBinaryPath(t)

	// Build binary if needed
	if _, err := os.Stat(juggleBinary); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", GetJuggleBinaryName(), "./cmd/juggle")
		buildCmd.Dir = GetRepoRoot(t)
		if output, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build juggle: %v\nOutput: %s", err, output)
		}
	}

	// Test update with --json
	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "update", ball.ID,
		"--priority", "urgent",
		"--json",
	)
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'juggle update --json' failed: %v\nOutput: %s", err, output)
	}

	// Verify it's valid JSON
	var updatedBall session.Ball
	if err := json.Unmarshal(output, &updatedBall); err != nil {
		t.Fatalf("Failed to parse ball JSON: %v\nOutput: %s", err, output)
	}

	// Verify the update was applied
	if updatedBall.Priority != session.PriorityUrgent {
		t.Errorf("Expected priority 'urgent', got %q", updatedBall.Priority)
	}
	if updatedBall.ID != ball.ID {
		t.Errorf("Expected same ball ID, got %q vs %q", updatedBall.ID, ball.ID)
	}
}

// TestUpdateJSONOutputMultipleFields tests update --json with multiple field changes
func TestUpdateJSONOutputMultipleFields(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball := env.CreateBall(t, "Multi-field update test", session.PriorityLow)

	juggleBinary := GetJuggleBinaryPath(t)

	// Build binary if needed
	if _, err := os.Stat(juggleBinary); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", GetJuggleBinaryName(), "./cmd/juggle")
		buildCmd.Dir = GetRepoRoot(t)
		if output, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build juggle: %v\nOutput: %s", err, output)
		}
	}

	// Update multiple fields
	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "update", ball.ID,
		"--intent", "Updated title",
		"--priority", "high",
		"--tags", "tag1,tag2",
		"--json",
	)
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'juggle update --json' failed: %v\nOutput: %s", err, output)
	}

	var updatedBall session.Ball
	if err := json.Unmarshal(output, &updatedBall); err != nil {
		t.Fatalf("Failed to parse ball JSON: %v\nOutput: %s", err, output)
	}

	if updatedBall.Title != "Updated title" {
		t.Errorf("Expected title 'Updated title', got %q", updatedBall.Title)
	}
	if updatedBall.Priority != session.PriorityHigh {
		t.Errorf("Expected priority 'high', got %q", updatedBall.Priority)
	}
	if len(updatedBall.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d: %v", len(updatedBall.Tags), updatedBall.Tags)
	}
}

// TestSessionsCreateJSONOutput tests that 'juggle sessions create ... --json' outputs created session as JSON
func TestSessionsCreateJSONOutput(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleBinary := GetJuggleBinaryPath(t)

	// Build binary if needed
	if _, err := os.Stat(juggleBinary); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", GetJuggleBinaryName(), "./cmd/juggle")
		buildCmd.Dir = GetRepoRoot(t)
		if output, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build juggle: %v\nOutput: %s", err, output)
		}
	}

	// Test sessions create with --json
	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "sessions", "create", "test-session",
		"-m", "Test session description",
		"--json",
	)
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'juggle sessions create --json' failed: %v\nOutput: %s", err, output)
	}

	// Verify it's valid JSON
	var sess session.JuggleSession
	if err := json.Unmarshal(output, &sess); err != nil {
		t.Fatalf("Failed to parse session JSON: %v\nOutput: %s", err, output)
	}

	// Verify session properties
	if sess.ID != "test-session" {
		t.Errorf("Expected ID 'test-session', got %q", sess.ID)
	}
	if sess.Description != "Test session description" {
		t.Errorf("Expected description 'Test session description', got %q", sess.Description)
	}
}

// TestSessionsCreateJSONOutputWithContext tests sessions create --json with context
func TestSessionsCreateJSONOutputWithContext(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleBinary := GetJuggleBinaryPath(t)

	// Build binary if needed
	if _, err := os.Stat(juggleBinary); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", GetJuggleBinaryName(), "./cmd/juggle")
		buildCmd.Dir = GetRepoRoot(t)
		if output, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build juggle: %v\nOutput: %s", err, output)
		}
	}

	// Test sessions create with context and ACs
	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "sessions", "create", "context-session",
		"-m", "Session with context",
		"--context", "This is the session context for agents",
		"--ac", "AC1: Must pass tests",
		"--ac", "AC2: Must be documented",
		"--json",
	)
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'juggle sessions create --json' with context failed: %v\nOutput: %s", err, output)
	}

	var sess session.JuggleSession
	if err := json.Unmarshal(output, &sess); err != nil {
		t.Fatalf("Failed to parse session JSON: %v\nOutput: %s", err, output)
	}

	if sess.Context != "This is the session context for agents" {
		t.Errorf("Expected context to be set, got %q", sess.Context)
	}
	if len(sess.AcceptanceCriteria) != 2 {
		t.Errorf("Expected 2 acceptance criteria, got %d", len(sess.AcceptanceCriteria))
	}
}

// TestSessionsContextJSONOutput tests that 'juggle sessions context ... --json' outputs updated session as JSON
func TestSessionsContextJSONOutput(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session first
	sessionStore, err := session.NewSessionStoreWithConfig(env.ProjectDir, session.StoreConfig{
		JuggleDirName: ".juggle",
	})
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	_, err = sessionStore.CreateSession("ctx-test", "Context test session")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	juggleBinary := GetJuggleBinaryPath(t)

	// Build binary if needed
	if _, err := os.Stat(juggleBinary); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", GetJuggleBinaryName(), "./cmd/juggle")
		buildCmd.Dir = GetRepoRoot(t)
		if output, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build juggle: %v\nOutput: %s", err, output)
		}
	}

	// Test sessions context --set with --json
	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "sessions", "context", "ctx-test",
		"--set", "New context for this session",
		"--json",
	)
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'juggle sessions context --set --json' failed: %v\nOutput: %s", err, output)
	}

	// Verify it's valid JSON
	var sess session.JuggleSession
	if err := json.Unmarshal(output, &sess); err != nil {
		t.Fatalf("Failed to parse session JSON: %v\nOutput: %s", err, output)
	}

	// Verify the context was updated
	if sess.Context != "New context for this session" {
		t.Errorf("Expected context 'New context for this session', got %q", sess.Context)
	}
	if sess.ID != "ctx-test" {
		t.Errorf("Expected session ID 'ctx-test', got %q", sess.ID)
	}
}

// TestSessionsContextJSONOutputViewOnly tests sessions context --json without update
func TestSessionsContextJSONOutputViewOnly(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session with context
	sessionStore, err := session.NewSessionStoreWithConfig(env.ProjectDir, session.StoreConfig{
		JuggleDirName: ".juggle",
	})
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	_, err = sessionStore.CreateSession("view-test", "View test session")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Set initial context
	err = sessionStore.UpdateSessionContext("view-test", "Initial context content")
	if err != nil {
		t.Fatalf("Failed to set context: %v", err)
	}

	juggleBinary := GetJuggleBinaryPath(t)

	// Build binary if needed
	if _, err := os.Stat(juggleBinary); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", GetJuggleBinaryName(), "./cmd/juggle")
		buildCmd.Dir = GetRepoRoot(t)
		if output, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build juggle: %v\nOutput: %s", err, output)
		}
	}

	// Test sessions context --json (view only, no --set)
	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "sessions", "context", "view-test", "--json")
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'juggle sessions context --json' failed: %v\nOutput: %s", err, output)
	}

	// Verify it's valid JSON
	var sess session.JuggleSession
	if err := json.Unmarshal(output, &sess); err != nil {
		t.Fatalf("Failed to parse session JSON: %v\nOutput: %s", err, output)
	}

	// Verify the session data
	if sess.Context != "Initial context content" {
		t.Errorf("Expected context 'Initial context content', got %q", sess.Context)
	}
}

// TestJSONOutputToStdout verifies that JSON output goes to stdout, not stderr
func TestJSONOutputToStdout(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleBinary := GetJuggleBinaryPath(t)

	// Build binary if needed
	if _, err := os.Stat(juggleBinary); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", GetJuggleBinaryName(), "./cmd/juggle")
		buildCmd.Dir = GetRepoRoot(t)
		if output, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build juggle: %v\nOutput: %s", err, output)
		}
	}

	// Test plan --json captures stdout only
	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "plan", "Stdout test", "--json")
	cmd.Dir = env.ProjectDir

	stdout, err := cmd.Output() // Output() returns stdout only
	if err != nil {
		// If there's an error, get combined output for debugging
		cmd2 := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "plan", "Stdout test 2", "--json")
		cmd2.Dir = env.ProjectDir
		combined, _ := cmd2.CombinedOutput()
		t.Fatalf("Command failed: %v\nCombined output: %s", err, combined)
	}

	// Verify stdout contains valid JSON
	var ball session.Ball
	if err := json.Unmarshal(stdout, &ball); err != nil {
		t.Fatalf("Failed to parse stdout as JSON: %v\nStdout: %s", err, stdout)
	}

	if ball.Title == "" {
		t.Error("Expected ball title to be set in stdout JSON")
	}
}

// TestUpdateJSONOutputError tests that update --json outputs errors as JSON
func TestUpdateJSONOutputError(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleBinary := GetJuggleBinaryPath(t)

	// Build binary if needed
	if _, err := os.Stat(juggleBinary); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", GetJuggleBinaryName(), "./cmd/juggle")
		buildCmd.Dir = GetRepoRoot(t)
		if output, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build juggle: %v\nOutput: %s", err, output)
		}
	}

	// Test update with non-existent ball
	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "update", "nonexistent-ball",
		"--priority", "high",
		"--json",
	)
	cmd.Dir = env.ProjectDir
	output, _ := cmd.CombinedOutput()

	// Should be valid JSON error
	var errResp map[string]string
	if err := json.Unmarshal(output, &errResp); err != nil {
		t.Fatalf("Failed to parse error JSON: %v\nOutput: %s", err, output)
	}

	if _, ok := errResp["error"]; !ok {
		t.Errorf("Expected 'error' field in JSON response, got: %s", output)
	}
}

// TestSessionsCreateJSONOutputError tests that sessions create --json outputs errors as JSON
func TestSessionsCreateJSONOutputError(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleBinary := GetJuggleBinaryPath(t)

	// Build binary if needed
	if _, err := os.Stat(juggleBinary); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", GetJuggleBinaryName(), "./cmd/juggle")
		buildCmd.Dir = GetRepoRoot(t)
		if output, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build juggle: %v\nOutput: %s", err, output)
		}
	}

	// Create a session first
	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "sessions", "create", "duplicate-session", "--json")
	cmd.Dir = env.ProjectDir
	_, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create first session: %v", err)
	}

	// Try to create duplicate session - should error
	cmd = exec.Command(juggleBinary, "--config-home", env.ConfigHome, "sessions", "create", "duplicate-session", "--json")
	cmd.Dir = env.ProjectDir
	output, _ := cmd.CombinedOutput()

	// Should be valid JSON error
	var errResp map[string]string
	if err := json.Unmarshal(output, &errResp); err != nil {
		t.Fatalf("Failed to parse error JSON: %v\nOutput: %s", err, output)
	}

	if _, ok := errResp["error"]; !ok {
		t.Errorf("Expected 'error' field in JSON response for duplicate session, got: %s", output)
	}
}

// TestSessionsContextJSONOutputError tests that sessions context --json outputs errors as JSON
func TestSessionsContextJSONOutputError(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleBinary := GetJuggleBinaryPath(t)

	// Build binary if needed
	if _, err := os.Stat(juggleBinary); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", GetJuggleBinaryName(), "./cmd/juggle")
		buildCmd.Dir = GetRepoRoot(t)
		if output, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build juggle: %v\nOutput: %s", err, output)
		}
	}

	// Test context for non-existent session
	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "sessions", "context", "nonexistent-session",
		"--set", "Some context",
		"--json",
	)
	cmd.Dir = env.ProjectDir
	output, _ := cmd.CombinedOutput()

	// Should be valid JSON error
	var errResp map[string]string
	if err := json.Unmarshal(output, &errResp); err != nil {
		t.Fatalf("Failed to parse error JSON: %v\nOutput: %s", err, output)
	}

	if _, ok := errResp["error"]; !ok {
		t.Errorf("Expected 'error' field in JSON response, got: %s", output)
	}
}

// ensureBinaryExists builds the juggle binary if it doesn't exist
func ensureBinaryExists(t *testing.T) string {
	t.Helper()

	juggleBinary := GetJuggleBinaryPath(t)

	if _, err := os.Stat(juggleBinary); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", GetJuggleBinaryName(), "./cmd/juggle")
		buildCmd.Dir = GetRepoRoot(t)
		if output, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build juggle: %v\nOutput: %s", err, output)
		}
	}

	return juggleBinary
}

// runJuggleCommandJSON runs a juggle command and expects JSON output
func runJuggleCommandJSON(t *testing.T, workingDir string, args ...string) []byte {
	t.Helper()

	juggleBinary := ensureBinaryExists(t)

	configHome := filepath.Join(workingDir, "..", "config")
	allArgs := append([]string{"--config-home", configHome}, args...)

	cmd := exec.Command(juggleBinary, allArgs...)
	cmd.Dir = workingDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command failed: %v\nArgs: %v\nOutput: %s", err, args, output)
	}

	return output
}
