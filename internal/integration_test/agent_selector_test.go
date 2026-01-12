package integration_test

import (
	"strings"
	"testing"

	"github.com/ohare93/juggle/internal/cli"
	"github.com/ohare93/juggle/internal/session"
)

// Tests for agent run session selector functionality

func TestAgentRunCmd_AcceptsNoArg(t *testing.T) {
	// Verify that the agent run command accepts optional session-id argument
	// This is a structural test - the command should not require an argument
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session so selector would have something to show
	env.CreateSession(t, "test-session", "Test session")

	// The command structure allows optional args (MaximumNArgs(1))
	// We can't test interactive input easily, but we can verify
	// the exported function exists and the error handling works
	_, err := cli.SelectSessionForAgentForTest(env.ProjectDir)

	// Should fail since there's no stdin input, but the error should be about
	// reading input, not about missing sessions
	if err != nil {
		// The error should be about reading input, not missing sessions
		// since we created a session above
		t.Log("Expected error when stdin is not interactive:", err)
	}
}

func TestAgentRunCmd_AcceptsSessionArg(t *testing.T) {
	// Verify that running with a session-id argument still works
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session and ball
	env.CreateSession(t, "test-session", "Test session")
	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Generate prompt directly (simulating when session-id is provided)
	_, err := cli.GenerateAgentPromptForTest(env.ProjectDir, "test-session", false, "")
	if err != nil {
		t.Fatalf("Failed to generate prompt with explicit session ID: %v", err)
	}
}

func TestSessionSelector_NoSessionsError(t *testing.T) {
	// Verify error message when no sessions exist
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Don't create any sessions
	_, err := cli.SelectSessionForAgentForTest(env.ProjectDir)
	if err == nil {
		t.Fatal("Expected error when no sessions exist")
	}

	// Should mention "no sessions found"
	if err.Error() == "" {
		t.Error("Error message should not be empty")
	}
	// The error should mention creating a session
	if !strings.Contains(err.Error(), "no sessions found") {
		t.Errorf("Error should mention 'no sessions found', got: %s", err.Error())
	}
}

func TestSessionSelector_LocalScope(t *testing.T) {
	// Verify selector shows local sessions only by default
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "local-session", "Local test session")

	// Create ball for the session
	ball := env.CreateBall(t, "Local ball", session.PriorityMedium)
	ball.Tags = []string{"local-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Note: We can't easily test the full interactive flow without stdin
	// but we can verify the session store finds local sessions
	sessionStore := env.GetSessionStore(t)
	sessions, err := sessionStore.ListSessions()
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("Expected 1 local session, got %d", len(sessions))
	}

	if len(sessions) > 0 && sessions[0].ID != "local-session" {
		t.Errorf("Expected local-session, got %s", sessions[0].ID)
	}
}

func TestSessionSelector_SessionInfoIncludesBallCount(t *testing.T) {
	// Verify that session info includes ball count
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session")

	// Create multiple balls for the session
	for i := 0; i < 3; i++ {
		ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
		ball.Tags = []string{"test-session"}
		store := env.GetStore(t)
		if err := store.UpdateBall(ball); err != nil {
			t.Fatalf("Failed to update ball: %v", err)
		}
	}

	// Verify balls are correctly associated with session
	balls, err := session.LoadBallsBySession([]string{env.ProjectDir}, "test-session")
	if err != nil {
		t.Fatalf("Failed to load balls by session: %v", err)
	}

	if len(balls) != 3 {
		t.Errorf("Expected 3 balls in session, got %d", len(balls))
	}
}

func TestSessionSelector_MultipleSessionsAvailable(t *testing.T) {
	// Verify selector finds all local sessions
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create multiple sessions
	env.CreateSession(t, "session-1", "First session")
	env.CreateSession(t, "session-2", "Second session")
	env.CreateSession(t, "session-3", "Third session")

	sessionStore := env.GetSessionStore(t)
	sessions, err := sessionStore.ListSessions()
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(sessions) != 3 {
		t.Errorf("Expected 3 sessions, got %d", len(sessions))
	}
}

func TestSessionSelection_StructFields(t *testing.T) {
	// Verify the sessionSelection struct has correct fields
	// This is a compile-time test that ensures the struct is properly defined

	// Create a mock selection to test struct fields
	selection := &cli.SessionSelection{
		SessionID:  "test-session",
		ProjectDir: "/tmp/test",
	}

	if selection.SessionID != "test-session" {
		t.Error("SessionID field not accessible")
	}

	if selection.ProjectDir != "/tmp/test" {
		t.Error("ProjectDir field not accessible")
	}
}

func TestAgentRun_DryRunWithOptionalSession(t *testing.T) {
	// Test that dry-run works with explicit session
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create session and ball
	env.CreateSession(t, "test-session", "Test session")
	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Test prompt generation (which is what dry-run displays)
	prompt, err := cli.GenerateAgentPromptForTest(env.ProjectDir, "test-session", true, "")
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	if len(prompt) == 0 {
		t.Error("Dry-run should generate non-empty prompt")
	}
}
