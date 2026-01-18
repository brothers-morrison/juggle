package integration_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ohare93/juggle/internal/cli"
	"github.com/ohare93/juggle/internal/vcs"
)

// TestInitCreatesJuggleDirectory tests that init creates .juggle directory structure
func TestInitCreatesJuggleDirectory(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a fresh directory for init
	initDir := filepath.Join(env.TempDir, "init-test")

	// Use InitProject directly (disable VCS init to avoid external commands in test)
	var output bytes.Buffer
	err := cli.InitProject(cli.InitOptions{
		TargetDir:     initDir,
		JuggleDirName: ".juggle",
		Force:         false,
		InitVCS:       false, // Disable VCS to keep test isolated
		Output:        &output,
	})
	if err != nil {
		t.Fatalf("InitProject failed: %v", err)
	}

	juggleDir := filepath.Join(initDir, ".juggle")

	// Verify structure was created
	if _, err := os.Stat(juggleDir); os.IsNotExist(err) {
		t.Error("Expected .juggle directory to exist")
	}

	if _, err := os.Stat(filepath.Join(juggleDir, "sessions")); os.IsNotExist(err) {
		t.Error("Expected .juggle/sessions directory to exist")
	}

	if _, err := os.Stat(filepath.Join(juggleDir, "archive")); os.IsNotExist(err) {
		t.Error("Expected .juggle/archive directory to exist")
	}

	if _, err := os.Stat(filepath.Join(juggleDir, "balls.jsonl")); os.IsNotExist(err) {
		t.Error("Expected .juggle/balls.jsonl to exist")
	}

	// Verify success message
	if !strings.Contains(output.String(), "Initialized juggle project") {
		t.Errorf("Expected success message, got: %s", output.String())
	}
}

// TestInitAtSpecifiedPath tests init with a path argument
func TestInitAtSpecifiedPath(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create target path
	targetPath := filepath.Join(env.TempDir, "specified-path")

	var output bytes.Buffer
	err := cli.InitProject(cli.InitOptions{
		TargetDir:     targetPath,
		JuggleDirName: ".juggle",
		Force:         false,
		InitVCS:       false,
		Output:        &output,
	})
	if err != nil {
		t.Fatalf("InitProject failed: %v", err)
	}

	juggleDir := filepath.Join(targetPath, ".juggle")

	// Verify the structure exists at specified path
	if _, err := os.Stat(juggleDir); os.IsNotExist(err) {
		t.Errorf("Expected .juggle directory at %s", juggleDir)
	}

	// Verify subdirectories
	if _, err := os.Stat(filepath.Join(juggleDir, "sessions")); os.IsNotExist(err) {
		t.Error("Expected sessions directory")
	}
	if _, err := os.Stat(filepath.Join(juggleDir, "archive")); os.IsNotExist(err) {
		t.Error("Expected archive directory")
	}
}

// TestInitWithExistingVCS tests that init doesn't initialize VCS if already exists
func TestInitWithExistingVCS(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	initDir := filepath.Join(env.TempDir, "existing-vcs")
	if err := os.MkdirAll(initDir, 0755); err != nil {
		t.Fatalf("Failed to create init dir: %v", err)
	}

	// Create a fake .git directory to simulate existing VCS
	gitDir := filepath.Join(initDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	// Verify VCS is detected as initialized
	if !vcs.IsVCSInitialized(initDir) {
		t.Error("Expected VCS to be detected as initialized")
	}

	// Run init with VCS enabled - it should NOT try to init VCS since .git exists
	var output bytes.Buffer
	err := cli.InitProject(cli.InitOptions{
		TargetDir:     initDir,
		JuggleDirName: ".juggle",
		Force:         false,
		InitVCS:       true, // Enable VCS init
		Output:        &output,
	})
	if err != nil {
		t.Fatalf("InitProject failed: %v", err)
	}

	// Should not contain "Initialized jj" or "Initialized git" since VCS already existed
	if strings.Contains(output.String(), "Initialized jj") || strings.Contains(output.String(), "Initialized git repository") {
		t.Errorf("Should not have initialized VCS when it already exists. Output: %s", output.String())
	}

	// .juggle should still be created
	if _, err := os.Stat(filepath.Join(initDir, ".juggle")); os.IsNotExist(err) {
		t.Error("Expected .juggle directory to exist")
	}
}

// TestInitWithExistingJJ tests that init detects existing jj
func TestInitWithExistingJJ(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	initDir := filepath.Join(env.TempDir, "existing-jj")
	if err := os.MkdirAll(initDir, 0755); err != nil {
		t.Fatalf("Failed to create init dir: %v", err)
	}

	// Create a fake .jj directory to simulate existing jj
	jjDir := filepath.Join(initDir, ".jj")
	if err := os.MkdirAll(jjDir, 0755); err != nil {
		t.Fatalf("Failed to create .jj dir: %v", err)
	}

	// Verify VCS is detected as initialized
	if !vcs.IsVCSInitialized(initDir) {
		t.Error("Expected jj VCS to be detected as initialized")
	}
}

// TestInitFailsIfJuggleExists tests that init fails if .juggle already exists
func TestInitFailsIfJuggleExists(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	initDir := filepath.Join(env.TempDir, "already-exists")
	juggleDir := filepath.Join(initDir, ".juggle")

	// Create existing .juggle directory
	if err := os.MkdirAll(juggleDir, 0755); err != nil {
		t.Fatalf("Failed to create existing .juggle dir: %v", err)
	}

	// Init without force should fail
	var output bytes.Buffer
	err := cli.InitProject(cli.InitOptions{
		TargetDir:     initDir,
		JuggleDirName: ".juggle",
		Force:         false,
		InitVCS:       false,
		Output:        &output,
	})

	if err == nil {
		t.Fatal("Expected error when .juggle already exists without --force")
	}

	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Expected 'already exists' error, got: %v", err)
	}
}

// TestInitWithForceReinitializes tests that --force allows reinitialization
func TestInitWithForceReinitializes(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	initDir := filepath.Join(env.TempDir, "force-reinit")
	juggleDir := filepath.Join(initDir, ".juggle")

	// Create existing .juggle directory (without subdirs)
	if err := os.MkdirAll(juggleDir, 0755); err != nil {
		t.Fatalf("Failed to create existing .juggle dir: %v", err)
	}

	// With --force, init should succeed and create/update the structure
	var output bytes.Buffer
	err := cli.InitProject(cli.InitOptions{
		TargetDir:     initDir,
		JuggleDirName: ".juggle",
		Force:         true,
		InitVCS:       false,
		Output:        &output,
	})
	if err != nil {
		t.Fatalf("InitProject with force failed: %v", err)
	}

	// Verify structure exists
	if _, err := os.Stat(filepath.Join(juggleDir, "sessions")); os.IsNotExist(err) {
		t.Error("Expected sessions directory to exist after force reinit")
	}
	if _, err := os.Stat(filepath.Join(juggleDir, "archive")); os.IsNotExist(err) {
		t.Error("Expected archive directory to exist after force reinit")
	}
	if _, err := os.Stat(filepath.Join(juggleDir, "balls.jsonl")); os.IsNotExist(err) {
		t.Error("Expected balls.jsonl to exist after force reinit")
	}
}

// TestVCSAutoDetect tests VCS auto-detection functions
func TestVCSAutoDetect(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Test with no VCS
	noVCSDir := filepath.Join(env.TempDir, "no-vcs")
	if err := os.MkdirAll(noVCSDir, 0755); err != nil {
		t.Fatalf("Failed to create no-vcs dir: %v", err)
	}

	if vcs.IsVCSInitialized(noVCSDir) {
		t.Error("Expected no VCS to be detected in empty directory")
	}

	// Test with .git
	gitDir := filepath.Join(env.TempDir, "with-git")
	if err := os.MkdirAll(filepath.Join(gitDir, ".git"), 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	if !vcs.IsVCSInitialized(gitDir) {
		t.Error("Expected git VCS to be detected")
	}

	// Test with .jj
	jjDir := filepath.Join(env.TempDir, "with-jj")
	if err := os.MkdirAll(filepath.Join(jjDir, ".jj"), 0755); err != nil {
		t.Fatalf("Failed to create .jj dir: %v", err)
	}

	if !vcs.IsVCSInitialized(jjDir) {
		t.Error("Expected jj VCS to be detected")
	}
}

// TestVCSAvailabilityCheck tests VCS availability detection
func TestVCSAvailabilityCheck(t *testing.T) {
	// These tests just verify the functions don't panic
	// The actual availability depends on the test environment
	_ = vcs.IsJJAvailable()
	_ = vcs.IsGitAvailable()
}

// TestInitCreatesEmptyBallsFile tests that balls.jsonl is created empty
func TestInitCreatesEmptyBallsFile(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	initDir := filepath.Join(env.TempDir, "empty-balls")

	var output bytes.Buffer
	err := cli.InitProject(cli.InitOptions{
		TargetDir:     initDir,
		JuggleDirName: ".juggle",
		Force:         false,
		InitVCS:       false,
		Output:        &output,
	})
	if err != nil {
		t.Fatalf("InitProject failed: %v", err)
	}

	ballsPath := filepath.Join(initDir, ".juggle", "balls.jsonl")

	// Verify file is empty
	content, err := os.ReadFile(ballsPath)
	if err != nil {
		t.Fatalf("Failed to read balls.jsonl: %v", err)
	}

	if len(content) != 0 {
		t.Errorf("Expected empty balls.jsonl, got %d bytes", len(content))
	}
}

// TestInitWithCustomJuggleDirName tests using a custom juggle directory name
func TestInitWithCustomJuggleDirName(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	initDir := filepath.Join(env.TempDir, "custom-name")

	var output bytes.Buffer
	err := cli.InitProject(cli.InitOptions{
		TargetDir:     initDir,
		JuggleDirName: ".my-juggle",
		Force:         false,
		InitVCS:       false,
		Output:        &output,
	})
	if err != nil {
		t.Fatalf("InitProject failed: %v", err)
	}

	customDir := filepath.Join(initDir, ".my-juggle")

	// Verify custom directory was created
	if _, err := os.Stat(customDir); os.IsNotExist(err) {
		t.Error("Expected custom juggle directory to exist")
	}
	if _, err := os.Stat(filepath.Join(customDir, "sessions")); os.IsNotExist(err) {
		t.Error("Expected sessions directory in custom juggle dir")
	}
	if _, err := os.Stat(filepath.Join(customDir, "archive")); os.IsNotExist(err) {
		t.Error("Expected archive directory in custom juggle dir")
	}
}

// TestInitRequiresTargetDir tests that empty target directory returns error
func TestInitRequiresTargetDir(t *testing.T) {
	var output bytes.Buffer
	err := cli.InitProject(cli.InitOptions{
		TargetDir:     "",
		JuggleDirName: ".juggle",
		Force:         false,
		InitVCS:       false,
		Output:        &output,
	})

	if err == nil {
		t.Fatal("Expected error when target directory is empty")
	}

	if !strings.Contains(err.Error(), "target directory is required") {
		t.Errorf("Expected 'target directory is required' error, got: %v", err)
	}
}
