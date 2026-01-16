# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Test Commands

### Building

```bash
# Enter devbox shell (sets up Go environment)
devbox shell

# Build the binary
go build -o juggle ./cmd/juggle

# Install locally for testing
go install ./cmd/juggle
```

### Testing

```bash
# Run integration tests (quiet - shows pass/fail summary only)
devbox run test-quiet

# Run integration tests (verbose - full output)
devbox run test-verbose
# or: go test -v ./internal/integration_test/...

# Run all tests
devbox run test-all
# or: go test -v ./...

# Generate coverage report
devbox run test-coverage
# or: go test -v -coverprofile=coverage.out ./internal/integration_test/...
#     go tool cover -html=coverage.out -o coverage.html

# Run single test
go test -v ./internal/integration_test/... -run TestExport
```

### Development

```bash
# Clean build artifacts
go clean

# Update dependencies
go mod tidy

# Check formatting
go fmt ./...
```

## Architecture Documentation

**Juggle** runs autonomous AI agent loops with good UX. Define tasks ("balls") with acceptance criteria via TUI or CLI, start the agent loop (`juggle agent run`), and add or modify tasks while it runs. No JSON editing - the TUI and CLI handle all task management.

For detailed architecture information, read these files as needed:

### Package Structure
- [Package Directory Structure](docs/arch/packages.md) - Complete directory tree with file descriptions

### Data Flows
- [Ball Lifecycle](docs/flows/ball-lifecycle.md) - How balls are created, activated, and completed
- [Agent Loop](docs/flows/agent-loop.md) - How the autonomous agent executes and signals completion
- [VCS Integration](docs/flows/vcs-integration.md) - How juggler tracks revisions in jj/git
- [Cross-Project Discovery](docs/flows/cross-project-discovery.md) - How --all flag finds balls across projects
- [Live TUI Updates](docs/flows/tui-updates.md) - How the TUI reacts to file changes in real-time

### Common Patterns (How to Add Features)
- [Adding a CLI Command](docs/patterns/add-cli-command.md)
- [Adding a Ball Field](docs/patterns/add-ball-field.md)
- [Adding a VCS Operation](docs/patterns/add-vcs-operation.md)
- [Adding an Agent Provider](docs/patterns/add-agent-provider.md)

### File Cross-References
- [Ball Lifecycle Files](docs/refs/ball-lifecycle.md)
- [CLI Command Files](docs/refs/cli-commands.md)
- [TUI Files](docs/refs/tui.md)
- [Agent System Files](docs/refs/agent-system.md)
- [VCS Integration Files](docs/refs/vcs.md)
- [Storage Files](docs/refs/storage.md)

**When working on a specific task, read only the relevant documentation files above.**

## Multi-Agent Support

When multiple agents/users work simultaneously, set `JUGGLER_CURRENT_BALL` environment variable to explicitly target a ball:

```bash
export JUGGLER_CURRENT_BALL="juggle-5"
```

This ensures operations go to the correct ball when:

- Multiple AI agents work in same repo
- Multiple terminal sessions are active
- You want explicit control over which ball is targeted

## Code Style Notes

- Use `lipgloss` for terminal styling (colors, formatting)
- Commands return `error`, not `fmt.Errorf()` directly - wrap with context
- JSONL append-only writes for better version control diffs
- Ball IDs format: `<directory-name>-<counter>` (e.g., `juggle-5`)
