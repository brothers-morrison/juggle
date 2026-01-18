package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var hooksCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Manage Claude Code hooks for enhanced progress tracking",
	Long:  `Commands for managing Claude Code hooks that provide enhanced progress tracking.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var hooksInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Claude Code hooks for juggler integration",
	Long: `Install Claude Code hooks that report progress to juggler.

These hooks automatically track:
  - Files changed (from Write/Edit tools)
  - Tool execution counts
  - Tool failures
  - Token usage
  - Turn counts
  - Session end events

By default, hooks are installed to .claude/settings.local.json in the current
project directory. This file is typically gitignored.

A backup is created before modifying the settings file.

Examples:
  juggle hooks install              # Install to .claude/settings.local.json (default)
  juggle hooks install --project    # Install to .claude/settings.json (version controlled)
  juggle hooks install --global     # Install to ~/.claude/settings.json (all projects)`,
	RunE: runHooksInstall,
}

var hooksStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check if Claude Code hooks are installed",
	RunE:  runHooksStatus,
}

var (
	hooksProjectFlag bool
	hooksGlobalFlag  bool
)

func init() {
	hooksInstallCmd.Flags().BoolVar(&hooksProjectFlag, "project", false, "Install to project .claude/settings.json (version controlled)")
	hooksInstallCmd.Flags().BoolVar(&hooksGlobalFlag, "global", false, "Install to ~/.claude/settings.json (all projects)")
	hooksCmd.AddCommand(hooksInstallCmd)
	hooksCmd.AddCommand(hooksStatusCmd)
	rootCmd.AddCommand(hooksCmd)
}

// ClaudeSettings represents the structure of .claude/settings.json
type ClaudeSettings struct {
	Hooks map[string][]HookMatcher `json:"hooks,omitempty"`
	// Preserve other fields
	Other map[string]json.RawMessage `json:"-"`
}

// HookMatcher represents a hook matcher configuration
type HookMatcher struct {
	Matcher string       `json:"matcher,omitempty"`
	Hooks   []HookConfig `json:"hooks"`
}

// HookConfig represents a single hook configuration
type HookConfig struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// JugglerHookConfig returns the hook configuration for juggler
func JugglerHookConfig() map[string][]HookMatcher {
	return map[string][]HookMatcher{
		"PostToolUse": {
			{
				Matcher: "Write|Edit|Bash",
				Hooks: []HookConfig{
					{Type: "command", Command: "juggle loop hook-event post-tool"},
				},
			},
		},
		"PostToolUseFailure": {
			{
				Matcher: "Write|Edit|Bash",
				Hooks: []HookConfig{
					{Type: "command", Command: "juggle loop hook-event tool-failure"},
				},
			},
		},
		"Stop": {
			{
				Hooks: []HookConfig{
					{Type: "command", Command: "juggle loop hook-event stop"},
				},
			},
		},
		"SessionEnd": {
			{
				Hooks: []HookConfig{
					{Type: "command", Command: "juggle loop hook-event session-end"},
				},
			},
		},
	}
}

func runHooksInstall(cmd *cobra.Command, args []string) error {
	settingsPath, err := getSettingsPath()
	if err != nil {
		return err
	}

	// Create backup if file exists
	if _, err := os.Stat(settingsPath); err == nil {
		backupPath := settingsPath + ".backup." + time.Now().Format("20060102-150405")
		data, err := os.ReadFile(settingsPath)
		if err != nil {
			return fmt.Errorf("failed to read existing settings for backup: %w", err)
		}
		if err := os.WriteFile(backupPath, data, 0644); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		fmt.Printf("Created backup: %s\n", backupPath)
	}

	// Load existing settings or create new
	settings, err := loadClaudeSettings(settingsPath)
	if err != nil {
		return err
	}

	// Merge juggler hooks
	jugglerHooks := JugglerHookConfig()
	if settings.Hooks == nil {
		settings.Hooks = make(map[string][]HookMatcher)
	}

	for hookType, matchers := range jugglerHooks {
		// Check if juggler hook already exists for this type
		if !hasJugglerHook(settings.Hooks[hookType]) {
			settings.Hooks[hookType] = append(settings.Hooks[hookType], matchers...)
		}
	}

	// Save updated settings
	if err := saveClaudeSettings(settingsPath, settings); err != nil {
		return err
	}

	fmt.Printf("Installed juggler hooks to: %s\n", settingsPath)
	fmt.Println("\nHooks installed for:")
	fmt.Println("  - PostToolUse (tracks file changes, tool counts)")
	fmt.Println("  - PostToolUseFailure (tracks errors)")
	fmt.Println("  - Stop (tracks turns, token usage)")
	fmt.Println("  - SessionEnd (marks session completion)")
	fmt.Println("\nNote: Hooks require JUGGLE_SESSION_ID env var to be set.")
	fmt.Println("This is automatically set by 'juggle agent start'.")

	return nil
}

func runHooksStatus(cmd *cobra.Command, args []string) error {
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Define all settings files to check (in priority order)
	settingsFiles := []struct {
		path  string
		label string
	}{
		{filepath.Join(cwd, ".claude", "settings.local.json"), "Project local (gitignored)"},
		{filepath.Join(cwd, ".claude", "settings.json"), "Project (version controlled)"},
		{filepath.Join(homeDir, ".claude", "settings.json"), "User global"},
	}

	hookTypes := []string{"PostToolUse", "PostToolUseFailure", "Stop", "SessionEnd"}
	foundAny := false

	for _, sf := range settingsFiles {
		if _, err := os.Stat(sf.path); os.IsNotExist(err) {
			continue
		}

		settings, err := loadClaudeSettings(sf.path)
		if err != nil {
			continue
		}

		// Check if any juggler hooks are in this file
		hasAnyHook := false
		for _, hookType := range hookTypes {
			if hasJugglerHook(settings.Hooks[hookType]) {
				hasAnyHook = true
				break
			}
		}

		if !hasAnyHook {
			continue
		}

		foundAny = true
		fmt.Printf("%s:\n  %s\n\n", sf.label, sf.path)

		allInstalled := true
		for _, hookType := range hookTypes {
			installed := hasJugglerHook(settings.Hooks[hookType])
			status := "not installed"
			if installed {
				status = "installed"
			} else {
				allInstalled = false
			}
			fmt.Printf("  %-20s %s\n", hookType+":", status)
		}
		fmt.Println()

		if allInstalled {
			fmt.Println("All juggler hooks are installed in this file.")
		}
		return nil // Found hooks, done
	}

	if !foundAny {
		fmt.Println("Juggler hooks are not installed in any settings file.")
		fmt.Println()
		fmt.Println("Checked locations:")
		for _, sf := range settingsFiles {
			exists := "not found"
			if _, err := os.Stat(sf.path); err == nil {
				exists = "exists (no juggler hooks)"
			}
			fmt.Printf("  %s: %s\n", sf.label, exists)
		}
		fmt.Println()
		fmt.Println("Run 'juggle hooks install' to install hooks.")
	}

	return nil
}

func getSettingsPath() (string, error) {
	if hooksGlobalFlag {
		// User-level settings (all projects)
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		return filepath.Join(homeDir, ".claude", "settings.json"), nil
	}

	cwd, err := GetWorkingDir()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	if hooksProjectFlag {
		// Project settings (version controlled)
		return filepath.Join(cwd, ".claude", "settings.json"), nil
	}

	// Default: project-local settings (gitignored)
	return filepath.Join(cwd, ".claude", "settings.local.json"), nil
}

func loadClaudeSettings(path string) (*ClaudeSettings, error) {
	settings := &ClaudeSettings{
		Hooks: make(map[string][]HookMatcher),
		Other: make(map[string]json.RawMessage),
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return settings, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read settings file: %w", err)
	}

	// First unmarshal into a generic map to preserve unknown fields
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse settings file: %w", err)
	}

	// Extract hooks field
	if hooksRaw, ok := raw["hooks"]; ok {
		if err := json.Unmarshal(hooksRaw, &settings.Hooks); err != nil {
			return nil, fmt.Errorf("failed to parse hooks field: %w", err)
		}
		delete(raw, "hooks")
	}

	// Store remaining fields
	settings.Other = raw

	return settings, nil
}

func saveClaudeSettings(path string, settings *ClaudeSettings) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create settings directory: %w", err)
	}

	// Reconstruct the full settings map
	output := make(map[string]interface{})

	// Add hooks
	if len(settings.Hooks) > 0 {
		output["hooks"] = settings.Hooks
	}

	// Add other preserved fields
	for key, value := range settings.Other {
		var v interface{}
		json.Unmarshal(value, &v)
		output[key] = v
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	return nil
}

// hasJugglerHook checks if any hook in the list contains a juggle command
func hasJugglerHook(matchers []HookMatcher) bool {
	for _, matcher := range matchers {
		for _, hook := range matcher.Hooks {
			if len(hook.Command) >= 6 && hook.Command[:6] == "juggle" {
				return true
			}
		}
	}
	return false
}

// AreHooksInstalled checks if juggler hooks are installed in any settings file
func AreHooksInstalled() bool {
	cwd, err := os.Getwd()
	if err != nil {
		return false
	}

	// Check project-local settings first (highest priority)
	localPath := filepath.Join(cwd, ".claude", "settings.local.json")
	if checkHooksInFile(localPath) {
		return true
	}

	// Check project settings
	projectPath := filepath.Join(cwd, ".claude", "settings.json")
	if checkHooksInFile(projectPath) {
		return true
	}

	// Check user settings (lowest priority)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	userPath := filepath.Join(homeDir, ".claude", "settings.json")
	return checkHooksInFile(userPath)
}

func checkHooksInFile(path string) bool {
	settings, err := loadClaudeSettings(path)
	if err != nil {
		return false
	}

	// Check if at least the core hooks are installed
	return hasJugglerHook(settings.Hooks["PostToolUse"]) &&
		hasJugglerHook(settings.Hooks["Stop"])
}
