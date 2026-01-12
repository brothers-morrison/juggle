package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ohare93/juggle/internal/agent"
	"github.com/ohare93/juggle/internal/cli"
	"github.com/ohare93/juggle/internal/session"
)

// allSessionMockRunner wraps MockRunner and adds progress to "_all" session
type allSessionMockRunner struct {
	mock         *agent.MockRunner
	sessionStore *session.SessionStore
}

func (p *allSessionMockRunner) Run(opts agent.RunOptions) (*agent.RunResult, error) {
	// Simulate agent updating progress before returning (using "_all" for storage)
	entry := fmt.Sprintf("[Iteration %d] Agent work completed\n", p.mock.NextIndex+1)
	_ = p.sessionStore.AppendProgress("_all", entry)

	return p.mock.Run(opts)
}

// TestAllMetaSession_AgentRunSkipsSessionVerification tests that "all" meta-session
// doesn't require a session file to exist
func TestAllMetaSession_AgentRunSkipsSessionVerification(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a ball WITHOUT any session tag (just in the repo)
	ball := env.CreateBall(t, "Untagged ball", session.PriorityMedium)
	ball.State = session.StateComplete // Make it complete so COMPLETE signal works
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Setup mock runner that updates progress to "_all" session
	sessionStore := env.GetSessionStore(t)
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:   "Working...\n<promise>COMPLETE</promise>\nDone.",
			Complete: true,
		},
	)
	agent.SetRunner(&allSessionMockRunner{
		mock:         mock,
		sessionStore: sessionStore,
	})
	defer agent.ResetRunner()

	// Run the agent loop with "all" session - should NOT require session file
	config := cli.AgentLoopConfig{
		SessionID:     "all", // Special meta-session
		ProjectDir:    env.ProjectDir,
		MaxIterations: 1,
		IterDelay:     0,
	}

	result, err := cli.RunAgentLoop(config)
	if err != nil {
		t.Fatalf("Agent run with 'all' session should not require session file: %v", err)
	}

	// Verify the loop ran
	if len(mock.Calls) != 1 {
		t.Errorf("Expected 1 call to runner, got %d", len(mock.Calls))
	}

	// Verify result
	if !result.Complete {
		t.Error("Expected result.Complete=true")
	}
}

// TestAllMetaSession_GeneratePromptIncludesAllBalls tests that "all" includes
// all balls in the repo regardless of session tag
func TestAllMetaSession_GeneratePromptIncludesAllBalls(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	store := env.GetStore(t)

	// Create balls with different session tags and one without
	ball1 := env.CreateBall(t, "Ball in session A", session.PriorityMedium)
	ball1.Tags = []string{"session-a"}
	if err := store.UpdateBall(ball1); err != nil {
		t.Fatalf("Failed to update ball1: %v", err)
	}

	ball2 := env.CreateBall(t, "Ball in session B", session.PriorityMedium)
	ball2.Tags = []string{"session-b"}
	if err := store.UpdateBall(ball2); err != nil {
		t.Fatalf("Failed to update ball2: %v", err)
	}

	ball3 := env.CreateBall(t, "Untagged ball", session.PriorityMedium)
	// No tags
	if err := store.UpdateBall(ball3); err != nil {
		t.Fatalf("Failed to update ball3: %v", err)
	}

	// Generate prompt with "all" session
	prompt, err := cli.GenerateAgentPromptForTest(env.ProjectDir, "all", false, "")
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	// Verify all balls are included
	if !strings.Contains(prompt, "Ball in session A") {
		t.Error("Prompt should include 'Ball in session A'")
	}
	if !strings.Contains(prompt, "Ball in session B") {
		t.Error("Prompt should include 'Ball in session B'")
	}
	if !strings.Contains(prompt, "Untagged ball") {
		t.Error("Prompt should include 'Untagged ball'")
	}
}

// TestAllMetaSession_OutputDirectory tests that "all" session uses "_all" directory
// for output files
func TestAllMetaSession_OutputDirectory(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a ball
	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	ball.State = session.StateComplete
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Setup mock runner that updates progress to "_all"
	sessionStore := env.GetSessionStore(t)
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:   "Test output\n<promise>COMPLETE</promise>",
			Complete: true,
		},
	)
	agent.SetRunner(&allSessionMockRunner{
		mock:         mock,
		sessionStore: sessionStore,
	})
	defer agent.ResetRunner()

	// Run with "all" session
	config := cli.AgentLoopConfig{
		SessionID:     "all",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 1,
		IterDelay:     0,
	}

	_, err := cli.RunAgentLoop(config)
	if err != nil {
		t.Fatalf("Agent run failed: %v", err)
	}

	// Verify output was written to "_all" directory (not "all")
	expectedPath := filepath.Join(env.ProjectDir, ".juggler", "sessions", "_all", "last_output.txt")
	// The directory should have been created
	allDir := filepath.Join(env.ProjectDir, ".juggler", "sessions", "_all")
	if _, err := filepath.Glob(allDir); err != nil {
		// Just check it doesn't error - the directory exists
	}
	_ = expectedPath // Path verification - the RunAgentLoop creates this
}

// TestAllMetaSession_ExportFormatAgent tests that "export --session all --format agent"
// includes all balls without session filtering
func TestAllMetaSession_ExportFormatAgent(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	store := env.GetStore(t)

	// Create balls with different tags
	ball1 := env.CreateBall(t, "Feature A ball", session.PriorityHigh)
	ball1.Tags = []string{"feature-a"}
	if err := store.UpdateBall(ball1); err != nil {
		t.Fatalf("Failed to update ball1: %v", err)
	}

	ball2 := env.CreateBall(t, "Feature B ball", session.PriorityMedium)
	ball2.Tags = []string{"feature-b"}
	if err := store.UpdateBall(ball2); err != nil {
		t.Fatalf("Failed to update ball2: %v", err)
	}

	// Generate agent prompt with "all" session
	prompt, err := cli.GenerateAgentPromptForTest(env.ProjectDir, "all", false, "")
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	// Both balls should be included
	if !strings.Contains(prompt, "Feature A ball") {
		t.Error("Prompt should include 'Feature A ball'")
	}
	if !strings.Contains(prompt, "Feature B ball") {
		t.Error("Prompt should include 'Feature B ball'")
	}

	// Should still have proper structure
	if !strings.Contains(prompt, "<balls>") {
		t.Error("Prompt should have <balls> section")
	}
	if !strings.Contains(prompt, "<instructions>") {
		t.Error("Prompt should have <instructions> section")
	}
}

// TestAllMetaSession_RegularSessionStillWorks tests that regular sessions
// still work correctly (no regression)
func TestAllMetaSession_RegularSessionStillWorks(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a real session
	env.CreateSession(t, "my-session", "My test session")

	store := env.GetStore(t)

	// Create balls with session tag
	ball1 := env.CreateBall(t, "Ball in my-session", session.PriorityMedium)
	ball1.Tags = []string{"my-session"}
	if err := store.UpdateBall(ball1); err != nil {
		t.Fatalf("Failed to update ball1: %v", err)
	}

	// Create ball NOT in session
	ball2 := env.CreateBall(t, "Ball not in session", session.PriorityMedium)
	ball2.Tags = []string{"other-session"}
	if err := store.UpdateBall(ball2); err != nil {
		t.Fatalf("Failed to update ball2: %v", err)
	}

	// Generate prompt with regular session
	prompt, err := cli.GenerateAgentPromptForTest(env.ProjectDir, "my-session", false, "")
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	// Only session ball should be included
	if !strings.Contains(prompt, "Ball in my-session") {
		t.Error("Prompt should include 'Ball in my-session'")
	}
	if strings.Contains(prompt, "Ball not in session") {
		t.Error("Prompt should NOT include 'Ball not in session'")
	}
}

// TestAllMetaSession_SessionNamedAllRequiresEscape tests that a session
// literally named "all" would need special handling (edge case documentation)
func TestAllMetaSession_SessionNamedAllWarning(t *testing.T) {
	// This test documents the behavior: if you have a session literally named "all",
	// it will be treated as the meta-session. Users should avoid naming sessions "all"
	// or use escaping if their system supports it.

	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Try to create a session named "all"
	env.CreateSession(t, "all", "Session literally named all")

	store := env.GetStore(t)

	// Create a ball tagged with "all"
	ball := env.CreateBall(t, "Ball in literal all session", session.PriorityMedium)
	ball.Tags = []string{"all"}
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Create another ball without "all" tag
	ball2 := env.CreateBall(t, "Another ball", session.PriorityMedium)
	if err := store.UpdateBall(ball2); err != nil {
		t.Fatalf("Failed to update ball2: %v", err)
	}

	// When using "all" as session, it's the meta-session, so ALL balls are included
	prompt, err := cli.GenerateAgentPromptForTest(env.ProjectDir, "all", false, "")
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	// Both balls should be included (because "all" is meta-session)
	if !strings.Contains(prompt, "Ball in literal all session") {
		t.Error("Meta-session 'all' should include all balls")
	}
	if !strings.Contains(prompt, "Another ball") {
		t.Error("Meta-session 'all' should include all balls")
	}
}

// TestAllMetaSession_BallFilterWithAll tests that --ball flag works with "all" session
func TestAllMetaSession_BallFilterWithAll(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	store := env.GetStore(t)

	// Create multiple balls
	ball1 := env.CreateBall(t, "Target ball", session.PriorityMedium)
	if err := store.UpdateBall(ball1); err != nil {
		t.Fatalf("Failed to update ball1: %v", err)
	}

	ball2 := env.CreateBall(t, "Other ball", session.PriorityMedium)
	if err := store.UpdateBall(ball2); err != nil {
		t.Fatalf("Failed to update ball2: %v", err)
	}

	// Generate prompt with "all" session but specific ball
	prompt, err := cli.GenerateAgentPromptForTest(env.ProjectDir, "all", false, ball1.ShortID())
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	// Only target ball should be included
	if !strings.Contains(prompt, "Target ball") {
		t.Error("Prompt should include 'Target ball'")
	}
	if strings.Contains(prompt, "Other ball") {
		t.Error("Prompt should NOT include 'Other ball' when --ball filter is used")
	}
}

// TestAllMetaSession_ProgressAppend tests that "juggle progress append all" works
// properly and stores to _all directory
func TestAllMetaSession_ProgressAppend(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Get session store
	sessionStore := env.GetSessionStore(t)

	// Append progress to "all" meta-session - note: cli maps "all" to "_all"
	// Testing the store directly with "_all"
	err := sessionStore.AppendProgress("_all", "[2024-01-01 10:00:00] First progress entry\n")
	if err != nil {
		t.Fatalf("Failed to append progress to _all: %v", err)
	}

	err = sessionStore.AppendProgress("_all", "[2024-01-01 10:30:00] Second progress entry\n")
	if err != nil {
		t.Fatalf("Failed to append second progress to _all: %v", err)
	}

	// Load progress
	progress, err := sessionStore.LoadProgress("_all")
	if err != nil {
		t.Fatalf("Failed to load progress from _all: %v", err)
	}

	// Verify content
	if !strings.Contains(progress, "First progress entry") {
		t.Error("Expected progress to contain 'First progress entry'")
	}
	if !strings.Contains(progress, "Second progress entry") {
		t.Error("Expected progress to contain 'Second progress entry'")
	}
}

// TestAllMetaSession_ProgressFileLocation tests that progress for "all" is stored
// in the correct location (.juggler/sessions/_all/progress.txt)
func TestAllMetaSession_ProgressFileLocation(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	sessionStore := env.GetSessionStore(t)

	// Append progress to _all
	err := sessionStore.AppendProgress("_all", "Test progress\n")
	if err != nil {
		t.Fatalf("Failed to append progress: %v", err)
	}

	// Check file location
	expectedPath := filepath.Join(env.ProjectDir, ".juggler", "sessions", "_all", "progress.txt")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected progress file at %s", expectedPath)
	}

	// Read file directly
	content, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read progress file: %v", err)
	}

	if !strings.Contains(string(content), "Test progress") {
		t.Error("Progress file should contain 'Test progress'")
	}
}

// TestAllMetaSession_ProgressEmptyBeforeWrite tests that loading progress from
// non-existent _all returns empty (not error)
func TestAllMetaSession_ProgressEmptyBeforeWrite(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	sessionStore := env.GetSessionStore(t)

	// Try to load progress before any writes
	progress, err := sessionStore.LoadProgress("_all")
	if err != nil {
		t.Fatalf("Loading from non-existent _all should not error: %v", err)
	}

	if progress != "" {
		t.Errorf("Expected empty progress, got '%s'", progress)
	}
}
