package main

import (
	"sync"

	"github.com/leofengg/Raft/raft"
)

type Node struct {
	rf          *raft.Raft
	sm          *StateMachine
	responseMap map[int]chan bool // log index -> waiting handler
	mu          sync.Mutex
}
