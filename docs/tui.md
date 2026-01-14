# Juggle TUI (Terminal User Interface)

## Overview

The Juggle TUI provides an interactive, full-screen terminal interface for managing balls (work sessions) across all your projects. It's built with the [Charm Bubbletea](https://github.com/charmbracelet/bubbletea) framework and offers a more visual, interactive experience compared to the CLI commands.

**TUI Main View** - Sessions on the left, balls (tasks) on the right, activity log at bottom:

![TUI Main Menu](assets/tui-main-menu.png)

**Ball Creation** - Define context, acceptance criteria, priority, dependencies:

![Create New Ball](assets/tui-create-new-ball.png)

## Features

### Current Implementation (MVP)

- **Ball List View**: See all balls across all projects with color-coded states
- **Ball Detail View**: View full ball information including todos, tags, and timestamps
- **Quick Actions**: Perform common operations with single keystrokes
- **State Filtering**: Filter balls by state (all/pending/in_progress/blocked)
- **Real-time Updates**: Refresh ball data on demand
- **Help View**: Built-in keyboard reference

## Usage

### Starting the TUI

```bash
# Launch TUI (all projects)
juggle tui

# Launch TUI (current project only)
juggle --local tui

# See help
juggle tui --help
```

### Workflow Example

1. Launch TUI: `juggle tui`
2. Use `↑`/`↓` to navigate balls
3. Press `tp` to toggle pending ball visibility
4. Press `ss` to start the selected ball
5. Press `Enter` to see ball details
6. Press `b` to go back to list
7. Press `s` then a state key to change state
8. Press `x` to delete a ball (with confirmation)
9. Press `p` to cycle priority levels
10. Press `q` to quit

### Quick Actions

The TUI supports several quick actions that work from the list view:

- **Start Ball (ss)**: Changes pending ball to in_progress
  - Only works on pending balls
  - Updates state immediately

- **Complete Ball (sc)**: Marks in_progress ball as complete
  - Only works on in_progress balls
  - Archives the ball

- **Block Ball (sb)**: Marks ball as blocked
  - Prompts for a reason
  - Works on pending or in_progress balls

- **Set Pending (sp)**: Changes ball to pending state
  - Works from any state
  - Useful for resetting balls

- **Archive (sa)**: Archives a completed ball
  - Only works on completed balls
  - Moves to archive

- **Delete Ball (x)**: Permanently deletes a ball
  - Shows confirmation dialog with ball details
  - Press `y` to confirm, `n` or `Esc` to cancel
  - Safe deletion with explicit confirmation

- **Cycle Priority (p)**: Changes ball priority
  - Order: low → medium → high → urgent → low
  - Works from any state
  - Updates immediately

- **Refresh (R)**: Reloads all balls from disk
  - Use shift+r
  - Shows "Reloading balls..." message
  - Updates after external changes

### Filtering

Use two-key sequences with `t` to toggle filter visibility by state:

- `ta` - Show all states (disables all filters)
- `tp` - Toggle pending ball visibility
- `ti` - Toggle in_progress ball visibility
- `tb` - Toggle blocked ball visibility
- `tc` - Toggle complete ball visibility

**Filter Behavior:**

- Filters are toggleable, not exclusive
- Multiple states can be visible simultaneously
- Example: Press `tp` then `ti` to see both pending and in_progress balls
- Press `ta` to reset all filters and show everything
- Filter state persists during TUI session
- Current filters shown in stats bar

The current filter is shown in the stats bar.

## Architecture

### Directory Structure

```
internal/tui/
├── model.go      # Main TUI model (bubbletea Model interface)
├── view.go       # Rendering logic for all views
├── update.go     # Event handling and state transitions
├── list.go       # Ball list rendering
├── detail.go     # Ball detail rendering
├── commands.go   # Bubbletea commands for async operations
├── styles.go     # Lipgloss styles and colors
└── tui_test.go   # Unit tests
```

### Key Components

**Model** (`model.go`):

- Holds application state (balls, current view, filters, cursor position)
- Implements `tea.Model` interface
- Manages navigation between views

**Update** (`update.go`):

- Handles keyboard events
- Manages state transitions
- Coordinates ball updates via Store

**View** (`view.go`):

- Renders current view based on mode
- Delegates to specialized renderers (list, detail, help)
- Shows messages and errors

**Commands** (`commands.go`):

- Async operations using bubbletea Cmd
- Load balls from all projects
- Update ball state in store

### State Management

The TUI maintains several state variables:

- `mode`: Current view (listView/detailView/helpView)
- `balls`: All loaded balls
- `filteredBalls`: Balls matching current filter
- `cursor`: Current selection in list
- `filterState`: Current filter ("all", "pending", "in_progress", "blocked")
- `message`: Success/error messages shown to user

### Ball Updates

When updating a ball (start/complete/drop):

1. Get ball from current cursor position
2. Validate state transition is allowed
3. Create Store for ball's working directory
4. Update ball state
5. Call Store.UpdateBall()
6. Reload all balls to refresh display
7. Show success/error message

## Testing

The TUI has comprehensive unit tests:

```bash
# Run TUI tests only
go test -v ./internal/tui/...

# Run all tests
devbox run test-all
```

Test coverage includes:

- Model initialization
- String truncation
- State formatting
- Ball counting
- Filter application
- View rendering (structure)

## Troubleshooting

### TUI Won't Launch

**Error**: "could not open a new TTY"

- **Cause**: Not running in a proper terminal
- **Solution**: Ensure you're in an interactive terminal, not a pipe or background process

### Colors Not Showing

**Issue**: Ball states not color-coded

- **Cause**: Terminal doesn't support colors
- **Solution**: Use a modern terminal emulator (iTerm2, Alacritty, etc.)

### Balls Not Loading

**Issue**: "No balls to display" when balls exist

- **Cause**: Discovery or loading error
- **Solution**: Check `~/.juggle/config.json` search paths are correct

### Updates Not Persisting

**Issue**: State changes don't save

- **Cause**: Store update failing
- **Solution**: Check `.juggle/` directory is writable

## References

- [Bubbletea Documentation](https://github.com/charmbracelet/bubbletea)
- [Lipgloss Styling](https://github.com/charmbracelet/lipgloss)
- [Bubbles Components](https://github.com/charmbracelet/bubbles)
