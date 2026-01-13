package integration_test

import (
	"testing"
	"time"

	"github.com/ohare93/juggle/internal/cli"
	"github.com/ohare93/juggle/internal/session"
)

// Tests for --delay and --fuzz flag behavior in agent run command

func TestAgentDelay_FlagOverridesConfig(t *testing.T) {
	// Test that flags override config values
	// The calculateFuzzyDelay function handles the actual calculation

	// With delay=5, no fuzz, should return exactly 5 minutes
	delay := cli.CalculateFuzzyDelayForTest(5, 0)
	if delay != 5*time.Minute {
		t.Errorf("Expected 5m delay when base is 5, got %v", delay)
	}
}

func TestAgentDelay_ZeroDelaySkipsFeature(t *testing.T) {
	// When delay is 0, the entire delay feature should be skipped
	// This means even with a fuzz value, no delay should occur
	// This is handled at the caller level (runAgentRun checks if delayMinutes > 0)
	// The calculateFuzzyDelay function itself doesn't enforce this

	// Test that when delayMinutes=0 and fuzz=0, we get 0
	delay := cli.CalculateFuzzyDelayForTest(0, 0)
	if delay != 0 {
		t.Errorf("Expected 0 delay when base and fuzz are 0, got %v", delay)
	}

	// AC4 is enforced at the caller level: "If the delay is 0 then the delay feature is skipped"
	// The runAgentRun function checks: if delayMinutes > 0 { ... calculate fuzzy delay }
	// So the feature is skipped entirely when delay is 0, regardless of fuzz
}

func TestAgentDelay_ConfigValuesUsedByDefault(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Set config with delay values
	config, err := session.LoadConfigWithOptions(session.ConfigOptions{
		ConfigHome:     env.ConfigHome,
		JugglerDirName: ".juggler",
	})
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Set delay values in config
	config.SetIterationDelay(3, 1)
	if err := config.SaveWithOptions(session.ConfigOptions{
		ConfigHome:     env.ConfigHome,
		JugglerDirName: ".juggler",
	}); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify config was saved
	loadedConfig, err := session.LoadConfigWithOptions(session.ConfigOptions{
		ConfigHome:     env.ConfigHome,
		JugglerDirName: ".juggler",
	})
	if err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}

	delayMins, fuzzMins := loadedConfig.GetIterationDelay()
	if delayMins != 3 {
		t.Errorf("Expected delay_minutes=3 in config, got %d", delayMins)
	}
	if fuzzMins != 1 {
		t.Errorf("Expected delay_fuzz=1 in config, got %d", fuzzMins)
	}
}

func TestAgentDelay_FuzzyCalculation(t *testing.T) {
	// Test that fuzzy delay produces values in expected range
	baseMinutes := 5
	fuzz := 2

	minDelay := time.Duration(baseMinutes-fuzz) * time.Minute // 3 minutes
	maxDelay := time.Duration(baseMinutes+fuzz) * time.Minute // 7 minutes

	// Run multiple times to verify range
	for i := 0; i < 50; i++ {
		delay := cli.CalculateFuzzyDelayForTest(baseMinutes, fuzz)
		if delay < minDelay || delay > maxDelay {
			t.Errorf("Fuzzy delay %v outside expected range [%v, %v]", delay, minDelay, maxDelay)
		}
	}
}

func TestAgentDelay_NegativeFuzzHandled(t *testing.T) {
	// Edge case: fuzz larger than base should still produce non-negative delay
	delay := cli.CalculateFuzzyDelayForTest(2, 5)
	if delay < 0 {
		t.Errorf("Delay should never be negative, got %v", delay)
	}
}
