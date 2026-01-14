package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handleShowHistory loads and displays agent run history
func (m Model) handleShowHistory() (tea.Model, tea.Cmd) {
	m.addActivity("Loading agent history...")
	m.message = "Loading history..."
	return m, loadAgentHistory(m.store.ProjectDir())
}

// handleHistoryViewKey handles keyboard input in history view
func (m Model) handleHistoryViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "H":
		// Return to split view
		m.mode = splitView
		m.message = ""
		return m, nil

	case "up", "k":
		// Move cursor up
		if m.historyCursor > 0 {
			m.historyCursor--
			// Adjust scroll if needed
			if m.historyCursor < m.historyScrollOffset {
				m.historyScrollOffset = m.historyCursor
			}
		}
		return m, nil

	case "down", "j":
		// Move cursor down
		if m.historyCursor < len(m.agentHistory)-1 {
			m.historyCursor++
			// Adjust scroll if needed (assuming 15 visible lines)
			visibleLines := 15
			if m.historyCursor >= m.historyScrollOffset+visibleLines {
				m.historyScrollOffset = m.historyCursor - visibleLines + 1
			}
		}
		return m, nil

	case "ctrl+d":
		// Page down
		pageSize := 7
		m.historyCursor += pageSize
		if m.historyCursor >= len(m.agentHistory) {
			m.historyCursor = len(m.agentHistory) - 1
		}
		if m.historyCursor < 0 {
			m.historyCursor = 0
		}
		// Adjust scroll
		visibleLines := 15
		if m.historyCursor >= m.historyScrollOffset+visibleLines {
			m.historyScrollOffset = m.historyCursor - visibleLines + 1
		}
		return m, nil

	case "ctrl+u":
		// Page up
		pageSize := 7
		m.historyCursor -= pageSize
		if m.historyCursor < 0 {
			m.historyCursor = 0
		}
		// Adjust scroll
		if m.historyCursor < m.historyScrollOffset {
			m.historyScrollOffset = m.historyCursor
		}
		return m, nil

	case "g":
		// Handle gg for go to top
		if m.lastKey == "g" {
			m.lastKey = ""
			m.historyCursor = 0
			m.historyScrollOffset = 0
			return m, nil
		}
		m.lastKey = "g"
		return m, nil

	case "G":
		// Go to bottom
		m.lastKey = ""
		if len(m.agentHistory) > 0 {
			m.historyCursor = len(m.agentHistory) - 1
			// Adjust scroll to show cursor at bottom
			visibleLines := 15
			if m.historyCursor >= visibleLines {
				m.historyScrollOffset = m.historyCursor - visibleLines + 1
			}
		}
		return m, nil

	case "enter", " ":
		// View output file for selected record
		if len(m.agentHistory) > 0 && m.historyCursor < len(m.agentHistory) {
			record := m.agentHistory[m.historyCursor]
			m.addActivity("Loading output for run: " + record.ID)
			return m, loadHistoryOutput(record.OutputFile)
		}
		return m, nil
	}

	// Reset gg detection for any other key
	m.lastKey = ""
	return m, nil
}

// handleHistoryOutputViewKey handles keyboard input in history output view
func (m Model) handleHistoryOutputViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "b":
		// Return to history view
		m.mode = historyView
		m.historyOutput = ""
		return m, nil

	case "up", "k":
		// Scroll up
		if m.historyOutputOffset > 0 {
			m.historyOutputOffset--
		}
		return m, nil

	case "down", "j":
		// Scroll down
		m.historyOutputOffset++
		return m, nil

	case "ctrl+d":
		// Page down
		m.historyOutputOffset += 15
		return m, nil

	case "ctrl+u":
		// Page up
		m.historyOutputOffset -= 15
		if m.historyOutputOffset < 0 {
			m.historyOutputOffset = 0
		}
		return m, nil

	case "g":
		// Handle gg for go to top
		if m.lastKey == "g" {
			m.lastKey = ""
			m.historyOutputOffset = 0
			return m, nil
		}
		m.lastKey = "g"
		return m, nil

	case "G":
		// Go to bottom (set to large value, will be clamped in render)
		m.lastKey = ""
		m.historyOutputOffset = 10000
		return m, nil
	}

	// Reset gg detection for any other key
	m.lastKey = ""
	return m, nil
}
