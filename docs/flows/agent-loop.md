# Agent Loop Flow

1. `juggle agent run <session-id>` invokes CLI handler
2. `agent/prompt.go` generates prompt from session (ralph format export)
3. `agent/runner.go` executes provider (Claude/OpenCode) with prompt
4. Provider spawns CLI subprocess, captures output to `.juggle/sessions/<id>/last_output.txt`
5. Parse output for `<promise>COMPLETE</promise>` or `<promise>BLOCKED: reason</promise>` signals
6. If COMPLETE → archive ball; if BLOCKED → update ball state; else continue iteration
7. Repeat until max iterations or completion

## Key Files
- CLI handler: `internal/cli/agent.go:58-150`
- Prompt generation: `internal/agent/prompt.go:50-200`
- Runner execution: `internal/agent/runner.go:47-78`
- Signal parsing: `internal/agent/provider/shared.go:100-200`
