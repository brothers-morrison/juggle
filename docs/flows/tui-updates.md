# Live TUI Updates Flow

1. `watcher.New()` starts fsnotify watcher
2. Monitors `.juggle/balls.jsonl`, `.juggle/sessions/<id>/{session.json,progress.txt}`
3. File changes trigger events sent as Bubble Tea messages
4. TUI `Update()` handler reloads affected data and re-renders panels

## Key Files
- Watcher implementation: `internal/watcher/watcher.go:30-200`
- TUI update handler: `internal/tui/list.go:300-500`
- Agent handlers: `internal/tui/agent_handlers.go:1-100`
