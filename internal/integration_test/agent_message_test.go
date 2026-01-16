package integration_test

import (
	"strings"
	"testing"

	"github.com/ohare93/juggle/internal/cli"
	"github.com/ohare93/juggle/internal/session"
)

// Tests for --message / -M flag functionality

func TestAgentPromptGeneration_WithMessage(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for message flag")

	// Create a ball tagged with the session
	ball := env.CreateBall(t, "Test ball for message", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	ball.AcceptanceCriteria = []string{"AC 1", "AC 2"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Generate prompt with a message
	message := "Focus on the authentication flow first and ensure proper error handling"
	prompt, err := cli.GenerateAgentPromptWithMessageForTest(env.ProjectDir, "test-session", false, "", message)
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	// Verify message section is included
	if !strings.Contains(prompt, "<user-message>") {
		t.Error("Prompt missing <user-message> tag")
	}
	if !strings.Contains(prompt, "</user-message>") {
		t.Error("Prompt missing </user-message> tag")
	}
	if !strings.Contains(prompt, message) {
		t.Error("Prompt missing the actual message content")
	}
}

func TestAgentPromptGeneration_WithoutMessage(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session without message")

	// Create a ball tagged with the session
	ball := env.CreateBall(t, "Test ball no message", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Generate prompt without a message (empty string)
	prompt, err := cli.GenerateAgentPromptWithMessageForTest(env.ProjectDir, "test-session", false, "", "")
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	// Verify user-message section is NOT included when message is empty
	if strings.Contains(prompt, "<user-message>") {
		t.Error("Prompt should NOT contain <user-message> tag when message is empty")
	}
	if strings.Contains(prompt, "</user-message>") {
		t.Error("Prompt should NOT contain </user-message> tag when message is empty")
	}
}

func TestAgentPromptGeneration_MessageAppearsAtEnd(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for message position")

	// Create a ball tagged with the session
	ball := env.CreateBall(t, "Test ball for position", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Generate prompt with a message
	message := "This is my custom instruction"
	prompt, err := cli.GenerateAgentPromptWithMessageForTest(env.ProjectDir, "test-session", false, "", message)
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	// Verify message appears after instructions (at the end)
	instructionsEnd := strings.Index(prompt, "</instructions>")
	messageStart := strings.Index(prompt, "<user-message>")

	if instructionsEnd == -1 {
		t.Fatal("Prompt missing </instructions>")
	}
	if messageStart == -1 {
		t.Fatal("Prompt missing <user-message>")
	}

	if messageStart < instructionsEnd {
		t.Error("User message should appear AFTER </instructions>")
	}
}

func TestAgentPromptGeneration_MultilineMessage(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for multiline message")

	// Create a ball tagged with the session
	ball := env.CreateBall(t, "Test ball for multiline", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Generate prompt with a multiline message
	message := "First line of instructions\nSecond line with more detail\nThird line with final notes"
	prompt, err := cli.GenerateAgentPromptWithMessageForTest(env.ProjectDir, "test-session", false, "", message)
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	// Verify all lines are present
	if !strings.Contains(prompt, "First line of instructions") {
		t.Error("Prompt missing first line of multiline message")
	}
	if !strings.Contains(prompt, "Second line with more detail") {
		t.Error("Prompt missing second line of multiline message")
	}
	if !strings.Contains(prompt, "Third line with final notes") {
		t.Error("Prompt missing third line of multiline message")
	}
}

func TestAgentPromptGeneration_MessageWithSpecialCharacters(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for special chars")

	// Create a ball tagged with the session
	ball := env.CreateBall(t, "Test ball for special chars", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Generate prompt with special characters in message
	message := "Handle <xml> tags & special \"characters\" correctly. Use 'quotes' too!"
	prompt, err := cli.GenerateAgentPromptWithMessageForTest(env.ProjectDir, "test-session", false, "", message)
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	// Verify the message with special characters is preserved
	if !strings.Contains(prompt, message) {
		t.Error("Prompt should preserve special characters in message")
	}
}

func TestAgentPromptGeneration_SingleBallWithMessage(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for single ball with message")

	// Create a ball tagged with the session
	ball := env.CreateBall(t, "Single target ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Generate prompt for specific ball with message
	message := "Focus only on this specific ball"
	prompt, err := cli.GenerateAgentPromptWithMessageForTest(env.ProjectDir, "test-session", false, ball.ShortID(), message)
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	// Verify single ball format is used
	if !strings.Contains(prompt, "<task>") {
		t.Error("Single ball prompt should use <task> format")
	}

	// Verify message is included
	if !strings.Contains(prompt, "<user-message>") {
		t.Error("Single ball prompt should include user message")
	}
	if !strings.Contains(prompt, message) {
		t.Error("Single ball prompt should contain the message content")
	}
}

func TestAgentPromptGeneration_DebugModeWithMessage(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for debug with message")

	// Create a ball tagged with the session
	ball := env.CreateBall(t, "Debug mode ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Generate prompt with debug mode AND message
	message := "Debug this specific issue"
	prompt, err := cli.GenerateAgentPromptWithMessageForTest(env.ProjectDir, "test-session", true, "", message)
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	// Verify both debug mode and message are present
	if !strings.Contains(prompt, "DEBUG MODE") {
		t.Error("Prompt should contain DEBUG MODE section")
	}
	if !strings.Contains(prompt, "<user-message>") {
		t.Error("Prompt should contain user message section")
	}
	if !strings.Contains(prompt, message) {
		t.Error("Prompt should contain the message content")
	}

	// Verify message appears after debug section (both are at the end of instructions)
	debugIdx := strings.Index(prompt, "DEBUG MODE")
	messageIdx := strings.Index(prompt, "<user-message>")
	if messageIdx < debugIdx {
		t.Error("User message should appear after DEBUG MODE section")
	}
}

// Tests for refine command message functionality

func TestRefinePromptGeneration_WithMessage(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	sessionStore := env.GetSessionStore(t)
	_, err := sessionStore.CreateSession("test-session", "Test session for refine message")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Create a ball tagged with the session
	ball := env.CreateBall(t, "Ball for refine message", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	ball.AcceptanceCriteria = []string{"AC 1"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Load balls for refine
	balls, err := cli.LoadBallsForRefineForTest(env.ProjectDir, "test-session")
	if err != nil {
		t.Fatalf("Failed to load balls for refine: %v", err)
	}

	// Generate refine prompt with message
	message := "Please focus on improving the acceptance criteria"
	prompt, err := cli.GenerateRefinePromptWithMessageForTest(env.ProjectDir, "test-session", balls, message)
	if err != nil {
		t.Fatalf("Failed to generate refine prompt: %v", err)
	}

	// Verify message section is included
	if !strings.Contains(prompt, "<user-message>") {
		t.Error("Refine prompt missing <user-message> tag")
	}
	if !strings.Contains(prompt, message) {
		t.Error("Refine prompt missing the actual message content")
	}
}

func TestRefinePromptGeneration_WithoutMessage(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	sessionStore := env.GetSessionStore(t)
	_, err := sessionStore.CreateSession("test-session", "Test session for refine no message")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Create a ball tagged with the session
	ball := env.CreateBall(t, "Ball for refine no message", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Load balls for refine
	balls, err := cli.LoadBallsForRefineForTest(env.ProjectDir, "test-session")
	if err != nil {
		t.Fatalf("Failed to load balls for refine: %v", err)
	}

	// Generate refine prompt without message
	prompt, err := cli.GenerateRefinePromptWithMessageForTest(env.ProjectDir, "test-session", balls, "")
	if err != nil {
		t.Fatalf("Failed to generate refine prompt: %v", err)
	}

	// Verify user-message section is NOT included when message is empty
	if strings.Contains(prompt, "<user-message>") {
		t.Error("Refine prompt should NOT contain <user-message> tag when message is empty")
	}
}

func TestRefinePromptGeneration_MessageAppearsAtEnd(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	sessionStore := env.GetSessionStore(t)
	_, err := sessionStore.CreateSession("test-session", "Test session for refine position")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Create a ball tagged with the session
	ball := env.CreateBall(t, "Ball for refine position", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Load balls for refine
	balls, err := cli.LoadBallsForRefineForTest(env.ProjectDir, "test-session")
	if err != nil {
		t.Fatalf("Failed to load balls for refine: %v", err)
	}

	// Generate refine prompt with message
	message := "Custom refine instruction"
	prompt, err := cli.GenerateRefinePromptWithMessageForTest(env.ProjectDir, "test-session", balls, message)
	if err != nil {
		t.Fatalf("Failed to generate refine prompt: %v", err)
	}

	// Verify message appears after instructions (at the end)
	instructionsEnd := strings.Index(prompt, "</instructions>")
	messageStart := strings.Index(prompt, "<user-message>")

	if instructionsEnd == -1 {
		t.Fatal("Refine prompt missing </instructions>")
	}
	if messageStart == -1 {
		t.Fatal("Refine prompt missing <user-message>")
	}

	if messageStart < instructionsEnd {
		t.Error("User message in refine prompt should appear AFTER </instructions>")
	}
}

// Tests for AgentLoopConfig.Message field

func TestAgentLoopConfig_MessageField(t *testing.T) {
	// Test that AgentLoopConfig properly accepts and stores Message field
	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    "/tmp/test",
		MaxIterations: 10,
		Message:       "Test message content",
	}

	if config.Message != "Test message content" {
		t.Errorf("Expected message 'Test message content', got '%s'", config.Message)
	}
}

func TestAgentLoopConfig_EmptyMessage(t *testing.T) {
	// Test that AgentLoopConfig handles empty Message field
	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    "/tmp/test",
		MaxIterations: 10,
		Message:       "",
	}

	if config.Message != "" {
		t.Errorf("Expected empty message, got '%s'", config.Message)
	}
}
