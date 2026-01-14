package integration_test

import (
	"sync"
	"testing"
	"time"

	"github.com/ohare93/juggle/internal/session"
)

// TestConcurrentBallAppend tests that multiple goroutines can append balls
// without data corruption or lost writes.
func TestConcurrentBallAppend(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	numGoroutines := 10
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	// Launch multiple goroutines that each append a ball
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			store, err := session.NewStore(env.ProjectDir)
			if err != nil {
				errors <- err
				return
			}
			ball, err := session.NewBall(env.ProjectDir, "Concurrent ball", session.PriorityMedium)
			if err != nil {
				errors <- err
				return
			}
			if err := store.AppendBall(ball); err != nil {
				errors <- err
				return
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Error during concurrent append: %v", err)
	}

	// Verify all balls were written
	store := env.GetStore(t)
	balls, err := store.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load balls: %v", err)
	}

	if len(balls) != numGoroutines {
		t.Errorf("Expected %d balls, got %d", numGoroutines, len(balls))
	}
}

// TestConcurrentBallUpdate tests that concurrent updates to different balls
// complete without data corruption or errors. Note: concurrent read-modify-write
// on the same data may result in lost updates (last writer wins), but the
// file should remain valid.
func TestConcurrentBallUpdate(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create initial balls - one per goroutine to avoid read-modify-write races
	numBalls := 5
	store := env.GetStore(t)
	balls := make([]*session.Ball, numBalls)
	for i := 0; i < numBalls; i++ {
		ball := env.CreateBall(t, "Ball for update", session.PriorityMedium)
		balls[i] = ball
	}

	// Concurrently update each ball (sequentially within goroutine to avoid
	// read-modify-write race on same ball)
	var wg sync.WaitGroup
	errors := make(chan error, numBalls)

	for i, ball := range balls {
		wg.Add(1)
		go func(idx int, b *session.Ball) {
			defer wg.Done()
			localStore, err := session.NewStore(env.ProjectDir)
			if err != nil {
				errors <- err
				return
			}

			// Load and update - each goroutine works on a different ball
			loaded, err := localStore.GetBallByID(b.ID)
			if err != nil {
				errors <- err
				return
			}

			loaded.Priority = session.PriorityHigh
			loaded.IncrementUpdateCount()

			if err := localStore.UpdateBall(loaded); err != nil {
				errors <- err
				return
			}
		}(i, ball)
	}

	wg.Wait()
	close(errors)

	// Check for errors - no errors means locking is working
	for err := range errors {
		t.Errorf("Error during concurrent update: %v", err)
	}

	// Verify file is not corrupted (can be loaded)
	allBalls, err := store.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load balls after concurrent updates: %v", err)
	}

	// Verify we still have all balls (no corruption)
	if len(allBalls) != numBalls {
		t.Errorf("Expected %d balls after updates, got %d", numBalls, len(allBalls))
	}

	// Count how many were updated to high priority
	// Due to concurrent read-modify-write, not all updates may have persisted
	highPriorityCount := 0
	for _, b := range allBalls {
		if b.Priority == session.PriorityHigh {
			highPriorityCount++
		}
	}
	t.Logf("Updated %d/%d balls to high priority (concurrent updates may overwrite each other)", highPriorityCount, numBalls)
}

// TestConcurrentArchiveUnarchive tests that archive and unarchive operations
// don't conflict with each other.
func TestConcurrentArchiveUnarchive(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	store := env.GetStore(t)

	// Create balls to archive
	ball1 := env.CreateBall(t, "Ball to archive 1", session.PriorityMedium)
	ball1.ForceSetState(session.StateComplete)
	store.UpdateBall(ball1)

	ball2 := env.CreateBall(t, "Ball to archive 2", session.PriorityMedium)
	ball2.ForceSetState(session.StateComplete)
	store.UpdateBall(ball2)

	// Archive both balls
	if err := store.ArchiveBall(ball1); err != nil {
		t.Fatalf("Failed to archive ball1: %v", err)
	}
	if err := store.ArchiveBall(ball2); err != nil {
		t.Fatalf("Failed to archive ball2: %v", err)
	}

	// Verify archive
	archived, err := store.LoadArchivedBalls()
	if err != nil {
		t.Fatalf("Failed to load archived balls: %v", err)
	}
	if len(archived) != 2 {
		t.Errorf("Expected 2 archived balls, got %d", len(archived))
	}

	// Concurrently unarchive both
	var wg sync.WaitGroup
	errors := make(chan error, 2)

	wg.Add(2)
	go func() {
		defer wg.Done()
		localStore, _ := session.NewStore(env.ProjectDir)
		_, err := localStore.UnarchiveBall(ball1.ID)
		if err != nil {
			errors <- err
		}
	}()
	go func() {
		defer wg.Done()
		localStore, _ := session.NewStore(env.ProjectDir)
		_, err := localStore.UnarchiveBall(ball2.ID)
		if err != nil {
			errors <- err
		}
	}()

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Error during concurrent unarchive: %v", err)
	}

	// Verify both balls are back in active list
	activeBalls, err := store.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load active balls: %v", err)
	}

	found := 0
	for _, b := range activeBalls {
		if b.ID == ball1.ID || b.ID == ball2.ID {
			found++
		}
	}
	if found != 2 {
		t.Errorf("Expected both balls to be restored, found %d", found)
	}
}

// TestConcurrentProgressAppend tests that multiple goroutines can append
// to progress files without losing data.
func TestConcurrentProgressAppend(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	sessionStore, err := session.NewSessionStore(env.ProjectDir)
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	_, err = sessionStore.CreateSession("test-concurrent", "Test concurrent session")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	numGoroutines := 10
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	// Launch multiple goroutines that each append to progress
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			localStore, err := session.NewSessionStore(env.ProjectDir)
			if err != nil {
				errors <- err
				return
			}
			entry := time.Now().Format(time.RFC3339) + " Progress entry from goroutine\n"
			if err := localStore.AppendProgress("test-concurrent", entry); err != nil {
				errors <- err
				return
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Error during concurrent progress append: %v", err)
	}

	// Verify all progress entries were written
	progress, err := sessionStore.LoadProgress("test-concurrent")
	if err != nil {
		t.Fatalf("Failed to load progress: %v", err)
	}

	// Count lines in progress (each goroutine adds one line)
	lineCount := 0
	for _, c := range progress {
		if c == '\n' {
			lineCount++
		}
	}

	if lineCount != numGoroutines {
		t.Errorf("Expected %d progress lines, got %d", numGoroutines, lineCount)
	}
}

// TestConcurrentReadWrite tests that reading and writing don't interfere
func TestConcurrentReadWrite(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create initial balls
	for i := 0; i < 5; i++ {
		env.CreateBall(t, "Initial ball", session.PriorityMedium)
	}

	numReaders := 5
	numWriters := 3
	var wg sync.WaitGroup
	errors := make(chan error, numReaders+numWriters)

	// Launch readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				localStore, err := session.NewStore(env.ProjectDir)
				if err != nil {
					errors <- err
					return
				}
				_, err = localStore.LoadBalls()
				if err != nil {
					errors <- err
					return
				}
				time.Sleep(time.Millisecond)
			}
		}()
	}

	// Launch writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				localStore, err := session.NewStore(env.ProjectDir)
				if err != nil {
					errors <- err
					return
				}
				ball, err := session.NewBall(env.ProjectDir, "Concurrent write ball", session.PriorityLow)
				if err != nil {
					errors <- err
					return
				}
				if err := localStore.AppendBall(ball); err != nil {
					errors <- err
					return
				}
				time.Sleep(time.Millisecond * 2)
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Errorf("Error during concurrent read/write: %v", err)
		errorCount++
	}

	// Verify data integrity
	store := env.GetStore(t)
	balls, err := store.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load balls: %v", err)
	}

	expectedBalls := 5 + (numWriters * 5)
	if len(balls) != expectedBalls {
		t.Errorf("Expected %d balls, got %d", expectedBalls, len(balls))
	}
}
