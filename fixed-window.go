package main

import (
	"sync"
	"time"
)

type FixedWindow struct {
	windowSize  time.Duration
	maxRequests int
	count       int
	windowStart time.Time
	mu          sync.Mutex
}

func NewFixedWindow(windowSize time.Duration, maxRequests int) *FixedWindow {

	if windowSize <= 0 || maxRequests <= 0 {
		panic("windowSize must be greater than 0 and maxRequests must be greater than 0")
	}

	return &FixedWindow{
		windowSize:  windowSize,
		maxRequests: maxRequests,
		count:       0,
		windowStart: time.Now(),
	}
}

func (fw *FixedWindow) Allow() bool {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	curTime := time.Now()

	if curTime.Sub(fw.windowStart) >= fw.windowSize {
		fw.windowStart = curTime
		fw.count = 0
	}
	if fw.count < fw.maxRequests {
		fw.count++
		return true
	}
	return false
}

func (fw *FixedWindow) Count() int {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	return fw.count
}
