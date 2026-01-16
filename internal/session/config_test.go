package session

import (
	"os"
	"path/filepath"
	"testing"
)

// TestProjectConfig_SetDefaultAcceptanceCriteria tests setting repo-level ACs
func TestProjectConfig_SetDefaultAcceptanceCriteria(t *testing.T) {
	config := DefaultProjectConfig()

	criteria := []string{"Tests pass", "Build succeeds"}
	config.SetDefaultAcceptanceCriteria(criteria)

	if len(config.DefaultAcceptanceCriteria) != 2 {
		t.Errorf("expected 2 acceptance criteria, got %d", len(config.DefaultAcceptanceCriteria))
	}
	if config.DefaultAcceptanceCriteria[0] != "Tests pass" {
		t.Errorf("expected first criterion 'Tests pass', got '%s'", config.DefaultAcceptanceCriteria[0])
	}
}

// TestProjectConfig_HasDefaultAcceptanceCriteria tests the Has method
func TestProjectConfig_HasDefaultAcceptanceCriteria(t *testing.T) {
	config := DefaultProjectConfig()

	if config.HasDefaultAcceptanceCriteria() {
		t.Error("expected HasDefaultAcceptanceCriteria to return false for empty")
	}

	config.SetDefaultAcceptanceCriteria([]string{"Test"})

	if !config.HasDefaultAcceptanceCriteria() {
		t.Error("expected HasDefaultAcceptanceCriteria to return true after setting")
	}
}

// TestUpdateProjectAcceptanceCriteria tests updating and getting repo-level ACs
func TestUpdateProjectAcceptanceCriteria(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggle-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Ensure .juggle directory exists
	juggleDir := filepath.Join(tmpDir, ".juggle")
	if err := os.MkdirAll(juggleDir, 0755); err != nil {
		t.Fatalf("failed to create .juggle dir: %v", err)
	}

	// Update acceptance criteria
	criteria := []string{"Tests pass", "Build succeeds", "Documentation updated"}
	err = UpdateProjectAcceptanceCriteria(tmpDir, criteria)
	if err != nil {
		t.Fatalf("failed to update acceptance criteria: %v", err)
	}

	// Get and verify
	loaded, err := GetProjectAcceptanceCriteria(tmpDir)
	if err != nil {
		t.Fatalf("failed to get acceptance criteria: %v", err)
	}

	if len(loaded) != 3 {
		t.Errorf("expected 3 acceptance criteria, got %d", len(loaded))
	}
	if loaded[0] != "Tests pass" {
		t.Errorf("expected first criterion 'Tests pass', got '%s'", loaded[0])
	}
}

// TestGetProjectAcceptanceCriteria_Empty tests getting ACs when none exist
func TestGetProjectAcceptanceCriteria_Empty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggle-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Ensure .juggle directory exists
	juggleDir := filepath.Join(tmpDir, ".juggle")
	if err := os.MkdirAll(juggleDir, 0755); err != nil {
		t.Fatalf("failed to create .juggle dir: %v", err)
	}

	// Get acceptance criteria (should be empty, not error)
	criteria, err := GetProjectAcceptanceCriteria(tmpDir)
	if err != nil {
		t.Fatalf("failed to get acceptance criteria: %v", err)
	}

	if len(criteria) != 0 {
		t.Errorf("expected 0 acceptance criteria, got %d", len(criteria))
	}
}

// TestProjectAcceptanceCriteria_Persistence tests ACs survive save/load
func TestProjectAcceptanceCriteria_Persistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggle-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Ensure .juggle directory exists
	juggleDir := filepath.Join(tmpDir, ".juggle")
	if err := os.MkdirAll(juggleDir, 0755); err != nil {
		t.Fatalf("failed to create .juggle dir: %v", err)
	}

	// Set acceptance criteria
	criteria := []string{"Run tests", "Check build"}
	if err := UpdateProjectAcceptanceCriteria(tmpDir, criteria); err != nil {
		t.Fatalf("failed to update ACs: %v", err)
	}

	// Load directly from config to verify persistence
	config, err := LoadProjectConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(config.DefaultAcceptanceCriteria) != 2 {
		t.Errorf("expected 2 ACs after reload, got %d", len(config.DefaultAcceptanceCriteria))
	}
}

// TestProjectAcceptanceCriteria_Clear tests clearing all ACs
func TestProjectAcceptanceCriteria_Clear(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggle-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Ensure .juggle directory exists
	juggleDir := filepath.Join(tmpDir, ".juggle")
	if err := os.MkdirAll(juggleDir, 0755); err != nil {
		t.Fatalf("failed to create .juggle dir: %v", err)
	}

	// Set some criteria
	criteria := []string{"Test 1", "Test 2"}
	if err := UpdateProjectAcceptanceCriteria(tmpDir, criteria); err != nil {
		t.Fatalf("failed to update ACs: %v", err)
	}

	// Clear by setting empty
	if err := UpdateProjectAcceptanceCriteria(tmpDir, []string{}); err != nil {
		t.Fatalf("failed to clear ACs: %v", err)
	}

	// Verify cleared
	loaded, err := GetProjectAcceptanceCriteria(tmpDir)
	if err != nil {
		t.Fatalf("failed to get ACs: %v", err)
	}

	if len(loaded) != 0 {
		t.Errorf("expected 0 ACs after clear, got %d", len(loaded))
	}
}

// TestProjectConfig_RunAliases tests the run alias functionality
func TestProjectConfig_RunAliases(t *testing.T) {
	config := DefaultProjectConfig()

	// Test initial state
	if config.HasRunAliases() {
		t.Error("expected HasRunAliases to return false for empty config")
	}

	if alias := config.GetRunAlias("build"); alias != "" {
		t.Errorf("expected empty string for non-existent alias, got %q", alias)
	}

	// Test setting an alias
	config.SetRunAlias("build", "devbox run build")
	if !config.HasRunAliases() {
		t.Error("expected HasRunAliases to return true after setting alias")
	}

	if alias := config.GetRunAlias("build"); alias != "devbox run build" {
		t.Errorf("expected 'devbox run build', got %q", alias)
	}

	// Test updating an alias
	config.SetRunAlias("build", "go build ./...")
	if alias := config.GetRunAlias("build"); alias != "go build ./..." {
		t.Errorf("expected updated alias 'go build ./...', got %q", alias)
	}

	// Test deleting an alias
	if !config.DeleteRunAlias("build") {
		t.Error("expected DeleteRunAlias to return true for existing alias")
	}

	if config.DeleteRunAlias("build") {
		t.Error("expected DeleteRunAlias to return false for non-existent alias")
	}

	if config.HasRunAliases() {
		t.Error("expected HasRunAliases to return false after deleting last alias")
	}
}

// TestProjectConfig_RunAliases_GetAll tests GetRunAliases
func TestProjectConfig_RunAliases_GetAll(t *testing.T) {
	config := DefaultProjectConfig()

	config.SetRunAlias("build", "go build ./...")
	config.SetRunAlias("test", "go test ./...")
	config.SetRunAlias("lint", "golangci-lint run")

	aliases := config.GetRunAliases()
	if len(aliases) != 3 {
		t.Errorf("expected 3 aliases, got %d", len(aliases))
	}

	if aliases["build"] != "go build ./..." {
		t.Errorf("expected build alias, got %q", aliases["build"])
	}
	if aliases["test"] != "go test ./..." {
		t.Errorf("expected test alias, got %q", aliases["test"])
	}
	if aliases["lint"] != "golangci-lint run" {
		t.Errorf("expected lint alias, got %q", aliases["lint"])
	}
}

// TestProjectConfig_RunAliases_Persistence tests aliases survive save/load
func TestProjectConfig_RunAliases_Persistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggle-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Ensure .juggle directory exists
	juggleDir := filepath.Join(tmpDir, ".juggle")
	if err := os.MkdirAll(juggleDir, 0755); err != nil {
		t.Fatalf("failed to create .juggle dir: %v", err)
	}

	// Create config with aliases
	config := DefaultProjectConfig()
	config.SetRunAlias("build", "devbox run build")
	config.SetRunAlias("test", "go test -v ./...")

	// Save
	if err := SaveProjectConfig(tmpDir, config); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Load and verify
	loaded, err := LoadProjectConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if !loaded.HasRunAliases() {
		t.Error("expected loaded config to have run aliases")
	}

	if alias := loaded.GetRunAlias("build"); alias != "devbox run build" {
		t.Errorf("expected 'devbox run build', got %q", alias)
	}

	if alias := loaded.GetRunAlias("test"); alias != "go test -v ./..." {
		t.Errorf("expected 'go test -v ./...', got %q", alias)
	}
}
