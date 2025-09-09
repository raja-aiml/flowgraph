package pregel

import (
    "context"
    "runtime"
    "testing"
)

type benchVertex struct{}

func (b *benchVertex) Compute(vertexID string, state map[string]interface{}, messages []*Message) (map[string]interface{}, []*Message, bool, error) {
    return state, nil, true, nil
}

func BenchmarkSchedulerThroughput(b *testing.B) {
    ws := NewWorkStealingScheduler(runtime.NumCPU(), 1024)
    ws.StartWorkers()
    defer ws.Stop()

    res := make(chan VertexResult, 1024)
    prog := &benchVertex{}
    task := VertexTask{VertexID: "A", Program: prog, State: map[string]interface{}{}, Messages: nil, Result: res}

    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        ws.Schedule(task)
        <-res
    }
}

func BenchmarkEngineRun(b *testing.B) {
    verts := map[string]VertexProgram{
        "A": &benchVertex{},
        "B": &benchVertex{},
        "C": &benchVertex{},
        "D": &benchVertex{},
    }
    states := map[string]map[string]interface{}{
        "A": {}, "B": {}, "C": {}, "D": {},
    }
    cfg := Config{MaxSupersteps: 1, Parallelism: runtime.NumCPU(), QueueCapacity: 1024}
    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        eng := NewEngine(verts, states, cfg)
        _ = eng.Run(context.Background())
        eng.Stop()
    }
}

