package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ohare93/juggle/internal/session"
)

func TestConfigPathsList(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	repoRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(repoRoot, "juggle")

	// Add some paths to config
	config, err := session.LoadConfigWithOptions(session.ConfigOptions{
		ConfigHome:    env.ConfigHome,
		JuggleDirName: ".juggle",
	})
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	config.AddSearchPath(env.ProjectDir)
	config.AddSearchPath("/nonexistent/path/for/test")

	if err := config.SaveWithOptions(session.ConfigOptions{
		ConfigHome:    env.ConfigHome,
		JuggleDirName: ".juggle",
	}); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Test list command
	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "config", "paths", "list")
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run config paths list: %v\nOutput: %s", err, output)
	}

	// Should show existing path with checkmark
	if !strings.Contains(string(output), env.ProjectDir) {
		t.Errorf("Expected output to contain %s, got: %s", env.ProjectDir, output)
	}

	// Should show nonexistent path with X
	if !strings.Contains(string(output), "nonexistent") {
		t.Errorf("Expected output to contain nonexistent path, got: %s", output)
	}
}

func TestConfigPathsPrune(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	repoRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(repoRoot, "juggle")

	// Add some paths to config
	config, err := session.LoadConfigWithOptions(session.ConfigOptions{
		ConfigHome:    env.ConfigHome,
		JuggleDirName: ".juggle",
	})
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	config.AddSearchPath(env.ProjectDir)
	config.AddSearchPath("/nonexistent/path/1")
	config.AddSearchPath("/nonexistent/path/2")

	if err := config.SaveWithOptions(session.ConfigOptions{
		ConfigHome:    env.ConfigHome,
		JuggleDirName: ".juggle",
	}); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Run prune command
	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "config", "paths", "prune", "-y")
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run config paths prune: %v\nOutput: %s", err, output)
	}

	// Should indicate 2 paths removed
	if !strings.Contains(string(output), "2 path(s)") {
		t.Errorf("Expected output to indicate 2 paths removed, got: %s", output)
	}

	// Verify config was updated
	configAfter, err := session.LoadConfigWithOptions(session.ConfigOptions{
		ConfigHome:    env.ConfigHome,
		JuggleDirName: ".juggle",
	})
	if err != nil {
		t.Fatalf("Failed to load config after prune: %v", err)
	}

	if len(configAfter.SearchPaths) != 1 {
		t.Errorf("Expected 1 path after prune, got %d: %v", len(configAfter.SearchPaths), configAfter.SearchPaths)
	}

	if configAfter.SearchPaths[0] != env.ProjectDir {
		t.Errorf("Expected remaining path to be %s, got %s", env.ProjectDir, configAfter.SearchPaths[0])
	}
}

func TestConfigPathsPruneNothingToRemove(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	repoRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(repoRoot, "juggle")

	// Add only existing path
	config, err := session.LoadConfigWithOptions(session.ConfigOptions{
		ConfigHome:    env.ConfigHome,
		JuggleDirName: ".juggle",
	})
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	config.AddSearchPath(env.ProjectDir)

	if err := config.SaveWithOptions(session.ConfigOptions{
		ConfigHome:    env.ConfigHome,
		JuggleDirName: ".juggle",
	}); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Run prune command
	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "config", "paths", "prune", "-y")
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run config paths prune: %v\nOutput: %s", err, output)
	}

	// Should indicate nothing to remove
	if !strings.Contains(string(output), "No non-existent paths to remove") {
		t.Errorf("Expected message about nothing to remove, got: %s", output)
	}
}

func TestJugglerConfigHomeEnvVar(t *testing.T) {
	// Test that JUGGLER_CONFIG_HOME environment variable is respected
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Verify the env var is set by SetupTestEnv
	configHome := os.Getenv(session.EnvConfigHome)
	if configHome != env.ConfigHome {
		t.Errorf("Expected JUGGLER_CONFIG_HOME to be %s, got %s", env.ConfigHome, configHome)
	}

	// Verify DefaultConfigOptions uses the env var
	opts := session.DefaultConfigOptions()
	if opts.ConfigHome != env.ConfigHome {
		t.Errorf("Expected DefaultConfigOptions.ConfigHome to be %s, got %s", env.ConfigHome, opts.ConfigHome)
	}
}
