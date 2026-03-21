package main

import (
	"sync"
	"testing"
	"time"
)

// --- Initialization tests ---

func TestNewBucketStartsFull(t *testing.T) {
	b := NewTokenBucket(10, 1)
	if b.Tokens() != 10 {
		t.Errorf("expected 10 tokens, got %f", b.Tokens())
	}
}

func TestNewBucketInvalidMaxTokens(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for maxTokens <= 0")
		}
	}()
	NewTokenBucket(0, 1)
}

func TestNewBucketInvalidRefillRate(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for refillRate <= 0")
		}
	}()
	NewTokenBucket(10, 0)
}

func TestNewBucketNegativeMaxTokens(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for negative maxTokens")
		}
	}()
	NewTokenBucket(-5, 1)
}

// --- Basic Allow() tests ---

func TestAllowDepletesTokens(t *testing.T) {
	b := NewTokenBucket(5, 1)
	for i := 0; i < 5; i++ {
		if !b.Allow(1) {
			t.Errorf("expected Allow to return true on request %d", i+1)
		}
	}
	if b.Tokens() != 0 {
		t.Errorf("expected 0 tokens after 5 requests, got %f", b.Tokens())
	}
}

func TestAllowReturnsFalseWhenEmpty(t *testing.T) {
	b := NewTokenBucket(2, 1)
	b.Allow(1)
	b.Allow(1)
	if b.Allow(1) {
		t.Error("expected Allow to return false when bucket is empty")
	}
}

func TestAllowWithCostGreaterThanOne(t *testing.T) {
	b := NewTokenBucket(10, 1)
	if !b.Allow(5) {
		t.Error("expected Allow(5) to succeed with 10 tokens")
	}
	if b.Tokens() != 5 {
		t.Errorf("expected 5 tokens remaining, got %f", b.Tokens())
	}
}

func TestAllowWithCostExceedingTokens(t *testing.T) {
	b := NewTokenBucket(3, 1)
	if b.Allow(5) {
		t.Error("expected Allow(5) to fail with only 3 tokens")
	}
	// tokens should be unchanged after a failed Allow
	if b.Tokens() != 3 {
		t.Errorf("expected tokens to remain 3 after failed Allow, got %f", b.Tokens())
	}
}

func TestAllowWithZeroCost(t *testing.T) {
	b := NewTokenBucket(5, 1)
	if !b.Allow(0) {
		t.Error("expected Allow(0) to always succeed")
	}
	if b.Tokens() != 5 {
		t.Error("expected tokens to be unchanged after Allow(0)")
	}
}

// --- Refill tests ---

func TestRefillAfterElapsedTime(t *testing.T) {
	b := NewTokenBucket(10, 10) // refills 10 tokens/sec
	b.Allow(10)                 // drain completely

	if b.Tokens() != 0 {
		t.Fatalf("expected 0 tokens after drain, got %f", b.Tokens())
	}

	time.Sleep(500 * time.Millisecond) // should refill ~5 tokens

	tokens := b.Tokens()
	if tokens < 4.5 || tokens > 5.5 {
		t.Errorf("expected ~5 tokens after 500ms, got %f", tokens)
	}
}

func TestRefillDoesNotExceedMax(t *testing.T) {
	b := NewTokenBucket(5, 100) // very fast refill rate
	b.Allow(1)

	time.Sleep(200 * time.Millisecond)

	if b.Tokens() > 5 {
		t.Errorf("tokens exceeded maxTokens: got %f", b.Tokens())
	}
}

func TestFullRefillAllowsRequests(t *testing.T) {
	b := NewTokenBucket(5, 10) // 10 tokens/sec
	b.Allow(5)                 // drain

	time.Sleep(600 * time.Millisecond) // refill ~6 tokens, capped at 5

	for i := 0; i < 5; i++ {
		if !b.Allow(1) {
			t.Errorf("expected Allow to succeed after refill, failed on request %d", i+1)
		}
	}
}

// --- Concurrency tests ---

func TestConcurrentAllowNoPanic(t *testing.T) {
	b := NewTokenBucket(1000, 100)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.Allow(1)
		}()
	}
	wg.Wait()
	// if we get here without a race condition or panic, test passes
}

func TestConcurrentAllowTokensNeverNegative(t *testing.T) {
	b := NewTokenBucket(50, 1)
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.Allow(1)
		}()
	}
	wg.Wait()
	if b.Tokens() < 0 {
		t.Errorf("tokens went negative: %f", b.Tokens())
	}
}

func TestConcurrentAllowTotalGrantedNeverExceedsInitial(t *testing.T) {
	maxTokens := float64(100)
	b := NewTokenBucket(maxTokens, 0.0001) // near-zero refill so we can count cleanly

	var mu sync.Mutex
	granted := 0
	var wg sync.WaitGroup

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if b.Allow(1) {
				mu.Lock()
				granted++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if float64(granted) > maxTokens {
		t.Errorf("granted %d requests but max tokens was %f", granted, maxTokens)
	}
}
