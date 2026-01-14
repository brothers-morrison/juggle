package integration_test

import (
	"strings"
	"testing"

	"github.com/ohare93/juggle/internal/cli"
)

// TestSuggestCommandSwap tests the command swap suggestion for reversed commands
func TestSuggestCommandSwap(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "run agent -> agent run",
			args:     []string{"run", "agent"},
			expected: "Did you mean 'juggle agent run'?",
		},
		{
			name:     "refine agent -> agent refine",
			args:     []string{"refine", "agent"},
			expected: "Did you mean 'juggle agent refine'?",
		},
		{
			name:     "add projects -> projects add",
			args:     []string{"add", "projects"},
			expected: "Did you mean 'juggle projects add'?",
		},
		{
			name:     "remove projects -> projects remove",
			args:     []string{"remove", "projects"},
			expected: "Did you mean 'juggle projects remove'?",
		},
		{
			name:     "ralph import -> import ralph",
			args:     []string{"ralph", "import"},
			expected: "Did you mean 'juggle import ralph'?",
		},
		{
			name:     "append progress -> progress append",
			args:     []string{"append", "progress"},
			expected: "Did you mean 'juggle progress append'?",
		},
		{
			name:     "not a command swap",
			args:     []string{"myball", "complete"},
			expected: "",
		},
		{
			name:     "single arg",
			args:     []string{"run"},
			expected: "",
		},
		{
			name:     "unknown second arg",
			args:     []string{"run", "unknown"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cli.SuggestCommandSwap(tt.args)
			if got != tt.expected {
				t.Errorf("SuggestCommandSwap(%v) = %q, want %q", tt.args, got, tt.expected)
			}
		})
	}
}

// TestIsKnownCommand tests command recognition
func TestIsKnownCommand(t *testing.T) {
	tests := []struct {
		cmd      string
		expected bool
	}{
		{"agent", true},
		{"balls", true},
		{"status", true},
		{"sessions", true},
		{"config", true},
		{"worktree", true},
		{"run", false},        // 'run' is a subcommand of agent, not a top-level command
		{"refine", false},     // 'refine' is a subcommand of agent, not a top-level command
		{"nonexistent", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			got := cli.IsKnownCommand(tt.cmd)
			if got != tt.expected {
				t.Errorf("IsKnownCommand(%q) = %v, want %v", tt.cmd, got, tt.expected)
			}
		})
	}
}

// TestIsKnownSubcommand tests subcommand recognition
func TestIsKnownSubcommand(t *testing.T) {
	tests := []struct {
		parent   string
		child    string
		expected bool
	}{
		{"agent", "run", true},
		{"agent", "refine", true},
		{"agent", "unknown", false},
		{"sessions", "create", true},
		{"sessions", "list", true},
		{"sessions", "delete", true},
		{"sessions", "run", false},
		{"nonexistent", "anything", false},
		{"balls", "anything", false}, // balls has no subcommands
	}

	for _, tt := range tests {
		t.Run(tt.parent+"_"+tt.child, func(t *testing.T) {
			got := cli.IsKnownSubcommand(tt.parent, tt.child)
			if got != tt.expected {
				t.Errorf("IsKnownSubcommand(%q, %q) = %v, want %v", tt.parent, tt.child, got, tt.expected)
			}
		})
	}
}

// TestEnhanceBallNotFoundError tests error message enhancement
func TestEnhanceBallNotFoundError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		ballID      string
		args        []string
		wantContain string
	}{
		{
			name:        "ball ID is a command with subcommands",
			err:         &testError{"ball not found in current project: agent"},
			ballID:      "agent",
			args:        []string{"agent"},
			wantContain: "'agent' is a command, not a ball ID",
		},
		{
			name:        "ball ID is a command without subcommands",
			err:         &testError{"ball not found in current project: balls"},
			ballID:      "balls",
			args:        []string{"balls"},
			wantContain: "'balls' is a command, not a ball ID",
		},
		{
			name:        "ball ID with unused arguments",
			err:         &testError{"ball not found in current project: myball"},
			ballID:      "myball",
			args:        []string{"myball", "extra", "args"},
			wantContain: "unused arguments after ball ID: extra args",
		},
		{
			name:        "non not-found error unchanged",
			err:         &testError{"ambiguous ID 'test' matches 2 balls"},
			ballID:      "test",
			args:        []string{"test"},
			wantContain: "ambiguous ID 'test' matches 2 balls",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cli.EnhanceBallNotFoundError(tt.err, tt.ballID, tt.args)
			if !strings.Contains(got.Error(), tt.wantContain) {
				t.Errorf("EnhanceBallNotFoundError() = %q, want it to contain %q", got.Error(), tt.wantContain)
			}
		})
	}
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
