package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handleAgentMonitorKey handles keyboard input in agent monitor view
func (m Model) handleAgentMonitorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Return to split view (agent keeps running)
		m.mode = splitView
		return m, nil

	case "q":
		// Detach from monitor - agent keeps running as daemon
		m.mode = splitView
		m.message = "Agent continues running in background"
		return m, nil

	case "X":
		// Cancel agent with confirmation
		if m.agentStatus.Running {
			return m.handleMonitorCancelAgent()
		}
		return m, nil

	case "p":
		// Toggle pause-on-next-iteration
		if m.agentStatus.Running {
			return m.handleMonitorPause()
		}
		return m, nil

	case "r":
		// Resume from pause
		if m.agentMonitorPaused {
			return m.handleMonitorResume()
		}
		return m, nil

	case "m":
		// Change model (would show model selector - for now just log)
		if m.agentStatus.Running {
			m.message = "Model change: use --model flag when starting agent"
		}
		return m, nil

	case "n":
		// Skip to next ball
		if m.agentStatus.Running {
			return m.handleMonitorSkipBall()
		}
		return m, nil

	case "O":
		// Toggle output panel expansion
		m.agentOutputExpanded = !m.agentOutputExpanded
		return m, nil

	// Scroll controls for output
	case "j", "down":
		return m.handleAgentOutputScrollDown()
	case "k", "up":
		return m.handleAgentOutputScrollUp()
	case "ctrl+d":
		return m.handleAgentOutputPageDown()
	case "ctrl+u":
		return m.handleAgentOutputPageUp()
	case "g":
		if m.lastKey == "g" {
			m.lastKey = ""
			return m.handleAgentOutputGoToTop()
		}
		m.lastKey = "g"
		return m, nil
	case "G":
		return m.handleAgentOutputGoToBottom()
	}

	return m, nil
}

// handleMonitorPause sends a pause command to the daemon
func (m Model) handleMonitorPause() (tea.Model, tea.Cmd) {
	m.agentMonitorPaused = true
	m.message = "Pausing after current iteration..."
	return m, sendDaemonControlCmd(m.store.ProjectDir(), m.agentStatus.SessionID, "pause", "")
}

// handleMonitorResume sends a resume command to the daemon
func (m Model) handleMonitorResume() (tea.Model, tea.Cmd) {
	m.agentMonitorPaused = false
	m.message = "Resuming..."
	return m, sendDaemonControlCmd(m.store.ProjectDir(), m.agentStatus.SessionID, "resume", "")
}

// handleMonitorSkipBall sends a skip_ball command to the daemon
func (m Model) handleMonitorSkipBall() (tea.Model, tea.Cmd) {
	m.message = "Skipping current ball..."
	return m, sendDaemonControlCmd(m.store.ProjectDir(), m.agentStatus.SessionID, "skip_ball", "")
}

// handleMonitorCancelAgent sends a cancel command to the daemon
func (m Model) handleMonitorCancelAgent() (tea.Model, tea.Cmd) {
	// Use existing agent cancel confirmation flow
	m.mode = confirmAgentCancel
	return m, nil
}
