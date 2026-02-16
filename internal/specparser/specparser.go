// Package specparser extracts ball definitions from spec.md and PRD.md files.
//
// The parser expects a markdown format where each H2 (##) section defines a
// ball/task. The heading text becomes the ball title, paragraph text beneath
// becomes context, and bullet/numbered/checkbox lists become acceptance criteria.
//
// Optional inline tags in the heading (e.g., [high], [small]) control priority
// and model size. Priority tags: [low], [medium], [high], [urgent].
// Model size tags: [small], [medium], [large].
//
// Example spec.md:
//
//	# My Project Spec
//
//	## Add user authentication [high]
//
//	Users need to be able to log in with email and password.
//
//	- Support email/password login
//	- Add password reset flow
//	- Rate limit login attempts
//
//	## Refactor database layer [medium] [small]
//
//	The current DB layer is tightly coupled to PostgreSQL.
//
//	1. Abstract database interface
//	2. Add connection pooling
//	3. Write migration tooling
package specparser

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// ParsedBall represents a ball extracted from a spec/PRD markdown file.
type ParsedBall struct {
	Title              string
	Context            string
	AcceptanceCriteria []string
	Priority           string // "low", "medium", "high", "urgent", or "" for default
	ModelSize          string // "small", "medium", "large", or "" for default
	Tags               []string
	SourceFile         string // Which file this was parsed from
}

// tagPattern matches bracketed tags in headings like [high], [small], etc.
var tagPattern = regexp.MustCompile(`\[([a-zA-Z]+)\]`)

// List item patterns
var (
	bulletPattern    = regexp.MustCompile(`^\s*[-*]\s+(.+)$`)
	numberedPattern  = regexp.MustCompile(`^\s*\d+\.\s+(.+)$`)
	checkboxPattern  = regexp.MustCompile(`^\s*-\s*\[[xX ]\]\s+(.+)$`)
)

// Known tag sets for classification
var priorityTags = map[string]bool{
	"low": true, "medium": true, "high": true, "urgent": true,
}

var modelSizeTags = map[string]bool{
	"small": true, "large": true,
	// "medium" is ambiguous between priority and model size;
	// we treat it as priority unless explicitly prefixed.
	// Users can use [model:medium] for model size disambiguation.
}

// ParseFile reads a markdown file and extracts ball definitions from H2 sections.
func ParseFile(path string) ([]ParsedBall, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	return parseMarkdown(scanner, path)
}

// ParseString parses markdown content from a string (useful for testing).
func ParseString(content, sourceName string) ([]ParsedBall, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	return parseMarkdown(scanner, sourceName)
}

// parseMarkdown does the actual parsing work from a scanner.
func parseMarkdown(scanner *bufio.Scanner, sourceName string) ([]ParsedBall, error) {
	var balls []ParsedBall
	var current *ParsedBall
	var contextLines []string
	inSection := false

	for scanner.Scan() {
		line := scanner.Text()

		// Check for H2 heading
		if strings.HasPrefix(line, "## ") {
			// Flush previous section
			if current != nil {
				current.Context = strings.TrimSpace(strings.Join(contextLines, "\n"))
				balls = append(balls, *current)
			}

			// Start new section
			heading := strings.TrimPrefix(line, "## ")
			current = parseHeading(heading, sourceName)
			contextLines = nil
			inSection = true
			continue
		}

		// Check for H1 or H3+ heading — these end the current H2 section
		// but don't start a new ball. H1 is typically the document title.
		if strings.HasPrefix(line, "# ") && !strings.HasPrefix(line, "## ") {
			if current != nil {
				current.Context = strings.TrimSpace(strings.Join(contextLines, "\n"))
				balls = append(balls, *current)
				current = nil
				contextLines = nil
			}
			inSection = false
			continue
		}

		if !inSection || current == nil {
			continue
		}

		// Try to match list items as acceptance criteria
		if criterion := extractListItem(line); criterion != "" {
			current.AcceptanceCriteria = append(current.AcceptanceCriteria, criterion)
			continue
		}

		// Non-list, non-empty lines are context
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			contextLines = append(contextLines, trimmed)
		}
	}

	// Flush last section
	if current != nil {
		current.Context = strings.TrimSpace(strings.Join(contextLines, "\n"))
		balls = append(balls, *current)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", sourceName, err)
	}

	return balls, nil
}

// parseHeading extracts title, priority, model size, and extra tags from an H2 heading.
func parseHeading(heading, sourceName string) *ParsedBall {
	ball := &ParsedBall{
		SourceFile: sourceName,
	}

	// Extract all bracketed tags
	matches := tagPattern.FindAllStringSubmatch(heading, -1)
	var extraTags []string

	for _, match := range matches {
		tag := strings.ToLower(match[1])

		if priorityTags[tag] {
			ball.Priority = tag
		} else if modelSizeTags[tag] {
			ball.ModelSize = tag
		} else if tag == "medium" {
			// Ambiguous: default to priority
			ball.Priority = tag
		} else {
			// Unknown tags become ball tags
			extraTags = append(extraTags, tag)
		}
	}

	// Remove all tags from the title
	title := tagPattern.ReplaceAllString(heading, "")
	title = strings.TrimSpace(title)
	ball.Title = title
	ball.Tags = extraTags

	return ball
}

// extractListItem tries to extract a list item from a line.
// Returns the item text if matched, empty string otherwise.
// Checkbox items are checked first (they're a subset of bullet syntax).
func extractListItem(line string) string {
	// Checkbox has highest priority (it's a special form of bullet)
	if m := checkboxPattern.FindStringSubmatch(line); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	if m := numberedPattern.FindStringSubmatch(line); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	if m := bulletPattern.FindStringSubmatch(line); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// FindSpecFiles looks for spec.md and PRD.md (case-insensitive) in the given directory.
// Returns the paths of files found.
func FindSpecFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	var found []string
	targetNames := map[string]bool{
		"spec.md": true,
		"prd.md":  true,
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		nameLower := strings.ToLower(entry.Name())
		if targetNames[nameLower] {
			found = append(found, entry.Name())
		}
	}

	return found, nil
}

// ParseDirectory finds and parses all spec.md and PRD.md files in a directory.
// Returns all extracted balls across all files found.
func ParseDirectory(dir string) ([]ParsedBall, error) {
	files, err := FindSpecFiles(dir)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no spec.md or PRD.md files found in %s", dir)
	}

	var allBalls []ParsedBall
	for _, file := range files {
		path := dir + "/" + file
		balls, err := ParseFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", file, err)
		}
		allBalls = append(allBalls, balls...)
	}

	return allBalls, nil
}
