package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var linkCmd = &cobra.Command{
	Use:   "link <ball-id> <beads-issue-id>",
	Short: "Link a ball to a beads issue",
	Long: `Link an existing ball to one or more beads issues.

Examples:
  juggle link juggler-5 bd-a1b2
  juggle link juggler-5 bd-a1b2 bd-c3d4

The first beads issue linked becomes the primary issue.`,
	Args: cobra.MinimumNArgs(2),
	RunE: runLink,
}

var unlinkCmd = &cobra.Command{
	Use:   "unlink <ball-id> <beads-issue-id>",
	Short: "Unlink a beads issue from a ball",
	Long: `Remove a beads issue link from a ball.

Examples:
  juggle unlink juggler-5 bd-a1b2

If you unlink the primary issue, the first remaining issue (if any) becomes primary.`,
	Args: cobra.ExactArgs(2),
	RunE: runUnlink,
}

func runLink(cmd *cobra.Command, args []string) error {
	ballID := args[0]
	beadsIssues := args[1:]

	// Find the ball across all discovered projects
	ball, store, err := findBallByID(ballID)
	if err != nil {
		return fmt.Errorf("failed to find ball %s: %w", ballID, err)
	}

	// Add the beads issues
	for _, beadsIssue := range beadsIssues {
		ball.AddBeadsIssue(beadsIssue)
	}

	// Update the ball
	if err := store.UpdateBall(ball); err != nil {
		return fmt.Errorf("failed to update ball: %w", err)
	}

	fmt.Printf("✓ Linked ball %s to beads issue(s): %s\n", ball.ID, strings.Join(beadsIssues, ", "))
	if ball.BeadsPrimary != "" {
		fmt.Printf("  Primary: %s\n", ball.BeadsPrimary)
	}
	if len(ball.BeadsIssues) > 0 {
		fmt.Printf("  All linked issues: %s\n", strings.Join(ball.BeadsIssues, ", "))
	}

	return nil
}

func runUnlink(cmd *cobra.Command, args []string) error {
	ballID := args[0]
	beadsIssue := args[1]

	// Find the ball across all discovered projects
	ball, store, err := findBallByID(ballID)
	if err != nil {
		return fmt.Errorf("failed to find ball %s: %w", ballID, err)
	}

	// Remove the beads issue
	if !ball.RemoveBeadsIssue(beadsIssue) {
		return fmt.Errorf("beads issue %s not found in ball %s", beadsIssue, ball.ID)
	}

	// Update the ball
	if err := store.UpdateBall(ball); err != nil {
		return fmt.Errorf("failed to update ball: %w", err)
	}

	fmt.Printf("✓ Unlinked beads issue %s from ball %s\n", beadsIssue, ball.ID)
	if len(ball.BeadsIssues) > 0 {
		fmt.Printf("  Remaining linked issues: %s\n", strings.Join(ball.BeadsIssues, ", "))
		if ball.BeadsPrimary != "" {
			fmt.Printf("  Primary: %s\n", ball.BeadsPrimary)
		}
	} else {
		fmt.Printf("  No beads issues linked\n")
	}

	return nil
}
