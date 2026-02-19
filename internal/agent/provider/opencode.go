package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// OpenCodeProvider implements Provider for OpenCode CLI
type OpenCodeProvider struct{}

// NewOpenCodeProvider creates a new OpenCode provider
func NewOpenCodeProvider() *OpenCodeProvider {
	return &OpenCodeProvider{}
}

// Type returns TypeOpenCode
func (o *OpenCodeProvider) Type() Type {
	return TypeOpenCode
}

// MapModel converts canonical model name to OpenCode format
// OpenCode uses: provider/model_id format (e.g., "anthropic/claude-opus-4-5")
func (o *OpenCodeProvider) MapModel(canonical string) string {
	switch canonical {
	case "haiku", "small":
		return "anthropic/claude-3-5-haiku-latest"
	case "sonnet", "medium":
		return "anthropic/claude-sonnet-4-5"
	case "opus", "large":
		return "anthropic/claude-opus-4-5"
	default:
		// Assume it's already in provider/model format or pass through
		return canonical
	}
}

// MapPermission converts PermissionMode to OpenCode's --agent flag
// OpenCode uses "agents" instead of permission modes:
// - build = full access (like acceptEdits/bypassPermissions)
// - plan = read-only (like plan mode)
func (o *OpenCodeProvider) MapPermission(mode PermissionMode) (flag, value string) {
	switch mode {
	case PermissionPlan:
		return "--agent", "plan"
	case PermissionAcceptEdits, PermissionBypass:
		return "--agent", "build"
	default:
		return "--agent", "build"
	}
}

// Run executes OpenCode CLI with the given options
func (o *OpenCodeProvider) Run(opts RunOptions) (*RunResult, error) {
	if opts.Mode == ModeInteractive {
		return o.runInteractive(opts)
	}
	return o.runHeadless(opts)
}

// runHeadless executes OpenCode in headless mode (opencode run "prompt")
func (o *OpenCodeProvider) runHeadless(opts RunOptions) (*RunResult, error) {
	result := &RunResult{}

	// OpenCode uses: opencode run "prompt"
	args := []string{"run"}

	// Set model if provided
	if opts.Model != "" {
		args = append(args, "--model", o.MapModel(opts.Model))
	}

	// Set agent (permission mode equivalent)
	flag, value := o.MapPermission(opts.Permission)
	args = append(args, flag, value)

	// OpenCode takes prompt as argument, not stdin
	args = append(args, opts.Prompt)

	// Create context with timeout if specified
	var ctx context.Context
	var cancel context.CancelFunc
	if opts.Timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), opts.Timeout)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	cmd := exec.CommandContext(ctx, "opencode", args...)
	if opts.WorkingDir != "" {
		cmd.Dir = opts.WorkingDir
	}

	var outputBuf strings.Builder

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start opencode: %w", err)
	}

	// Stream output to console and capture
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		streamOutput(stdout, &outputBuf, os.Stdout)
	}()
	go func() {
		defer wg.Done()
		streamOutput(stderr, &outputBuf, os.Stderr)
	}()

	// Wait for command to complete
	err = cmd.Wait()
	wg.Wait()
	result.Output = outputBuf.String()

	if err != nil {
		// Check if this was a timeout
		if ctx.Err() == context.DeadlineExceeded {
			result.TimedOut = true
			result.Error = fmt.Errorf("iteration timed out after %v", opts.Timeout)
			return result, nil
		}

		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		result.Error = fmt.Errorf("opencode exited with error: %w", err)
	}

	// Parse signals - same format as Claude since the prompt instructs the LLM
	parseSignals(result)

	// Signal recovery: if no signal found in stdout, try opencode export
	// OpenCode's stdout capture is unreliable - signals may be lost
	if !result.Complete && !result.Continue && !result.Blocked && !result.RateLimited && result.Error == nil {
		if recovered := o.recoverSignalsFromExport(opts.WorkingDir); recovered != nil {
			if recovered.Complete {
				result.Complete = true
				result.CommitMessage = recovered.CommitMessage
			}
			if recovered.Continue {
				result.Continue = true
				result.CommitMessage = recovered.CommitMessage
			}
			if recovered.Blocked {
				result.Blocked = true
				result.BlockedReason = recovered.BlockedReason
			}
		}
	}

	// Parse rate limits with OpenCode-specific patterns
	o.parseRateLimit(result)

	return result, nil
}

// runInteractive executes OpenCode in interactive mode (terminal TUI)
func (o *OpenCodeProvider) runInteractive(opts RunOptions) (*RunResult, error) {
	result := &RunResult{}

	// OpenCode interactive mode - no "run" subcommand
	args := []string{}

	// Set model if provided
	if opts.Model != "" {
		args = append(args, "--model", o.MapModel(opts.Model))
	}

	// Set agent (permission mode equivalent)
	flag, value := o.MapPermission(opts.Permission)
	args = append(args, flag, value)

	// Pass prompt via --prompt flag
	if opts.Prompt != "" {
		args = append(args, "--prompt", opts.Prompt)
	}

	// Create context with timeout if specified
	var ctx context.Context
	var cancel context.CancelFunc
	if opts.Timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), opts.Timeout)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	cmd := exec.CommandContext(ctx, "opencode", args...)
	if opts.WorkingDir != "" {
		cmd.Dir = opts.WorkingDir
	}

	// Inherit terminal for full TUI
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start opencode: %w", err)
	}

	// Wait for command to complete
	err := cmd.Wait()

	if err != nil {
		// Check if this was a timeout
		if ctx.Err() == context.DeadlineExceeded {
			result.TimedOut = true
			result.Error = fmt.Errorf("session timed out after %v", opts.Timeout)
			return result, nil
		}

		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		result.Error = fmt.Errorf("opencode exited with error: %w", err)
	}

	return result, nil
}

// parseRateLimit detects rate limit errors with OpenCode/OpenAI-specific patterns
func (o *OpenCodeProvider) parseRateLimit(result *RunResult) {
	output := strings.ToLower(result.Output)

	// Rate limit patterns - includes both Anthropic and OpenAI patterns
	// since OpenCode supports multiple providers
	rateLimitPatterns := []string{
		"rate limit",
		"rate_limit",
		"too many requests",
		"429",
		"overloaded",
		"capacity",
		"try again",
		"throttl",
		"quota",         // OpenAI specific
		"tpm limit",     // Tokens per minute
		"rpm limit",     // Requests per minute
		"exceeded your", // "exceeded your quota"
	}

	for _, pattern := range rateLimitPatterns {
		if strings.Contains(output, pattern) {
			result.RateLimited = true
			break
		}
	}

	// Also check error message if present
	if result.Error != nil {
		errStr := strings.ToLower(result.Error.Error())
		for _, pattern := range rateLimitPatterns {
			if strings.Contains(errStr, pattern) {
				result.RateLimited = true
				break
			}
		}
	}

	// Extract retry-after time if specified
	if result.RateLimited {
		result.RetryAfter = parseRetryAfter(result.Output)
	}

	// Check for overload exhaustion
	o.parseOverloadExhausted(result)
}

// recoverSignalsFromExport attempts to recover missed <promise> signals by
// running `opencode export` on the most recent session. This handles the common
// case where OpenCode's stdout doesn't reliably flush the LLM's signal output.
func (o *OpenCodeProvider) recoverSignalsFromExport(workingDir string) *RunResult {
	// Get the most recent session ID
	sessionID := o.getMostRecentSession(workingDir)
	if sessionID == "" {
		return nil
	}

	// Export session data
	exportOutput, err := o.runOpenCodeExport(sessionID, workingDir)
	if err != nil || exportOutput == "" {
		return nil
	}

	// Parse the export JSON and extract text from the last assistant message
	lastAssistantText := extractLastAssistantText(exportOutput)
	if lastAssistantText == "" {
		return nil
	}

	// Check for signals in the extracted text
	recovered := &RunResult{Output: lastAssistantText}
	parseSignals(recovered)

	if recovered.Complete || recovered.Continue || recovered.Blocked {
		fmt.Fprintf(os.Stderr, "[juggle] Recovered signal from OpenCode export (session %s)\n", sessionID)
		return recovered
	}

	// Also check for juggle tool calls that indicate completion
	// e.g., the agent ran `juggle loop update --state complete` but the signal was lost
	if o.checkForJuggleToolCalls(exportOutput) {
		fmt.Fprintf(os.Stderr, "[juggle] Detected juggle tool calls in OpenCode export (session %s), treating as CONTINUE\n", sessionID)
		return &RunResult{Continue: true}
	}

	return nil
}

// getMostRecentSession runs `opencode session list` and returns the most recent session ID
func (o *OpenCodeProvider) getMostRecentSession(workingDir string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "opencode", "session", "list")
	if workingDir != "" {
		cmd.Dir = workingDir
	}

	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Parse the table output - first data line after the separator has the most recent session
	// Format:
	// Session ID                      Title                           Updated
	// ──────────────────────────────────────────────────────────────────────
	// ses_xxxxx                        ...                             ...
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ses_") {
			// Extract session ID (first whitespace-delimited field)
			fields := strings.Fields(line)
			if len(fields) > 0 {
				return fields[0]
			}
		}
	}

	return ""
}

// runOpenCodeExport runs `opencode export <sessionID>` and returns the JSON output
func (o *OpenCodeProvider) runOpenCodeExport(sessionID, workingDir string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "opencode", "export", sessionID)
	if workingDir != "" {
		cmd.Dir = workingDir
	}

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// openCodeExport represents the top-level structure of opencode export JSON
type openCodeExport struct {
	Messages []openCodeMessage `json:"messages"`
}

// openCodeMessage represents a message in the export
type openCodeMessage struct {
	Info  openCodeMessageInfo `json:"info"`
	Parts []openCodePart      `json:"parts"`
}

// openCodeMessageInfo contains message metadata
type openCodeMessageInfo struct {
	Role string `json:"role"`
}

// openCodePart represents a part of a message (text, tool call, etc.)
type openCodePart struct {
	Type   string `json:"type"`
	Text   string `json:"text"`
	Tool   string `json:"tool"`
	Input  any    `json:"input"`
	Output string `json:"output"`
}

// extractLastAssistantText parses export JSON and returns concatenated text
// from the last assistant message's text parts
func extractLastAssistantText(exportJSON string) string {
	var export openCodeExport
	if err := json.Unmarshal([]byte(exportJSON), &export); err != nil {
		return ""
	}

	// Find the last assistant message
	var lastAssistant *openCodeMessage
	for i := len(export.Messages) - 1; i >= 0; i-- {
		if export.Messages[i].Info.Role == "assistant" {
			lastAssistant = &export.Messages[i]
			break
		}
	}

	if lastAssistant == nil {
		return ""
	}

	// Concatenate all text parts and tool output parts
	var texts []string
	for _, part := range lastAssistant.Parts {
		if part.Type == "text" && part.Text != "" {
			texts = append(texts, part.Text)
		}
		if part.Type == "tool" && part.Output != "" {
			texts = append(texts, part.Output)
		}
	}

	return strings.Join(texts, "\n")
}

// checkForJuggleToolCalls scans the export JSON for recent juggle tool invocations
// that indicate the agent was updating state (e.g., `juggle loop update`, `juggle update`)
func (o *OpenCodeProvider) checkForJuggleToolCalls(exportJSON string) bool {
	var export openCodeExport
	if err := json.Unmarshal([]byte(exportJSON), &export); err != nil {
		return false
	}

	// Check the last few assistant messages for juggle-related tool calls
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

			// Check tool name
			toolName := strings.ToLower(part.Tool)
			if toolName == "bash" || toolName == "terminal" || toolName == "shell" {
				// Check if the command input references juggle
				inputStr := fmt.Sprintf("%v", part.Input)
				if strings.Contains(inputStr, "juggle") &&
					(strings.Contains(inputStr, "update") ||
						strings.Contains(inputStr, "complete") ||
						strings.Contains(inputStr, "blocked") ||
						strings.Contains(inputStr, "progress")) {
					return true
				}
			}
		}
	}

	return false
}

// parseOverloadExhausted detects when the agent has exited after exhausting retries
func (o *OpenCodeProvider) parseOverloadExhausted(result *RunResult) {
	output := strings.ToLower(result.Output)

	exhaustionPatterns := []string{
		"529",
		"overloaded_error",
		"api is overloaded",
		"exhausted.*retry",
		"maximum.*retries",
		"quota exceeded",
	}

	if result.Error == nil && result.ExitCode == 0 {
		return
	}

	for _, pattern := range exhaustionPatterns {
		if strings.Contains(output, pattern) {
			result.OverloadExhausted = true
			return
		}
	}

	if result.ExitCode != 0 && strings.Contains(output, "overloaded") {
		result.OverloadExhausted = true
	}
}
