package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ohare93/juggle/internal/cli"
)

// TestLoadSavePreservesFullSandboxConfig verifies that loading and saving settings
// preserves all sandbox configuration fields, not just the known struct fields.
// This is a regression test for a bug where fields like excludedCommands, network,
// and filesystem were lost during save because they weren't in the SandboxConfig struct.
func TestLoadSavePreservesFullSandboxConfig(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create .claude directory
	claudeDir := filepath.Join(env.TempDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create .claude dir: %v", err)
	}

	// Create settings.json with full sandbox config (as a real user would have)
	fullSettings := `{
  "sandbox": {
    "enabled": true,
    "autoAllowBashIfSandboxed": true,
    "allowUnsandboxedCommands": true,
    "excludedCommands": [],
    "network": {
      "allowLocalBinding": true
    },
    "filesystem": {
      "write": {
        "additionalAllow": ["~/.cache/nix", "~/go"]
      }
    }
  },
  "permissions": {
    "allow": [
      "Bash(go:*)",
      "Bash(gofmt:*)"
    ],
    "deny": [
      "Read(./.env)"
    ],
    "ask": [
      "Bash(git push:*)"
    ]
  }
}`
	settingsPath := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(fullSettings), 0644); err != nil {
		t.Fatalf("Failed to write settings: %v", err)
	}

	// Load settings using the cli function
	settings, err := cli.LoadClaudeSettings(settingsPath)
	if err != nil {
		t.Fatalf("Failed to load settings: %v", err)
	}

	// Save settings back
	if err := cli.SaveClaudeSettings(settingsPath, settings); err != nil {
		t.Fatalf("Failed to save settings: %v", err)
	}

	// Read the saved file and verify all fields are present
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("Failed to read saved settings: %v", err)
	}

	// Parse into generic map to check structure
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to parse saved settings: %v", err)
	}

	// Verify sandbox section exists and has all fields
	sandbox, ok := parsed["sandbox"].(map[string]interface{})
	if !ok {
		t.Fatal("sandbox section missing or wrong type")
	}

	// Check all expected sandbox fields
	if _, ok := sandbox["enabled"]; !ok {
		t.Error("sandbox.enabled field missing")
	}
	if _, ok := sandbox["autoAllowBashIfSandboxed"]; !ok {
		t.Error("sandbox.autoAllowBashIfSandboxed field missing")
	}
	if _, ok := sandbox["allowUnsandboxedCommands"]; !ok {
		t.Error("sandbox.allowUnsandboxedCommands field missing")
	}
	if _, ok := sandbox["excludedCommands"]; !ok {
		t.Error("sandbox.excludedCommands field missing - field was lost during save!")
	}
	if _, ok := sandbox["network"]; !ok {
		t.Error("sandbox.network field missing - field was lost during save!")
	}
	if _, ok := sandbox["filesystem"]; !ok {
		t.Error("sandbox.filesystem field missing - field was lost during save!")
	}

	// Verify network nested structure
	network, ok := sandbox["network"].(map[string]interface{})
	if !ok {
		t.Error("sandbox.network should be an object")
	} else {
		if _, ok := network["allowLocalBinding"]; !ok {
			t.Error("sandbox.network.allowLocalBinding field missing")
		}
	}

	// Verify filesystem nested structure
	filesystem, ok := sandbox["filesystem"].(map[string]interface{})
	if !ok {
		t.Error("sandbox.filesystem should be an object")
	} else {
		write, ok := filesystem["write"].(map[string]interface{})
		if !ok {
			t.Error("sandbox.filesystem.write should be an object")
		} else {
			if _, ok := write["additionalAllow"]; !ok {
				t.Error("sandbox.filesystem.write.additionalAllow field missing")
			}
		}
	}

	// Verify permissions are also preserved
	permissions, ok := parsed["permissions"].(map[string]interface{})
	if !ok {
		t.Fatal("permissions section missing")
	}
	allow, ok := permissions["allow"].([]interface{})
	if !ok || len(allow) != 2 {
		t.Error("permissions.allow should have 2 items")
	}
}

// TestLoadSavePreservesUnknownTopLevelFields verifies that unknown top-level fields
// in settings.json are preserved during load/save cycle.
func TestLoadSavePreservesUnknownTopLevelFields(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create .claude directory
	claudeDir := filepath.Join(env.TempDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create .claude dir: %v", err)
	}

	// Settings with unknown top-level field
	settingsWithUnknown := `{
  "sandbox": {
    "enabled": true
  },
  "someUnknownField": {
    "nested": "value"
  },
  "anotherUnknown": ["item1", "item2"]
}`
	settingsPath := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(settingsWithUnknown), 0644); err != nil {
		t.Fatalf("Failed to write settings: %v", err)
	}

	// Load and save
	settings, err := cli.LoadClaudeSettings(settingsPath)
	if err != nil {
		t.Fatalf("Failed to load settings: %v", err)
	}
	if err := cli.SaveClaudeSettings(settingsPath, settings); err != nil {
		t.Fatalf("Failed to save settings: %v", err)
	}

	// Verify unknown fields preserved
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("Failed to read saved settings: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to parse saved settings: %v", err)
	}

	if _, ok := parsed["someUnknownField"]; !ok {
		t.Error("someUnknownField was lost during save")
	}
	if _, ok := parsed["anotherUnknown"]; !ok {
		t.Error("anotherUnknown was lost during save")
	}
}
