# Ball Lifecycle Flow

1. User creates ball (CLI `plan` or TUI `a`)
2. `session.NewBall()` creates Ball with `pending` state
3. `store.AppendBall()` writes to `.juggle/balls.jsonl`
4. User activates ball: `juggle <id>` calls `ball.Start()` → `in_progress` state
5. `store.UpdateBall()` rewrites entire `balls.jsonl` with updated ball
6. Work happens (manual or agent-driven)
7. Ball completion: `juggle <id> complete` calls `ball.Complete()` → `complete` state
8. `store.ArchiveBall()` moves ball to `.juggle/archive/balls.jsonl`

## Key Files
- Ball state machine: `internal/session/ball.go:262-280`
- Storage operations: `internal/session/store.go:100-150`
- CLI activation: `internal/cli/juggling.go:606-636`
