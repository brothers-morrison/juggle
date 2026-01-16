# Commit Message Format

Use conventional commits format: `type(scope): ball-id - brief summary`

## Types (use the most specific)

| Type | Use When |
|------|----------|
| `feat` | Adding new functionality or features |
| `fix` | Fixing a bug or correcting behavior |
| `docs` | Documentation-only changes |
| `style` | Formatting, whitespace (no code logic change) |
| `refactor` | Code restructuring without changing behavior |
| `perf` | Performance improvements |
| `test` | Adding or modifying tests |
| `build` | Build system or dependency changes |
| `ci` | CI/CD configuration changes |
| `chore` | Maintenance tasks, tooling updates |
| `revert` | Reverting a previous commit |

## Scope

Optional component/feature name in parentheses (e.g., `tui`, `agent`, `vcs`, `cli`).

## Examples

- `feat(tui): juggle-42 - Add sort toggle for ball list`
- `fix(agent): juggle-17 - Resolve timeout in preflight check`
- `docs: juggle-8 - Update CLI command reference`
- `refactor(vcs): juggle-55 - Extract commit logic to interface`
- `test: juggle-23 - Add integration tests for export`
