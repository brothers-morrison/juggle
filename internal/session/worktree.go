package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	linkFile       = "link"
	worktreesField = "worktrees"
)

// WorktreeConfig represents the worktree-related configuration stored in .juggle/config.json
type WorktreeConfig struct {
	Worktrees []string `json:"worktrees,omitempty"`
}

// ResolveStorageDir resolves the directory where .juggle/ storage should be located.
// If the given directory contains a .juggle/link file, returns the linked main repo path.
// Otherwise, returns the directory unchanged.
func ResolveStorageDir(dir string, juggleDirName string) (string, error) {
	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		dir = cwd
	}

	if juggleDirName == "" {
		juggleDirName = projectStorePath
	}

	// Check for .juggle/link file
	linkPath := filepath.Join(dir, juggleDirName, linkFile)
	data, err := os.ReadFile(linkPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No link file, use directory as-is
			return dir, nil
		}
		return "", fmt.Errorf("failed to read link file: %w", err)
	}

	// Link file contains the main repo path
	mainRepoPath := strings.TrimSpace(string(data))
	if mainRepoPath == "" {
		return dir, nil
	}

	// Validate the linked path exists and has a .juggle directory
	linkedJuggleDir := filepath.Join(mainRepoPath, juggleDirName)
	if _, err := os.Stat(linkedJuggleDir); os.IsNotExist(err) {
		return "", fmt.Errorf("linked repo %s does not have a %s directory", mainRepoPath, juggleDirName)
	}

	return mainRepoPath, nil
}

// RegisterWorktree adds a worktree to the main repo's config and creates the link file in the worktree.
// Must be run from the main repo directory (or mainDir must point to it).
func RegisterWorktree(mainDir, worktreeDir string, juggleDirName string) error {
	if juggleDirName == "" {
		juggleDirName = projectStorePath
	}

	// Validate main directory has .juggle
	mainJuggleDir := filepath.Join(mainDir, juggleDirName)
	if _, err := os.Stat(mainJuggleDir); os.IsNotExist(err) {
		return fmt.Errorf("main repo %s does not have a %s directory", mainDir, juggleDirName)
	}

	// Validate worktree directory exists
	if _, err := os.Stat(worktreeDir); os.IsNotExist(err) {
		return fmt.Errorf("worktree directory %s does not exist", worktreeDir)
	}

	// Get absolute paths for consistency
	absMainDir, err := filepath.Abs(mainDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for main dir: %w", err)
	}
	absWorktreeDir, err := filepath.Abs(worktreeDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for worktree dir: %w", err)
	}

	// Prevent registering main repo as its own worktree
	if absMainDir == absWorktreeDir {
		return fmt.Errorf("cannot register main repo as a worktree of itself")
	}

	// Load existing worktree config
	config, err := loadWorktreeConfig(mainDir, juggleDirName)
	if err != nil {
		config = &WorktreeConfig{Worktrees: []string{}}
	}

	// Check if already registered
	for _, wt := range config.Worktrees {
		if wt == absWorktreeDir {
			return fmt.Errorf("worktree %s is already registered", absWorktreeDir)
		}
	}

	// Add to config
	config.Worktrees = append(config.Worktrees, absWorktreeDir)

	// Save updated config
	if err := saveWorktreeConfig(mainDir, juggleDirName, config); err != nil {
		return fmt.Errorf("failed to save worktree config: %w", err)
	}

	// Create .juggle directory in worktree if it doesn't exist
	worktreeJuggleDir := filepath.Join(absWorktreeDir, juggleDirName)
	if err := os.MkdirAll(worktreeJuggleDir, 0755); err != nil {
		return fmt.Errorf("failed to create %s directory in worktree: %w", juggleDirName, err)
	}

	// Create link file in worktree
	linkPath := filepath.Join(worktreeJuggleDir, linkFile)
	if err := os.WriteFile(linkPath, []byte(absMainDir+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to create link file: %w", err)
	}

	return nil
}

// ForgetWorktree removes a worktree from the main repo's config and deletes the link file.
func ForgetWorktree(mainDir, worktreeDir string, juggleDirName string) error {
	if juggleDirName == "" {
		juggleDirName = projectStorePath
	}

	// Get absolute paths
	absMainDir, err := filepath.Abs(mainDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for main dir: %w", err)
	}
	absWorktreeDir, err := filepath.Abs(worktreeDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for worktree dir: %w", err)
	}

	// Load existing config
	config, err := loadWorktreeConfig(absMainDir, juggleDirName)
	if err != nil {
		return fmt.Errorf("failed to load worktree config: %w", err)
	}

	// Find and remove the worktree
	found := false
	newWorktrees := make([]string, 0, len(config.Worktrees))
	for _, wt := range config.Worktrees {
		if wt == absWorktreeDir {
			found = true
			continue
		}
		newWorktrees = append(newWorktrees, wt)
	}

	if !found {
		return fmt.Errorf("worktree %s is not registered", absWorktreeDir)
	}

	config.Worktrees = newWorktrees

	// Save updated config
	if err := saveWorktreeConfig(absMainDir, juggleDirName, config); err != nil {
		return fmt.Errorf("failed to save worktree config: %w", err)
	}

	// Remove link file from worktree (if it exists)
	linkPath := filepath.Join(absWorktreeDir, juggleDirName, linkFile)
	if err := os.Remove(linkPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove link file: %w", err)
	}

	return nil
}

// ListWorktrees returns all registered worktrees for the given main repo.
func ListWorktrees(mainDir string, juggleDirName string) ([]string, error) {
	if juggleDirName == "" {
		juggleDirName = projectStorePath
	}

	config, err := loadWorktreeConfig(mainDir, juggleDirName)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	return config.Worktrees, nil
}

// IsWorktree checks if the given directory is a registered worktree (has a link file).
func IsWorktree(dir string, juggleDirName string) (bool, error) {
	if juggleDirName == "" {
		juggleDirName = projectStorePath
	}

	linkPath := filepath.Join(dir, juggleDirName, linkFile)
	_, err := os.Stat(linkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetLinkedMainRepo returns the main repo path if this directory is a worktree.
// Returns empty string if not a worktree.
func GetLinkedMainRepo(dir string, juggleDirName string) (string, error) {
	if juggleDirName == "" {
		juggleDirName = projectStorePath
	}

	linkPath := filepath.Join(dir, juggleDirName, linkFile)
	data, err := os.ReadFile(linkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

// loadWorktreeConfig loads the worktree configuration from the main repo's config.json
func loadWorktreeConfig(mainDir, juggleDirName string) (*WorktreeConfig, error) {
	configPath := filepath.Join(mainDir, juggleDirName, "config.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &WorktreeConfig{Worktrees: []string{}}, nil
		}
		return nil, err
	}

	// Parse as generic map first to preserve other fields
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse config.json: %w", err)
	}

	config := &WorktreeConfig{Worktrees: []string{}}
	if worktreesRaw, ok := raw[worktreesField]; ok {
		if err := json.Unmarshal(worktreesRaw, &config.Worktrees); err != nil {
			return nil, fmt.Errorf("failed to parse worktrees field: %w", err)
		}
	}

	return config, nil
}

// saveWorktreeConfig saves the worktree configuration to the main repo's config.json
// Preserves other fields in the config file.
func saveWorktreeConfig(mainDir, juggleDirName string, config *WorktreeConfig) error {
	configPath := filepath.Join(mainDir, juggleDirName, "config.json")

	// Load existing config to preserve other fields
	var raw map[string]interface{}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			raw = make(map[string]interface{})
		} else {
			return err
		}
	} else {
		if err := json.Unmarshal(data, &raw); err != nil {
			return fmt.Errorf("failed to parse existing config.json: %w", err)
		}
	}

	// Update worktrees field
	if len(config.Worktrees) == 0 {
		delete(raw, worktreesField)
	} else {
		raw[worktreesField] = config.Worktrees
	}

	// Write back
	newData, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, newData, 0644); err != nil {
		return fmt.Errorf("failed to write config.json: %w", err)
	}

	return nil
}
