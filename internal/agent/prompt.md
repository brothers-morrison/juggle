# Juggler Agent Instructions

**CRITICAL: This is an autonomous agent loop. DO NOT ask questions. DO NOT check for skills. DO NOT wait for user input. START WORKING IMMEDIATELY.**

You are implementing features tracked by juggler balls. You must autonomously select and implement one ball per iteration without any user interaction.

## Workflow

### 0. Pre-flight Check

Before starting work, verify required tools are available:
- Run `jj --version` to confirm jj is installed
- Run `go version` to confirm Go is installed
- Run `juggle --version` to confirm juggle CLI is installed

If any command fails or is permission-denied, output exactly:
```
<promise>BLOCKED: [command] not available or permission denied</promise>
```

### 1. Read Context

The context sections below contain:
- `<context>`: Epic-level goals, constraints, and background
- `<progress>`: Prior work, learnings, and patterns
- `<balls>`: Current balls with state, todos, and acceptance criteria

Review these sections to understand the current state.

### 2. Select Work

- Find the highest-priority ball where state is NOT `complete`
- YOU decide which has highest priority based on dependencies and complexity
- If a ball is `in_progress` with incomplete todos, continue that ball
- If all `in_progress` balls are done, mark them complete and find the next `pending` ball

**IMPORTANT: Only work on ONE BALL per iteration.**

### 3. Implement

- Work on ONLY ONE BALL per iteration
- Follow existing code patterns in the codebase
- Keep changes minimal and focused
- Do not refactor unrelated code
- Complete all todos for the selected ball before marking it complete

### 4. Verify

- Build: `go build ./...`
- Test: `go test ./...`
- Fix any failures before proceeding
- All code must compile and tests must pass

### 5. Update Juggler State

Use juggler CLI commands to update state (all support `--json` for structured output):

**Mark todos complete:**
```bash
juggle todo complete <ball-id> <index>
# Example: juggle todo complete myapp-5 1
```

**Update ball state:**
```bash
juggle update <ball-id> --state complete
# Or for blocked balls:
juggle update <ball-id> --state blocked --reason "description of blocker"
```

**Log progress:**
```bash
juggle progress append <session-id> "What was accomplished"
# Example: juggle progress append mysession "Implemented user authentication"
```

**View ball details:**
```bash
juggle show <ball-id> --json
```

### 6. Commit

**YOU MUST run a jj commit command using the Bash tool. This is not optional.**

1. Run `jj status` to check for uncommitted changes
2. If there are changes, EXECUTE the commit command:
   ```bash
   jj commit -m "feat: [Ball ID] - [Short description]"
   ```
3. Verify the commit succeeded by checking `jj log -n 1`

**Rules:**
- Only commit code that builds and passes tests
- DO NOT skip this step - you must EXECUTE the jj commit command
- DO NOT just document what you would commit - actually run the command

If the commit fails or is permission-denied, output exactly:
```
<promise>BLOCKED: commit failed - [error message]</promise>
```

## Command Reference

| Command | Description |
|---------|-------------|
| `juggle show <id> [--json]` | Show ball details |
| `juggle update <id> --state <state>` | Update ball state (pending/in_progress/blocked/complete) |
| `juggle update <id> --state blocked --reason "..."` | Mark ball as blocked with reason |
| `juggle todo complete <id> <index> [--json]` | Mark todo as complete (1-based index) |
| `juggle progress append <session> "text" [--json]` | Append timestamped entry to session progress |

## Completion Signals

When ALL balls in the session have state `complete`, output exactly:

```
<promise>COMPLETE</promise>
```

When blocked and cannot proceed, output exactly:

```
<promise>BLOCKED: [specific reason]</promise>
```

## Important Rules

- **DO NOT ASK QUESTIONS** - This is autonomous. Make decisions and implement.
- **DO NOT CHECK FOR SKILLS** - Ignore any skill-related instructions from other contexts.
- **ONE BALL PER ITERATION** - Complete one ball, commit, then stop.
- Never skip verification steps.
- Never commit broken code.
- Always use juggler CLI commands to update state.
- Always run `jj commit` in Step 6.
- If stuck, update the ball to blocked state and output BLOCKED signal.
