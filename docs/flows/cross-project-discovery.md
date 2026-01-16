# Cross-Project Discovery Flow

1. Commands with `--all` flag trigger discovery
2. `config.SearchPaths` contains directories to scan (default: ~/Development, ~/projects, ~/work)
3. `session.DiscoverProjects()` finds all directories with `.juggle/` subdirectory
4. Load balls from each project's `balls.jsonl`
5. Aggregate and display combined view

## Key Files
- Config loading: `internal/session/config.go:50-150`
- Discovery logic: `internal/session/discovery.go:20-80`
- CLI flag handling: `internal/cli/root.go:88-102`
