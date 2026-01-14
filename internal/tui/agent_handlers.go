package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handleCancelAgent shows confirmation dialog for cancelling a running agent
func (m Model) handleCancelAgent() (tea.Model, tea.Cmd) {
	// Check if agent is running
	if !m.agentStatus.Running {
		m.message = "No agent is running"
		return m, nil
	}

	// Show confirmation dialog
	m.mode = confirmAgentCancel
	return m, nil
}

// handleAgentCancelConfirm handles the agent cancel confirmation
func (m Model) handleAgentCancelConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		// Confirm cancellation
		m.mode = splitView
		m.addActivity("Cancelling agent...")
		m.message = "Cancelling agent..."

		// Kill the process if we have a reference
		if m.agentProcess != nil {
			if err := m.agentProcess.Kill(); err != nil {
				m.addActivity("Error killing agent: " + err.Error())
				m.message = "Error killing agent: " + err.Error()
			} else {
				m.addActivity("Agent process terminated")
				m.addAgentOutput("=== Agent cancelled by user ===", true)
			}
		}

		// Clear agent status
		m.agentStatus.Running = false
		m.agentProcess = nil
		m.message = "Agent cancelled"

		// Reload balls to reflect any changes made before cancellation
		return m, loadBalls(m.store, m.config, m.localOnly)

	case "n", "N", "esc", "q":
		// Don't cancel
		m.mode = splitView
		m.message = "Agent still running"
		return m, nil
	}

	return m, nil
}
