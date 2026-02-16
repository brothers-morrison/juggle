package cli

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Lipgloss styles for markdown rendering (vanilla terminal safe - ANSI 256 colors only)
var (
	mdH2Style        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14")).MarginBottom(1) // Cyan bold
	mdH3Style        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))                // Yellow bold
	mdBlockquoteBar  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))                            // Gray bar
	mdBlockquoteText = lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Italic(true)                // Light gray italic
	mdBoldStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))                // Bright white bold
	mdCodeBlockStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))                           // Green (like terminal output)
	mdInlineCode     = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))                           // Blue
	mdCommentStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))                            // Gray for # comments
	mdTextStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))                            // Default light gray
)

// RenderMarkdown renders a markdown string with lipgloss styling for terminal output.
// Handles the subset of markdown used in quickstart.md: headers, blockquotes,
// fenced code blocks, inline bold, and inline code.
func RenderMarkdown(md string) string {
	lines := strings.Split(md, "\n")
	var out []string
	inCodeBlock := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Fenced code blocks
		if strings.HasPrefix(line, "```") {
			inCodeBlock = !inCodeBlock
			if inCodeBlock {
				// Opening fence — skip the line itself (```bash etc)
				continue
			}
			// Closing fence — skip
			continue
		}
		if inCodeBlock {
			out = append(out, renderCodeLine(line))
			continue
		}

		// Blank lines
		if strings.TrimSpace(line) == "" {
			out = append(out, "")
			continue
		}

		// Headers
		if strings.HasPrefix(line, "### ") {
			text := strings.TrimPrefix(line, "### ")
			out = append(out, mdH3Style.Render(text))
			continue
		}
		if strings.HasPrefix(line, "## ") {
			text := strings.TrimPrefix(line, "## ")
			out = append(out, mdH2Style.Render(text))
			continue
		}

		// Blockquotes
		if strings.HasPrefix(line, "> ") {
			text := strings.TrimPrefix(line, "> ")
			text = renderInline(text)
			out = append(out, mdBlockquoteBar.Render("│ ")+mdBlockquoteText.Render(text))
			continue
		}

		// Regular text with inline formatting
		out = append(out, renderInline(line))
	}

	return strings.Join(out, "\n")
}

// renderCodeLine styles a single line inside a fenced code block.
// Shell comments (lines starting with #) get dimmed.
func renderCodeLine(line string) string {
	trimmed := strings.TrimSpace(line)
	prefix := "  "

	if strings.HasPrefix(trimmed, "#") {
		return prefix + mdCommentStyle.Render(line)
	}
	return prefix + mdCodeBlockStyle.Render(line)
}

// renderInline applies inline formatting: **bold** and `code`.
func renderInline(s string) string {
	var result strings.Builder
	i := 0
	for i < len(s) {
		// Bold: **text**
		if i+1 < len(s) && s[i] == '*' && s[i+1] == '*' {
			end := strings.Index(s[i+2:], "**")
			if end >= 0 {
				inner := s[i+2 : i+2+end]
				result.WriteString(mdBoldStyle.Render(inner))
				i = i + 2 + end + 2
				continue
			}
		}
		// Inline code: `text`
		if s[i] == '`' {
			end := strings.Index(s[i+1:], "`")
			if end >= 0 {
				inner := s[i+1 : i+1+end]
				result.WriteString(mdInlineCode.Render(inner))
				i = i + 1 + end + 1
				continue
			}
		}
		// Links: [text](url) — render as text only
		if s[i] == '[' {
			closeBracket := strings.Index(s[i:], "](")
			if closeBracket >= 0 {
				closeParen := strings.Index(s[i+closeBracket:], ")")
				if closeParen >= 0 {
					linkText := s[i+1 : i+closeBracket]
					result.WriteString(mdInlineCode.Render(linkText))
					i = i + closeBracket + closeParen + 1
					continue
				}
			}
		}
		result.WriteByte(s[i])
		i++
	}
	return result.String()
}
