// Package vcs provides a unified interface for version control systems.
package vcs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// VCSType represents the version control system type.
type VCSType string

const (
	VCSTypeJJ  VCSType = "jj"
	VCSTypeGit VCSType = "git"
)

// String returns the string representation of VCSType.
func (v VCSType) String() string {
	return string(v)
}

// IsValid returns true if the VCSType is a known valid type.
func (v VCSType) IsValid() bool {
	return v == VCSTypeJJ || v == VCSTypeGit
}

// CommitResult represents the outcome of a commit operation.
type CommitResult struct {
	Success      bool   // Whether the commit succeeded
	CommitHash   string // Short hash of the new commit (if successful)
	StatusOutput string // Output from status after commit
	ErrorMessage string // Error message if commit failed
}

// VCS defines the interface for version control operations.
type VCS interface {
	// Type returns the VCS type (jj or git)
	Type() VCSType

	// Status returns the current status output
	Status(projectDir string) (string, error)

	// HasChanges returns true if there are uncommitted changes
	HasChanges(projectDir string) (bool, error)

	// Commit creates a commit with the given message
	Commit(projectDir, message string) (*CommitResult, error)

	// GetLastCommitHash returns the short hash of the last commit
	GetLastCommitHash(projectDir string) (string, error)

	// DescribeWorkingCopy updates the working copy description with the given message.
	// For jj: runs "jj desc -m <message>"
	// For git: this is a no-op (git doesn't have working copy descriptions)
	DescribeWorkingCopy(projectDir, message string) error

	// IsolateAndReset creates a new working copy based on a target revision,
	// leaving the current changes in a separate revision.
	// For jj: runs "jj new <targetRevision>" to create a new change from the target
	// For git: creates a branch for the current work and checks out the target revision
	// If targetRevision is empty, uses a sensible default (parent for jj, main/master for git).
	// Returns the revision ID of the isolated changes.
	IsolateAndReset(projectDir, targetRevision string) (string, error)

	// GetCurrentRevision returns the current working copy revision/change ID.
	// For jj: returns the change_id of the working copy
	// For git: returns the current commit hash or branch name
	GetCurrentRevision(projectDir string) (string, error)
}

// GetBackend returns the appropriate VCS backend for the given type.
func GetBackend(vcsType VCSType) VCS {
	switch vcsType {
	case VCSTypeJJ:
		return NewJJBackend()
	case VCSTypeGit:
		return NewGitBackend()
	default:
		return NewGitBackend() // Default to git
	}
}

// GetBackendForProject returns the VCS backend for a project, using config resolution.
func GetBackendForProject(projectDir string, projectVCS, globalVCS VCSType) VCS {
	vcsType := Detect(projectDir, projectVCS, globalVCS)
	return GetBackend(vcsType)
}

// IsVCSInitialized checks if a VCS is already initialized in the directory.
// Returns true if either .jj or .git exists.
func IsVCSInitialized(projectDir string) bool {
	return AutoDetectExists(projectDir)
}

// AutoDetectExists checks if any VCS directory exists.
// Returns true if .jj or .git exists.
func AutoDetectExists(projectDir string) bool {
	jjPath := filepath.Join(projectDir, ".jj")
	gitPath := filepath.Join(projectDir, ".git")

	if _, err := os.Stat(jjPath); err == nil {
		return true
	}
	if _, err := os.Stat(gitPath); err == nil {
		return true
	}
	return false
}

// IsJJAvailable checks if the jj command is available in PATH.
func IsJJAvailable() bool {
	_, err := exec.LookPath("jj")
	return err == nil
}

// IsGitAvailable checks if the git command is available in PATH.
func IsGitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// InitJJ initializes a jj repository in the given directory.
func InitJJ(projectDir string) error {
	cmd := exec.Command("jj", "git", "init")
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("jj git init: %w\n%s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// InitGit initializes a git repository in the given directory.
func InitGit(projectDir string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git init: %w\n%s", err, strings.TrimSpace(string(output)))
	}
	return nil
}
