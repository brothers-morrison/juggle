package cli

import (
	"testing"
)

func TestWorktreeCmdHasWorkspaceAlias(t *testing.T) {
	// Verify the worktree command has "workspace" as an alias
	aliases := worktreeCmd.Aliases
	found := false
	for _, alias := range aliases {
		if alias == "workspace" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("worktree command should have 'workspace' alias, got aliases: %v", aliases)
	}
}

func TestWorktreeSubcommands(t *testing.T) {
	// Verify worktree has expected subcommands that will work via workspace alias
	expectedSubcmds := []string{"add", "forget", "list", "status", "run", "sync", "jump"}

	for _, name := range expectedSubcmds {
		found := false
		for _, cmd := range worktreeCmd.Commands() {
			if cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("worktree command should have '%s' subcommand", name)
		}
	}
}

func TestWorktreeJumpCmdFlags(t *testing.T) {
	// Verify jump command has expected flags
	flag := worktreeJumpCmd.Flags().Lookup("print")
	if flag == nil {
		t.Error("jump command should have --print flag")
	}
	if flag != nil && flag.Shorthand != "p" {
		t.Errorf("--print flag should have shorthand 'p', got '%s'", flag.Shorthand)
	}
}

func TestWorktreeRunCmdFlags(t *testing.T) {
	// Verify run command has expected flags
	testCases := []struct {
		name      string
		shorthand string
	}{
		{"continue-on-error", ""},
		{"list", "l"},
		{"set", "s"},
		{"delete", "d"},
	}

	for _, tc := range testCases {
		flag := worktreeRunCmd.Flags().Lookup(tc.name)
		if flag == nil {
			t.Errorf("run command should have --%s flag", tc.name)
			continue
		}
		if tc.shorthand != "" && flag.Shorthand != tc.shorthand {
			t.Errorf("--%s flag should have shorthand '%s', got '%s'", tc.name, tc.shorthand, flag.Shorthand)
		}
	}
}
