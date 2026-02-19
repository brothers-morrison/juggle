package supervisor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// OpenCodeBridge reads OpenCode session data and extracts state information
// for use by the supervisor to detect missed signals and sync state.
type OpenCodeBridge struct{}

// NewOpenCodeBridge creates a new OpenCode bridge
func NewOpenCodeBridge() *OpenCodeBridge {
	return &OpenCodeBridge{}
}

// SessionExport represents the structure returned by `opencode export`
type SessionExport struct {
	Info     SessionInfo      `json:"info"`
	Messages []SessionMessage `json:"messages"`
}

// SessionInfo contains session metadata
type SessionInfo struct {
	ID        string       `json:"id"`
	Title     string       `json:"title"`
	Directory string       `json:"directory"`
	TimeInfo  SessionTime  `json:"time"`
}

// SessionTime contains timing information
type SessionTime struct {
	Created int64 `json:"created"`
	Updated int64 `json:"updated"`
}

// SessionMessage represents a message in the export
type SessionMessage struct {
	Info  MessageInfo    `json:"info"`
	Parts []MessagePart  `json:"parts"`
}

// MessageInfo contains message metadata
type MessageInfo struct {
	Role   string `json:"role"`
	Finish string `json:"finish,omitempty"`
}

// MessagePart represents a part of a message
type MessagePart struct {
	Type   string `json:"type"`
	Text   string `json:"text,omitempty"`
	Tool   string `json:"tool,omitempty"`
	Input  any    `json:"input,omitempty"`
	Output string `json:"output,omitempty"`
}

// RecoverSession attempts to recover missed signals for a juggle session
// by reading the most recent OpenCode session data
func (b *OpenCodeBridge) RecoverSession(projectDir, sessionID string) {
	// Get the most recent OpenCode session for this project
	ocSessionID := b.findSessionForProject(projectDir)
	if ocSessionID == "" {
		return
	}

	export, err := b.exportSession(ocSessionID, projectDir)
	if err != nil {
		return
	}

	// Look for juggle tool calls in the last assistant messages
	juggleCalls := b.extractJuggleCalls(export)
	if len(juggleCalls) == 0 {
		return
	}

	// Replay any missed juggle commands
	for _, call := range juggleCalls {
		fmt.Fprintf(os.Stderr, "[supervisor/bridge] Replaying missed juggle call: %s\n", call)
		b.replayJuggleCall(call, projectDir)
	}
}

// GetRecentSessions returns a list of recent OpenCode sessions
func (b *OpenCodeBridge) GetRecentSessions(projectDir string) ([]SessionInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "opencode", "session", "list")
	if projectDir != "" {
		cmd.Dir = projectDir
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	var sessions []SessionInfo
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ses_") {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				sessions = append(sessions, SessionInfo{ID: fields[0]})
			}
		}
	}

	return sessions, nil
}

// findSessionForProject finds the most recent OpenCode session for a project directory
func (b *OpenCodeBridge) findSessionForProject(projectDir string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "opencode", "session", "list")
	if projectDir != "" {
		cmd.Dir = projectDir
	}

	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Parse output for session IDs
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ses_") {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				return fields[0]
			}
		}
	}

	return ""
}

// exportSession runs `opencode export` and parses the result
func (b *OpenCodeBridge) exportSession(sessionID, workingDir string) (*SessionExport, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "opencode", "export", sessionID)
	if workingDir != "" {
		cmd.Dir = workingDir
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var export SessionExport
	if err := json.Unmarshal(output, &export); err != nil {
		return nil, fmt.Errorf("failed to parse export: %w", err)
	}

	return &export, nil
}

// extractJuggleCalls finds bash tool calls that invoke juggle commands
func (b *OpenCodeBridge) extractJuggleCalls(export *SessionExport) []string {
	var calls []string

	// Only check the last few messages
	checkedMessages := 0
	for i := len(export.Messages) - 1; i >= 0 && checkedMessages < 5; i-- {
		msg := export.Messages[i]
		if msg.Info.Role != "assistant" {
			continue
		}
		checkedMessages++

		for _, part := range msg.Parts {
			if part.Type != "tool" {
				continue
			}

			toolName := strings.ToLower(part.Tool)
			if toolName != "bash" && toolName != "terminal" && toolName != "shell" {
				continue
			}

			// Extract the command from input
			inputStr := fmt.Sprintf("%v", part.Input)
			if !strings.Contains(inputStr, "juggle") {
				continue
			}

			// Look for juggle update/progress/complete/blocked commands
			if strings.Contains(inputStr, "juggle loop update") ||
				strings.Contains(inputStr, "juggle update") ||
				strings.Contains(inputStr, "juggle progress") ||
				strings.Contains(inputStr, "juggle blocked") ||
				strings.Contains(inputStr, "juggle complete") {
				// Extract just the juggle command
				cmd := extractJuggleCommand(inputStr)
				if cmd != "" {
					calls = append(calls, cmd)
				}
			}
		}
	}

	return calls
}

// extractJuggleCommand extracts a juggle command from a bash input string
func extractJuggleCommand(input string) string {
	// Find "juggle" in the input and extract to end of line or semicolon
	idx := strings.Index(input, "juggle")
	if idx < 0 {
		return ""
	}

	rest := input[idx:]

	// Trim at newline or semicolon
	if nlIdx := strings.IndexAny(rest, "\n;"); nlIdx >= 0 {
		rest = rest[:nlIdx]
	}

	// Trim at closing bracket/quote
	for _, ch := range []string{"]", "}", "'", "\""} {
		if chIdx := strings.Index(rest, ch); chIdx >= 0 {
			rest = rest[:chIdx]
		}
	}

	return strings.TrimSpace(rest)
}

// replayJuggleCall executes a juggle command that was missed
func (b *OpenCodeBridge) replayJuggleCall(command, projectDir string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	parts := strings.Fields(command)
	if len(parts) < 2 {
		return
	}

	// Find juggle binary
	juggleBin, err := exec.LookPath("juggle")
	if err != nil {
		return
	}

	// Execute: strip "juggle" prefix and run the rest
	cmd := exec.CommandContext(ctx, juggleBin, parts[1:]...)
	cmd.Dir = projectDir
	cmd.Stdout = os.Stderr // Log to stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "[supervisor/bridge] Replay failed for '%s': %v\n", command, err)
	}
}
