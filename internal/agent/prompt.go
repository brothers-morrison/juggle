// Package agent provides the agent prompt template and related utilities
// for running AI agents with juggle.
package agent

import (
	_ "embed"
)

//go:embed prompt.md
var PromptTemplate string

// GetPromptTemplate returns the embedded agent prompt template.
func GetPromptTemplate() string {
	return PromptTemplate
}
