package main

import (
	"sync"
	"time"
)

type SlidingWindow struct {
	windowSize  time.Duration
	maxRequests int
	timestamps  []time.Time
	mu          sync.Mutex
}

func NewSlidingWindow(windowSize time.Duration, maxRequests int) *SlidingWindow {

	if windowSize <= 0 || maxRequests <= 0 {
		panic("windowSize must be greater than 0 and maxRequests must be greater than 0")
	}

	return &SlidingWindow{
		windowSize:  windowSize,
		maxRequests: maxRequests,
		timestamps:  []time.Time{},
	}

}

func (sw *SlidingWindow) Allow() bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	curTime := time.Now()
	window := curTime.Add(-sw.windowSize)

	i := 0
	for i < len(sw.timestamps) && sw.timestamps[i].Before(window) {
		i++
	}
	sw.timestamps = sw.timestamps[i:]

	if len(sw.timestamps) < sw.maxRequests {
		sw.timestamps = append(sw.timestamps, curTime)
		return true
	}

	return false
}

func (sw *SlidingWindow) Count() int {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	return len(sw.timestamps)

}
