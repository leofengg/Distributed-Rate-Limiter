package main

import (
	"sync"
	"testing"
	"time"
)

// --- Initialization tests ---

func TestSlidingWindowNewStartsEmpty(t *testing.T) {
	sw := NewSlidingWindow(time.Second, 10)
	if sw.Count() != 0 {
		t.Errorf("expected 0 count on new window, got %d", sw.Count())
	}
}

func TestSlidingWindowInvalidWindowSize(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for windowSize <= 0")
		}
	}()
	NewSlidingWindow(0, 10)
}

func TestSlidingWindowInvalidMaxRequests(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for maxRequests <= 0")
		}
	}()
	NewSlidingWindow(time.Second, 0)
}

func TestSlidingWindowNegativeWindowSize(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for negative windowSize")
		}
	}()
	NewSlidingWindow(-time.Second, 10)
}

// --- Basic Allow() tests ---

func TestSlidingWindowAllowsUpToMax(t *testing.T) {
	sw := NewSlidingWindow(time.Second, 5)
	for i := 0; i < 5; i++ {
		if !sw.Allow() {
			t.Errorf("expected Allow to return true on request %d", i+1)
		}
	}
}

func TestSlidingWindowBlocksAfterMax(t *testing.T) {
	sw := NewSlidingWindow(time.Second, 3)
	sw.Allow()
	sw.Allow()
	sw.Allow()
	if sw.Allow() {
		t.Error("expected Allow to return false after reaching max requests")
	}
}

func TestSlidingWindowCountMatchesAllowed(t *testing.T) {
	sw := NewSlidingWindow(time.Second, 10)
	for i := 0; i < 7; i++ {
		sw.Allow()
	}
	if sw.Count() != 7 {
		t.Errorf("expected count of 7, got %d", sw.Count())
	}
}

func TestSlidingWindowBlockedRequestDoesNotIncrementCount(t *testing.T) {
	sw := NewSlidingWindow(time.Second, 2)
	sw.Allow()
	sw.Allow()
	sw.Allow() // this should be blocked
	if sw.Count() != 2 {
		t.Errorf("expected count to remain 2 after blocked request, got %d", sw.Count())
	}
}

// --- Sliding window behavior tests ---

func TestSlidingWindowAllowsAfterExpiry(t *testing.T) {
	sw := NewSlidingWindow(200*time.Millisecond, 2)
	sw.Allow()
	sw.Allow()

	if sw.Allow() {
		t.Fatal("expected third request to be blocked")
	}

	time.Sleep(250 * time.Millisecond) // wait for window to expire

	if !sw.Allow() {
		t.Error("expected Allow to succeed after window expiry")
	}
}

func TestSlidingWindowOldTimestampsAreEvicted(t *testing.T) {
	sw := NewSlidingWindow(200*time.Millisecond, 5)
	sw.Allow()
	sw.Allow()

	time.Sleep(250 * time.Millisecond) // both timestamps now outside window

	sw.Allow() // triggers eviction
	if sw.Count() != 1 {
		t.Errorf("expected count of 1 after eviction, got %d", sw.Count())
	}
}

func TestSlidingWindowPartialEviction(t *testing.T) {
	sw := NewSlidingWindow(300*time.Millisecond, 5)
	sw.Allow() // will expire
	sw.Allow() // will expire

	time.Sleep(200 * time.Millisecond)

	sw.Allow() // still within window
	sw.Allow() // still within window

	time.Sleep(150 * time.Millisecond) // first two now expired, last two still valid

	sw.Allow() // triggers eviction of first two
	if sw.Count() != 3 {
		t.Errorf("expected count of 3 after partial eviction, got %d", sw.Count())
	}
}

func TestSlidingWindowDoesNotResetLikeFixedWindow(t *testing.T) {
	// This test verifies sliding behavior vs fixed window behavior.
	// With a fixed window, requests at t=0 and t=window would both be fresh.
	// With sliding window, requests at t=0 should still count at t=window-epsilon.
	sw := NewSlidingWindow(300*time.Millisecond, 3)
	sw.Allow()
	sw.Allow()
	sw.Allow()

	time.Sleep(150 * time.Millisecond) // halfway through window

	// all 3 requests still within the sliding window, should be blocked
	if sw.Allow() {
		t.Error("expected request to be blocked — old requests still within sliding window")
	}
}

// --- Concurrency tests ---

func TestSlidingWindowConcurrentAllowNoPanic(t *testing.T) {
	sw := NewSlidingWindow(time.Second, 1000)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sw.Allow()
		}()
	}
	wg.Wait()
}

func TestSlidingWindowConcurrentCountNeverExceedsMax(t *testing.T) {
	max := 50
	sw := NewSlidingWindow(time.Second, max)
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sw.Allow()
		}()
	}
	wg.Wait()
	if sw.Count() > max {
		t.Errorf("count %d exceeded max %d", sw.Count(), max)
	}
}

func TestSlidingWindowConcurrentGrantsNeverExceedMax(t *testing.T) {
	max := 50
	sw := NewSlidingWindow(time.Second, max)
	var mu sync.Mutex
	granted := 0
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if sw.Allow() {
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
