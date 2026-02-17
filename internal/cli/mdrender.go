package cli

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Lipgloss styles for markdown rendering (vanilla terminal safe - ANSI 256 colors only)

type StyleSet struct {
        Name string
        H2 lipgloss.Style
        H3 lipgloss.Style
		BlockquoteBar lipgloss.Style
		BlockquoteText lipgloss.Style
		BoldStyle lipgloss.Style
		CodeBlockStyle lipgloss.Style
		InlineCode lipgloss.Style
		CommentStyle lipgloss.Style
		TextStyle lipgloss.Style
}
var ANSI256 = StyleSet{
        H2:             lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14")).MarginBottom(1), // Cyan bold
        H3:             lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11")),                // Yellow bold
        BlockquoteBar:  lipgloss.NewStyle().Foreground(lipgloss.Color("8")),                            // Gray bar
        BlockquoteText: lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Italic(true),               // Light gray italic
        BoldStyle:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")),                // Bright white bold
        CodeBlockStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("10")),                           // Green (like terminal output)
        InlineCode:     lipgloss.NewStyle().Foreground(lipgloss.Color("12")),                           // Blue
        CommentStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("8")),                            // Gray for # comments
        TextStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("7")),                            // Default light gray
}
var Harlequin = StyleSet{
        H2:             lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFBA08")).MarginBottom(1),
        H3:             lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF006E")),
        BlockquoteBar:  lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
        BlockquoteText: lipgloss.NewStyle().Foreground(lipgloss.Color("#E6EDF3")).Italic(true),
        BoldStyle:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#D7263D")),
        CodeBlockStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("10")),
        InlineCode:     lipgloss.NewStyle().Foreground(lipgloss.Color("12")),
        CommentStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("#D7263D")),
        TextStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("#F8F9FA")),
}
// TBD: check for current terminal support, and fallback to ANSI256 if these HEXCodes are not supported.
var ChosenStyle = Harlequin


var (
        mdH2Style        = ChosenStyle.H2
        mdH3Style        = ChosenStyle.H3
        mdBlockquoteBar  = ChosenStyle.BlockquoteBar
        mdBlockquoteText = ChosenStyle.BlockquoteText
        mdBoldStyle      = ChosenStyle.BoldStyle
        mdCodeBlockStyle = ChosenStyle.CodeBlockStyle
        mdInlineCode     = ChosenStyle.InlineCode
        mdCommentStyle   = ChosenStyle.CommentStyle
        mdTextStyle      = ChosenStyle.TextStyle
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
