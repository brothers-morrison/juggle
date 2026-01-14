package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handleLaunchAgent shows confirmation dialog for launching an agent
func (m Model) handleLaunchAgent() (tea.Model, tea.Cmd) {
	// Check if agent is already running
	if m.agentStatus.Running {
		m.message = "Agent already running for: " + m.agentStatus.SessionID
		return m, nil
	}

	// Check if session is selected
	if m.selectedSession == nil {
		m.message = "No session selected"
		return m, nil
	}

	// Prevent launching on untagged pseudo-session (but allow "All" which maps to meta-session "all")
	if m.selectedSession.ID == PseudoSessionUntagged {
		m.message = "Cannot launch agent on untagged session"
		return m, nil
	}

	// Show confirmation dialog
	m.mode = confirmAgentLaunch
	return m, nil
}

// handleAgentLaunchConfirm handles the agent launch confirmation
func (m Model) handleAgentLaunchConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		// Confirm launch
		sessionID := m.selectedSession.ID
		// Map PseudoSessionAll to "all" meta-session for agent command
		if sessionID == PseudoSessionAll {
			sessionID = "all"
		}
		m.mode = splitView
		m.addActivity("Launching agent for: " + sessionID)
		m.message = "Starting agent for: " + sessionID

		// Clear previous output and create new output channel
		m.clearAgentOutput()
		m.agentOutputCh = make(chan agentOutputMsg, 100)
		m.agentOutputVisible = true // Auto-show agent output when launching
		m.addAgentOutput("=== Starting agent for session: "+sessionID+" ===", false)

		// Set initial agent status
		m.agentStatus = AgentStatus{
			Running:       true,
			SessionID:     sessionID,
			Iteration:     0,
			MaxIterations: 10, // Default iterations
		}

		// Launch agent in background with output streaming
		return m, tea.Batch(
			launchAgentWithOutputCmd(sessionID, m.agentOutputCh),
			listenForAgentOutput(m.agentOutputCh),
		)

	case "n", "N", "esc", "q":
		// Cancel
		m.mode = splitView
		m.message = "Agent launch cancelled"
		return m, nil
	}

	return m, nil
}

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
