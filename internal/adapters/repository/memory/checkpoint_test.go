package adapters

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/flowgraph/flowgraph/internal/core/checkpoint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemorySaver_BasicOperations(t *testing.T) {
	ctx := context.Background()
	saver := DefaultInMemorySaver()
	defer func() { _ = saver.Close() }()

	// Create test checkpoint
	cp := &checkpoint.Checkpoint{
		ID:       "test-1",
		GraphID:  "graph-1",
		ThreadID: "thread-1",
		State: map[string]interface{}{
			"key": "value",
		},
		Timestamp: time.Now(),
	}

	t.Run("Save checkpoint", func(t *testing.T) {
		err := saver.Save(ctx, cp)
		require.NoError(t, err)
	})

	t.Run("Load checkpoint", func(t *testing.T) {
		loaded, err := saver.Load(ctx, "test-1")
		require.NoError(t, err)
		assert.Equal(t, cp.ID, loaded.ID)
		assert.Equal(t, cp.GraphID, loaded.GraphID)
		assert.Equal(t, cp.ThreadID, loaded.ThreadID)
		assert.Equal(t, cp.State, loaded.State)
	})

	t.Run("Delete checkpoint", func(t *testing.T) {
		err := saver.Delete(ctx, "test-1")
		require.NoError(t, err)

		// Verify deletion
		_, err = saver.Load(ctx, "test-1")
		assert.ErrorIs(t, err, checkpoint.ErrCheckpointNotFound)
	})
}

func TestInMemorySaver_TTL(t *testing.T) {
	ctx := context.Background()

	// Create saver with short TTL for testing
	config := InMemoryConfig{
		DefaultTTL:      100 * time.Millisecond,
		CleanupInterval: 50 * time.Millisecond,
	}
	saver := NewInMemorySaver(config)
	defer func() { _ = saver.Close() }()

	cp := &checkpoint.Checkpoint{
		ID:        "ttl-test",
		GraphID:   "graph-1",
		ThreadID:  "thread-1",
		State:     map[string]interface{}{"key": "value"},
		Timestamp: time.Now(),
	}

	// Save checkpoint
	err := saver.Save(ctx, cp)
	require.NoError(t, err)

	// Should be available immediately
	_, err = saver.Load(ctx, "ttl-test")
	require.NoError(t, err)

	// Wait for TTL to expire
	time.Sleep(200 * time.Millisecond)

	// Should be expired and cleaned up
	_, err = saver.Load(ctx, "ttl-test")
	assert.ErrorIs(t, err, checkpoint.ErrCheckpointNotFound)
}

func TestInMemorySaver_MemoryManagement(t *testing.T) {
	ctx := context.Background()

	// Create saver with very small memory limit
	config := InMemoryConfig{
		MaxMemoryMB: 1, // 1MB limit
	}
	saver := NewInMemorySaver(config)
	defer func() { _ = saver.Close() }()

	// Create large checkpoints to exceed memory limit
	largeData := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largeData[fmt.Sprintf("key%d", i)] = string(make([]byte, 1024)) // 1KB each
	}

	// Save several large checkpoints
	checkpoints := make([]*checkpoint.Checkpoint, 5)
	for i := 0; i < 5; i++ {
		checkpoints[i] = &checkpoint.Checkpoint{
			ID:        fmt.Sprintf("large-%d", i),
			GraphID:   "graph-1",
			ThreadID:  fmt.Sprintf("thread-%d", i),
			State:     largeData,
			Timestamp: time.Now(),
		}

		err := saver.Save(ctx, checkpoints[i])
		if i < 2 {
			require.NoError(t, err)
		}
		// Later saves might trigger LRU eviction

		time.Sleep(10 * time.Millisecond) // Ensure different access times
	}

	// Check memory stats
	stats := saver.GetStats()
	assert.True(t, stats.SizeMB <= config.MaxMemoryMB+1) // Allow small buffer for overhead
	assert.True(t, stats.Count > 0)
	assert.True(t, stats.UtilizationPercent <= 110) // Allow small buffer
}

func TestInMemorySaver_ConcurrentOperations(t *testing.T) {
	ctx := context.Background()
	saver := DefaultInMemorySaver()
	defer func() { _ = saver.Close() }()

	const numGoroutines = 10
	const operationsPerGoroutine = 100

	var wg sync.WaitGroup

	// Concurrent saves
	t.Run("Concurrent saves", func(t *testing.T) {
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(routineID int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					cp := &checkpoint.Checkpoint{
						ID:       fmt.Sprintf("concurrent-%d-%d", routineID, j),
						GraphID:  fmt.Sprintf("graph-%d", routineID),
						ThreadID: fmt.Sprintf("thread-%d-%d", routineID, j),
						State: map[string]interface{}{
							"routine": routineID,
							"op":      j,
						},
						Timestamp: time.Now(),
					}

					err := saver.Save(ctx, cp)
					assert.NoError(t, err)
				}
			}(i)
		}

		wg.Wait()
	})

	// Concurrent reads
	t.Run("Concurrent reads", func(t *testing.T) {
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(routineID int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					id := fmt.Sprintf("concurrent-%d-%d", routineID, j)
					_, err := saver.Load(ctx, id)
					// Some might not exist due to LRU eviction, that's OK
					if err != nil {
						assert.ErrorIs(t, err, checkpoint.ErrCheckpointNotFound)
					}
				}
			}(i)
		}

		wg.Wait()
	})
}

func TestInMemorySaver_List(t *testing.T) {
	ctx := context.Background()
	saver := DefaultInMemorySaver()
	defer func() { _ = saver.Close() }()

	// Create test checkpoints
	checkpoints := []*checkpoint.Checkpoint{
		{
			ID:        "list-1",
			GraphID:   "graph-1",
			ThreadID:  "thread-1",
			State:     map[string]interface{}{"index": 1},
			Timestamp: time.Now().Add(-2 * time.Hour),
		},
		{
			ID:        "list-2",
			GraphID:   "graph-1",
			ThreadID:  "thread-2",
			State:     map[string]interface{}{"index": 2},
			Timestamp: time.Now().Add(-1 * time.Hour),
		},
		{
			ID:        "list-3",
			GraphID:   "graph-2",
			ThreadID:  "thread-3",
			State:     map[string]interface{}{"index": 3},
			Timestamp: time.Now(),
		},
	}

	// Save all checkpoints
	for _, cp := range checkpoints {
		err := saver.Save(ctx, cp)
		require.NoError(t, err)
	}

	t.Run("List all checkpoints", func(t *testing.T) {
		filter := checkpoint.Filter{}
		list, err := saver.List(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, list, 3)
	})

	t.Run("Filter by GraphID", func(t *testing.T) {
		filter := checkpoint.Filter{GraphID: "graph-1"}
		list, err := saver.List(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, list, 2)

		for _, cp := range list {
			assert.Equal(t, "graph-1", cp.GraphID)
		}
	})

	t.Run("Filter by ThreadID", func(t *testing.T) {
		filter := checkpoint.Filter{ThreadID: "thread-2"}
		list, err := saver.List(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, list, 1)
		assert.Equal(t, "list-2", list[0].ID)
	})

	t.Run("Filter with Since", func(t *testing.T) {
		since := time.Now().Add(-30 * time.Minute)
		filter := checkpoint.Filter{Since: &since}
		list, err := saver.List(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, list, 1)
		assert.Equal(t, "list-3", list[0].ID)
	})

	t.Run("Filter with Limit", func(t *testing.T) {
		filter := checkpoint.Filter{Limit: 2}
		list, err := saver.List(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, list, 2)
	})

	t.Run("Filter with Offset", func(t *testing.T) {
		filter := checkpoint.Filter{Offset: 1, Limit: 2}
		list, err := saver.List(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, list, 2)
	})
}

func TestInMemorySaver_ErrorHandling(t *testing.T) {
	ctx := context.Background()
	saver := DefaultInMemorySaver()
	defer func() { _ = saver.Close() }()

	t.Run("Save invalid checkpoint", func(t *testing.T) {
		cp := &checkpoint.Checkpoint{
			ID: "", // Invalid: empty ID
		}
		err := saver.Save(ctx, cp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})

	t.Run("Load non-existent checkpoint", func(t *testing.T) {
		_, err := saver.Load(ctx, "non-existent")
		assert.ErrorIs(t, err, checkpoint.ErrCheckpointNotFound)
	})

	t.Run("Delete non-existent checkpoint", func(t *testing.T) {
		err := saver.Delete(ctx, "non-existent")
		assert.ErrorIs(t, err, checkpoint.ErrCheckpointNotFound)
	})

	t.Run("List with invalid filter", func(t *testing.T) {
		filter := checkpoint.Filter{Limit: -1} // Invalid limit
		_, err := saver.List(ctx, filter)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})
}

// TestInMemorySaver_EdgeCases tests TTL expiration to increase coverage
func TestInMemorySaver_EdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("TTL expiration", func(t *testing.T) {
		config := InMemoryConfig{
			DefaultTTL:      50 * time.Millisecond, // Very short TTL
			CleanupInterval: 10 * time.Millisecond, // Frequent cleanup
		}
		saver := NewInMemorySaver(config)
		defer func() { _ = saver.Close() }()

		cp := &checkpoint.Checkpoint{
			ID:        "ttl-test",
			GraphID:   "graph-1",
			ThreadID:  "thread-1",
			State:     map[string]interface{}{"data": "test"},
			Timestamp: time.Now(),
		}

		err := saver.Save(ctx, cp)
		require.NoError(t, err)

		// Initially accessible
		_, err = saver.Load(ctx, "ttl-test")
		assert.NoError(t, err)

		// Wait for TTL to expire and cleanup to run
		time.Sleep(100 * time.Millisecond)

		// Should be expired and cleaned up
		_, err = saver.Load(ctx, "ttl-test")
		assert.Error(t, err)
	})

	t.Run("isExpired coverage", func(t *testing.T) {
		saver := DefaultInMemorySaver()
		defer func() { _ = saver.Close() }()

		cp := &checkpoint.Checkpoint{
			ID:        "expire-test",
			GraphID:   "graph-1",
			ThreadID:  "thread-1",
			State:     map[string]interface{}{"data": "test"},
			Timestamp: time.Now(),
		}

		err := saver.Save(ctx, cp)
		require.NoError(t, err)

		// Load to test isExpired check in Load function
		_, err = saver.Load(ctx, "expire-test")
		assert.NoError(t, err)

		// List to test isExpired check in List function
		filter := checkpoint.Filter{}
		list, err := saver.List(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, list, 1)
	})
}

func TestInMemorySaver_Stats(t *testing.T) {
	ctx := context.Background()

	config := InMemoryConfig{
		MaxMemoryMB: 100,
	}
	saver := NewInMemorySaver(config)
	defer func() { _ = saver.Close() }()

	// Initially empty
	stats := saver.GetStats()
	assert.Equal(t, int64(0), stats.Count)
	assert.Equal(t, int64(0), stats.SizeMB)
	assert.Equal(t, int64(100), stats.MaxSizeMB)
	assert.Equal(t, float64(0), stats.UtilizationPercent)

	// Add some checkpoints
	for i := 0; i < 5; i++ {
		cp := &checkpoint.Checkpoint{
			ID:        fmt.Sprintf("stats-%d", i),
			GraphID:   "graph-1",
			ThreadID:  fmt.Sprintf("thread-%d", i),
			State:     map[string]interface{}{"data": string(make([]byte, 1000))},
			Timestamp: time.Now(),
		}
		err := saver.Save(ctx, cp)
		require.NoError(t, err)
	}

	// Check updated stats
	stats = saver.GetStats()
	assert.Equal(t, int64(5), stats.Count)
	assert.True(t, stats.SizeMB > 0)
	assert.True(t, stats.UtilizationPercent > 0)
	assert.True(t, stats.UtilizationPercent < 100)
}

func BenchmarkInMemorySaver_Save(b *testing.B) {
	ctx := context.Background()
	saver := DefaultInMemorySaver()
	defer func() { _ = saver.Close() }()

	cp := &checkpoint.Checkpoint{
		ID:       "benchmark",
		GraphID:  "graph-1",
		ThreadID: "thread-1",
		State: map[string]interface{}{
			"data": "benchmark data",
		},
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cp.ID = fmt.Sprintf("benchmark-%d", i)
		_ = saver.Save(ctx, cp)
	}
}

func BenchmarkInMemorySaver_Load(b *testing.B) {
	ctx := context.Background()
	saver := DefaultInMemorySaver()
	defer func() { _ = saver.Close() }()

	// Pre-populate with data
	for i := 0; i < 1000; i++ {
		cp := &checkpoint.Checkpoint{
			ID:        fmt.Sprintf("benchmark-%d", i),
			GraphID:   "graph-1",
			ThreadID:  fmt.Sprintf("thread-%d", i),
			State:     map[string]interface{}{"data": "benchmark data"},
			Timestamp: time.Now(),
		}
		_ = saver.Save(ctx, cp)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := fmt.Sprintf("benchmark-%d", i%1000)
		_, _ = saver.Load(ctx, id)
	}
}

func BenchmarkInMemorySaver_ConcurrentAccess(b *testing.B) {
	ctx := context.Background()
	saver := DefaultInMemorySaver()
	defer func() { _ = saver.Close() }()

	// Pre-populate
	for i := 0; i < 100; i++ {
		cp := &checkpoint.Checkpoint{
			ID:        fmt.Sprintf("concurrent-%d", i),
			GraphID:   "graph-1",
			ThreadID:  fmt.Sprintf("thread-%d", i),
			State:     map[string]interface{}{"data": "test"},
			Timestamp: time.Now(),
		}
		_ = saver.Save(ctx, cp)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			id := fmt.Sprintf("concurrent-%d", i%100)
			_, _ = saver.Load(ctx, id)
			i++
		}
	})
}
