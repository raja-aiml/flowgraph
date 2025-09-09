package pregel

import (
    "context"
    "math"
    "runtime"
    "testing"

    "github.com/stretchr/testify/assert"
)

// Ensure scheduler defaults to >=1 workers and uses NumCPU when input <=0
func TestNewWorkStealingScheduler_Defaults(t *testing.T) {
    ws := NewWorkStealingScheduler(0, 100)
    assert.GreaterOrEqual(t, ws.numWorkers, 1)
    assert.Equal(t, runtime.NumCPU(), ws.numWorkers)

    ws2 := NewWorkStealingScheduler(-5, 50)
    assert.GreaterOrEqual(t, ws2.numWorkers, 1)
}

// Ensure engine runs with zero/negative parallelism by defaulting to NumCPU
func TestEngine_ParallelismGuards(t *testing.T) {
    vertices := map[string]VertexProgram{
        "A": &nopVertex{},
    }
    initial := map[string]map[string]interface{}{
        "A": {},
    }
    cfg := Config{MaxSupersteps: 1, Parallelism: 0}
    eng := NewEngine(vertices, initial, cfg)
    assert.GreaterOrEqual(t, eng.Config.Parallelism, 1)
    assert.Equal(t, runtime.NumCPU(), eng.Config.Parallelism)
    assert.NoError(t, eng.Run(context.Background()))

    cfg2 := Config{MaxSupersteps: 1, Parallelism: -10}
    eng2 := NewEngine(vertices, initial, cfg2)
    assert.GreaterOrEqual(t, eng2.Config.Parallelism, 1)
    assert.NoError(t, eng2.Run(context.Background()))
}

type nopVertex struct{}

func (n *nopVertex) Compute(vertexID string, state map[string]interface{}, messages []*Message) (map[string]interface{}, []*Message, bool, error) {
    return state, nil, true, nil
}

// Ensure ParallelismFactor scales workers when Parallelism not set.
func TestEngine_ParallelismFactor(t *testing.T) {
    vertices := map[string]VertexProgram{"A": &nopVertex{}}
    initial := map[string]map[string]interface{}{"A": {}}
    factor := 1.5
    cfg := Config{MaxSupersteps: 1, Parallelism: 0, ParallelismFactor: factor}
    eng := NewEngine(vertices, initial, cfg)
    expected := int(math.Ceil(factor * float64(runtime.NumCPU())))
    if expected < 1 { expected = 1 }
    stats := eng.Stats()
    assert.Equal(t, expected, stats.Workers)
}
