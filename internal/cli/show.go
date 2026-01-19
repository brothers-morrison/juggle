package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var showJSONFlag bool

var showCmd = &cobra.Command{
	Use:   "show <session-id>",
	Short: "Show detailed information about a session",
	Long:  `Display detailed information about a specific session.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runShow,
}

func init() {
	showCmd.Flags().BoolVar(&showJSONFlag, "json", false, "Output as JSON")
}

func runShow(cmd *cobra.Command, args []string) error {
	id := args[0]

	// Try to find a ball first
	foundBall, _, err := findBallByID(id)
	if err == nil {
		if showJSONFlag {
			return printBallJSON(foundBall)
		}
		renderBallDetails(foundBall)
		return nil
	}

	// Ball not found, try to find a session
	cwd, cwdErr := GetWorkingDir()
	if cwdErr != nil {
		if showJSONFlag {
			return printJSONError(cwdErr)
		}
		return cwdErr
	}

	store, storeErr := session.NewSessionStoreWithConfig(cwd, GetStoreConfig())
	if storeErr != nil {
		if showJSONFlag {
			return printJSONError(storeErr)
		}
		return storeErr
	}

	sess, sessErr := store.LoadSession(id)
	if sessErr != nil {
		// Neither ball nor session found
		if showJSONFlag {
			return printJSONError(fmt.Errorf("no ball or session found with id: %s", id))
		}
		return fmt.Errorf("no ball or session found with id: %s", id)
	}

	// Found a session - load linked balls and progress
	ballStore, _ := NewStoreForCommand(cwd)
	var sessionBalls []*session.Ball
	if ballStore != nil {
		allBalls, _ := ballStore.LoadBalls()
		for _, ball := range allBalls {
			for _, tag := range ball.Tags {
				if tag == id {
					sessionBalls = append(sessionBalls, ball)
					break
				}
			}
		}
	}
	progress, _ := store.LoadProgress(id)

	if showJSONFlag {
		return printSessionJSON(sess, sessionBalls, progress)
	}

	renderSessionDetails(sess, sessionBalls, progress)
	return nil
}

// printSessionJSON outputs a session with linked balls as JSON
func printSessionJSON(sess *session.JuggleSession, balls []*session.Ball, progress string) error {
	response := struct {
		Session  *session.JuggleSession `json:"session"`
		Balls    []*session.Ball        `json:"balls"`
		Progress string                 `json:"progress,omitempty"`
	}{
		Session:  sess,
		Balls:    balls,
		Progress: progress,
	}
	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return printJSONError(err)
	}
	fmt.Println(string(data))
	return nil
}

// printBallJSON outputs the ball as JSON
func printBallJSON(ball *session.Ball) error {
	data, err := json.MarshalIndent(ball, "", "  ")
	if err != nil {
		return printJSONError(err)
	}
	fmt.Println(string(data))
	return nil
}

// printJSONError outputs an error in JSON format to stdout
// Returns nil to prevent cobra from printing the error again to stderr
// Note: This means exit code will be 0 even on errors when using --json
// Callers can check the JSON "error" field to detect failures
func printJSONError(err error) error {
	errResp := map[string]string{"error": err.Error()}
	data, _ := json.Marshal(errResp)
	fmt.Println(string(data))
	return nil
}

func renderBallDetails(ball *session.Ball) {
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	valueStyle := lipgloss.NewStyle()

	fmt.Println(labelStyle.Render("Ball ID:"), valueStyle.Render(ball.ID))
	fmt.Println(labelStyle.Render("Working Dir:"), valueStyle.Render(ball.WorkingDir))
	if ball.Context != "" {
		fmt.Println(labelStyle.Render("Context:"), valueStyle.Render(ball.Context))
	}
	fmt.Println(labelStyle.Render("Title:"), valueStyle.Render(ball.Title))
	fmt.Println(labelStyle.Render("Priority:"), valueStyle.Render(string(ball.Priority)))
	fmt.Println(labelStyle.Render("State:"), valueStyle.Render(string(ball.State)))

	if ball.BlockedReason != "" {
		fmt.Println(labelStyle.Render("Blocked:"), valueStyle.Render(ball.BlockedReason))
	}

	fmt.Println(labelStyle.Render("Started:"), valueStyle.Render(ball.StartedAt.Format("2006-01-02 15:04:05")))
	fmt.Println(labelStyle.Render("Last Activity:"), valueStyle.Render(ball.LastActivity.Format("2006-01-02 15:04:05")))
	fmt.Println(labelStyle.Render("Updates:"), valueStyle.Render(fmt.Sprintf("%d", ball.UpdateCount)))

	if len(ball.Tags) > 0 {
		fmt.Println(labelStyle.Render("Tags:"), valueStyle.Render(strings.Join(ball.Tags, ", ")))
	}

	if len(ball.DependsOn) > 0 {
		fmt.Println(labelStyle.Render("Depends On:"), valueStyle.Render(strings.Join(ball.DependsOn, ", ")))
	}

	if len(ball.AcceptanceCriteria) > 0 {
		fmt.Printf("\n%s\n", labelStyle.Render("Acceptance Criteria:"))
		for i, ac := range ball.AcceptanceCriteria {
			fmt.Printf("  %d. %s\n", i+1, ac)
		}
	}

	if ball.CompletionNote != "" {
		fmt.Println(labelStyle.Render("\nCompletion Note:"), valueStyle.Render(ball.CompletionNote))
	}

	if ball.Output != "" {
		fmt.Printf("\n%s\n", labelStyle.Render("Output:"))
		fmt.Println(valueStyle.Render(ball.Output))
	}
}

func renderSessionDetails(sess *session.JuggleSession, balls []*session.Ball, progress string) {
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	valueStyle := lipgloss.NewStyle()
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))

	fmt.Println(headerStyle.Render("Session: " + sess.ID))
	fmt.Println()

	if sess.Description != "" {
		fmt.Println(labelStyle.Render("Description:"), valueStyle.Render(sess.Description))
	}
	fmt.Println(labelStyle.Render("Created:"), valueStyle.Render(sess.CreatedAt.Format(time.RFC3339)))
	fmt.Println(labelStyle.Render("Updated:"), valueStyle.Render(sess.UpdatedAt.Format(time.RFC3339)))

	// Acceptance criteria section
	fmt.Println()
	fmt.Printf("%s (%d)\n", labelStyle.Render("Acceptance Criteria:"), len(sess.AcceptanceCriteria))
	if len(sess.AcceptanceCriteria) > 0 {
		for i, ac := range sess.AcceptanceCriteria {
			fmt.Printf("  %d. %s\n", i+1, ac)
		}
	} else {
		fmt.Println("  (no session-level acceptance criteria)")
	}

	// Context section
	fmt.Println()
	fmt.Println(labelStyle.Render("Context:"))
	if sess.Context != "" {
		lines := strings.Split(sess.Context, "\n")
		for _, line := range lines {
			fmt.Printf("  %s\n", line)
		}
	} else {
		fmt.Println("  (no context set)")
	}

	// Balls section
	fmt.Println()
	fmt.Printf("%s (%d)\n", labelStyle.Render("Balls:"), len(balls))
	if len(balls) > 0 {
		for _, ball := range balls {
			stateStyle := lipgloss.NewStyle()
			switch ball.State {
			case session.StateInProgress:
				stateStyle = stateStyle.Foreground(lipgloss.Color("10"))
			case session.StatePending:
				stateStyle = stateStyle.Foreground(lipgloss.Color("14"))
			case session.StateBlocked:
				stateStyle = stateStyle.Foreground(lipgloss.Color("11"))
			case session.StateComplete:
				stateStyle = stateStyle.Foreground(lipgloss.Color("8"))
			}
			fmt.Printf("  - %s [%s] %s\n", ball.ID, stateStyle.Render(string(ball.State)), ball.Title)
		}
	} else {
		fmt.Println("  (no balls linked to this session)")
	}

	// Progress section
	fmt.Println()
	fmt.Println(labelStyle.Render("Progress:"))
	if progress != "" {
		lines := strings.Split(progress, "\n")
		for _, line := range lines {
			fmt.Printf("  %s\n", line)
		}
	} else {
		fmt.Println("  (no progress logged)")
	}
}
