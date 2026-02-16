package specparser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseString_BasicH2Sections(t *testing.T) {
	content := `# Project Spec

## Add user authentication

Users need to be able to log in.

- Support email/password login
- Add password reset flow
- Rate limit login attempts

## Refactor database layer

The DB layer is tightly coupled.

1. Abstract database interface
2. Add connection pooling
3. Write migration tooling
`

	balls, err := ParseString(content, "spec.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(balls) != 2 {
		t.Fatalf("expected 2 balls, got %d", len(balls))
	}

	// First ball
	b := balls[0]
	if b.Title != "Add user authentication" {
		t.Errorf("expected title 'Add user authentication', got %q", b.Title)
	}
	if b.Context != "Users need to be able to log in." {
		t.Errorf("expected context about logging in, got %q", b.Context)
	}
	if len(b.AcceptanceCriteria) != 3 {
		t.Errorf("expected 3 acceptance criteria, got %d", len(b.AcceptanceCriteria))
	}
	if b.AcceptanceCriteria[0] != "Support email/password login" {
		t.Errorf("expected first criterion 'Support email/password login', got %q", b.AcceptanceCriteria[0])
	}
	if b.SourceFile != "spec.md" {
		t.Errorf("expected source file 'spec.md', got %q", b.SourceFile)
	}

	// Second ball
	b = balls[1]
	if b.Title != "Refactor database layer" {
		t.Errorf("expected title 'Refactor database layer', got %q", b.Title)
	}
	if len(b.AcceptanceCriteria) != 3 {
		t.Errorf("expected 3 acceptance criteria, got %d", len(b.AcceptanceCriteria))
	}
	if b.AcceptanceCriteria[0] != "Abstract database interface" {
		t.Errorf("expected first criterion 'Abstract database interface', got %q", b.AcceptanceCriteria[0])
	}
}

func TestParseString_PriorityTags(t *testing.T) {
	content := `## Low priority task [low]

Some context.

- Criterion 1

## High priority task [high]

Important work.

- Criterion 1

## Urgent task [urgent]

Critical fix needed.

- Fix immediately
`

	balls, err := ParseString(content, "test.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(balls) != 3 {
		t.Fatalf("expected 3 balls, got %d", len(balls))
	}

	tests := []struct {
		title    string
		priority string
	}{
		{"Low priority task", "low"},
		{"High priority task", "high"},
		{"Urgent task", "urgent"},
	}

	for i, tt := range tests {
		if balls[i].Title != tt.title {
			t.Errorf("ball %d: expected title %q, got %q", i, tt.title, balls[i].Title)
		}
		if balls[i].Priority != tt.priority {
			t.Errorf("ball %d: expected priority %q, got %q", i, tt.priority, balls[i].Priority)
		}
	}
}

func TestParseString_ModelSizeTags(t *testing.T) {
	content := `## Simple docs task [low] [small]

Quick documentation update.

- Update README

## Complex architecture [high] [large]

Major refactoring effort.

- Redesign API layer
`

	balls, err := ParseString(content, "test.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(balls) != 2 {
		t.Fatalf("expected 2 balls, got %d", len(balls))
	}

	if balls[0].Priority != "low" {
		t.Errorf("expected priority 'low', got %q", balls[0].Priority)
	}
	if balls[0].ModelSize != "small" {
		t.Errorf("expected model size 'small', got %q", balls[0].ModelSize)
	}

	if balls[1].Priority != "high" {
		t.Errorf("expected priority 'high', got %q", balls[1].Priority)
	}
	if balls[1].ModelSize != "large" {
		t.Errorf("expected model size 'large', got %q", balls[1].ModelSize)
	}
}

func TestParseString_MediumTag(t *testing.T) {
	// [medium] is ambiguous - should default to priority
	content := `## Some task [medium]

Context here.

- Do the thing
`

	balls, err := ParseString(content, "test.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(balls) != 1 {
		t.Fatalf("expected 1 ball, got %d", len(balls))
	}

	if balls[0].Priority != "medium" {
		t.Errorf("expected priority 'medium', got %q", balls[0].Priority)
	}
	if balls[0].ModelSize != "" {
		t.Errorf("expected empty model size, got %q", balls[0].ModelSize)
	}
}

func TestParseString_ExtraTags(t *testing.T) {
	content := `## Add feature [high] [frontend] [api]

Some context.

- Build UI
`

	balls, err := ParseString(content, "test.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(balls) != 1 {
		t.Fatalf("expected 1 ball, got %d", len(balls))
	}

	b := balls[0]
	if b.Title != "Add feature" {
		t.Errorf("expected title 'Add feature', got %q", b.Title)
	}
	if b.Priority != "high" {
		t.Errorf("expected priority 'high', got %q", b.Priority)
	}
	if len(b.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d: %v", len(b.Tags), b.Tags)
	}
	if b.Tags[0] != "frontend" || b.Tags[1] != "api" {
		t.Errorf("expected tags [frontend, api], got %v", b.Tags)
	}
}

func TestParseString_CheckboxLists(t *testing.T) {
	content := `## Task with checkboxes

Some context here.

- [ ] Unchecked item
- [x] Checked item
- [X] Also checked
`

	balls, err := ParseString(content, "test.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(balls) != 1 {
		t.Fatalf("expected 1 ball, got %d", len(balls))
	}

	if len(balls[0].AcceptanceCriteria) != 3 {
		t.Fatalf("expected 3 acceptance criteria, got %d", len(balls[0].AcceptanceCriteria))
	}
	if balls[0].AcceptanceCriteria[0] != "Unchecked item" {
		t.Errorf("expected 'Unchecked item', got %q", balls[0].AcceptanceCriteria[0])
	}
	if balls[0].AcceptanceCriteria[1] != "Checked item" {
		t.Errorf("expected 'Checked item', got %q", balls[0].AcceptanceCriteria[1])
	}
}

func TestParseString_MixedListTypes(t *testing.T) {
	content := `## Mixed list task

Context paragraph.

- Bullet item
* Star bullet item
1. Numbered item
2. Another numbered item
`

	balls, err := ParseString(content, "test.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(balls) != 1 {
		t.Fatalf("expected 1 ball, got %d", len(balls))
	}

	if len(balls[0].AcceptanceCriteria) != 4 {
		t.Fatalf("expected 4 acceptance criteria, got %d: %v", len(balls[0].AcceptanceCriteria), balls[0].AcceptanceCriteria)
	}
}

func TestParseString_NoBalls(t *testing.T) {
	content := `# Just a Title

Some regular text without any H2 sections.

More text here.
`

	balls, err := ParseString(content, "test.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(balls) != 0 {
		t.Errorf("expected 0 balls, got %d", len(balls))
	}
}

func TestParseString_EmptyFile(t *testing.T) {
	balls, err := ParseString("", "test.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(balls) != 0 {
		t.Errorf("expected 0 balls, got %d", len(balls))
	}
}

func TestParseString_NoContext(t *testing.T) {
	content := `## Task without context

- Just criteria
- Nothing else
`

	balls, err := ParseString(content, "test.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(balls) != 1 {
		t.Fatalf("expected 1 ball, got %d", len(balls))
	}

	if balls[0].Context != "" {
		t.Errorf("expected empty context, got %q", balls[0].Context)
	}
	if len(balls[0].AcceptanceCriteria) != 2 {
		t.Errorf("expected 2 criteria, got %d", len(balls[0].AcceptanceCriteria))
	}
}

func TestParseString_NoCriteria(t *testing.T) {
	content := `## Task without criteria

Just context text here with no list items.
Multi-line context paragraph continues.
`

	balls, err := ParseString(content, "test.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(balls) != 1 {
		t.Fatalf("expected 1 ball, got %d", len(balls))
	}

	if balls[0].Context == "" {
		t.Error("expected non-empty context")
	}
	if len(balls[0].AcceptanceCriteria) != 0 {
		t.Errorf("expected 0 criteria, got %d", len(balls[0].AcceptanceCriteria))
	}
}

func TestParseString_H3DoesNotCreateBalls(t *testing.T) {
	content := `## Main task

Context for main task.

### Sub-section

This is a sub-section and should not create a new ball.

- Criterion 1
`

	balls, err := ParseString(content, "test.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(balls) != 1 {
		t.Fatalf("expected 1 ball, got %d", len(balls))
	}

	if balls[0].Title != "Main task" {
		t.Errorf("expected title 'Main task', got %q", balls[0].Title)
	}
}

func TestParseString_MultipleH1Sections(t *testing.T) {
	content := `# Part 1

## Task A

- Criterion A

# Part 2

## Task B

- Criterion B
`

	balls, err := ParseString(content, "test.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(balls) != 2 {
		t.Fatalf("expected 2 balls, got %d", len(balls))
	}

	if balls[0].Title != "Task A" {
		t.Errorf("expected 'Task A', got %q", balls[0].Title)
	}
	if balls[1].Title != "Task B" {
		t.Errorf("expected 'Task B', got %q", balls[1].Title)
	}
}

func TestParseString_DefaultPriorityAndModelSize(t *testing.T) {
	content := `## Plain task

- Do something
`

	balls, err := ParseString(content, "test.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(balls) != 1 {
		t.Fatalf("expected 1 ball, got %d", len(balls))
	}

	if balls[0].Priority != "" {
		t.Errorf("expected empty priority (caller sets default), got %q", balls[0].Priority)
	}
	if balls[0].ModelSize != "" {
		t.Errorf("expected empty model size, got %q", balls[0].ModelSize)
	}
}

func TestParseFile(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "spec.md")

	content := `# Test Spec

## First task [high]

Context for first task.

- Criterion 1
- Criterion 2

## Second task

- Criterion A
`

	if err := os.WriteFile(specPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	balls, err := ParseFile(specPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(balls) != 2 {
		t.Fatalf("expected 2 balls, got %d", len(balls))
	}

	if balls[0].Title != "First task" {
		t.Errorf("expected 'First task', got %q", balls[0].Title)
	}
	if balls[0].Priority != "high" {
		t.Errorf("expected priority 'high', got %q", balls[0].Priority)
	}
}

func TestParseFile_NotFound(t *testing.T) {
	_, err := ParseFile("/nonexistent/path/spec.md")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestFindSpecFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create spec.md and PRD.md
	os.WriteFile(filepath.Join(tmpDir, "spec.md"), []byte("# spec"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "PRD.md"), []byte("# prd"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "other.md"), []byte("# other"), 0644)

	files, err := FindSpecFiles(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(files), files)
	}

	// Check that both spec.md and PRD.md are found (order may vary)
	found := make(map[string]bool)
	for _, f := range files {
		found[f] = true
	}
	if !found["spec.md"] {
		t.Error("spec.md not found")
	}
	if !found["PRD.md"] {
		t.Error("PRD.md not found")
	}
}

func TestFindSpecFiles_CaseInsensitive(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files with various cases
	os.WriteFile(filepath.Join(tmpDir, "SPEC.MD"), []byte("# spec"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "Prd.Md"), []byte("# prd"), 0644)

	files, err := FindSpecFiles(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(files), files)
	}
}

func TestFindSpecFiles_NoFiles(t *testing.T) {
	tmpDir := t.TempDir()

	files, err := FindSpecFiles(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestParseDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	specContent := `## Task from spec

- Spec criterion 1
`

	prdContent := `## Task from PRD [high]

PRD context.

- PRD criterion 1
- PRD criterion 2
`

	os.WriteFile(filepath.Join(tmpDir, "spec.md"), []byte(specContent), 0644)
	os.WriteFile(filepath.Join(tmpDir, "PRD.md"), []byte(prdContent), 0644)

	balls, err := ParseDirectory(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(balls) != 2 {
		t.Fatalf("expected 2 balls, got %d", len(balls))
	}

	// Find ball from each file
	var specBall, prdBall *ParsedBall
	for i := range balls {
		if balls[i].Title == "Task from spec" {
			specBall = &balls[i]
		}
		if balls[i].Title == "Task from PRD" {
			prdBall = &balls[i]
		}
	}

	if specBall == nil {
		t.Fatal("spec ball not found")
	}
	if prdBall == nil {
		t.Fatal("prd ball not found")
	}

	if prdBall.Priority != "high" {
		t.Errorf("expected prd ball priority 'high', got %q", prdBall.Priority)
	}
	if len(prdBall.AcceptanceCriteria) != 2 {
		t.Errorf("expected 2 criteria for prd ball, got %d", len(prdBall.AcceptanceCriteria))
	}
}

func TestParseDirectory_NoFiles(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := ParseDirectory(tmpDir)
	if err == nil {
		t.Error("expected error when no spec files found")
	}
}

func TestParseString_ContextWithMultipleLines(t *testing.T) {
	content := `## Complex task

This is the first line of context.
This is the second line of context.
And a third line.

- Criterion 1
`

	balls, err := ParseString(content, "test.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(balls) != 1 {
		t.Fatalf("expected 1 ball, got %d", len(balls))
	}

	expected := "This is the first line of context.\nThis is the second line of context.\nAnd a third line."
	if balls[0].Context != expected {
		t.Errorf("expected context %q, got %q", expected, balls[0].Context)
	}
}

func TestParseString_IndentedListItems(t *testing.T) {
	content := `## Task

  - Indented bullet
  1. Indented number
  - [ ] Indented checkbox
`

	balls, err := ParseString(content, "test.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(balls) != 1 {
		t.Fatalf("expected 1 ball, got %d", len(balls))
	}

	if len(balls[0].AcceptanceCriteria) != 3 {
		t.Fatalf("expected 3 criteria, got %d: %v", len(balls[0].AcceptanceCriteria), balls[0].AcceptanceCriteria)
	}
}

func TestParseString_RealisticPRD(t *testing.T) {
	content := `# Product Requirements Document

## Overview

This document describes the requirements for our new feature.

## User Login [high]

As a user, I want to log in securely to access my account.

- [ ] Email and password authentication
- [ ] OAuth2 support (Google, GitHub)
- [ ] Two-factor authentication
- [ ] Session management with JWT tokens
- [ ] Logout functionality

## User Profile [medium]

Users should be able to manage their profile information.

1. Display user profile with avatar
2. Edit name, email, and bio
3. Upload profile picture
4. Change password
5. Delete account with confirmation

## Admin Dashboard [high] [large]

Administrators need a dashboard to manage users and content.

- View all registered users
- Ban/suspend user accounts
- View system metrics
- Manage content moderation queue

## API Documentation [low] [small]

We need comprehensive API documentation.

- Generate OpenAPI spec from code
- Host interactive docs at /api/docs
- Include authentication examples
`

	balls, err := ParseString(content, "PRD.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not include "Overview" since it has no list items and is
	// just introductory text
	if len(balls) != 5 {
		t.Fatalf("expected 5 balls, got %d", len(balls))
	}

	// Check the first real ball (Overview is also captured since it has context)
	// Actually Overview will be captured as a ball (it's an H2 section)
	// Let's verify each one

	expected := []struct {
		title     string
		priority  string
		modelSize string
		criteria  int
	}{
		{"Overview", "", "", 0},
		{"User Login", "high", "", 5},
		{"User Profile", "medium", "", 5},
		{"Admin Dashboard", "high", "large", 4},
		{"API Documentation", "low", "small", 3},
	}

	for i, exp := range expected {
		if balls[i].Title != exp.title {
			t.Errorf("ball %d: expected title %q, got %q", i, exp.title, balls[i].Title)
		}
		if balls[i].Priority != exp.priority {
			t.Errorf("ball %d (%s): expected priority %q, got %q", i, exp.title, exp.priority, balls[i].Priority)
		}
		if balls[i].ModelSize != exp.modelSize {
			t.Errorf("ball %d (%s): expected model size %q, got %q", i, exp.title, exp.modelSize, balls[i].ModelSize)
		}
		if len(balls[i].AcceptanceCriteria) != exp.criteria {
			t.Errorf("ball %d (%s): expected %d criteria, got %d", i, exp.title, exp.criteria, len(balls[i].AcceptanceCriteria))
		}
	}
}

func TestExtractListItem(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"- bullet item", "bullet item"},
		{"* star bullet", "star bullet"},
		{"1. numbered", "numbered"},
		{"10. double digit", "double digit"},
		{"- [ ] unchecked", "unchecked"},
		{"- [x] checked", "checked"},
		{"- [X] also checked", "also checked"},
		{"  - indented bullet", "indented bullet"},
		{"  1. indented number", "indented number"},
		{"regular text", ""},
		{"## heading", ""},
		{"", ""},
		{"   ", ""},
	}

	for _, tt := range tests {
		got := extractListItem(tt.input)
		if got != tt.expected {
			t.Errorf("extractListItem(%q): expected %q, got %q", tt.input, tt.expected, got)
		}
	}
}
