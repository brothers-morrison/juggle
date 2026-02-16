package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ohare93/juggle/internal/session"
	"github.com/ohare93/juggle/internal/specparser"
	"github.com/spf13/cobra"
)

var (
	importSpecSessionID string
	importSpecDryRun    bool
	importSpecFiles     []string
)

// importSpecCmd imports spec.md and PRD.md as balls
var importSpecCmd = &cobra.Command{
	Use:   "spec [files...]",
	Short: "Import balls from spec.md and PRD.md files",
	Long: `Import tasks from spec.md and PRD.md files as juggle balls.

Automatically searches the current directory for spec.md and PRD.md files
(case-insensitive). You can also specify files explicitly.

Each H2 (##) section in the markdown becomes a ball:
  - Heading text       -> ball title
  - Paragraph text     -> ball context
  - Bullet/numbered/checkbox lists -> acceptance criteria
  - Inline tags like [high], [urgent] -> priority
  - Inline tags like [small], [large] -> model size

Skips sections that already exist as balls (matching by title).

Examples:
  # Auto-detect and import from spec.md and PRD.md in current dir
  juggle import spec

  # Import from specific files
  juggle import spec docs/spec.md docs/PRD.md

  # Preview what would be imported (dry run)
  juggle import spec --dry-run

  # Import and tag with a session
  juggle import spec --session my-feature

Example spec.md format:
  ## Add user authentication [high]

  Users need to log in with email and password.

  - Support email/password login
  - Add password reset flow
  - Rate limit login attempts

  ## Refactor database layer [medium] [small]

  The current DB layer is tightly coupled.

  1. Abstract database interface
  2. Add connection pooling`,
	RunE: runImportSpec,
}

// ballsFromSpecCmd is a top-level convenience command (--balls-from-spec)
var ballsFromSpecCmd = &cobra.Command{
	Use:    "balls-from-spec [files...]",
	Short:  "Generate balls from spec.md and PRD.md files",
	Long:   `Convenience alias for 'juggle import spec'. See 'juggle import spec --help' for full documentation.`,
	RunE:   runImportSpec,
	Hidden: true, // Available but not shown in main help (use import spec instead)
}

func init() {
	// Flags for import spec subcommand
	importSpecCmd.Flags().StringVarP(&importSpecSessionID, "session", "s", "", "Session ID to tag imported balls with")
	importSpecCmd.Flags().BoolVar(&importSpecDryRun, "dry-run", false, "Preview what would be imported without creating balls")

	// Flags for top-level convenience command
	ballsFromSpecCmd.Flags().StringVarP(&importSpecSessionID, "session", "s", "", "Session ID to tag imported balls with")
	ballsFromSpecCmd.Flags().BoolVar(&importSpecDryRun, "dry-run", false, "Preview what would be imported without creating balls")

	// Register import spec as subcommand of import
	importCmd.AddCommand(importSpecCmd)

	// Register top-level convenience command
	rootCmd.AddCommand(ballsFromSpecCmd)
}

func runImportSpec(cmd *cobra.Command, args []string) error {
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Validate session exists if specified
	if importSpecSessionID != "" {
		sessionStore, err := session.NewSessionStore(cwd)
		if err != nil {
			return fmt.Errorf("failed to create session store: %w", err)
		}
		if _, err := sessionStore.LoadSession(importSpecSessionID); err != nil {
			return fmt.Errorf("session not found: %s", importSpecSessionID)
		}
	}

	// Determine which files to parse
	var parsedBalls []specparser.ParsedBall

	if len(args) > 0 {
		// Parse explicitly specified files
		for _, file := range args {
			path := file
			if !filepath.IsAbs(path) {
				path = filepath.Join(cwd, path)
			}
			balls, err := specparser.ParseFile(path)
			if err != nil {
				return fmt.Errorf("failed to parse %s: %w", file, err)
			}
			parsedBalls = append(parsedBalls, balls...)
		}
	} else {
		// Auto-detect spec.md and PRD.md in current directory
		parsedBalls, err = specparser.ParseDirectory(cwd)
		if err != nil {
			return err
		}
	}

	if len(parsedBalls) == 0 {
		fmt.Println("No ball definitions found in the spec files.")
		return nil
	}

	// Dry run: just show what would be imported
	if importSpecDryRun {
		return printDryRun(parsedBalls)
	}

	// Create store and import balls
	return importSpecBalls(parsedBalls, cwd, importSpecSessionID)
}

// printDryRun displays what would be imported without creating balls
func printDryRun(balls []specparser.ParsedBall) error {
	fmt.Printf("Found %d ball(s) to import:\n\n", len(balls))

	for i, b := range balls {
		priority := b.Priority
		if priority == "" {
			priority = "medium"
		}
		fmt.Printf("  %d. %s\n", i+1, b.Title)
		fmt.Printf("     Priority: %s\n", priority)
		if b.ModelSize != "" {
			fmt.Printf("     Model size: %s\n", b.ModelSize)
		}
		if b.Context != "" {
			ctx := b.Context
			if len(ctx) > 80 {
				ctx = ctx[:77] + "..."
			}
			fmt.Printf("     Context: %s\n", ctx)
		}
		if len(b.AcceptanceCriteria) > 0 {
			fmt.Printf("     Acceptance criteria: %d\n", len(b.AcceptanceCriteria))
			for _, ac := range b.AcceptanceCriteria {
				fmt.Printf("       - %s\n", ac)
			}
		}
		if len(b.Tags) > 0 {
			fmt.Printf("     Tags: %s\n", strings.Join(b.Tags, ", "))
		}
		fmt.Printf("     Source: %s\n", b.SourceFile)
		fmt.Println()
	}

	fmt.Println("Run without --dry-run to import these balls.")
	return nil
}

// importSpecBalls creates juggle balls from parsed spec data
func importSpecBalls(parsedBalls []specparser.ParsedBall, projectDir, sessionID string) error {
	store, err := NewStoreForCommand(projectDir)
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}

	// Load existing balls to check for duplicates
	existingBalls, err := store.LoadBalls()
	if err != nil {
		return fmt.Errorf("failed to load existing balls: %w", err)
	}

	existingTitles := make(map[string]bool)
	for _, ball := range existingBalls {
		existingTitles[ball.Title] = true
	}

	var imported, skipped int

	for _, pb := range parsedBalls {
		if pb.Title == "" {
			continue
		}

		// Check for existing ball with same title
		if existingTitles[pb.Title] {
			fmt.Printf("Skipped: \"%s\" (already exists)\n", pb.Title)
			skipped++
			continue
		}

		// Determine priority
		priority := pb.Priority
		if priority == "" {
			priority = "medium"
		}
		if !session.ValidatePriority(priority) {
			fmt.Printf("Warning: invalid priority %q for \"%s\", using medium\n", priority, pb.Title)
			priority = "medium"
		}

		// Create ball
		ball, err := session.NewBall(projectDir, pb.Title, session.Priority(priority))
		if err != nil {
			fmt.Printf("Warning: failed to create ball for \"%s\": %v\n", pb.Title, err)
			continue
		}

		ball.State = session.StatePending

		// Set context
		if pb.Context != "" {
			ball.Context = pb.Context
		}

		// Set acceptance criteria
		if len(pb.AcceptanceCriteria) > 0 {
			ball.SetAcceptanceCriteria(pb.AcceptanceCriteria)
		}

		// Set model size
		if pb.ModelSize != "" {
			ms := session.ModelSize(pb.ModelSize)
			if session.ValidateModelSize(pb.ModelSize) {
				ball.ModelSize = ms
			}
		}

		// Add spec-related tags
		for _, tag := range pb.Tags {
			ball.AddTag(tag)
		}

		// Add source file as tag
		ball.AddTag("spec:" + filepath.Base(pb.SourceFile))

		// Add session tag if specified
		if sessionID != "" {
			ball.AddTag(sessionID)
		}

		// Save ball
		if err := store.AppendBall(ball); err != nil {
			fmt.Printf("Warning: failed to save ball for \"%s\": %v\n", pb.Title, err)
			continue
		}

		imported++
		fmt.Printf("Imported: \"%s\" -> %s (%s)\n", pb.Title, ball.ID, ball.Priority)

		// Track title to avoid duplicates within this import
		existingTitles[pb.Title] = true
	}

	fmt.Printf("\nImport complete: %d imported, %d skipped\n", imported, skipped)

	// Ensure project is in search paths for discovery
	_ = session.EnsureProjectInSearchPaths(projectDir)

	return nil
}
