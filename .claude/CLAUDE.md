# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Test Commands

### Building

```bash
# Enter devbox shell (sets up Go environment)
devbox shell

# Build the binary
go build -o juggle ./cmd/juggle

# Install locally for testing
go install ./cmd/juggle
```

### Testing

```bash
# Run integration tests (quiet - shows pass/fail summary only)
devbox run test-quiet

# Run integration tests (verbose - full output)
devbox run test-verbose
# or: go test -v ./internal/integration_test/...

# Run all tests
devbox run test-all
# or: go test -v ./...

# Generate coverage report
devbox run test-coverage
# or: go test -v -coverprofile=coverage.out ./internal/integration_test/...
#     go tool cover -html=coverage.out -o coverage.html

# Run single test
go test -v ./internal/integration_test/... -run TestExport
```

### Development

```bash
# Clean build artifacts
go clean

# Update dependencies
go mod tidy

# Check formatting
go fmt ./...
```

## Architecture Overview

### Core Concepts

**Juggle** runs autonomous AI agent loops with good UX. Define tasks ("balls") with acceptance criteria via TUI or CLI, start the agent loop (`juggle agent run`), and add or modify tasks while it runs. No JSON editing - the TUI and CLI handle all task management.

### Directory Structure

```
juggler/
├── cmd/
│   └── juggle/
│       └── main.go              # Entry point, initializes CLI
├── internal/
│   ├── agent/                   # Agent execution and prompt generation
│   │   ├── provider/            # Multi-provider support (Claude, OpenCode)
│   │   │   ├── provider.go      # Provider interface definition
│   │   │   ├── claude.go        # Claude provider implementation
│   │   │   ├── opencode.go      # OpenCode provider implementation
│   │   │   ├── detect.go        # Auto-detect provider from environment
│   │   │   └── shared.go        # Shared provider utilities
│   │   ├── runner.go            # Agent runner interface and default impl
│   │   ├── prompt.go            # Prompt template generation
│   │   └── refine.go            # Interactive ball refinement
│   ├── cli/                     # Command-line interface
│   │   ├── root.go              # Root command and global flags
│   │   ├── agent.go             # Agent run/refine commands
│   │   ├── sessions.go          # Session CRUD commands
│   │   ├── export.go            # Export to various formats
│   │   ├── start.go, status.go, # Individual ball operations
│   │   ├── config.go            # Config management commands
│   │   └── ...                  # Other CLI commands
│   ├── session/                 # Core data model and storage
│   │   ├── ball.go              # Ball struct and state machine
│   │   ├── store.go             # JSONL persistence layer
│   │   ├── juggle_session.go   # Session entity and store
│   │   ├── config.go            # Global config (~/.juggle/config.json)
│   │   ├── discovery.go         # Cross-project ball discovery
│   │   ├── archive.go           # Completed ball archival
│   │   ├── agent_history.go     # Agent execution history tracking
│   │   ├── worktree.go          # Git worktree management
│   │   └── lock.go              # Concurrent access locking
│   ├── tui/                     # Terminal UI (Bubble Tea)
│   │   ├── list.go              # Split-view ball list (default)
│   │   ├── detail.go            # Legacy single-panel view
│   │   ├── agent_handlers.go   # Agent-related UI handlers
│   │   └── styles.go            # Lipgloss styling
│   ├── vcs/                     # Version control abstraction
│   │   ├── vcs.go               # VCS interface definition
│   │   ├── jj.go                # Jujutsu backend
│   │   ├── git.go               # Git backend
│   │   └── detect.go            # Auto-detect VCS from project
│   ├── watcher/                 # File system watcher
│   │   └── watcher.go           # fsnotify integration for live updates
│   └── integration_test/        # Integration test suite
│       └── testutil_test.go     # Test utilities and helpers
├── .juggle/                     # Per-project juggle data
│   ├── balls.jsonl              # Active balls (JSONL format)
│   ├── archive/
│   │   └── balls.jsonl          # Completed balls
│   ├── sessions/
│   │   └── <session-id>/
│   │       ├── session.json     # Session metadata
│   │       ├── progress.txt     # Progress log
│   │       └── last_output.txt  # Last agent output
│   └── config.json              # Project-local config
└── ~/.juggle/                   # Global user config
    └── config.json              # Global settings (search paths, VCS preferences)
```

### State Machine

Balls use a 5-state model:

- **pending** → Ball is planned but not started
- **in_progress** → Ball is actively being worked on
- **complete** → Task finished and archived
- **researched** → Investigation task finished with findings (stored in `Output`) but no code changes
- **blocked** → Task is blocked (with optional `BlockedReason` for context)

State transitions:
- `pending` → `in_progress` (via `start`)
- `in_progress` → `complete` (via completion commands)
- `in_progress` → `researched` (via research completion - produces findings, not code)
- `in_progress` → `blocked` (via block command with reason)
- `blocked` → `in_progress` (via unblock/resume)
- Any state → `pending` (reset)

### Key Components

#### 1. Session Package (`internal/session/`)

**`ball.go`** - Core data model:

- `Ball` struct fields:
  - `ID`, `Title`: Identity and short description
  - `Context`: Background info for the agent (rich text, can include @file references)
  - `AcceptanceCriteria`: Specific, testable conditions for completion
  - `State`: pending/in_progress/complete/researched/blocked
  - `BlockedReason`: Context when blocked (human needed, waiting for dependency, etc.)
  - `Priority`: low/medium/high/urgent
  - `ModelSize`: small (haiku), medium (sonnet), large (opus) - hints for agent model selection
  - `Output`: Research findings for `researched` state balls
  - `CompletionNote`: Summary of what was done when completing
  - `DependsOn`: List of ball IDs that must complete first
  - `Tags`: For filtering and session grouping
- Methods for state transitions, todo management

**`store.go`** - Persistent storage:

- JSONL format: `.juggle/balls.jsonl` (active), `.juggle/archive/balls.jsonl` (completed)
- `Store` type handles CRUD operations for balls
- Methods: `AppendBall()`, `LoadBalls()`, `UpdateBall()`, `ArchiveBall()`
- Ball resolution by ID or short ID

**`config.go`** - Global configuration:

- Location: `~/.juggle/config.json`
- Manages search paths for discovering projects with `.juggle/` directories
- Default paths: `~/Development`, `~/projects`, `~/work`

**`discovery.go`** - Cross-project discovery:

- `DiscoverProjects()`: Scans search paths for `.juggle/` directories
- `LoadAllBalls()`, `LoadInProgressBalls()`: Load balls across all discovered projects
- Enables global views like `juggle status` and `juggle next`

**`archive.go`** - Archival operations:

- `ArchiveBall()`: Moves completed balls to archive
- `LoadArchive()`: Query historical completed work

**`juggle_session.go`** - JuggleSession entity:

- `JuggleSession` struct: Groups balls by tag with context and acceptance criteria
- Fields: ID, Description, Context, DefaultModel, AcceptanceCriteria, CreatedAt, UpdatedAt
- `SessionStore` handles CRUD: `CreateSession()`, `LoadSession()`, `ListSessions()`, `DeleteSession()`
- Progress tracking: `AppendProgress()`, `LoadProgress()`

#### 2. CLI Package (`internal/cli/`)

**Command structure:**

- `root.go`: Main command dispatcher, handles `juggle` with no args (shows in-progress balls) or `juggle <ball-id> <action>`
- Each major command has its own file (e.g., `start.go`, `status.go`, `todo.go`)
- Helper functions: `GetWorkingDir()`, `NewStoreForCommand()`, `LoadConfigForCommand()`

**Key command patterns:**

- Commands operating on current ball: Get store → resolve current ball → operate → update store
- Cross-project commands: Load config → discover projects → load balls → operate
- Ball-specific commands: Find ball by ID across all projects → create store for that ball's directory → operate

**Session commands (`sessions.go`):**

- `juggle sessions list` - List all sessions
- `juggle sessions create <id> [-m description]` - Create new session
- `juggle sessions delete <id>` - Delete session
- `juggle sessions show <id>` - Show session details
- `juggle sessions context <id> [--edit]` - View/edit session context
- `juggle sessions progress <id>` - View progress log
- `juggle sessions edit <id>` - Edit session properties

**Agent commands (`agent.go`):**

- `juggle agent run` - Run autonomous agent loop
  - `--session <id>` or `--all` for meta-session
  - `--iterations N` (default: 10)
  - `--model <opus|sonnet|haiku>`
  - `--headless` for non-interactive mode
  - `--dry-run` for testing without execution
- `juggle agent refine` - Interactive ball refinement

**Export command (`export.go`):**

- `juggle export --session <id> --format <json|csv|ralph|agent>`
- Ralph format outputs `<context>`, `<progress>`, `<tasks>` sections

#### 3. TUI Package (`internal/tui/`)

**Bubble Tea-based terminal UI:**

**Split View (Default)** - Three-panel layout:
- Left Panel (25%): Sessions list with ball counts
- Right Panel (75%): Balls list with expandable todos
- Bottom Panel: Activity log

Navigation: `Tab` cycles panels, `j/k` navigates, `Enter` selects, `Esc` goes back

CRUD: `a` add, `e` edit, `x` delete, `s` start, `c` complete, `b` block, `Space` toggle todo

Flags: `--legacy` for single-panel mode, `--session <id>` pre-select session

**Legacy View** (`--legacy`): Single-panel ball list with detail view

#### 4. Watcher Package (`internal/watcher/`)

**File system watcher for live updates:**

- Uses `fsnotify` to monitor `.juggle/` directory
- Watches `balls.jsonl`, `session.json`, `progress.txt`
- Sends events to TUI for real-time updates
- Event types: `BallsChanged`, `ProgressChanged`, `SessionChanged`

## Storage Format

### JSONL Structure

Each ball is one line of JSON in `.juggle/balls.jsonl`:

```json
{
  "id": "juggle-5",
  "title": "Add search feature",
  "context": "Need full-text search across all balls",
  "acceptance_criteria": ["Search returns relevant results", "Handles special chars"],
  "priority": "high",
  "state": "in_progress",
  "blocked_reason": "",
  "model_size": "medium",
  "depends_on": ["juggle-3"],
  "started_at": "2025-10-16T10:30:00Z",
  "last_activity": "2025-10-16T11:45:00Z",
  "tags": ["feature", "backend"]
}
```

### File Locations

- Per-project balls: `.juggle/balls.jsonl` (active), `.juggle/archive/balls.jsonl` (complete)
- Global config: `~/.juggle/config.json`

### Session Storage

Sessions use directory structure:
- `.juggle/sessions/<id>/session.json` - Session data
- `.juggle/sessions/<id>/progress.txt` - Progress log

## Important Patterns

### Resolving Current Ball

When multiple balls exist in a project, resolution logic:

1. Check for explicit ball ID argument
2. If no ID provided, find current ball:
   - If exactly one in-progress ball exists → use it
   - If multiple in-progress balls → error, require explicit ID

### Cross-Project Operations

Commands like `status`, `next`, `search`, `history`:

1. Load config via `LoadConfigForCommand()`
2. Discover all projects with `session.DiscoverProjects(config)`
3. Load balls from all projects
4. Operate on aggregated data
5. When updating a ball, create a store for that ball's working directory

### State Transitions

Valid transitions (enforced in command handlers):

- `pending` → `in_progress` (via `start`)
- `in_progress` → `complete` (via complete command)
- `in_progress` → `blocked` (via block command)
- `blocked` → `in_progress` (via resume)

### Testing Utilities

Integration tests use `testutil_test.go`:

- `TestEnv`: Sets up isolated test environment with temp directories
- `SetupTestStore()`: Creates store with temp config
- Environment variable mocking for testing

## Multi-Agent Support

When multiple agents/users work simultaneously, set `JUGGLER_CURRENT_BALL` environment variable to explicitly target a ball:

```bash
export JUGGLER_CURRENT_BALL="juggle-5"
```

This ensures operations go to the correct ball when:

- Multiple AI agents work in same repo
- Multiple terminal sessions are active
- You want explicit control over which ball is targeted

## Key Data Flows

### Ball Lifecycle
1. User creates ball (CLI `plan` or TUI `a`)
2. `session.NewBall()` creates Ball with `pending` state
3. `store.AppendBall()` writes to `.juggle/balls.jsonl`
4. User activates ball: `juggle <id>` calls `ball.Start()` → `in_progress` state
5. `store.UpdateBall()` rewrites entire `balls.jsonl` with updated ball
6. Work happens (manual or agent-driven)
7. Ball completion: `juggle <id> complete` calls `ball.Complete()` → `complete` state
8. `store.ArchiveBall()` moves ball to `.juggle/archive/balls.jsonl`

### Agent Loop
1. `juggle agent run <session-id>` invokes CLI handler
2. `agent/prompt.go` generates prompt from session (ralph format export)
3. `agent/runner.go` executes provider (Claude/OpenCode) with prompt
4. Provider spawns CLI subprocess, captures output to `.juggle/sessions/<id>/last_output.txt`
5. Parse output for `<promise>COMPLETE</promise>` or `<promise>BLOCKED: reason</promise>` signals
6. If COMPLETE → archive ball; if BLOCKED → update ball state; else continue iteration
7. Repeat until max iterations or completion

### VCS Integration
1. On ball activation: `vcs.GetCurrentRevision()` stores in `ball.StartingRevision`
2. Optional: `vcs.DescribeWorkingCopy()` updates VCS description with ball info
3. On ball complete/block: `vcs.GetCurrentRevision()` stores in `ball.RevisionID`
4. VCS backend determined by: project config → global config → auto-detect (.jj/ or .git/)

### Cross-Project Discovery
1. Commands with `--all` flag trigger discovery
2. `config.SearchPaths` contains directories to scan (default: ~/Development, ~/projects, ~/work)
3. `session.DiscoverProjects()` finds all directories with `.juggle/` subdirectory
4. Load balls from each project's `balls.jsonl`
5. Aggregate and display combined view

### Live TUI Updates
1. `watcher.New()` starts fsnotify watcher
2. Monitors `.juggle/balls.jsonl`, `.juggle/sessions/<id>/{session.json,progress.txt}`
3. File changes trigger events sent as Bubble Tea messages
4. TUI `Update()` handler reloads affected data and re-renders panels

## Common Patterns for Adding Features

### Pattern 1: Adding a New CLI Command

Based on `internal/cli/show.go`:

1. **Create new file** `internal/cli/mycommand.go`:
   ```go
   package cli

   import (
       "github.com/ohare93/juggle/internal/session"
       "github.com/spf13/cobra"
   )

   var myCmd = &cobra.Command{
       Use:   "my <args>",
       Short: "Brief description",
       Long:  `Longer description...`,
       Args:  cobra.ExactArgs(1), // or other validator
       RunE:  runMy,
   }

   func init() {
       // Add flags if needed
       myCmd.Flags().BoolVar(&myFlag, "flag", false, "description")
   }

   func runMy(cmd *cobra.Command, args []string) error {
       // Get working directory
       cwd, err := GetWorkingDir()
       if err != nil {
           return fmt.Errorf("failed to get current directory: %w", err)
       }

       // Create store for accessing balls
       store, err := NewStoreForCommand(cwd)
       if err != nil {
           return fmt.Errorf("failed to initialize store: %w", err)
       }

       // Your command logic here
       return nil
   }
   ```

2. **Register in `internal/cli/root.go`** init() function:
   ```go
   rootCmd.AddCommand(myCmd)
   ```

3. **Add integration test** in `internal/integration_test/`

**Reference files:**
- Example: `internal/cli/show.go` (simple read command)
- Registration: `internal/cli/root.go:206-223`

### Pattern 2: Adding a New Ball Field

Based on commit `7ed3384` which added `AgentProvider` and `ModelOverride`:

1. **Add field to Ball struct** in `internal/session/ball.go:76-97`:
   ```go
   type Ball struct {
       // ... existing fields ...
       MyNewField string `json:"my_new_field,omitempty"`
   }
   ```

2. **Add validation function** (if needed) in `internal/session/ball.go`:
   ```go
   func ValidateMyNewField(s string) bool {
       // Validation logic
       return true
   }
   ```

3. **Add setter method** in `internal/session/ball.go`:
   ```go
   func (b *Ball) SetMyNewField(value string) {
       b.MyNewField = value
       b.UpdateActivity()  // Updates LastActivity and UpdateCount
   }
   ```

4. **Update CLI** in `internal/cli/update.go` to allow setting the field via `juggle update`

5. **Update TUI** in:
   - `internal/tui/ball_form.go` - Add form field for editing
   - `internal/tui/view.go` - Display field in ball view
   - `internal/tui/split_handlers.go` - Add keyboard handlers if needed

6. **Update export** in `internal/cli/export.go` if field should be included in exports

**Reference files:**
- Ball struct: `internal/session/ball.go:76-97`
- Example setters: `internal/session/ball.go:553-593` (SetModelSize, SetAgentProvider, SetModelOverride)
- CLI update: `internal/cli/update.go`
- TUI form: `internal/tui/ball_form.go`

### Pattern 3: Adding a VCS Operation

Based on `internal/vcs/vcs.go` interface:

1. **Add method to VCS interface** in `internal/vcs/vcs.go:31-64`:
   ```go
   type VCS interface {
       // ... existing methods ...

       // MyOperation performs a new VCS operation
       MyOperation(projectDir string) error
   }
   ```

2. **Implement for JJ** in `internal/vcs/jj.go`:
   ```go
   func (j *JJBackend) MyOperation(projectDir string) error {
       cmd := exec.Command("jj", "my-command", "args")
       cmd.Dir = projectDir
       output, err := cmd.CombinedOutput()
       if err != nil {
           return fmt.Errorf("jj my-command failed: %w\n%s", err, output)
       }
       return nil
   }
   ```

3. **Implement for Git** in `internal/vcs/git.go`:
   ```go
   func (g *GitBackend) MyOperation(projectDir string) error {
       cmd := exec.Command("git", "my-command", "args")
       cmd.Dir = projectDir
       output, err := cmd.CombinedOutput()
       if err != nil {
           return fmt.Errorf("git my-command failed: %w\n%s", err, output)
       }
       return nil
   }
   ```

4. **Add tests** in `internal/vcs/vcs_test.go` or `internal/integration_test/vcs_*.go`

**Reference files:**
- VCS interface: `internal/vcs/vcs.go:31-64`
- JJ implementation example: `internal/vcs/jj.go:101-108` (DescribeWorkingCopy)
- Git implementation example: `internal/vcs/git.go:118-124` (DescribeWorkingCopy)

### Pattern 4: Adding a New Agent Provider

Based on `internal/agent/provider/opencode.go`:

1. **Create provider file** `internal/agent/provider/myprovider.go`:
   ```go
   package provider

   import "os/exec"

   type MyProvider struct{}

   func NewMyProvider() *MyProvider {
       return &MyProvider{}
   }

   func (p *MyProvider) Type() Type {
       return Type("myprovider")
   }

   func (p *MyProvider) Name() string {
       return "My Provider"
   }

   func (p *MyProvider) Run(opts RunOptions) (*RunResult, error) {
       // Build command with provider-specific flags
       args := []string{"run", "--prompt", opts.Prompt}

       if opts.Model != "" {
           args = append(args, "--model", opts.Model)
       }

       cmd := exec.Command("myprovider", args...)
       cmd.Dir = opts.WorkingDir

       // Execute and capture output
       output, err := cmd.CombinedOutput()

       result := &RunResult{
           Output:   string(output),
           ExitCode: cmd.ProcessState.ExitCode(),
       }

       // Parse signals from output (COMPLETE, BLOCKED, etc.)
       parsePromiseSignals(result)

       return result, err
   }
   ```

2. **Add Type constant** in `internal/agent/provider/provider.go:12-17`:
   ```go
   const (
       TypeClaude   Type = "claude"
       TypeOpenCode Type = "opencode"
       TypeMyProvider Type = "myprovider"  // Add this
   )
   ```

3. **Update IsValid()** in `internal/agent/provider/provider.go:24-27`

4. **Add to detection logic** in `internal/agent/provider/detect.go`

5. **Update CLI flag** in `internal/cli/agent.go` to accept new provider name

**Reference files:**
- Provider interface: `internal/agent/provider/provider.go:78-94`
- Example implementation: `internal/agent/provider/opencode.go`
- Detection logic: `internal/agent/provider/detect.go`

## Key File Cross-References

### Ball Lifecycle
- **Ball struct definition**: `internal/session/ball.go:76-97`
- **State transitions**: `internal/session/ball.go:262-280` (Start, Complete, Block methods)
- **JSONL persistence**: `internal/session/store.go:100-150` (AppendBall, UpdateBall)
- **Archival**: `internal/session/archive.go:20-80`

### CLI Commands
- **Root command dispatcher**: `internal/cli/root.go:32`
- **Ball command router**: `internal/cli/juggling.go:136-154` (runRootCommand, handleBallCommand)
- **Ball activation**: `internal/cli/juggling.go:606-636` (activateBall)
- **Start command**: `internal/cli/start.go:26-321`
- **Agent run**: `internal/cli/agent.go:58-150`

### TUI
- **Split view (default)**: `internal/tui/list.go`
- **Key handlers**: `internal/tui/update.go:545-595`
- **State change sequence**: `internal/tui/split_handlers.go:20-44` (ss=start, sc=complete, etc.)
- **Ball form editor**: `internal/tui/ball_form.go`

### Agent System
- **Prompt generation**: `internal/agent/prompt.go:50-200`
- **Runner interface**: `internal/agent/runner.go:47-78`
- **Provider interface**: `internal/agent/provider/provider.go:78-94`
- **Claude provider**: `internal/agent/provider/claude.go`
- **Signal parsing**: `internal/agent/provider/shared.go:100-200`

### VCS Integration
- **VCS interface**: `internal/vcs/vcs.go:31-64`
- **Detection logic**: `internal/vcs/detect.go:20-50`
- **JJ backend**: `internal/vcs/jj.go`
- **Git backend**: `internal/vcs/git.go`

### Cross-Project Discovery
- **Config loading**: `internal/session/config.go:50-150`
- **Project discovery**: `internal/session/discovery.go:20-80`
- **Global flag handling**: `internal/cli/root.go:88-102` (DiscoverProjectsForCommand)

### Storage
- **Store interface**: `internal/session/store.go:30-80`
- **JSONL read/write**: `internal/session/store.go:100-250`
- **Session storage**: `internal/session/juggle_session.go:80-200`
- **File watching**: `internal/watcher/watcher.go:30-200`

## Code Style Notes

- Use `lipgloss` for terminal styling (colors, formatting)
- Commands return `error`, not `fmt.Errorf()` directly - wrap with context
- JSONL append-only writes for better version control diffs
- Ball IDs format: `<directory-name>-<counter>` (e.g., `juggle-5`)
