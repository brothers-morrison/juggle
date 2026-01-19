# Claude Code Integration

Juggle works best with Claude Code's sandbox mode enabled for autonomous agent loops. This guide explains the integration and configuration options.

## Quick Setup

Run `juggle init` in your project directory to automatically configure:
- Sandbox mode for OS-level security isolation
- Hooks for progress tracking
- Secret file protection
- Push confirmation prompts

For project-specific permissions, run `juggle agent setup-repo` after initialization.

## Why Sandbox Mode Matters

Claude Code's sandbox mode replaces approval prompts with OS-level security boundaries. Instead of clicking through "Allow this command?" prompts, the sandbox enables commands to run autonomously within predefined limits.

**Benefits:**
- **Reduced approval fatigue** - Commands run headlessly without interruption
- **Better security** - OS-enforced restrictions (not just application-level)
- **Improved success rate** - Agent loops complete without stalling on prompts

**How it works:**
- Filesystem isolation: Claude can only read/write within allowed paths
- Network isolation: Only approved domains are accessible
- These are kernel-level restrictions, not bypassable prompts

Reference: https://www.nathanonn.com/claude-code-sandbox-explained/

## Default Settings

`juggle init` creates `.claude/settings.json` with:

```json
{
  "sandbox": {
    "enabled": true,
    "autoAllowBashIfSandboxed": true
  },
  "permissions": {
    "allow": ["Bash(juggle:*)"],
    "deny": [
      "Read(./.env)",
      "Read(./.env.*)",
      "Read(./secrets/**)"
    ],
    "ask": [
      "Bash(juggle agent:*)",
      "Bash(jj git push:*)",
      "Bash(git push:*)"
    ]
  },
  "hooks": { ... }
}
```

### Key Settings Explained

| Setting | Purpose |
|---------|---------|
| `sandbox.enabled` | Activates OS-level isolation |
| `autoAllowBashIfSandboxed` | Allows commands within sandbox without prompts |
| `permissions.allow` | Commands that run without approval |
| `permissions.deny` | Files/commands that are blocked |
| `permissions.ask` | Commands that prompt for confirmation |

## Hooks Integration

Juggler integrates with Claude Code hooks to provide enhanced progress tracking during agent runs. Hooks automatically capture metrics like file changes, tool counts, and token usage without requiring explicit updates from the agent.

### Overview

When Claude Code hooks are installed, juggler receives real-time updates about:

- **Files changed** - Which files were modified during the session
- **Tool counts** - How many Write, Edit, Bash operations occurred
- **Tool failures** - Count of failed tool executions
- **Token usage** - Input, output, and cache tokens (cumulative)
- **Turn count** - Number of agent responses
- **Session state** - Whether the session ended cleanly

This supplements the semantic state from `juggle loop update` (which ball is active, which AC is being tested) with mechanical observations about the agent's activity.

### Installation

Hooks are installed automatically when you run `juggle init`. You can also manage them manually:

```bash
juggle hooks install           # Install to .claude/settings.json (default, version controlled)
juggle hooks install --local   # Install to .claude/settings.local.json (gitignored)
juggle hooks install --global  # Install to ~/.claude/settings.json (all projects)
juggle hooks status            # Check installation status
```

When you run `juggle agent run`, juggler will check if hooks are installed and offer to install them if missing.

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

### How It Works

1. **Agent starts**: `juggle agent run` sets the `JUGGLE_SESSION_ID` environment variable
2. **Claude runs tools**: Hooks fire after each tool execution
3. **Hooks update metrics**: `juggle loop hook-event` reads JSON from stdin and updates `.juggle/sessions/<id>/agent-metrics.json`
4. **TUI displays metrics**: File watcher detects changes and updates the monitor view

#### Environment Variable

Hooks only activate when `JUGGLE_SESSION_ID` is set. This environment variable is automatically set by `juggle agent run` and passed to the Claude process.

If you're running Claude manually and want hook metrics:

```bash
export JUGGLE_SESSION_ID=my-session
claude --print ...
```

### Viewing Metrics

Metrics are displayed in the agent monitor view (launched via `juggle agent run --monitor` or by pressing `W` on a running daemon in the TUI).

The metrics panel shows:
- **Tools**: Total tool calls (and failures if any)
- **Files**: Number of unique files modified
- **Tokens**: Total tokens used (with cache hits if applicable)

### Comparison: Hooks vs Loop Update

| Feature | Hooks | Loop Update |
|---------|-------|-------------|
| Automatic | Yes | No - requires agent call |
| File tracking | Yes | No |
| Token usage | Yes | No |
| Semantic state | No | Yes (which AC, phase) |
| Ball context | No | Yes |
| Setup required | Yes (one-time) | No |

**Best practice**: Use both. Loop update provides semantic context ("Testing AC #3"), hooks provide mechanical metrics (files changed, tokens used).

## Project-Specific Setup

For tailored configuration beyond the defaults, run:

```bash
juggle agent setup-repo
```

This launches an interactive AI-assisted configuration that:
1. Analyzes your codebase for technology stack
2. Asks about your development workflow
3. Generates appropriate permissions for your tools
4. Merges with existing settings

## Manual Configuration

### Adding Tool Permissions

To allow additional commands, add to `permissions.allow`:

```json
{
  "permissions": {
    "allow": [
      "Bash(juggle:*)",
      "Bash(npm:*)",
      "Bash(go:*)",
      "Bash(cargo:*)"
    ]
  }
}
```

### Network Access

To allow additional domains for package registries:

```json
{
  "sandbox": {
    "network": {
      "allowedHosts": ["registry.npmjs.org", "proxy.golang.org"]
    }
  }
}
```

### File System Access

To allow writing to additional directories:

```json
{
  "sandbox": {
    "filesystem": {
      "write": {
        "additionalAllow": ["~/.cache/go-build", "~/go"]
      }
    }
  }
}
```

## Troubleshooting

### Sandbox Issues

**Commands failing with permission errors:**
1. Check if the command is in `permissions.allow`
2. For sandboxed commands, verify the sandbox allows the operation
3. Run `juggle agent setup-repo` to regenerate permissions

**Network requests failing:**
1. Add the domain to `sandbox.network.allowedHosts`
2. Or add `WebFetch(domain:example.com)` to `permissions.allow`

### Hooks Issues

**Hooks not working:**
1. Check if hooks are installed: `juggle hooks status`
2. Verify `JUGGLE_SESSION_ID` is set in the Claude process
3. Check that `juggle` is in your PATH
4. Look for errors in Claude's output or logs

**Metrics not updating:**
1. Ensure the agent is running through `juggle agent run` (not manually)
2. Check that the session directory exists: `.juggle/sessions/<id>/`
3. Verify the metrics file is being written: `cat .juggle/sessions/<id>/agent-metrics.json`

**Hooks conflicting with existing hooks:**
Juggler hooks are appended to existing hook configurations. If you have conflicting hooks:
1. Edit `~/.claude/settings.json` or `.claude/settings.json`
2. Adjust hook order or matchers as needed
3. Juggler hooks are identified by commands starting with `juggle`

## Files

- **Claude settings**: `.claude/settings.json` (version controlled)
- **Local overrides**: `.claude/settings.local.json` (gitignored, higher priority)
- **User settings**: `~/.claude/settings.json` (global, lowest priority)
- **Metrics file**: `.juggle/sessions/<session-id>/agent-metrics.json`
