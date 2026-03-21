package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

type Command struct {
	UserID string
	Cost   float64
}

type StateMachine struct {
	limiters   map[string]*TokenBucket
	mu         sync.Mutex
	maxTokens  float64
	refillRate float64
}

func NewStateMachine(maxTokens, refillRate float64) *StateMachine {

	return &StateMachine{
		limiters:   make(map[string]*TokenBucket),
		maxTokens:  maxTokens,
		refillRate: refillRate,
	}
}

func (sm *StateMachine) Apply(cmd Command) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	_, ok := sm.limiters[cmd.UserID]
	if !ok {
		sm.limiters[cmd.UserID] = NewTokenBucket(sm.maxTokens, sm.refillRate)
	}

	return sm.limiters[cmd.UserID].Allow(cmd.Cost)
}

func SerializeCommand(userID string, cost float64) string {
	return fmt.Sprintf("allow:%s:%f", userID, cost)
}

func ParseCommand(cmd string) (Command, error) {

	substrings := strings.Split(cmd, ":")
	if len(substrings) != 3 || substrings[0] != "allow" {
		return Command{}, fmt.Errorf("invalid command format")
	}

	if substrings[1] == "" {
		return Command{}, fmt.Errorf("userID cannot be empty")
	}

	cost, err := strconv.ParseFloat(substrings[2], 64)
	if err != nil {
		return Command{}, fmt.Errorf("invalid cost value: %v", err)
	}

	return Command{
		UserID: substrings[1],
		Cost:   cost,
	}, nil

}
