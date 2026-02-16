## Quick Start

> **Prerequisite:** You need [Claude Code](https://claude.ai/code) or [OpenCode](https://opencode.ai/) already set up and authenticated. This is what Juggle will be running, with flags.

### Create a session and add tasks

```bash
cd ~/your-project
juggle sessions create my-feature
juggle tui                        # Add tasks via TUI (recommended)
```

### Run the agent loop

```bash
juggle agent run                  # Interactive session selector
juggle agent run my-feature       # Or specify session directly
```

### Refine already existing tasks interactively

```bash
juggle agent refine
juggle agent refine my-feature
```

### Manage while it runs

Open the TUI:

```bash
juggle                            # Add/edit/reorder tasks live
```

Or just add / edit / view tasks directly in the terminal:

```bash
juggle plan

juggle update 162b4eb0 --tags bug-fixes,loop-a

# Large updates for the agent to run during refinement. Users would use the TUI instead
juggle update cc58e434 --ac "juggle worktree add <path> registers worktree in main repo config" \
       --ac "juggle worktree add creates .juggle/link file in worktree pointing to main repo" \
       --ac "All juggle commands in worktree use main repo's .juggle/ for storage" \
       --ac "Ball WorkingDir reflects actual worktree path (not main repo)" \
       --ac "juggle worktree remove <path> unregisters and removes link file" \
       --ac "juggle worktree list shows registered worktrees" \
       --ac "Integration tests for worktree registration and ball sharing" \
       --ac "devbox run test passes"
```
