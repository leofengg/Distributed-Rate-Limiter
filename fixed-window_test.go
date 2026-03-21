package main

import (
	"sync"
	"testing"
	"time"
)

// --- Initialization tests ---

func TestFixedWindowNewStartsEmpty(t *testing.T) {
	fw := NewFixedWindow(time.Second, 10)
	if fw.Count() != 0 {
		t.Errorf("expected count 0 on new window, got %d", fw.Count())
	}
}

func TestFixedWindowInvalidWindowSize(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for windowSize <= 0")
		}
	}()
	NewFixedWindow(0, 10)
}

func TestFixedWindowInvalidMaxRequests(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for maxRequests <= 0")
		}
	}()
	NewFixedWindow(time.Second, 0)
}

func TestFixedWindowNegativeWindowSize(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for negative windowSize")
		}
	}()
	NewFixedWindow(-time.Second, 10)
}

// --- Basic Allow() tests ---

func TestFixedWindowAllowsUpToMax(t *testing.T) {
	fw := NewFixedWindow(time.Second, 5)
	for i := 0; i < 5; i++ {
		if !fw.Allow() {
			t.Errorf("expected Allow to return true on request %d", i+1)
		}
	}
}

func TestFixedWindowBlocksAfterMax(t *testing.T) {
	fw := NewFixedWindow(time.Second, 3)
	fw.Allow()
	fw.Allow()
	fw.Allow()
	if fw.Allow() {
		t.Error("expected Allow to return false after reaching max requests")
	}
}

func TestFixedWindowCountMatchesAllowed(t *testing.T) {
	fw := NewFixedWindow(time.Second, 10)
	for i := 0; i < 6; i++ {
		fw.Allow()
	}
	if fw.Count() != 6 {
		t.Errorf("expected count of 6, got %d", fw.Count())
	}
}

func TestFixedWindowBlockedRequestDoesNotIncrementCount(t *testing.T) {
	fw := NewFixedWindow(time.Second, 2)
	fw.Allow()
	fw.Allow()
	fw.Allow() // blocked
	if fw.Count() != 2 {
		t.Errorf("expected count to remain 2 after blocked request, got %d", fw.Count())
	}
}

// --- Window reset tests ---

func TestFixedWindowResetsAfterExpiry(t *testing.T) {
	fw := NewFixedWindow(200*time.Millisecond, 2)
	fw.Allow()
	fw.Allow()

	if fw.Allow() {
		t.Fatal("expected third request to be blocked")
	}

	time.Sleep(250 * time.Millisecond)

	if !fw.Allow() {
		t.Error("expected Allow to succeed after window reset")
	}
}

func TestFixedWindowCountResetsToOne(t *testing.T) {
	fw := NewFixedWindow(200*time.Millisecond, 3)
	fw.Allow()
	fw.Allow()
	fw.Allow()

	time.Sleep(250 * time.Millisecond)

	fw.Allow() // triggers reset, count should be 1
	if fw.Count() != 1 {
		t.Errorf("expected count of 1 after reset, got %d", fw.Count())
	}
}

func TestFixedWindowAllowsFullQuotaAfterReset(t *testing.T) {
	fw := NewFixedWindow(200*time.Millisecond, 3)
	fw.Allow()
	fw.Allow()
	fw.Allow()

	time.Sleep(250 * time.Millisecond)

	for i := 0; i < 3; i++ {
		if !fw.Allow() {
			t.Errorf("expected Allow to succeed after reset on request %d", i+1)
		}
	}
}

// --- Boundary exploit test ---

// This test demonstrates the known weakness of fixed windows:
// a client can make 2x maxRequests by sending maxRequests just before
// the window ends and maxRequests immediately after it resets.
// This test documents the behavior rather than asserting it is wrong —
// it is an inherent property of the fixed window algorithm.
func TestFixedWindowBoundaryExploit(t *testing.T) {
	fw := NewFixedWindow(300*time.Millisecond, 5)

	// exhaust quota near end of window
	for i := 0; i < 5; i++ {
		fw.Allow()
	}

	time.Sleep(250 * time.Millisecond) // close to window boundary

	// wait for reset
	time.Sleep(100 * time.Millisecond)

	// immediately fire full quota again in new window
	granted := 0
	for i := 0; i < 5; i++ {
		if fw.Allow() {
			granted++
		}
	}

	// in a ~100ms span around the boundary, 10 requests were effectively allowed
	// this is expected behavior for fixed window — document it, don't fail on it
	t.Logf("boundary exploit: granted %d requests across window boundary (max per window: 5)", granted)
	if granted != 5 {
		t.Errorf("expected 5 grants in new window, got %d", granted)
	}
}

// --- Concurrency tests ---

func TestFixedWindowConcurrentAllowNoPanic(t *testing.T) {
	fw := NewFixedWindow(time.Second, 1000)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fw.Allow()
		}()
	}
	wg.Wait()
}

func TestFixedWindowConcurrentCountNeverExceedsMax(t *testing.T) {
	max := 50
	fw := NewFixedWindow(time.Second, max)
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fw.Allow()
		}()
	}
	wg.Wait()
	if fw.Count() > max {
		t.Errorf("count %d exceeded max %d", fw.Count(), max)
	}
}

func TestFixedWindowConcurrentGrantsNeverExceedMax(t *testing.T) {
	max := 50
	fw := NewFixedWindow(time.Second, max)
	var mu sync.Mutex
	granted := 0
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if fw.Allow() {
				mu.Lock()
				granted++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if granted > max {
		t.Errorf("granted %d requests but max was %d", granted, max)
	}
}
