package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var progressAppendJSONFlag bool

var progressCmd = &cobra.Command{
	Use:   "progress",
	Short: "Manage session progress logs",
	Long:  `Commands for managing session progress logs (progress.txt files).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var progressAppendCmd = &cobra.Command{
	Use:   "append [session-id] <text>",
	Short: "Append a timestamped entry to session progress",
	Long: `Append a timestamped entry to a session's progress.txt file.

The session-id can be provided as the first argument, or via the
JUGGLE_SESSION_ID environment variable.

Creates progress.txt if it doesn't exist.

Examples:
  juggle progress append my-session "Completed user story US-001"
  JUGGLE_SESSION_ID=my-session juggle progress append "Fixed auth bug"
  juggle progress append my-session "Message" --json`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runProgressAppend,
}

func init() {
	progressAppendCmd.Flags().BoolVar(&progressAppendJSONFlag, "json", false, "Output as JSON")
	progressCmd.AddCommand(progressAppendCmd)
	rootCmd.AddCommand(progressCmd)
}

func runProgressAppend(cmd *cobra.Command, args []string) error {
	var sessionID, text string

	// Parse args: either (session-id, text) or just (text) with env var
	if len(args) == 2 {
		sessionID = args[0]
		text = args[1]
	} else {
		// Single arg - use env var for session ID
		sessionID = os.Getenv("JUGGLE_SESSION_ID")
		if sessionID == "" {
			err := fmt.Errorf("session ID required: provide as first argument or set JUGGLE_SESSION_ID")
			if progressAppendJSONFlag {
				return printProgressAppendJSONError(err)
			}
			return err
		}
		text = args[0]
	}

	cwd, err := GetWorkingDir()
	if err != nil {
		err = fmt.Errorf("failed to get current directory: %w", err)
		if progressAppendJSONFlag {
			return printProgressAppendJSONError(err)
		}
		return err
	}

	store, err := session.NewSessionStoreWithConfig(cwd, GetStoreConfig())
	if err != nil {
		err = fmt.Errorf("failed to initialize session store: %w", err)
		if progressAppendJSONFlag {
			return printProgressAppendJSONError(err)
		}
		return err
	}

	// Format timestamped entry
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("[%s] %s\n", timestamp, text)

	// Map "all" meta-session to "_all" for storage
	storageID := sessionID
	if sessionID == "all" {
		storageID = "_all"
	}

	// Append to progress file
	if err := store.AppendProgress(storageID, entry); err != nil {
		err = fmt.Errorf("failed to append progress: %w", err)
		if progressAppendJSONFlag {
			return printProgressAppendJSONError(err)
		}
		return err
	}

	if progressAppendJSONFlag {
		return printProgressAppendJSONSuccess(sessionID, text, timestamp)
	}

	// Success message for agent confirmation
	fmt.Printf("Appended to session %s progress.txt\n", sessionID)
	return nil
}

// ProgressAppendResponse is the JSON response for progress append command
type ProgressAppendResponse struct {
	Success   bool   `json:"success"`
	SessionID string `json:"session_id"`
	Text      string `json:"text"`
	Timestamp string `json:"timestamp"`
}

func printProgressAppendJSONSuccess(sessionID, text, timestamp string) error {
	resp := ProgressAppendResponse{
		Success:   true,
		SessionID: sessionID,
		Text:      text,
		Timestamp: timestamp,
	}
	data, _ := json.Marshal(resp)
	fmt.Println(string(data))
	return nil
}

func printProgressAppendJSONError(err error) error {
	errResp := map[string]string{"error": err.Error()}
	data, _ := json.Marshal(errResp)
	fmt.Println(string(data))
	return nil // Return nil so the error is in JSON, not stderr
}
