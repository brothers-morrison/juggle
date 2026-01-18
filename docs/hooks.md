# Claude Code Hooks Integration

Juggler integrates with Claude Code hooks to provide enhanced progress tracking during agent runs. Hooks automatically capture metrics like file changes, tool counts, and token usage without requiring explicit updates from the agent.

## Overview

When Claude Code hooks are installed, juggler receives real-time updates about:

- **Files changed** - Which files were modified during the session
- **Tool counts** - How many Write, Edit, Bash operations occurred
- **Tool failures** - Count of failed tool executions
- **Token usage** - Input, output, and cache tokens (cumulative)
- **Turn count** - Number of agent responses
- **Session state** - Whether the session ended cleanly

This supplements the semantic state from `juggle loop update` (which ball is active, which AC is being tested) with mechanical observations about the agent's activity.

## Installation

### Automatic Installation

When you run `juggle agent run`, juggler will check if hooks are installed and offer to install them:

```
Claude Code hooks are not installed.
Hooks provide enhanced progress tracking: file changes, tool counts, token usage.

Install hooks now? [Y/n]
```

Press Enter or `y` to install. Use `--skip-hooks-check` to bypass this prompt.

### Manual Installation

```bash
juggle hooks install           # Install to ~/.claude/settings.json
juggle hooks install --project # Install to project .claude/settings.json
juggle hooks status           # Check installation status
```

### What Gets Installed

The following hooks are added to your Claude settings:

```json
{
  "hooks": {
    "PostToolUse": [{
      "matcher": "Write|Edit|Bash",
      "hooks": [{ "type": "command", "command": "juggle loop hook-event post-tool" }]
    }],
    "PostToolUseFailure": [{
      "matcher": "Write|Edit|Bash",
      "hooks": [{ "type": "command", "command": "juggle loop hook-event tool-failure" }]
    }],
    "Stop": [{
      "hooks": [{ "type": "command", "command": "juggle loop hook-event stop" }]
    }],
    "SessionEnd": [{
      "hooks": [{ "type": "command", "command": "juggle loop hook-event session-end" }]
    }]
  }
}
```

Hooks are merged with existing configuration - your other hooks are preserved.

## How It Works

1. **Agent starts**: `juggle agent run` sets the `JUGGLE_SESSION_ID` environment variable
2. **Claude runs tools**: Hooks fire after each tool execution
3. **Hooks update metrics**: `juggle loop hook-event` reads JSON from stdin and updates `.juggle/sessions/<id>/agent-metrics.json`
4. **TUI displays metrics**: File watcher detects changes and updates the monitor view

### Environment Variable

Hooks only activate when `JUGGLE_SESSION_ID` is set. This environment variable is automatically set by `juggle agent run` and passed to the Claude process.

If you're running Claude manually and want hook metrics:

```bash
export JUGGLE_SESSION_ID=my-session
claude --print ...
```

## Viewing Metrics

Metrics are displayed in the agent monitor view (launched via `juggle agent run --monitor` or by pressing `W` on a running daemon in the TUI).

The metrics panel shows:
- **Tools**: Total tool calls (and failures if any)
- **Files**: Number of unique files modified
- **Tokens**: Total tokens used (with cache hits if applicable)

## Troubleshooting

### Hooks not working

1. Check if hooks are installed: `juggle hooks status`
2. Verify `JUGGLE_SESSION_ID` is set in the Claude process
3. Check that `juggle` is in your PATH
4. Look for errors in Claude's output or logs

### Metrics not updating

1. Ensure the agent is running through `juggle agent run` (not manually)
2. Check that the session directory exists: `.juggle/sessions/<id>/`
3. Verify the metrics file is being written: `cat .juggle/sessions/<id>/agent-metrics.json`

### Hooks conflicting with existing hooks

Juggler hooks are appended to existing hook configurations. If you have conflicting hooks:

1. Edit `~/.claude/settings.json` or `.claude/settings.json`
2. Adjust hook order or matchers as needed
3. Juggler hooks are identified by commands starting with `juggle`

## Files

- **Metrics file**: `.juggle/sessions/<session-id>/agent-metrics.json`
- **User settings**: `~/.claude/settings.json`
- **Project settings**: `.claude/settings.json`

## Comparison: Hooks vs Loop Update

| Feature | Hooks | Loop Update |
|---------|-------|-------------|
| Automatic | Yes | No - requires agent call |
| File tracking | Yes | No |
| Token usage | Yes | No |
| Semantic state | No | Yes (which AC, phase) |
| Ball context | No | Yes |
| Setup required | Yes (one-time) | No |

**Best practice**: Use both. Loop update provides semantic context ("Testing AC #3"), hooks provide mechanical metrics (files changed, tokens used).
