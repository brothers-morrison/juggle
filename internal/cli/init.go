package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ohare93/juggle/internal/vcs"
	"github.com/spf13/cobra"
)

var initForce bool

var initCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Initialize a new juggle project",
	Long: `Initialize a new juggle project in the current directory or at the specified path.

Creates the .juggle directory structure:
  .juggle/
  ├── balls.jsonl    # Active tasks
  ├── sessions/      # Session data
  └── archive/       # Completed tasks

If no VCS (jj or git) is detected:
  - Initializes jj if available
  - Falls back to git otherwise

If a VCS already exists, only creates the .juggle directory structure.

Examples:
  juggle init              # Initialize in current directory
  juggle init ./myproject  # Initialize at specified path
  juggle init --force      # Reinitialize even if .juggle exists`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

func init() {
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Reinitialize even if .juggle already exists")
}

// InitOptions configures the InitProject function.
type InitOptions struct {
	TargetDir     string // Directory to initialize (required)
	JuggleDirName string // Name of juggle directory (default: ".juggle")
	Force         bool   // Allow reinitialization if .juggle exists
	InitVCS       bool   // Whether to initialize VCS if not present (default: true)
	Output        io.Writer // Where to write status messages (default: os.Stdout)
}

// InitProject initializes a juggle project at the specified directory.
// This is the core logic extracted for testability.
func InitProject(opts InitOptions) error {
	if opts.TargetDir == "" {
		return fmt.Errorf("target directory is required")
	}

	if opts.JuggleDirName == "" {
		opts.JuggleDirName = ".juggle"
	}

	if opts.Output == nil {
		opts.Output = os.Stdout
	}

	// Create target directory if it doesn't exist
	if err := os.MkdirAll(opts.TargetDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", opts.TargetDir, err)
	}

	juggleDir := filepath.Join(opts.TargetDir, opts.JuggleDirName)

	// Check if .juggle already exists
	if _, err := os.Stat(juggleDir); err == nil {
		if !opts.Force {
			return fmt.Errorf("%s already exists (use --force to reinitialize)", juggleDir)
		}
	}

	// Initialize VCS if needed (default behavior unless explicitly disabled)
	if opts.InitVCS {
		vcsExists := vcs.IsVCSInitialized(opts.TargetDir)
		if !vcsExists {
			if vcs.IsJJAvailable() {
				if err := vcs.InitJJ(opts.TargetDir); err != nil {
					return fmt.Errorf("failed to initialize jj: %w", err)
				}
				fmt.Fprintf(opts.Output, "Initialized jj repository in %s\n", opts.TargetDir)
			} else if vcs.IsGitAvailable() {
				if err := vcs.InitGit(opts.TargetDir); err != nil {
					return fmt.Errorf("failed to initialize git: %w", err)
				}
				fmt.Fprintf(opts.Output, "Initialized git repository in %s\n", opts.TargetDir)
			} else {
				fmt.Fprintln(opts.Output, "Warning: Neither jj nor git is available. Continuing without VCS.")
			}
		}
	}

	// Create .juggle directory structure
	if err := os.MkdirAll(juggleDir, 0755); err != nil {
		return fmt.Errorf("failed to create %s directory: %w", opts.JuggleDirName, err)
	}

	// Create sessions directory
	sessionsDir := filepath.Join(juggleDir, "sessions")
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return fmt.Errorf("failed to create sessions directory: %w", err)
	}

	// Create archive directory
	archiveDir := filepath.Join(juggleDir, "archive")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return fmt.Errorf("failed to create archive directory: %w", err)
	}

	// Create empty balls.jsonl file
	ballsPath := filepath.Join(juggleDir, "balls.jsonl")
	if _, err := os.Stat(ballsPath); os.IsNotExist(err) {
		f, err := os.Create(ballsPath)
		if err != nil {
			return fmt.Errorf("failed to create balls.jsonl: %w", err)
		}
		f.Close()
	}

	fmt.Fprintf(opts.Output, "Initialized juggle project at %s\n", opts.TargetDir)
	return nil
}

func runInit(cmd *cobra.Command, args []string) error {
	// Determine target directory
	var targetDir string
	if len(args) > 0 {
		targetDir = args[0]
		// Convert to absolute path
		absPath, err := filepath.Abs(targetDir)
		if err != nil {
			return fmt.Errorf("failed to resolve path: %w", err)
		}
		targetDir = absPath
	} else {
		cwd, err := GetWorkingDir()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		targetDir = cwd
	}

	// Get juggle directory name from config
	juggleDirName := GlobalOpts.JuggleDir
	if juggleDirName == "" {
		juggleDirName = ".juggle"
	}

	return InitProject(InitOptions{
		TargetDir:     targetDir,
		JuggleDirName: juggleDirName,
		Force:         initForce,
		InitVCS:       true,
		Output:        os.Stdout,
	})
}
