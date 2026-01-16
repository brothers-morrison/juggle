# Package Directory Structure

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
