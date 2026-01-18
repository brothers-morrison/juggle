package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var loopUpdateJSONFlag bool

var loopCmd = &cobra.Command{
	Use:   "loop",
	Short: "Manage agent loop status and progress",
	Long:  `Commands for managing agent loop status updates (agent-update.txt files).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var loopUpdateCmd = &cobra.Command{
	Use:   "update [session-id] <ball-id> <state> <message>",
	Short: "Update agent loop progress for a ball",
	Long: `Update the agent loop's progress tracking for a specific ball.

The session-id can be provided as the first argument, or via the
JUGGLE_SESSION_ID environment variable.

State should be one of: starting, working, blocked, testing, complete

Creates or overwrites agent-update.txt with the current status.

Examples:
  juggle loop update my-session juggle-123 starting "Beginning work on authentication"
  JUGGLE_SESSION_ID=my-session juggle loop update juggle-123 working "Implementing OAuth flow"
  juggle loop update my-session juggle-123 blocked "Waiting for API credentials"
  juggle loop update my-session juggle-123 testing "Running test suite"
  juggle loop update my-session juggle-123 complete "All ACs satisfied"`,
	Args: cobra.RangeArgs(3, 4),
	RunE: runLoopUpdate,
}

func init() {
	loopUpdateCmd.Flags().BoolVar(&loopUpdateJSONFlag, "json", false, "Output as JSON")
	loopCmd.AddCommand(loopUpdateCmd)
	rootCmd.AddCommand(loopCmd)
}

func runLoopUpdate(cmd *cobra.Command, args []string) error {
	var sessionID, ballID, state, message string

	// Parse args: either (session-id, ball-id, state, message) or (ball-id, state, message) with env var
	if len(args) == 4 {
		sessionID = args[0]
		ballID = args[1]
		state = args[2]
		message = args[3]
	} else {
		// Three args - use env var for session ID
		sessionID = os.Getenv("JUGGLE_SESSION_ID")
		if sessionID == "" {
			err := fmt.Errorf("session ID required: provide as first argument or set JUGGLE_SESSION_ID")
			if loopUpdateJSONFlag {
				return printLoopUpdateJSONError(err)
			}
			return err
		}
		ballID = args[0]
		state = args[1]
		message = args[2]
	}

	// Validate state
	validStates := map[string]bool{
		"starting": true,
		"working":  true,
		"blocked":  true,
		"testing":  true,
		"complete": true,
	}
	if !validStates[state] {
		err := fmt.Errorf("invalid state '%s': must be one of starting, working, blocked, testing, complete", state)
		if loopUpdateJSONFlag {
			return printLoopUpdateJSONError(err)
		}
		return err
	}

	cwd, err := GetWorkingDir()
	if err != nil {
		err = fmt.Errorf("failed to get current directory: %w", err)
		if loopUpdateJSONFlag {
			return printLoopUpdateJSONError(err)
		}
		return err
	}

	store, err := session.NewSessionStoreWithConfig(cwd, GetStoreConfig())
	if err != nil {
		err = fmt.Errorf("failed to initialize session store: %w", err)
		if loopUpdateJSONFlag {
			return printLoopUpdateJSONError(err)
		}
		return err
	}

	// Format timestamped entry
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("[%s] ball=%s state=%s message=%s\n", timestamp, ballID, state, message)

	// Map "all" meta-session to "_all" for storage
	storageID := sessionID
	if sessionID == "all" {
		storageID = "_all"
	}

	// Write to agent update file (overwrites existing content)
	if err := store.WriteAgentUpdate(storageID, entry); err != nil {
		err = fmt.Errorf("failed to write agent update: %w", err)
		if loopUpdateJSONFlag {
			return printLoopUpdateJSONError(err)
		}
		return err
	}

	if loopUpdateJSONFlag {
		return printLoopUpdateJSONSuccess(sessionID, ballID, state, message, timestamp)
	}

	// Success message for agent confirmation
	fmt.Printf("Updated agent status for session %s\n", sessionID)
	return nil
}

// LoopUpdateResponse is the JSON response for loop update command
type LoopUpdateResponse struct {
	Success   bool   `json:"success"`
	SessionID string `json:"session_id"`
	BallID    string `json:"ball_id"`
	State     string `json:"state"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

func printLoopUpdateJSONSuccess(sessionID, ballID, state, message, timestamp string) error {
	resp := LoopUpdateResponse{
		Success:   true,
		SessionID: sessionID,
		BallID:    ballID,
		State:     state,
		Message:   message,
		Timestamp: timestamp,
	}
	data, _ := json.Marshal(resp)
	fmt.Println(string(data))
	return nil
}

func printLoopUpdateJSONError(err error) error {
	errResp := map[string]string{"error": err.Error()}
	data, _ := json.Marshal(errResp)
	fmt.Println(string(data))
	return nil // Return nil so the error is in JSON, not stderr
}
