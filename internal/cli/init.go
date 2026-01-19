package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ohare93/juggle/internal/vcs"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var initCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Initialize a new juggle project",
	Long: `Initialize a new juggle project in the current directory or at the specified path.

Creates the .juggle directory structure:
  .juggle/
  ├── balls.jsonl    # Active tasks
  ├── sessions/      # Session data
  └── archive/       # Completed tasks

Also creates .claude/settings.json with sensible defaults for autonomous
agent loops (sandbox mode, hooks, secret protection).

If no VCS (jj or git) is detected:
  - Initializes jj if available
  - Falls back to git otherwise

Safe to run on existing projects - only creates missing files.

Examples:
  juggle init              # Initialize in current directory
  juggle init ./myproject  # Initialize at specified path`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

// InitOptions configures the InitProject function.
type InitOptions struct {
	TargetDir            string    // Directory to initialize (required)
	JuggleDirName        string    // Name of juggle directory (default: ".juggle")
	InitVCS              bool      // Whether to initialize VCS if not present (default: true)
	CreateClaudeSettings bool      // Whether to create .claude/settings.json (default: true when not set)
	SkipSetupPrompt      bool      // Skip interactive setup-repo prompt
	Output               io.Writer // Where to write status messages (default: os.Stdout)
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
	juggleDirExists := false

	// Check if .juggle already exists
	if _, err := os.Stat(juggleDir); err == nil {
		juggleDirExists = true
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

	if juggleDirExists {
		fmt.Fprintf(opts.Output, "Juggle project already initialized at %s\n", opts.TargetDir)
	} else {
		fmt.Fprintf(opts.Output, "Initialized juggle project at %s\n", opts.TargetDir)
	}

	// Create or update Claude settings if requested (default behavior)
	if opts.CreateClaudeSettings {
		claudeSettingsPath := filepath.Join(opts.TargetDir, ".claude", "settings.json")
		result, err := ensureClaudeSettings(claudeSettingsPath)
		if err != nil {
			return fmt.Errorf("failed to configure Claude settings: %w", err)
		}
		if len(result.Added) > 0 || len(result.Preserved) > 0 {
			printClaudeSettingsResult(opts.Output, result)
		}
	}

	return nil
}

// ClaudeSettingsResult contains both added and preserved settings info.
type ClaudeSettingsResult struct {
	Added     []string
	Preserved []string
}

// ensureClaudeSettings creates or updates .claude/settings.json with default settings.
// Returns what was added and what was preserved (empty if nothing changed).
func ensureClaudeSettings(path string) (*ClaudeSettingsResult, error) {
	result := &ClaudeSettingsResult{}
	defaults := DefaultClaudeSettings()

	// Create .claude directory if needed
	claudeDir := filepath.Dir(path)
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .claude directory: %w", err)
	}

	// Load existing settings or start fresh
	existing, err := LoadClaudeSettings(path)
	if err != nil || existing == nil {
		existing = &ClaudeSettings{}
	}

	// Merge sandbox settings
	if existing.GetSandboxConfig() == nil {
		// Copy defaults' sandbox config to existing
		defaultSandbox := defaults.GetSandboxConfig()
		if defaultSandbox != nil {
			_ = existing.SetSandboxConfig(defaultSandbox)
		}
		result.Added = append(result.Added, "Sandbox mode enabled (OS-level security boundaries)")
	} else {
		result.Preserved = append(result.Preserved, "Sandbox settings")
	}

	// Merge permissions
	if existing.Permissions == nil {
		existing.Permissions = defaults.Permissions
		result.Added = append(result.Added, "Secret file protection (.env, secrets/)")
		result.Added = append(result.Added, "Push confirmation prompts")
	} else {
		result.Preserved = append(result.Preserved, "Permission rules")
	}

	// Merge hooks (check if juggler hooks are missing)
	if !hasJugglerHook(existing.Hooks["PostToolUse"]) {
		if existing.Hooks == nil {
			existing.Hooks = make(map[string][]HookMatcher)
		}
		for k, v := range defaults.Hooks {
			existing.Hooks[k] = v
		}
		result.Added = append(result.Added, "Hooks for progress tracking")
	} else {
		result.Preserved = append(result.Preserved, "Hooks")
	}

	// Only save if we added something
	if len(result.Added) > 0 {
		if err := SaveClaudeSettings(path, existing); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func printClaudeSettingsResult(w io.Writer, result *ClaudeSettingsResult) {
	fmt.Fprintln(w, "")
	if len(result.Added) > 0 {
		fmt.Fprintln(w, "Updated .claude/settings.json:")
		for _, item := range result.Added {
			fmt.Fprintf(w, "  + %s\n", item)
		}
	}
	if len(result.Preserved) > 0 {
		if len(result.Added) > 0 {
			fmt.Fprintln(w, "")
		}
		fmt.Fprintln(w, "Preserved existing settings:")
		for _, item := range result.Preserved {
			fmt.Fprintf(w, "  ✓ %s\n", item)
		}
	}
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "These defaults reduce approval prompts for headless agent loops")
	fmt.Fprintln(w, "while improving security by restricting what agents can access.")
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

	err := InitProject(InitOptions{
		TargetDir:            targetDir,
		JuggleDirName:        juggleDirName,
		InitVCS:              true,
		CreateClaudeSettings: true,
		Output:               os.Stdout,
	})
	if err != nil {
		return err
	}

	// Offer interactive setup if running in terminal
	if term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Println("")
		fmt.Println("To complete setup with project-specific permissions (build tools,")
		fmt.Println("package managers, dev servers), run interactive configuration now.")
		fmt.Println("")
		confirmed, err := ConfirmSingleKey("Configure project-specific settings?")
		if err == nil && confirmed {
			if err := runAgentSetupRepo(nil, nil); err != nil {
				fmt.Printf("Setup failed: %v\n", err)
				fmt.Println("You can run 'juggle agent setup-repo' later to configure.")
			}
		}
	}

	return nil
}
