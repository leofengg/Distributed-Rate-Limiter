package main

import (
	"sync"
	"time"
)

type TokenBucket struct {
	tokens         float64
	maxTokens      float64
	refillRate     float64
	lastRefillTime time.Time
	mu             sync.Mutex
}

func NewTokenBucket(maxTokens, refillRate float64) *TokenBucket {

	if maxTokens <= 0 || refillRate <= 0 {
		panic("maxTokens and refillRate must be greater than 0")
	}

	return &TokenBucket{
		tokens:         maxTokens,
		maxTokens:      maxTokens,
		refillRate:     refillRate,
		lastRefillTime: time.Now(),
	}

}

func (b *TokenBucket) Allow(cost float64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.refill()

	if b.tokens >= cost {
		b.tokens -= cost
		return true
	}

	return false

}

func (b *TokenBucket) refill() {

	now := time.Now()
	elapsed := now.Sub(b.lastRefillTime).Seconds()
	b.tokens += elapsed * b.refillRate
	b.lastRefillTime = now
	if b.tokens > b.maxTokens {
		b.tokens = b.maxTokens
	}
}

func (b *TokenBucket) Tokens() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.refill()
	return b.tokens
}
