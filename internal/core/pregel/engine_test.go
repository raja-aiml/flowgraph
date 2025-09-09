package pregel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type simpleVertex struct{}

func (s *simpleVertex) Compute(vertexID string, state map[string]interface{}, messages []*Message) (map[string]interface{}, []*Message, bool, error) {
	// Increment state counter and send message to next vertex
	count := 0
	if v, ok := state["count"].(int); ok {
		count = v
	}
	count++
	state["count"] = count
	msgs := []*Message{}
	halt := false
	if next, ok := state["next"].(string); ok && count < 3 {
		msgs = append(msgs, &Message{From: vertexID, To: next, Value: count, Step: 0})
	} else {
		halt = true
	}
	return state, msgs, halt, nil
}

func TestPregelEngine_Run(t *testing.T) {
	vertices := map[string]VertexProgram{
		"A": &simpleVertex{},
		"B": &simpleVertex{},
	}
	initialStates := map[string]map[string]interface{}{
		"A": {"count": 0, "next": "B"},
		"B": {"count": 0, "next": "A"},
	}
	config := Config{
		MaxSupersteps: 4,
		Parallelism:   2,
	}
	engine := NewEngine(vertices, initialStates, config)
	ctx := context.Background()
	assert.NoError(t, engine.Run(ctx))
	assert.GreaterOrEqual(t, engine.VertexStates["A"]["count"].(int), 2)
	assert.GreaterOrEqual(t, engine.VertexStates["B"]["count"].(int), 2)
}
