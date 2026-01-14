package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

const (
	projectStorePath = ".juggle"
	ballsFile        = "balls.jsonl"
	archiveDir       = "archive"
	archiveBallsFile = "balls.jsonl"
)

// StoreConfig holds configurable options for Store.
type StoreConfig struct {
	JuggleDirName string // Name of the juggle directory (default: ".juggle")
}

// DefaultStoreConfig returns the default store configuration.
func DefaultStoreConfig() StoreConfig {
	return StoreConfig{
		JuggleDirName: projectStorePath,
	}
}

// Store handles persistence of balls in a project directory.
//
// Store manages balls stored in JSONL format at .juggle/balls.jsonl (active)
// and .juggle/archive/balls.jsonl (completed). It provides thread-safe
// CRUD operations using file locking.
//
// Key features:
//   - JSONL format for append-friendly version control
//   - File locking for concurrent access safety
//   - Atomic writes via temp file + rename pattern
//   - Ball resolution by full ID, short ID, or prefix
//   - Worktree-aware: resolves to main repo when in a git worktree
//
// Create a Store with NewStore or NewStoreWithConfig:
//
//	store, err := session.NewStore("/path/to/project")
//	balls, err := store.LoadBalls()
type Store struct {
	projectDir  string
	ballsPath   string
	archivePath string
	config      StoreConfig
}

// ProjectDir returns the project directory for this store
func (s *Store) ProjectDir() string {
	return s.projectDir
}

// NewStore creates a new store for the given project directory
func NewStore(projectDir string) (*Store, error) {
	return NewStoreWithConfig(projectDir, DefaultStoreConfig())
}

// NewStoreWithConfig creates a new store with custom configuration.
// If running in a worktree (has .juggle/link file), uses the linked main repo for storage.
func NewStoreWithConfig(projectDir string, config StoreConfig) (*Store, error) {
	if projectDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
		projectDir = cwd
	}

	// Resolve to main repo if this is a worktree
	storageDir, err := ResolveStorageDir(projectDir, config.JuggleDirName)
	if err != nil {
		// If resolution fails, fall back to projectDir
		storageDir = projectDir
	}

	storePath := filepath.Join(storageDir, config.JuggleDirName)
	ballsPath := filepath.Join(storePath, ballsFile)
	archivePath := filepath.Join(storePath, archiveDir, archiveBallsFile)

	// Ensure directories exist
	if err := os.MkdirAll(storePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create %s directory: %w", config.JuggleDirName, err)
	}

	archiveDirPath := filepath.Join(storePath, archiveDir)
	if err := os.MkdirAll(archiveDirPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create archive directory: %w", err)
	}

	return &Store{
		projectDir:  projectDir,
		ballsPath:   ballsPath,
		archivePath: archivePath,
		config:      config,
	}, nil
}

// acquireFileLock acquires an exclusive lock on a file
// Returns the file handle and cleanup function. The cleanup function should be deferred.
func acquireFileLock(path string) (*os.File, func(), error) {
	lockPath := path + ".lock"
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open lock file %s: %w", lockPath, err)
	}

	// Acquire exclusive lock (blocking)
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, nil, fmt.Errorf("failed to acquire lock on %s: %w", lockPath, err)
	}

	cleanup := func() {
		syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
	}

	return f, cleanup, nil
}

// AppendBall adds a new ball to the JSONL file
func (s *Store) AppendBall(ball *Ball) error {
	data, err := json.Marshal(ball)
	if err != nil {
		return fmt.Errorf("failed to marshal ball: %w", err)
	}

	// Acquire file lock
	_, unlock, err := acquireFileLock(s.ballsPath)
	if err != nil {
		return err
	}
	defer unlock()

	// Open file in append mode
	f, err := os.OpenFile(s.ballsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open balls file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("failed to write ball: %w", err)
	}

	if _, err := f.WriteString("\n"); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	return nil
}

// ballJSON is used for JSON unmarshaling with migration support
// It includes both old (intent) and new (title) field names
type ballJSON struct {
	Ball
	Intent string `json:"intent,omitempty"` // Legacy field, migrated to Title
}

// LoadBalls reads all balls from the JSONL file
func (s *Store) LoadBalls() ([]*Ball, error) {
	// If file doesn't exist, return empty slice
	if _, err := os.Stat(s.ballsPath); os.IsNotExist(err) {
		return []*Ball{}, nil
	}

	f, err := os.Open(s.ballsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open balls file: %w", err)
	}
	defer f.Close()

	balls := make([]*Ball, 0)
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue // Skip empty lines
		}

		var ballData ballJSON
		if err := json.Unmarshal([]byte(line), &ballData); err != nil {
			// Log error but continue
			fmt.Fprintf(os.Stderr, "Warning: failed to parse ball line: %v\n", err)
			continue
		}

		ball := ballData.Ball

		// Migrate legacy "intent" field to "title"
		if ball.Title == "" && ballData.Intent != "" {
			ball.Title = ballData.Intent
		}

		// Set WorkingDir from store location (not stored in JSON)
		ball.WorkingDir = s.projectDir

		balls = append(balls, &ball)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading balls file: %w", err)
	}

	return balls, nil
}

// LoadArchivedBalls reads all balls from the archive JSONL file
func (s *Store) LoadArchivedBalls() ([]*Ball, error) {
	// If file doesn't exist, return empty slice
	if _, err := os.Stat(s.archivePath); os.IsNotExist(err) {
		return []*Ball{}, nil
	}

	f, err := os.Open(s.archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open archive file: %w", err)
	}
	defer f.Close()

	balls := make([]*Ball, 0)
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue // Skip empty lines
		}

		var ballData ballJSON
		if err := json.Unmarshal([]byte(line), &ballData); err != nil {
			// Log error but continue
			fmt.Fprintf(os.Stderr, "Warning: failed to parse archived ball line: %v\n", err)
			continue
		}

		ball := ballData.Ball

		// Migrate legacy "intent" field to "title"
		if ball.Title == "" && ballData.Intent != "" {
			ball.Title = ballData.Intent
		}

		// Set WorkingDir from store location (not stored in JSON)
		ball.WorkingDir = s.projectDir

		balls = append(balls, &ball)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading archive file: %w", err)
	}

	return balls, nil
}

// UpdateBall updates an existing ball by rewriting the JSONL file
func (s *Store) UpdateBall(updated *Ball) error {
	balls, err := s.LoadBalls()
	if err != nil {
		return err
	}

	// Find and update the ball
	found := false
	for i, ball := range balls {
		if ball.ID == updated.ID {
			balls[i] = updated
			found = true
			break
		}
	}

	if !found {
		return NewBallNotFoundError(updated.ID)
	}

	// Rewrite entire file
	return s.writeBalls(balls)
}

// DeleteBall removes a ball from the JSONL file
func (s *Store) DeleteBall(id string) error {
	balls, err := s.LoadBalls()
	if err != nil {
		return err
	}

	// Filter out the ball to delete
	filtered := make([]*Ball, 0, len(balls))
	for _, ball := range balls {
		if ball.ID != id {
			filtered = append(filtered, ball)
		}
	}

	return s.writeBalls(filtered)
}

// ArchiveBall moves a ball to the archive.
// This operation is atomic: both files are locked, and changes are applied
// atomically using temp file + rename pattern.
func (s *Store) ArchiveBall(ball *Ball) error {
	// Acquire locks on both files to ensure atomic operation
	_, unlockBalls, err := acquireFileLock(s.ballsPath)
	if err != nil {
		return fmt.Errorf("failed to lock balls file: %w", err)
	}
	defer unlockBalls()

	_, unlockArchive, err := acquireFileLock(s.archivePath)
	if err != nil {
		return fmt.Errorf("failed to lock archive file: %w", err)
	}
	defer unlockArchive()

	// Load current balls
	balls, err := s.LoadBalls()
	if err != nil {
		return fmt.Errorf("failed to load balls: %w", err)
	}

	// Load current archive
	archived, err := s.LoadArchivedBalls()
	if err != nil {
		return fmt.Errorf("failed to load archived balls: %w", err)
	}

	// Find and remove the ball from active list
	found := false
	filtered := make([]*Ball, 0, len(balls))
	for _, b := range balls {
		if b.ID != ball.ID {
			filtered = append(filtered, b)
		} else {
			found = true
		}
	}

	if !found {
		return NewBallNotFoundError(ball.ID)
	}

	// Add ball to archive
	archived = append(archived, ball)

	// Write both files atomically
	// First, write the new archive (safer to add first)
	if err := s.writeArchivedBallsUnlocked(archived); err != nil {
		return fmt.Errorf("failed to update archive: %w", err)
	}

	// Then, write the active balls (without the archived ball)
	if err := s.writeBallsUnlocked(filtered); err != nil {
		// Attempt to restore archive on failure (remove the ball we just added)
		// This is best-effort; in worst case we have a duplicate in archive
		s.writeArchivedBallsUnlocked(archived[:len(archived)-1])
		return fmt.Errorf("failed to remove ball from active: %w", err)
	}

	return nil
}

// GetInProgressBalls returns all balls currently in progress in this project
func (s *Store) GetInProgressBalls() ([]*Ball, error) {
	balls, err := s.LoadBalls()
	if err != nil {
		return nil, err
	}

	// Filter for in_progress balls
	inProgress := make([]*Ball, 0)
	for _, ball := range balls {
		if ball.State == StateInProgress {
			inProgress = append(inProgress, ball)
		}
	}

	// Sort by most recently active first
	sort.Slice(inProgress, func(i, j int) bool {
		return inProgress[i].LastActivity.After(inProgress[j].LastActivity)
	})

	return inProgress, nil
}

// GetBallsByState returns all balls with the given state
func (s *Store) GetBallsByState(state BallState) ([]*Ball, error) {
	all, err := s.LoadBalls()
	if err != nil {
		return nil, err
	}

	filtered := make([]*Ball, 0)
	for _, ball := range all {
		if ball.State == state {
			filtered = append(filtered, ball)
		}
	}

	return filtered, nil
}

// GetBallByID finds a ball by its ID
func (s *Store) GetBallByID(id string) (*Ball, error) {
	balls, err := s.LoadBalls()
	if err != nil {
		return nil, err
	}

	for _, ball := range balls {
		if ball.ID == id {
			return ball, nil
		}
	}

	return nil, NewBallNotFoundError(id)
}


// GetBallByShortID finds a ball by its short ID (numeric part)
// If multiple balls match, returns the most recently active
func (s *Store) GetBallByShortID(shortID string) (*Ball, error) {
	balls, err := s.LoadBalls()
	if err != nil {
		return nil, err
	}

	matches := make([]*Ball, 0)
	for _, ball := range balls {
		if ball.ShortID() == shortID {
			matches = append(matches, ball)
		}
	}

	if len(matches) == 0 {
		return nil, NewBallNotFoundShortError(shortID)
	}

	// If multiple matches, return most recently active
	if len(matches) > 1 {
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].LastActivity.After(matches[j].LastActivity)
		})
	}

	return matches[0], nil
}

// ResolveBallID resolves a ball ID from either full ID, short ID, or prefix match.
// For prefix matching, input "0" will match a ball with short ID "01234abc" if
// no other ball's short ID starts with "0".
func (s *Store) ResolveBallID(id string) (*Ball, error) {
	// Try as full ID first
	ball, err := s.GetBallByID(id)
	if err == nil {
		return ball, nil
	}

	// Load all balls for prefix matching
	balls, err := s.LoadBalls()
	if err != nil {
		return nil, err
	}

	// Use prefix matching to find candidates
	matches := ResolveBallByPrefix(balls, id)

	if len(matches) == 0 {
		return nil, NewBallNotFoundError(id)
	}

	if len(matches) == 1 {
		return matches[0], nil
	}

	// Multiple matches - return most recently active (for backward compatibility)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].LastActivity.After(matches[j].LastActivity)
	})
	return matches[0], nil
}

// ResolveBallIDStrict resolves a ball ID with strict uniqueness requirement.
// Returns an error if the prefix matches multiple balls.
func (s *Store) ResolveBallIDStrict(id string) (*Ball, error) {
	// Try as full ID first
	ball, err := s.GetBallByID(id)
	if err == nil {
		return ball, nil
	}

	// Load all balls for prefix matching
	balls, err := s.LoadBalls()
	if err != nil {
		return nil, err
	}

	// Use prefix matching to find candidates
	matches := ResolveBallByPrefix(balls, id)

	if len(matches) == 0 {
		return nil, NewBallNotFoundError(id)
	}

	if len(matches) > 1 {
		// Build list of matching IDs for error message
		matchingIDs := make([]string, len(matches))
		for i, m := range matches {
			matchingIDs[i] = m.ID
		}
		return nil, NewAmbiguousIDError(id, matchingIDs)
	}

	return matches[0], nil
}

// writeBalls rewrites the entire balls.jsonl file
func (s *Store) writeBalls(balls []*Ball) error {
	// Acquire file lock
	_, unlock, err := acquireFileLock(s.ballsPath)
	if err != nil {
		return err
	}
	defer unlock()

	return s.writeBallsUnlocked(balls)
}

// writeBallsUnlocked rewrites the entire balls.jsonl file without acquiring a lock.
// Caller must hold the lock.
func (s *Store) writeBallsUnlocked(balls []*Ball) error {
	// Write to temp file first
	tempPath := s.ballsPath + ".tmp"
	f, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	for _, ball := range balls {
		data, err := json.Marshal(ball)
		if err != nil {
			f.Close()
			os.Remove(tempPath)
			return fmt.Errorf("failed to marshal ball: %w", err)
		}

		if _, err := f.Write(data); err != nil {
			f.Close()
			os.Remove(tempPath)
			return fmt.Errorf("failed to write ball: %w", err)
		}

		if _, err := f.WriteString("\n"); err != nil {
			f.Close()
			os.Remove(tempPath)
			return fmt.Errorf("failed to write newline: %w", err)
		}
	}

	if err := f.Close(); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, s.ballsPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// UnarchiveBall restores a completed ball from archive back to ready state.
// This operation is atomic: both files are locked, and changes are applied
// atomically using temp file + rename pattern.
func (s *Store) UnarchiveBall(ballID string) (*Ball, error) {
	// Acquire locks on both files to ensure atomic operation
	_, unlockBalls, err := acquireFileLock(s.ballsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to lock balls file: %w", err)
	}
	defer unlockBalls()

	_, unlockArchive, err := acquireFileLock(s.archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to lock archive file: %w", err)
	}
	defer unlockArchive()

	// Load archived balls (within lock)
	archived, err := s.LoadArchivedBalls()
	if err != nil {
		return nil, fmt.Errorf("failed to load archived balls: %w", err)
	}

	// Find ball with matching ID
	var ball *Ball
	var ballIndex int
	for i, b := range archived {
		if b.ID == ballID {
			ball = b
			ballIndex = i
			break
		}
	}
	if ball == nil {
		return nil, NewBallNotFoundError(ballID)
	}

	// Change state to pending using new state model
	ball.State = StatePending
	ball.BlockedReason = ""
	ball.CompletedAt = nil
	ball.CompletionNote = ""

	// Load current balls
	balls, err := s.LoadBalls()
	if err != nil {
		return nil, fmt.Errorf("failed to load balls: %w", err)
	}

	// Add the unarchived ball to the list
	balls = append(balls, ball)

	// Prepare the updated archive (without the ball being restored)
	updatedArchive := make([]*Ball, 0, len(archived)-1)
	for i, b := range archived {
		if i != ballIndex {
			updatedArchive = append(updatedArchive, b)
		}
	}

	// Write both files atomically (temp file + rename pattern)
	// First, write the new archive
	if err := s.writeArchivedBallsUnlocked(updatedArchive); err != nil {
		return nil, fmt.Errorf("failed to update archive: %w", err)
	}

	// Then, write the active balls
	if err := s.writeBallsUnlocked(balls); err != nil {
		// Attempt to restore archive on failure
		// This is best-effort; in worst case we have inconsistent state
		s.writeArchivedBallsUnlocked(archived)
		return nil, fmt.Errorf("failed to add ball to active: %w", err)
	}

	return ball, nil
}

// writeArchivedBalls rewrites the entire archive/balls.jsonl file
func (s *Store) writeArchivedBalls(balls []*Ball) error {
	// Acquire file lock
	_, unlock, err := acquireFileLock(s.archivePath)
	if err != nil {
		return err
	}
	defer unlock()

	return s.writeArchivedBallsUnlocked(balls)
}

// writeArchivedBallsUnlocked rewrites the entire archive/balls.jsonl file without acquiring a lock.
// Caller must hold the lock.
func (s *Store) writeArchivedBallsUnlocked(balls []*Ball) error {
	// Write to temp file first
	tempPath := s.archivePath + ".tmp"
	f, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	for _, ball := range balls {
		data, err := json.Marshal(ball)
		if err != nil {
			f.Close()
			os.Remove(tempPath)
			return fmt.Errorf("failed to marshal ball: %w", err)
		}

		if _, err := f.Write(data); err != nil {
			f.Close()
			os.Remove(tempPath)
			return fmt.Errorf("failed to write ball: %w", err)
		}

		if _, err := f.WriteString("\n"); err != nil {
			f.Close()
			os.Remove(tempPath)
			return fmt.Errorf("failed to write newline: %w", err)
		}
	}

	if err := f.Close(); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, s.archivePath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// Save is an alias for UpdateBall for backwards compatibility.
//
// Deprecated: Use UpdateBall for existing balls or AppendBall for new balls instead.
// Save auto-detects whether a ball exists and calls the appropriate method,
// but explicit calls to UpdateBall or AppendBall are clearer.
func (s *Store) Save(ball *Ball) error {
	// Check if ball already exists
	existing, err := s.GetBallByID(ball.ID)
	if err != nil || existing == nil {
		// New ball, append it
		return s.AppendBall(ball)
	}

	// Existing ball, update it
	return s.UpdateBall(ball)
}
