package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

By default, hooks are installed to .claude/settings.json in the current
project directory. This file is version controlled and shared with the team.

A backup is created before modifying the settings file.

Examples:
  juggle hooks install              # Install to .claude/settings.json (default, version controlled)
  juggle hooks install --local      # Install to .claude/settings.local.json (gitignored)
  juggle hooks install --global     # Install to ~/.claude/settings.json (all projects)`,
	RunE: runHooksInstall,
}

var hooksStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check if Claude Code hooks are installed",
	RunE:  runHooksStatus,
}

var (
	hooksLocalFlag  bool
	hooksGlobalFlag bool
)

func init() {
	hooksInstallCmd.Flags().BoolVar(&hooksLocalFlag, "local", false, "Install to .claude/settings.local.json (gitignored)")
	hooksInstallCmd.Flags().BoolVar(&hooksGlobalFlag, "global", false, "Install to ~/.claude/settings.json (all projects)")
	hooksCmd.AddCommand(hooksInstallCmd)
	hooksCmd.AddCommand(hooksStatusCmd)
	rootCmd.AddCommand(hooksCmd)
}

// ClaudeSettings represents the structure of .claude/settings.json
type ClaudeSettings struct {
	// SandboxRaw stores the complete sandbox config as raw JSON to preserve unknown fields.
	// Use GetSandboxConfig() to access parsed values, SetSandboxConfig() to set values.
	SandboxRaw  json.RawMessage            `json:"-"`
	Permissions *PermissionsConfig         `json:"permissions,omitempty"`
	Hooks       map[string][]HookMatcher   `json:"hooks,omitempty"`
	Other       map[string]json.RawMessage `json:"-"` // Preserves unknown fields
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

// SandboxConfig represents known Claude Code sandbox settings.
// Note: This struct intentionally only contains fields that juggler uses.
// The full sandbox config is preserved in ClaudeSettings.SandboxRaw.
type SandboxConfig struct {
	Enabled                  bool `json:"enabled,omitempty"`
	AutoAllowBashIfSandboxed bool `json:"autoAllowBashIfSandboxed,omitempty"`
	AllowUnsandboxedCommands bool `json:"allowUnsandboxedCommands,omitempty"`
}

// GetSandboxConfig parses and returns the known sandbox config fields.
// Returns nil if no sandbox config is set.
func (s *ClaudeSettings) GetSandboxConfig() *SandboxConfig {
	if len(s.SandboxRaw) == 0 {
		return nil
	}
	var config SandboxConfig
	if err := json.Unmarshal(s.SandboxRaw, &config); err != nil {
		return nil
	}
	return &config
}

// SetSandboxConfig merges the given config into the existing sandbox config,
// preserving any unknown fields. If no existing config, creates a new one.
func (s *ClaudeSettings) SetSandboxConfig(config *SandboxConfig) error {
	if config == nil {
		s.SandboxRaw = nil
		return nil
	}

	// Parse existing config into a map to preserve unknown fields
	var existing map[string]interface{}
	if len(s.SandboxRaw) > 0 {
		if err := json.Unmarshal(s.SandboxRaw, &existing); err != nil {
			existing = make(map[string]interface{})
		}
	} else {
		existing = make(map[string]interface{})
	}

	// Merge known fields
	existing["enabled"] = config.Enabled
	existing["autoAllowBashIfSandboxed"] = config.AutoAllowBashIfSandboxed
	existing["allowUnsandboxedCommands"] = config.AllowUnsandboxedCommands

	// Marshal back
	data, err := json.Marshal(existing)
	if err != nil {
		return err
	}
	s.SandboxRaw = data
	return nil
}

// PermissionsConfig represents Claude Code permission rules
type PermissionsConfig struct {
	Allow []string `json:"allow,omitempty"`
	Deny  []string `json:"deny,omitempty"`
	Ask   []string `json:"ask,omitempty"`
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

// DefaultClaudeSettings returns the bare-bones Claude settings for juggle projects.
// These settings enable sandbox mode and hooks while protecting secrets.
func DefaultClaudeSettings() *ClaudeSettings {
	settings := &ClaudeSettings{
		Permissions: &PermissionsConfig{
			Allow: []string{"Bash(juggle:*)"},
			Deny:  []string{"Read(./.env)", "Read(./.env.*)", "Read(./secrets/**)"},
			Ask:   []string{"Bash(juggle agent:*)", "Bash(jj git push:*)", "Bash(git push:*)"},
		},
		Hooks: JugglerHookConfig(),
	}
	// Set sandbox config
	_ = settings.SetSandboxConfig(&SandboxConfig{
		Enabled:                  true,
		AutoAllowBashIfSandboxed: true,
		AllowUnsandboxedCommands: true,
	})
	return settings
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
	settings, err := LoadClaudeSettings(settingsPath)
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
	if err := SaveClaudeSettings(settingsPath, settings); err != nil {
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

		settings, err := LoadClaudeSettings(sf.path)
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

	if hooksLocalFlag {
		// Project-local settings (gitignored)
		return filepath.Join(cwd, ".claude", "settings.local.json"), nil
	}

	// Default: project settings (version controlled)
	return filepath.Join(cwd, ".claude", "settings.json"), nil
}

// LoadClaudeSettings loads Claude settings from the given path.
// It preserves unknown fields in the Other map for round-trip safety.
func LoadClaudeSettings(path string) (*ClaudeSettings, error) {
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

	// Extract sandbox field as raw JSON to preserve all fields
	if sandboxRaw, ok := raw["sandbox"]; ok {
		settings.SandboxRaw = sandboxRaw
		delete(raw, "sandbox")
	}

	// Extract permissions field
	if permissionsRaw, ok := raw["permissions"]; ok {
		settings.Permissions = &PermissionsConfig{}
		if err := json.Unmarshal(permissionsRaw, settings.Permissions); err != nil {
			return nil, fmt.Errorf("failed to parse permissions field: %w", err)
		}
		delete(raw, "permissions")
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

// SaveClaudeSettings saves Claude settings to the given path.
// It preserves unknown fields from the Other map.
func SaveClaudeSettings(path string, settings *ClaudeSettings) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create settings directory: %w", err)
	}

	// Reconstruct the full settings map
	output := make(map[string]interface{})

	// Add sandbox (as raw JSON to preserve all fields)
	if len(settings.SandboxRaw) > 0 {
		var sandbox interface{}
		if err := json.Unmarshal(settings.SandboxRaw, &sandbox); err == nil {
			output["sandbox"] = sandbox
		}
	}

	// Add permissions
	if settings.Permissions != nil {
		output["permissions"] = settings.Permissions
	}

	// Add hooks
	if len(settings.Hooks) > 0 {
		output["hooks"] = settings.Hooks
	}

	// Add other preserved fields
	for key, value := range settings.Other {
		var v interface{}
		if err := json.Unmarshal(value, &v); err != nil {
			continue // Skip corrupted fields
		}
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
			if strings.HasPrefix(hook.Command, "juggle") {
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
	settings, err := LoadClaudeSettings(path)
	if err != nil {
		return false
	}

	// Check if at least the core hooks are installed
	return hasJugglerHook(settings.Hooks["PostToolUse"]) &&
		hasJugglerHook(settings.Hooks["Stop"])
}
