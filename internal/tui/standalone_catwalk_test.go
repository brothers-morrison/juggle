package tui

import (
	"os"
	"testing"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/knz/catwalk"
	"github.com/ohare93/juggle/internal/session"
)

// createTestStandaloneBallModel creates a StandaloneBallModel for testing.
// Uses a temp directory for the store to avoid nil pointer issues.
func createTestStandaloneBallModel(t *testing.T) StandaloneBallModel {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "juggler-tui-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	store, err := session.NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Create text input for title field
	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = 60
	ti.Placeholder = "What is this ball about? (50 char recommended)"
	ti.Blur()

	// Create textarea for context field
	ta := textarea.New()
	ta.Placeholder = "Background context for this task"
	ta.CharLimit = 2000
	ta.SetWidth(60)
	ta.SetHeight(1)
	ta.ShowLineNumbers = false
	ta.Focus()

	return StandaloneBallModel{
		store:               store,
		textInput:           ti,
		contextInput:        ta,
		pendingBallPriority: 1, // Default to medium
		fileAutocomplete:    NewAutocompleteState(store.ProjectDir()),
		width:               80,
		height:              24,
	}
}

// TestStandaloneBallForm tests the standalone ball creation form using catwalk.
// Run with -rewrite to update golden files.
func TestStandaloneBallForm(t *testing.T) {
	model := createTestStandaloneBallModel(t)
	catwalk.RunModel(t, "testdata/standalone_ball_form", model)
}

// TestStandaloneBallFormWithData tests the form with pre-populated data.
func TestStandaloneBallFormWithData(t *testing.T) {
	model := createTestStandaloneBallModel(t)
	model.pendingBallIntent = "Test task intent"
	model.pendingBallContext = "Some context for the task"
	model.contextInput.SetValue("Some context for the task")
	model.pendingBallTags = "feature, backend"
	model.pendingAcceptanceCriteria = []string{
		"First criterion",
		"Second criterion",
	}
	catwalk.RunModel(t, "testdata/standalone_ball_form_with_data", model)
}

// TestStandaloneBallFormNavigation tests navigating through form fields.
func TestStandaloneBallFormNavigation(t *testing.T) {
	model := createTestStandaloneBallModel(t)
	catwalk.RunModel(t, "testdata/standalone_ball_form_navigation", model)
}

// TestStandaloneBallFormLongContext tests that long context text wraps correctly.
func TestStandaloneBallFormLongContext(t *testing.T) {
	model := createTestStandaloneBallModel(t)
	longContext := "One Two Three Four Five Six Seven Eight Nine Ten Eleven Twelve"
	model.pendingBallContext = longContext
	model.contextInput.SetValue(longContext)
	// Move focus away from context field so we see the wrapped display
	model.pendingBallFormField = 1 // fieldIntent
	model.contextInput.Blur()
	model.textInput.Focus()
	catwalk.RunModel(t, "testdata/standalone_ball_form_long_context", model)
}

// TestStandaloneBallFormLongContextEditing tests long context while editing (field focused).
func TestStandaloneBallFormLongContextEditing(t *testing.T) {
	model := createTestStandaloneBallModel(t)
	longContext := "One Two Three Four Five Six Seven Eight Nine Ten Eleven Twelve"
	model.PrePopulate("Test intent", longContext, nil, "", "medium", "", nil, nil)
	catwalk.RunModel(t, "testdata/standalone_ball_form_long_context_editing", model)
}

// TestStandaloneBallFormVeryLongContext tests with even longer context (3+ lines).
func TestStandaloneBallFormVeryLongContext(t *testing.T) {
	model := createTestStandaloneBallModel(t)
	longContext := "One Two Three Four Five Six Seven Eight Nine Ten Eleven Twelve Thirteen Fourteen Fifteen Sixteen Seventeen Eighteen Nineteen Twenty TwentyOne TwentyTwo TwentyThree TwentyFour TwentyFive"
	model.PrePopulate("Test intent", longContext, nil, "", "medium", "", nil, nil)
	catwalk.RunModel(t, "testdata/standalone_ball_form_very_long_context", model)
}

// TestStandaloneBallFormRealConstructor tests using the actual NewStandaloneBallModel constructor.
func TestStandaloneBallFormRealConstructor(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-tui-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	store, err := session.NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	model := NewStandaloneBallModel(store, nil)
	longContext := "One Two Three Four Five Six Seven Eight Nine Ten Eleven Twelve"
	model.PrePopulate("Test intent", longContext, nil, "", "medium", "", nil, nil)
	catwalk.RunModel(t, "testdata/standalone_ball_form_real_constructor", model)
}

// TestStandaloneBallFormTypingLongContext tests typing long text into context field.
func TestStandaloneBallFormTypingLongContext(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-tui-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	store, err := session.NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	model := NewStandaloneBallModel(store, nil)
	catwalk.RunModel(t, "testdata/standalone_ball_form_typing_long_context", model)
}
