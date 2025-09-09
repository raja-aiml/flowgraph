package pregel

import (
	"sync"
	"time"
)

// Superstep manages synchronization barriers for BSP model
// PRINCIPLES:
// - Bulk Synchronous Parallel (BSP) barrier
// - Aggregates messages and coordinates vertex execution

type Superstep struct {
	StepNumber     int
	Aggregator     *MessageAggregator
	Barrier        *sync.WaitGroup
	ActiveVertices map[string]bool
	StartTime      time.Time
	EndTime        time.Time
	mu             sync.RWMutex
}

func NewSuperstep(step int, combiner MessageCombiner) *Superstep {
	return &Superstep{
		StepNumber:     step,
		Aggregator:     NewMessageAggregator(combiner),
		Barrier:        &sync.WaitGroup{},
		ActiveVertices: make(map[string]bool),
		StartTime:      time.Now(),
	}
}

func (s *Superstep) AddVertex(vertexID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ActiveVertices[vertexID] = true
	s.Barrier.Add(1)
}

func (s *Superstep) CompleteVertex(vertexID string, halt bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ActiveVertices[vertexID] = !halt
	s.Barrier.Done()
}

func (s *Superstep) Wait() {
	s.Barrier.Wait()
	s.EndTime = time.Now()
}

func (s *Superstep) Duration() time.Duration {
	return s.EndTime.Sub(s.StartTime)
}

func (s *Superstep) HasActiveVertices() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, active := range s.ActiveVertices {
		if active {
			return true
		}
	}
	return false
}

func (s *Superstep) GetActiveVertexCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, active := range s.ActiveVertices {
		if active {
			count++
		}
	}
	return count
}

func (s *Superstep) GetActiveVertices() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var vertices []string
	for vid, active := range s.ActiveVertices {
		if active {
			vertices = append(vertices, vid)
		}
	}
	return vertices
}

func (s *Superstep) IsComplete() bool {
	return !s.EndTime.IsZero()
}

func (s *Superstep) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Barrier = &sync.WaitGroup{}
	s.ActiveVertices = make(map[string]bool)
	s.StartTime = time.Now()
	s.EndTime = time.Time{}
	s.Aggregator.Clear()
}

func (s *Superstep) GetMessageCount() int {
	return s.Aggregator.GetMessageCount()
}
