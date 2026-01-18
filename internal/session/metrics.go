package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

const (
	metricsFile = "agent-metrics.json"
)

// AgentMetrics tracks real-time metrics from Claude Code hooks.
// Updated automatically via hooks installed by `juggle hooks install`.
type AgentMetrics struct {
	// Activity tracking (from PostToolUse)
	FilesChanged []string       `json:"files_changed"`
	ToolCounts   map[string]int `json:"tool_counts"`
	ToolFailures int            `json:"tool_failures"`
	LastActivity time.Time      `json:"last_activity"`
	TotalTools   int            `json:"total_tools"`

	// Turn tracking (from Stop)
	TurnCount int `json:"turn_count"`

	// Token usage (from Stop - cumulative)
	InputTokens     int `json:"input_tokens"`
	OutputTokens    int `json:"output_tokens"`
	CacheReadTokens int `json:"cache_read_tokens"`

	// Session state (from SessionEnd)
	SessionEnded bool `json:"session_ended"`
}

// NewAgentMetrics creates an empty metrics struct
func NewAgentMetrics() *AgentMetrics {
	return &AgentMetrics{
		FilesChanged: []string{},
		ToolCounts:   make(map[string]int),
	}
}

// metricsFilePath returns the path to a session's metrics file
func (s *SessionStore) metricsFilePath(id string) string {
	return filepath.Join(s.sessionPath(id), metricsFile)
}

// LoadMetrics reads the metrics from a session's agent-metrics.json file
func (s *SessionStore) LoadMetrics(id string) (*AgentMetrics, error) {
	metricsPath := s.metricsFilePath(id)

	data, err := os.ReadFile(metricsPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty metrics if file doesn't exist
			return NewAgentMetrics(), nil
		}
		return nil, fmt.Errorf("failed to read metrics file: %w", err)
	}

	var metrics AgentMetrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, fmt.Errorf("failed to parse metrics file: %w", err)
	}

	// Initialize maps if nil (from old files)
	if metrics.ToolCounts == nil {
		metrics.ToolCounts = make(map[string]int)
	}
	if metrics.FilesChanged == nil {
		metrics.FilesChanged = []string{}
	}

	return &metrics, nil
}

// SaveMetrics writes metrics to a session's agent-metrics.json file
func (s *SessionStore) SaveMetrics(id string, metrics *AgentMetrics) error {
	// Ensure session directory exists
	sessionDir := s.sessionPath(id)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	metricsPath := s.metricsFilePath(id)
	lockPath := metricsPath + ".lock"

	// Acquire file lock
	fileLock := flock.New(lockPath)
	if err := fileLock.Lock(); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer fileLock.Unlock()

	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	if err := os.WriteFile(metricsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metrics file: %w", err)
	}

	return nil
}

// UpdateMetricsFromPostTool updates metrics based on a PostToolUse hook event
func (s *SessionStore) UpdateMetricsFromPostTool(id, toolName, filePath string) error {
	metrics, err := s.LoadMetrics(id)
	if err != nil {
		return err
	}

	// Update tool counts
	metrics.ToolCounts[toolName]++
	metrics.TotalTools++
	metrics.LastActivity = time.Now()

	// Track file changes
	if filePath != "" {
		metrics.FilesChanged = appendUnique(metrics.FilesChanged, filePath)
	}

	return s.SaveMetrics(id, metrics)
}

// UpdateMetricsFromToolFailure updates metrics based on a PostToolUseFailure hook event
func (s *SessionStore) UpdateMetricsFromToolFailure(id, toolName string) error {
	metrics, err := s.LoadMetrics(id)
	if err != nil {
		return err
	}

	metrics.ToolFailures++
	metrics.LastActivity = time.Now()

	return s.SaveMetrics(id, metrics)
}

// UpdateMetricsFromStop updates metrics based on a Stop hook event
func (s *SessionStore) UpdateMetricsFromStop(id string, inputTokens, outputTokens, cacheReadTokens int) error {
	metrics, err := s.LoadMetrics(id)
	if err != nil {
		return err
	}

	metrics.TurnCount++
	metrics.InputTokens += inputTokens
	metrics.OutputTokens += outputTokens
	metrics.CacheReadTokens += cacheReadTokens
	metrics.LastActivity = time.Now()

	return s.SaveMetrics(id, metrics)
}

// UpdateMetricsFromSessionEnd updates metrics based on a SessionEnd hook event
func (s *SessionStore) UpdateMetricsFromSessionEnd(id string) error {
	metrics, err := s.LoadMetrics(id)
	if err != nil {
		return err
	}

	metrics.SessionEnded = true
	metrics.LastActivity = time.Now()

	return s.SaveMetrics(id, metrics)
}

// appendUnique appends a string to a slice if it's not already present
func appendUnique(slice []string, item string) []string {
	for _, existing := range slice {
		if existing == item {
			return slice
		}
	}
	return append(slice, item)
}
