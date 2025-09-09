package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "math/rand"
    "path/filepath"
    "time"

    ch "github.com/flowgraph/flowgraph/internal/core/channel"
    pregel "github.com/flowgraph/flowgraph/internal/core/pregel"
)

// benchVertex halts immediately; keeps engine lightweight while bumping metrics.
type benchVertex struct{}

func (b *benchVertex) Compute(vertexID string, state map[string]interface{}, messages []*pregel.Message) (map[string]interface{}, []*pregel.Message, bool, error) {
    return state, nil, true, nil
}

func runEngineLoop(ctx context.Context, hz time.Duration) {
    ticker := time.NewTicker(hz)
    defer ticker.Stop()
    verts := map[string]pregel.VertexProgram{"A": &benchVertex{}, "B": &benchVertex{}}
    states := map[string]map[string]interface{}{"A": {}, "B": {}}
    cfg := pregel.Config{MaxSupersteps: 1, Parallelism: 0, QueueCapacity: 256}
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            eng := pregel.NewEngine(verts, states, cfg)
            _ = eng.Run(ctx)
            eng.Stop()
        }
    }
}

func runChannelLoop(ctx context.Context, kind string, hz time.Duration) {
    ticker := time.NewTicker(hz)
    defer ticker.Stop()
    var c ch.Channel
    switch kind {
    case "inmemory":
        c = ch.DefaultInMemoryChannel()
    case "buffered":
        c = ch.DefaultBufferedChannel()
    case "persistent":
        dir := filepath.Join("./.metrics-load", fmt.Sprintf("data-%d", time.Now().UnixNano()))
        pc, err := ch.NewPersistentChannel(ch.PersistentChannelConfig{DataDir: dir, MaxSizeMB: 10, Timeout: time.Second})
        if err != nil { log.Printf("persistent channel error: %v", err); return }
        c = pc
    default:
        log.Printf("unknown channel kind: %s", kind)
        return
    }
    defer c.Close()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            msg := ch.Message{ID: fmt.Sprintf("m-%d", rand.Int63()), Type: ch.MessageTypeData, Source: "load", Target: "sink", Payload: rand.Int()}
            _ = c.Send(context.Background(), msg)
            _, _ = c.Receive(context.Background())
        }
    }
}

func main() {
    var (
        duration = flag.Duration("duration", time.Minute, "how long to run")
        engHz    = flag.Duration("engine-rate", 200*time.Millisecond, "engine tick rate")
        chHz     = flag.Duration("channel-rate", 50*time.Millisecond, "channel tick rate")
        chKind   = flag.String("channel", "inmemory", "channel kind: inmemory|buffered|persistent")
    )
    flag.Parse()

    ctx, cancel := context.WithTimeout(context.Background(), *duration)
    defer cancel()

    go runEngineLoop(ctx, *engHz)
    go runChannelLoop(ctx, *chKind, *chHz)

    <-ctx.Done()
    log.Println("metrics-load finished")
}

