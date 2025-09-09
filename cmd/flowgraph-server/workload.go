package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	ch "github.com/flowgraph/flowgraph/internal/core/channel"
	pregel "github.com/flowgraph/flowgraph/internal/core/pregel"
)

type workloadManager struct {
	mu        sync.Mutex
	engCancel context.CancelFunc
	chCancel  context.CancelFunc
}

var wm workloadManager

func (m *workloadManager) startEngine(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.engCancel != nil {
		http.Error(w, "engine workload already running", http.StatusConflict)
		return
	}
	rate := 200 * time.Millisecond
	if v := r.URL.Query().Get("rate_ms"); v != "" {
		if ms, err := time.ParseDuration(v + "ms"); err == nil {
			rate = ms
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.engCancel = cancel
	go func() { runEngineLoop(ctx, rate) }()
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(w, "engine workload started at %v\n", rate)
}

func (m *workloadManager) stopEngine(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.engCancel != nil {
		m.engCancel()
		m.engCancel = nil
	}
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, "engine workload stopped\n")
}

func (m *workloadManager) startChannel(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.chCancel != nil {
		http.Error(w, "channel workload already running", http.StatusConflict)
		return
	}
	kind := r.URL.Query().Get("kind")
	if kind == "" {
		kind = "inmemory"
	}
	rate := 50 * time.Millisecond
	if v := r.URL.Query().Get("rate_ms"); v != "" {
		if ms, err := time.ParseDuration(v + "ms"); err == nil {
			rate = ms
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.chCancel = cancel
	go func() { runChannelLoop(ctx, kind, rate) }()
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(w, "channel workload started: kind=%s rate=%v\n", kind, rate)
}

func (m *workloadManager) stopChannel(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.chCancel != nil {
		m.chCancel()
		m.chCancel = nil
	}
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, "channel workload stopped\n")
}

// Reuse same helpers as example
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
		dir := filepath.Join(osTempDirSafe(), ".fg-metrics", fmt.Sprintf("data-%d", time.Now().UnixNano()))
		pc, err := ch.NewPersistentChannel(ch.PersistentChannelConfig{DataDir: dir, MaxSizeMB: 10, Timeout: time.Second})
		if err != nil {
			log.Printf("persistent channel error: %v", err)
			return
		}
		c = pc
	default:
		log.Printf("unknown channel kind: %s", kind)
		return
	}
	defer func() { _ = c.Close() }()
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

func osTempDirSafe() string { return filepath.Clean(filepath.FromSlash(os.TempDir())) }

// Optional JSON helpers (unused yet, reserved for future controls)
