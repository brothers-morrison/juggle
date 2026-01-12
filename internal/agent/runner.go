// Package agent provides the agent prompt template and runner interface
// for running AI agents with juggler.
package agent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// RunMode defines how Claude should be executed
type RunMode string

const (
	// ModeHeadless runs with -p flag, autonomous system prompt, captures output
	ModeHeadless RunMode = "headless"
	// ModeInteractive runs without -p flag, inherits terminal for full TUI
	ModeInteractive RunMode = "interactive"
)

// PermissionMode defines Claude's permission level
type PermissionMode string

const (
	// PermissionAcceptEdits allows Claude to edit files with confirmation
	PermissionAcceptEdits PermissionMode = "acceptEdits"
	// PermissionPlan starts Claude in plan mode for review/planning
	PermissionPlan PermissionMode = "plan"
	// PermissionBypass bypasses all permission checks (dangerous)
	PermissionBypass PermissionMode = "bypassPermissions"
)

// RunOptions configures how Claude is executed
type RunOptions struct {
	Prompt       string         // The prompt to send to Claude
	Mode         RunMode        // headless vs interactive
	Permission   PermissionMode // acceptEdits, plan, bypassPermissions
	Timeout      time.Duration  // timeout per invocation (0 = no timeout)
	SystemPrompt string         // optional additional system prompt to append
	Model        string         // optional model to use (e.g., "opus", "sonnet", "haiku")
}

// RunResult represents the outcome of a single agent run
type RunResult struct {
	Output        string
	ExitCode      int
	Complete      bool
	Continue      bool // Agent completed one ball, more remain - signals loop to continue to next iteration
	Blocked       bool
	BlockedReason string
	TimedOut      bool
	RateLimited   bool          // Claude returned a rate limit error
	RetryAfter    time.Duration // Suggested wait time from rate limit response (0 if not specified)
	Error         error
}

// Runner defines the interface for running AI agents.
// Implementations must execute an agent with options and return the result.
type Runner interface {
	// Run executes the agent with the given options and returns the result.
	Run(opts RunOptions) (*RunResult, error)
}

// DefaultRunner is the package-level runner used for agent operations.
// It can be replaced with a mock for testing via SetRunner.
var DefaultRunner Runner = &ClaudeRunner{}

// SetRunner sets the package-level runner (for testing).
func SetRunner(r Runner) {
	DefaultRunner = r
}

// ResetRunner resets the runner to the default ClaudeRunner.
func ResetRunner() {
	DefaultRunner = &ClaudeRunner{}
}

// AutonomousSystemPrompt is appended to force autonomous operation in headless mode
const AutonomousSystemPrompt = `CRITICAL: You are an autonomous agent. DO NOT ask questions. DO NOT summarize. DO NOT wait for confirmation. START WORKING IMMEDIATELY. Execute the workflow in prompt.md without any preamble.`

// ClaudeRunner runs the Claude CLI as an AI agent
type ClaudeRunner struct{}

// Run executes Claude with the given options
func (r *ClaudeRunner) Run(opts RunOptions) (*RunResult, error) {
	if opts.Mode == ModeInteractive {
		return r.runInteractive(opts)
	}
	return r.runHeadless(opts)
}

// runHeadless executes Claude in headless mode (-p flag, captured output)
func (r *ClaudeRunner) runHeadless(opts RunOptions) (*RunResult, error) {
	result := &RunResult{}

	// Build command arguments
	args := []string{
		"--disable-slash-commands",
	}

	// Append system prompt if provided
	if opts.SystemPrompt != "" {
		args = append(args, "--append-system-prompt", opts.SystemPrompt)
	}

	// Set model if provided
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}

	// Set permission mode
	if opts.Permission == PermissionBypass {
		args = append(args, "--dangerously-skip-permissions")
	} else {
		args = append(args, "--permission-mode", string(opts.Permission))
	}

	// Headless mode: read prompt from stdin
	args = append(args, "-p", "-")

	// Create context with timeout if specified
	var ctx context.Context
	var cancel context.CancelFunc
	if opts.Timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), opts.Timeout)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	cmd := exec.CommandContext(ctx, "claude", args...)

	var outputBuf strings.Builder

	// Pipe prompt through stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

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
		return nil, fmt.Errorf("failed to start claude: %w", err)
	}

	// Write prompt to stdin
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, opts.Prompt)
	}()

	// Stream output to console and capture
	go streamOutput(stdout, &outputBuf, os.Stdout)
	go streamOutput(stderr, &outputBuf, os.Stderr)

	// Wait for command to complete
	err = cmd.Wait()
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
		result.Error = fmt.Errorf("claude exited with error: %w", err)
	}

	// Parse completion signals from output
	r.parseSignals(result)

	return result, nil
}

// runInteractive executes Claude in interactive mode (terminal TUI)
func (r *ClaudeRunner) runInteractive(opts RunOptions) (*RunResult, error) {
	result := &RunResult{}

	// Build command arguments
	args := []string{
		"--disable-slash-commands",
	}

	// Append system prompt if provided
	if opts.SystemPrompt != "" {
		args = append(args, "--append-system-prompt", opts.SystemPrompt)
	}

	// Set model if provided
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}

	// Set permission mode
	if opts.Permission == PermissionBypass {
		args = append(args, "--dangerously-skip-permissions")
	} else {
		args = append(args, "--permission-mode", string(opts.Permission))
	}

	// Interactive mode: pass prompt as argument
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

	cmd := exec.CommandContext(ctx, "claude", args...)

	// Inherit terminal for full TUI
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start claude: %w", err)
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
		result.Error = fmt.Errorf("claude exited with error: %w", err)
	}

	return result, nil
}

// parseSignals checks the output for COMPLETE/CONTINUE/BLOCKED signals and rate limits
func (r *ClaudeRunner) parseSignals(result *RunResult) {
	if strings.Contains(result.Output, "<promise>COMPLETE</promise>") {
		result.Complete = true
	}

	if strings.Contains(result.Output, "<promise>CONTINUE</promise>") {
		result.Continue = true
	}

	if idx := strings.Index(result.Output, "<promise>BLOCKED:"); idx != -1 {
		endIdx := strings.Index(result.Output[idx:], "</promise>")
		if endIdx != -1 {
			reason := strings.TrimSpace(result.Output[idx+len("<promise>BLOCKED:") : idx+endIdx])
			result.Blocked = true
			result.BlockedReason = reason
		}
	}

	// Check for rate limit indicators in output
	r.parseRateLimit(result)
}

// parseRateLimit detects rate limit errors and extracts retry-after time if available
func (r *ClaudeRunner) parseRateLimit(result *RunResult) {
	output := strings.ToLower(result.Output)

	// Common rate limit patterns from Claude API
	rateLimitPatterns := []string{
		"rate limit",
		"rate_limit",
		"too many requests",
		"429",
		"overloaded",
		"capacity",
		"try again",
		"throttl",
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
}

// parseRetryAfter extracts wait time from rate limit messages
// Looks for patterns like "try again in 30 seconds", "retry after 1 minute", etc.
func parseRetryAfter(output string) time.Duration {
	output = strings.ToLower(output)

	// Pattern: "X seconds" or "X minutes" or "X hours"
	patterns := []struct {
		unit       string
		multiplier time.Duration
	}{
		{"second", time.Second},
		{"minute", time.Minute},
		{"hour", time.Hour},
	}

	for _, p := range patterns {
		// Look for number followed by unit
		idx := strings.Index(output, p.unit)
		if idx > 0 {
			// Search backwards for a number
			numStr := ""
			for i := idx - 1; i >= 0 && i >= idx-5; i-- {
				c := output[i]
				if c >= '0' && c <= '9' {
					numStr = string(c) + numStr
				} else if len(numStr) > 0 {
					break
				}
			}
			if len(numStr) > 0 {
				var num int
				fmt.Sscanf(numStr, "%d", &num)
				if num > 0 {
					return time.Duration(num) * p.multiplier
				}
			}
		}
	}

	return 0
}

// streamOutput reads from reader and writes to both buffer and writer
func streamOutput(reader io.Reader, buf *strings.Builder, writer io.Writer) {
	scanner := bufio.NewScanner(reader)
	// Increase scanner buffer for long lines
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		buf.WriteString(line)
		buf.WriteString("\n")
		fmt.Fprintln(writer, line)
	}
}

// MockRunner is a test implementation of Runner
type MockRunner struct {
	// Responses is a queue of results to return (FIFO)
	Responses []*RunResult
	// Calls records all calls made to Run
	Calls []RunOptions
	// NextIndex tracks which response to return next
	NextIndex int
}

// NewMockRunner creates a new MockRunner with the given responses
func NewMockRunner(responses ...*RunResult) *MockRunner {
	return &MockRunner{
		Responses: responses,
		Calls:     make([]RunOptions, 0),
	}
}

// Run records the call and returns the next queued response
func (m *MockRunner) Run(opts RunOptions) (*RunResult, error) {
	m.Calls = append(m.Calls, opts)

	if m.NextIndex >= len(m.Responses) {
		// Return a default blocked result if no more responses queued
		return &RunResult{
			Output:        "No more mock responses",
			Blocked:       true,
			BlockedReason: "MockRunner exhausted",
		}, nil
	}

	result := m.Responses[m.NextIndex]
	m.NextIndex++
	return result, nil
}

// Reset clears call history and resets response index
func (m *MockRunner) Reset() {
	m.Calls = make([]RunOptions, 0)
	m.NextIndex = 0
}

// SetResponses replaces the response queue
func (m *MockRunner) SetResponses(responses ...*RunResult) {
	m.Responses = responses
	m.NextIndex = 0
}
