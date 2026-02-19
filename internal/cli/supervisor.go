package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ohare93/juggle/internal/agent/supervisor"
	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var supervisorCmd = &cobra.Command{
	Use:   "supervisor",
	Short: "Manage the juggle supervisor daemon",
	Long: `The supervisor daemon monitors running juggle agent sessions, detects stalled
daemons, recovers missed signals (especially from OpenCode), and optionally
auto-launches daemons for sessions with pending work.

Examples:
  juggle supervisor start           # Start the supervisor in the foreground
  juggle supervisor start --daemon  # Start in the background (not yet implemented)
  juggle supervisor stop            # Stop the running supervisor
  juggle supervisor status          # Show supervisor and session status
  juggle supervisor once            # Run a single poll cycle and exit`,
}

var supervisorDaemonFlag bool

var supervisorStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the supervisor daemon",
	Long: `Start the supervisor polling loop. By default runs in the foreground.

The supervisor will:
  1. Periodically scan all projects for sessions with running/pending daemons
  2. Detect stalled daemons (no state update within stall_timeout)
  3. Recover missed signals from OpenCode session exports
  4. Optionally auto-restart stalled daemons
  5. Optionally auto-launch daemons for sessions with pending balls

Configuration is read from ~/.juggle/config.json under the "supervisor" key.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := loadSupervisorConfig()
		if err != nil {
			return err
		}

		// Check if already running
		if running, pid := supervisor.IsSupervisorRunning(); running {
			return fmt.Errorf("supervisor is already running (PID %d)", pid)
		}

		sv := supervisor.New(config, GetConfigOptions())

		fmt.Printf("Starting supervisor (poll every %d min, stall timeout %d min, max concurrent %d)\n",
			config.GetPollInterval(), config.GetStallTimeout(), config.GetMaxConcurrent())

		if config.AutoRestart {
			fmt.Println("  Auto-restart: enabled")
		}
		if config.AutoLaunch {
			fmt.Println("  Auto-launch:  enabled")
		}

		if err := sv.Start(); err != nil {
			return err
		}

		// Wait for interrupt
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		fmt.Println("Supervisor running. Press Ctrl+C to stop.")

		<-sigCh
		fmt.Println("\nStopping supervisor...")
		sv.Stop()
		fmt.Println("Supervisor stopped.")

		return nil
	},
}

var supervisorStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running supervisor",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := supervisor.StopSupervisor(); err != nil {
			return err
		}
		fmt.Println("Supervisor stopped.")
		return nil
	},
}

var supervisorStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show supervisor and session status",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check supervisor itself
		running, pid := supervisor.IsSupervisorRunning()
		if running {
			fmt.Printf("Supervisor: running (PID %d)\n", pid)
		} else {
			fmt.Println("Supervisor: not running")
		}
		fmt.Println()

		// Run a one-shot poll to show session status
		config, err := loadSupervisorConfig()
		if err != nil {
			return err
		}

		sv := supervisor.New(config, GetConfigOptions())
		statuses, err := sv.RunOnce()
		if err != nil {
			return fmt.Errorf("failed to poll sessions: %w", err)
		}

		if len(statuses) == 0 {
			fmt.Println("No sessions found.")
			return nil
		}

		if GlobalOpts.JSONOutput {
			data, _ := json.MarshalIndent(statuses, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("%-40s %-15s %-8s %s\n", "SESSION", "STATUS", "BALLS", "DETAILS")
		fmt.Println("--------------------------------------------------------------------------------------------")

		for _, st := range statuses {
			sessionLabel := st.SessionID
			if len(sessionLabel) > 38 {
				sessionLabel = sessionLabel[:38] + ".."
			}

			balls := fmt.Sprintf("P:%d I:%d C:%d B:%d",
				st.Pending, st.InProgress, st.Complete, st.Blocked)

			runStatus := "stopped"
			if st.Running && !st.Stalled {
				runStatus = fmt.Sprintf("running/%d", st.DaemonPID)
			} else if st.Stalled {
				runStatus = "STALLED"
			}

			fmt.Printf("%-40s %-15s %-8s %s\n",
				sessionLabel, runStatus, balls, st.Status)
		}

		return nil
	},
}

var supervisorOnceCmd = &cobra.Command{
	Use:   "once",
	Short: "Run a single poll cycle and exit",
	Long:  `Run one poll cycle: check all sessions, recover signals, restart stalled daemons, then exit.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := loadSupervisorConfig()
		if err != nil {
			return err
		}

		sv := supervisor.New(config, GetConfigOptions())
		statuses, err := sv.RunOnce()
		if err != nil {
			return fmt.Errorf("failed to poll sessions: %w", err)
		}

		if GlobalOpts.JSONOutput {
			data, _ := json.MarshalIndent(statuses, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Polled %d sessions:\n", len(statuses))
		for _, st := range statuses {
			icon := " "
			if st.Stalled {
				icon = "!"
			} else if st.Running {
				icon = ">"
			}
			fmt.Printf("  %s %-30s  %s\n", icon, st.SessionID, st.Status)
		}

		return nil
	},
}

var supervisorEnsureCmd = &cobra.Command{
	Use:   "ensure-running",
	Short: "Start supervisor if not already running",
	Long:  `Check if supervisor is running; if not, start it. Useful for cron or init scripts.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if running, pid := supervisor.IsSupervisorRunning(); running {
			fmt.Printf("Supervisor already running (PID %d)\n", pid)
			return nil
		}

		// Not running - start it
		fmt.Println("Supervisor not running, starting...")
		config, err := loadSupervisorConfig()
		if err != nil {
			return err
		}

		sv := supervisor.New(config, GetConfigOptions())
		if err := sv.Start(); err != nil {
			return err
		}

		// Wait for interrupt
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		fmt.Println("Supervisor running. Press Ctrl+C to stop.")
		<-sigCh
		fmt.Println("\nStopping supervisor...")
		sv.Stop()

		return nil
	},
}

func loadSupervisorConfig() (*session.SupervisorConfig, error) {
	config, err := LoadConfigForCommand()
	if err != nil {
		return session.DefaultSupervisorConfig(), nil
	}

	if config.Supervisor != nil {
		return config.Supervisor, nil
	}

	return session.DefaultSupervisorConfig(), nil
}

func init() {
	supervisorStartCmd.Flags().BoolVar(&supervisorDaemonFlag, "daemon", false, "Run in background (not yet implemented)")

	supervisorCmd.AddCommand(supervisorStartCmd)
	supervisorCmd.AddCommand(supervisorStopCmd)
	supervisorCmd.AddCommand(supervisorStatusCmd)
	supervisorCmd.AddCommand(supervisorOnceCmd)
	supervisorCmd.AddCommand(supervisorEnsureCmd)
}
