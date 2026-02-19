package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var cronCmd = &cobra.Command{
	Use:   "cron",
	Short: "Manage juggle cron jobs",
	Long: `Install or remove cron jobs for automatic juggle daemon management.

Two cron jobs are managed:
  1. auto-launcher.sh  - Runs hourly, launches daemons for sessions with pending work
  2. status-checker.sh - Runs every 5 minutes, syncs state and recovers missed signals

The scripts are stored in ~/.juggle/ and cron entries are tagged with
a marker comment for safe install/removal.`,
}

var cronInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install juggle cron jobs",
	Long: `Install the auto-launcher and status-checker cron jobs.

This adds two entries to your crontab:
  0 * * * *   ~/.juggle/auto-launcher.sh   (hourly)
  */5 * * * * ~/.juggle/status-checker.sh   (every 5 minutes)

Both scripts must exist and be executable in ~/.juggle/.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCronScript("install")
	},
}

var cronRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove juggle cron jobs",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCronScript("remove")
	},
}

var cronStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current juggle cron entries",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCronScript("status")
	},
}

// runCronScript delegates to ~/.juggle/install-cron.sh
func runCronScript(action string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	script := filepath.Join(home, ".juggle", "install-cron.sh")
	if _, err := os.Stat(script); os.IsNotExist(err) {
		return fmt.Errorf("cron management script not found at %s\n\nThe script should be created during juggle setup", script)
	}

	// Check if script is executable
	info, err := os.Stat(script)
	if err != nil {
		return fmt.Errorf("failed to stat %s: %w", script, err)
	}
	if info.Mode()&0111 == 0 {
		return fmt.Errorf("%s is not executable. Run: chmod +x %s", script, script)
	}

	// Also verify the subscripts exist for install
	if action == "install" {
		missing := checkCronScripts(home)
		if len(missing) > 0 {
			return fmt.Errorf("required scripts not found:\n  %s\n\nCreate them first or run setup", strings.Join(missing, "\n  "))
		}
	}

	cmd := exec.Command(script, action)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("cron %s failed (exit %d)", action, exitErr.ExitCode())
		}
		return fmt.Errorf("cron %s failed: %w", action, err)
	}

	return nil
}

// checkCronScripts verifies that required scripts exist
func checkCronScripts(home string) []string {
	required := []string{
		filepath.Join(home, ".juggle", "auto-launcher.sh"),
		filepath.Join(home, ".juggle", "status-checker.sh"),
	}

	var missing []string
	for _, path := range required {
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			missing = append(missing, path+" (not found)")
		} else if err == nil && info.Mode()&0111 == 0 {
			missing = append(missing, path+" (not executable)")
		}
	}

	return missing
}

func init() {
	cronCmd.AddCommand(cronInstallCmd)
	cronCmd.AddCommand(cronRemoveCmd)
	cronCmd.AddCommand(cronStatusCmd)
}
