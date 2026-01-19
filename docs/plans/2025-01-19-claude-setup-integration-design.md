# Claude Code Setup Integration Design

## Overview

Enhance `juggle init` to configure Claude Code for optimal headless agent execution, reducing approval fatigue while improving security.

## Problem

Running autonomous agent loops requires Claude Code configuration:
- Sandbox mode for OS-level security boundaries
- Hooks for progress tracking
- Project-specific permissions for build tools, package managers

Currently users must configure this manually, leading to either:
- Approval fatigue (clicking through prompts)
- Security gaps (overly permissive settings)
- Broken metrics (missing hooks)

## Solution

### 1. Enhanced `juggle init` Flow

```
$ juggle init

Initialized jj repository in /home/user/myproject
Initialized juggle project at /home/user/myproject

Created .claude/settings.json with:
  ✓ Sandbox mode enabled (OS-level security boundaries)
  ✓ Hooks for progress tracking
  ✓ Secret file protection (.env, secrets/)
  ✓ Push confirmation prompts

These defaults reduce approval prompts for headless agent loops while
improving security by restricting what agents can access.

To complete setup with project-specific permissions (build tools,
package managers, dev servers), run interactive configuration now.

Configure project-specific settings? [Y/n]
```

**If Y:** Launches `juggle agent setup-repo` interactively.
**If N:** Bare-bones config ready, user can run `juggle agent setup-repo` later.

### 2. Bare-Bones Default Settings

`juggle init` creates `.claude/settings.json`:

```json
{
  "sandbox": {
    "enabled": true,
    "autoAllowBashIfSandboxed": true
  },
  "permissions": {
    "allow": [
      "Bash(juggle:*)"
    ],
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
  "hooks": {
    "PostToolUse": [{
      "matcher": "Write|Edit|Bash",
      "hooks": [{"type": "command", "command": "juggle loop hook-event post-tool"}]
    }],
    "PostToolUseFailure": [{
      "matcher": "Write|Edit|Bash",
      "hooks": [{"type": "command", "command": "juggle loop hook-event tool-failure"}]
    }],
    "Stop": [{
      "hooks": [{"type": "command", "command": "juggle loop hook-event stop"}]
    }],
    "SessionEnd": [{
      "hooks": [{"type": "command", "command": "juggle loop hook-event session-end"}]
    }]
  }
}
```

### 3. New `juggle agent setup-repo` Command

Launches agent with sandbox-setup skill to:
1. Analyze codebase and detect technology stack
2. Ask interactive questions about security posture
3. Generate tailored permissions for detected tools
4. Merge with existing settings (preserves hooks, sandbox, deny rules)

### 4. First-Run Checks

**Check A: Any `juggle` command without `.juggle/`**

```
$ juggle balls

No juggle project found in this directory.

Initialize now? [Y/n]
```

**Check B: `juggle agent run` without Claude settings**

```
$ juggle agent run --session my-feature

Claude Code is not configured for optimal headless execution:
  ✗ Sandbox mode not enabled
  ✓ Hooks installed
  ✗ Settings file missing

Run setup now? [Y/n/never]
```

- **Y**: Runs `juggle init` flow
- **N**: Continues anyway
- **never**: Stores `"skip_claude_setup_check": true` in `.juggle/config.json`

### 5. `juggle hooks install` Behavior Change

**Before (old behavior):**
- Default → `.claude/settings.local.json` (gitignored)
- `--project` flag → `.claude/settings.json`

**After (implemented):**
- Default → `.claude/settings.json` (version controlled)
- `--local` flag → `.claude/settings.local.json` (gitignored)

### 6. Sandbox Setup Skill

Location: `.claude/skills/sandbox-setup/`

Lightweight skill that:
- References nathanonn/claude-skills-sandbox-architect for stack detection
- Adds juggle-specific knowledge (preserve hooks, explain juggle features)
- Merges generated config with existing settings

### 7. Documentation Updates

**Rename:** `docs/hooks.md` → `docs/claude-integration.md`

New structure:
- Overview: Why sandbox and hooks matter for agent loops
- Sandbox Mode: OS-level isolation, key settings
- Hooks: Progress tracking integration
- Setup: `juggle init` and `juggle agent setup-repo`
- Troubleshooting

**README addition:**
```markdown
## Claude Code Setup

Juggle works best with Claude Code's sandbox mode enabled for headless
agent loops. Run `juggle init` to configure automatically, or see
[Claude Integration](docs/claude-integration.md) for details.
```

## Files Changed

### New Files
- `.claude/skills/sandbox-setup/SKILL.md` - Setup skill definition
- `internal/cli/agent_setup_repo.go` - New command

### Modified Files
- `internal/cli/init.go` - Add Claude settings creation, setup-repo prompt
- `internal/cli/hooks.go` - Flip default to settings.json
- `internal/cli/agent.go` - Add first-run check
- `internal/cli/root.go` - Add .juggle existence check
- `docs/hooks.md` → `docs/claude-integration.md` - Expand docs
- `README.md` - Add Claude setup section

### Settings File Updates (this repo)
- `.claude/settings.json` - Clean example with bare-bones defaults
- `.claude/settings.local.json` - Project-specific permissions only
