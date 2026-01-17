package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Monitor view styles
var (
	monitorTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("3")). // Yellow
				Padding(0, 1)

	monitorProgressBarFilled = lipgloss.NewStyle().
					Foreground(lipgloss.Color("2")). // Green
					Render("█")

	monitorProgressBarEmpty = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")). // Gray
				Render("░")

	monitorMetricLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Width(15)

	monitorMetricValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("6")) // Cyan

	monitorControlsStyle = lipgloss.NewStyle().
				Faint(true)

	monitorSeparatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))
)

// renderAgentMonitorView renders the full-screen agent monitoring dashboard
func (m Model) renderAgentMonitorView() string {
	// Guard against rendering before window size is received
	if m.width < 40 || m.height < 15 {
		return "Loading..."
	}

	var b strings.Builder

	// Title bar with session info
	b.WriteString(m.renderMonitorTitleBar())
	b.WriteString("\n")

	// Progress bar
	b.WriteString(m.renderMonitorProgressBar())
	b.WriteString("\n")

	// Calculate output section height
	outputHeight := m.height - 12 // title(2) + progress(2) + metrics(4) + controls(2) + margins
	if outputHeight < 5 {
		outputHeight = 5
	}

	// Output section (reuses existing agent output panel rendering)
	b.WriteString(m.renderMonitorOutputSection(outputHeight))

	// Separator
	b.WriteString(monitorSeparatorStyle.Render(strings.Repeat("─", m.width)))
	b.WriteString("\n")

	// Metrics panel
	b.WriteString(m.renderMonitorMetricsPanel())

	// Controls panel
	b.WriteString(m.renderMonitorControlsPanel())

	return b.String()
}

// renderMonitorTitleBar renders the top title bar with status
func (m Model) renderMonitorTitleBar() string {
	var status string
	statusColor := lipgloss.Color("2") // Green

	if m.agentMonitorPaused {
		status = "Pausing..."
		statusColor = lipgloss.Color("3") // Yellow
	} else if !m.agentStatus.Running {
		status = "Stopped"
		statusColor = lipgloss.Color("1") // Red
	} else {
		// Running with spinner animation
		status = m.agentSpinner.View() + " Running"
	}

	statusStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(statusColor)

	title := fmt.Sprintf("Agent Monitor: %s [%s] - Iteration %d/%d",
		m.agentStatus.SessionID,
		statusStyle.Render(status),
		m.agentStatus.Iteration,
		m.agentStatus.MaxIterations)

	if m.agentMonitorReconnected {
		title += lipgloss.NewStyle().Faint(true).Render(" (reconnected)")
	}

	return monitorTitleStyle.Render(title)
}

// renderMonitorProgressBar renders the iteration progress bar
func (m Model) renderMonitorProgressBar() string {
	width := m.width - 12 // Leave room for percentage
	if width < 20 {
		width = 20
	}

	var progress float64
	if m.agentStatus.MaxIterations > 0 {
		progress = float64(m.agentStatus.Iteration) / float64(m.agentStatus.MaxIterations)
	}

	filled := int(float64(width) * progress)
	if filled > width {
		filled = width
	}
	empty := width - filled

	percentage := fmt.Sprintf(" %3.0f%%", progress*100)

	filledStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	return "  " + filledStyle.Render(strings.Repeat("█", filled)) +
		emptyStyle.Render(strings.Repeat("░", empty)) +
		percentage + "\n"
}

// renderMonitorOutputSection renders the scrollable output section
func (m Model) renderMonitorOutputSection(height int) string {
	var b strings.Builder

	// Section title with scroll position
	title := "Output"
	if len(m.agentOutput) > 0 {
		title = fmt.Sprintf("Output [%d/%d]", m.agentOutputOffset+1, len(m.agentOutput))
	}
	titleStyled := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6")).
		Render(title)
	b.WriteString("  " + titleStyled + "\n")
	b.WriteString("  " + monitorSeparatorStyle.Render(strings.Repeat("─", m.width-4)) + "\n")

	if len(m.agentOutput) == 0 {
		emptyMsg := "  No agent output"
		if !m.agentStatus.Running {
			emptyMsg += " - Press Esc to return"
		}
		b.WriteString(helpStyle.Render(emptyMsg) + "\n")
		// Pad remaining height
		for i := 0; i < height-3; i++ {
			b.WriteString("\n")
		}
		return b.String()
	}

	// Calculate visible range
	visibleLines := height - 2 // Account for title and separator
	if visibleLines < 1 {
		visibleLines = 1
	}

	startIdx := m.agentOutputOffset
	if startIdx < 0 {
		startIdx = 0
	}
	endIdx := startIdx + visibleLines
	if endIdx > len(m.agentOutput) {
		endIdx = len(m.agentOutput)
	}

	// Check if we need scroll indicators
	needTopIndicator := startIdx > 0
	needBottomIndicator := endIdx < len(m.agentOutput)

	// Adjust visible lines for scroll indicators
	if needTopIndicator {
		visibleLines--
		b.WriteString(helpStyle.Render("  ▲ more above") + "\n")
	}
	if needBottomIndicator {
		visibleLines--
	}

	// Recalculate end index after indicator adjustment
	endIdx = startIdx + visibleLines
	if endIdx > len(m.agentOutput) {
		endIdx = len(m.agentOutput)
	}

	// Render visible lines
	for i := startIdx; i < endIdx; i++ {
		entry := m.agentOutput[i]
		line := entry.Line
		// Truncate long lines
		if len(line) > m.width-4 {
			line = line[:m.width-7] + "..."
		}
		if entry.IsError {
			b.WriteString("  " + errorStyle.Render(line) + "\n")
		} else {
			b.WriteString("  " + line + "\n")
		}
	}

	// Pad if we have fewer lines than space
	linesRendered := endIdx - startIdx
	if needTopIndicator {
		linesRendered++
	}
	for i := linesRendered; i < height-2; i++ {
		b.WriteString("\n")
	}

	if needBottomIndicator {
		b.WriteString(helpStyle.Render("  ▼ more below") + "\n")
	}

	return b.String()
}

// renderMonitorMetricsPanel renders the metrics section
func (m Model) renderMonitorMetricsPanel() string {
	var b strings.Builder

	// Get current ball info from agent status or output
	ballID := "—"
	ballTitle := "—"
	elapsed := "—"
	model := "—"

	if m.agentStatus.Running || m.agentStatus.SessionID != "" {
		ballID = m.agentStatus.SessionID
	}

	if !m.agentMonitorStartTime.IsZero() {
		elapsed = formatDuration(time.Since(m.agentMonitorStartTime))
	}

	// Row 1
	b.WriteString(fmt.Sprintf("  %s %s  %s\n",
		monitorMetricLabelStyle.Render("Session:"),
		monitorMetricValueStyle.Render(ballID),
		monitorMetricValueStyle.Render(ballTitle)))

	// Row 2
	b.WriteString(fmt.Sprintf("  %s %s    %s %s\n",
		monitorMetricLabelStyle.Render("Elapsed:"),
		monitorMetricValueStyle.Render(elapsed),
		monitorMetricLabelStyle.Render("Model:"),
		monitorMetricValueStyle.Render(model)))

	return b.String()
}

// renderMonitorControlsPanel renders the controls help line
func (m Model) renderMonitorControlsPanel() string {
	var controls []string

	if m.agentStatus.Running {
		if m.agentMonitorPaused {
			controls = append(controls, "r:Resume")
		} else {
			controls = append(controls, "p:Pause")
		}
		controls = append(controls,
			"m:Model",
			"n:Skip ball",
			"X:Cancel",
		)
	}

	controls = append(controls,
		"O:Expand",
		"Esc:Back",
		"q:Detach",
	)

	return "\n  " + monitorControlsStyle.Render(strings.Join(controls, " | "))
}
