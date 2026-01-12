package agent

import (
	_ "embed"
)

//go:embed refine_prompt.md
var RefinePromptTemplate string

// GetRefinePromptTemplate returns the embedded refinement prompt template.
func GetRefinePromptTemplate() string {
	return RefinePromptTemplate
}
