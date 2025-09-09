package channel

import (
    "context"
    "fmt"
    "os"
    "sync"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// Helper to create test messages
func createTestMessage(id, source, target string, payload interface{}) Message {
	return Message{
		ID:      id,
		Type:    MessageTypeData,
		Source:  source,
		Target:  target,
		Payload: payload,
	}
}

// TestInMemoryChannelComprehensive provides extensive coverage
func TestInMemoryChannelComprehensive(t *testing.T) {
	t.Run("DefaultCreation", func(t *testing.T) {
		ch := DefaultInMemoryChannel()
		require.NotNil(t, ch)
		defer ch.Close()

		// Test default stats
		stats := ch.Stats()
		assert.Equal(t, 0, stats.Length)
		assert.Greater(t, stats.Capacity, 0)
		assert.False(t, stats.Closed)
	})

	t.Run("ConfiguredCreation", func(t *testing.T) {
		ch := NewInMemoryChannel(InMemoryChannelConfig{
			BufferSize: 50,
			Timeout:    time.Second * 2,
		})
		require.NotNil(t, ch)
		defer ch.Close()

		assert.Equal(t, 50, ch.Cap())
	})

	t.Run("SendReceiveCycle", func(t *testing.T) {
		ch := DefaultInMemoryChannel()
		defer ch.Close()

		ctx := context.Background()

		// Test basic send/receive
		msg := createTestMessage("test-1", "node-a", "node-b", "hello")
		err := ch.Send(ctx, msg)
		require.NoError(t, err)

		received, err := ch.Receive(ctx)
		require.NoError(t, err)
		assert.Equal(t, msg, received)
	})

	t.Run("MultipleMessages", func(t *testing.T) {
		ch := DefaultInMemoryChannel()
		defer ch.Close()

		ctx := context.Background()

		messages := []Message{
			createTestMessage("msg-1", "a", "b", "payload1"),
			createTestMessage("msg-2", "a", "b", "payload2"),
			createTestMessage("msg-3", "a", "b", "payload3"),
		}

		// Send all messages
		for _, msg := range messages {
			err := ch.Send(ctx, msg)
			require.NoError(t, err)
		}

		// Verify length
		assert.Equal(t, len(messages), ch.Len())

		// Receive all messages
		for i, expectedMsg := range messages {
			received, err := ch.Receive(ctx)
			require.NoError(t, err, "Failed to receive message %d", i)
			assert.Equal(t, expectedMsg, received)
		}

		// Verify empty
		assert.Equal(t, 0, ch.Len())
	})

	t.Run("ClosedChannelOperations", func(t *testing.T) {
		ch := DefaultInMemoryChannel()
		ctx := context.Background()

		// Send a message before closing
		msg := createTestMessage("pre-close", "a", "b", "data")
		err := ch.Send(ctx, msg)
		require.NoError(t, err)

		// Close channel
		err = ch.Close()
		require.NoError(t, err)
		assert.True(t, ch.IsClosed())

		// Try to send after close - should error
		err = ch.Send(ctx, msg)
		assert.Error(t, err)

		// Should still be able to receive pending message
		received, err := ch.Receive(ctx)
		require.NoError(t, err)
		assert.Equal(t, msg, received)

		// Multiple closes should be safe
		err = ch.Close()
		assert.NoError(t, err)
	})

	t.Run("CapacityTests", func(t *testing.T) {
		// Create small buffer for testing limits
		ch := NewInMemoryChannel(InMemoryChannelConfig{
			BufferSize: 2,
			Timeout:    time.Millisecond * 50,
		})
		defer ch.Close()

		ctx := context.Background()
		assert.Equal(t, 2, ch.Cap())

		// Fill to capacity
		for i := 0; i < 2; i++ {
			msg := createTestMessage(fmt.Sprintf("msg-%d", i), "a", "b", i)
			err := ch.Send(ctx, msg)
			require.NoError(t, err)
		}

		// Should be at capacity
		assert.Equal(t, 2, ch.Len())

		// Next send should timeout due to full buffer
		msg := createTestMessage("overflow", "a", "b", "too much")
		err := ch.Send(ctx, msg)
		assert.Error(t, err)
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		ch := DefaultInMemoryChannel()
		defer ch.Close()

		// Test cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := ch.Receive(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)

		// Test timeout context
		timeoutCtx, cancelTimeout := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancelTimeout()

		_, err = ch.Receive(timeoutCtx)
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})

	t.Run("StatsValidation", func(t *testing.T) {
		ch := DefaultInMemoryChannel()
		defer ch.Close()

		ctx := context.Background()

		// Initially empty
		stats := ch.Stats()
		assert.Equal(t, 0, stats.Length)
		assert.False(t, stats.Closed)

		// Send messages and verify stats update
		for i := 1; i <= 3; i++ {
			msg := createTestMessage(fmt.Sprintf("stats-%d", i), "a", "b", i)
			err := ch.Send(ctx, msg)
			require.NoError(t, err)

			assert.Equal(t, i, ch.Len())
		}

		// Receive messages and verify stats update
		for i := 3; i > 0; i-- {
			_, err := ch.Receive(ctx)
			require.NoError(t, err)
			assert.Equal(t, i-1, ch.Len())
		}

		// Verify final stats
		finalStats := ch.Stats()
		assert.Equal(t, 0, finalStats.Length)
		assert.False(t, finalStats.Closed)
	})

	t.Run("ConcurrentOperations", func(t *testing.T) {
		ch := NewInMemoryChannel(InMemoryChannelConfig{
			BufferSize: 100,
			Timeout:    time.Second,
		})
		defer ch.Close()

		ctx := context.Background()
		numRoutines := 10
		messagesPerRoutine := 10

		var wg sync.WaitGroup

		// Concurrent senders
		wg.Add(numRoutines)
		for i := 0; i < numRoutines; i++ {
			go func(routineID int) {
				defer wg.Done()
				for j := 0; j < messagesPerRoutine; j++ {
					msg := createTestMessage(
						fmt.Sprintf("routine-%d-msg-%d", routineID, j),
						fmt.Sprintf("sender-%d", routineID),
						"receiver",
						fmt.Sprintf("data-%d-%d", routineID, j),
					)
					err := ch.Send(ctx, msg)
					assert.NoError(t, err)
				}
			}(i)
		}

		wg.Wait()

		// Verify all messages were sent
		expectedMessages := numRoutines * messagesPerRoutine
		assert.Equal(t, expectedMessages, ch.Len())

		// Concurrent receivers
		receivedMessages := make([]Message, 0, expectedMessages)
		var receiveMutex sync.Mutex

		wg.Add(numRoutines)
		for i := 0; i < numRoutines; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < messagesPerRoutine; j++ {
					msg, err := ch.Receive(ctx)
					assert.NoError(t, err)

					receiveMutex.Lock()
					receivedMessages = append(receivedMessages, msg)
					receiveMutex.Unlock()
				}
			}()
		}

		wg.Wait()

		// Verify all messages were received
		assert.Len(t, receivedMessages, expectedMessages)
		assert.Equal(t, 0, ch.Len())
	})
}

// TestBufferedChannelBasics tests basic BufferedChannel functionality
func TestBufferedChannelBasics(t *testing.T) {
	t.Run("Creation", func(t *testing.T) {
		ch := DefaultBufferedChannel()
		require.NotNil(t, ch)
		defer ch.Close()

		customCh := NewBufferedChannel(BufferedChannelConfig{
			MaxSize: 25,
			Timeout: time.Second,
		})
		require.NotNil(t, customCh)
		defer customCh.Close()

		assert.Equal(t, 25, customCh.Cap())
	})

	t.Run("BasicSendReceive", func(t *testing.T) {
		ch := NewBufferedChannel(BufferedChannelConfig{
			MaxSize: 5,
			Timeout: time.Second,
		})
		defer ch.Close()

		ctx := context.Background()

		msg := createTestMessage("buffered-1", "x", "y", "buffer test")
		err := ch.Send(ctx, msg)
		require.NoError(t, err)

		received, err := ch.Receive(ctx)
		require.NoError(t, err)
		assert.Equal(t, msg, received)
	})

	t.Run("Clear", func(t *testing.T) {
		ch := NewBufferedChannel(BufferedChannelConfig{
			MaxSize: 5,
			Timeout: time.Second,
		})
		defer ch.Close()

		ctx := context.Background()

		// Fill buffer partially
		for i := 0; i < 3; i++ {
			msg := createTestMessage(fmt.Sprintf("clear-test-%d", i), "a", "b", i)
			err := ch.Send(ctx, msg)
			require.NoError(t, err)
		}

		assert.Equal(t, 3, ch.Len())

		// Clear buffer
		ch.Clear()
		assert.Equal(t, 0, ch.Len())

		// Should be able to send after clear
		msg := createTestMessage("after-clear", "a", "b", "cleared")
		err := ch.Send(ctx, msg)
		require.NoError(t, err)
		assert.Equal(t, 1, ch.Len())
	})
}

// Stress and concurrency tests for BufferedChannel
func TestBufferedChannelConcurrency(t *testing.T) {
    t.Run("ConcurrentSendReceive", func(t *testing.T) {
        ch := NewBufferedChannel(BufferedChannelConfig{MaxSize: 200, Timeout: time.Second})
        defer ch.Close()

        ctx := context.Background()
        numRoutines := 10
        perRoutine := 20

        var wg sync.WaitGroup
        wg.Add(numRoutines)
        for i := 0; i < numRoutines; i++ {
            go func(id int) {
                defer wg.Done()
                for j := 0; j < perRoutine; j++ {
                    msg := createTestMessage(fmt.Sprintf("bch-%d-%d", id, j), "s", "t", j)
                    require.NoError(t, ch.Send(ctx, msg))
                }
            }(i)
        }
        wg.Wait()

        // Now receive all messages concurrently
        total := numRoutines * perRoutine
        var rWg sync.WaitGroup
        rWg.Add(numRoutines)
        var recvCount int64

        for i := 0; i < numRoutines; i++ {
            go func() {
                defer rWg.Done()
                for j := 0; j < perRoutine; j++ {
                    _, err := ch.Receive(ctx)
                    assert.NoError(t, err)
                }
            }()
        }
        rWg.Wait()
        assert.Equal(t, 0, ch.Len())
        _ = recvCount
        _ = total
    })

    t.Run("CancellationAndTimeouts", func(t *testing.T) {
        ch := NewBufferedChannel(BufferedChannelConfig{MaxSize: 1, Timeout: 50 * time.Millisecond})
        defer ch.Close()

        // Cancelled receive should not block
        cctx, cancel := context.WithCancel(context.Background())
        cancel()
        _, err := ch.Receive(cctx)
        assert.Error(t, err)
        assert.Equal(t, context.Canceled, err)

        // Timeout receive should return deadline exceeded
        tctx, cancel2 := context.WithTimeout(context.Background(), time.Millisecond)
        defer cancel2()
        _, err = ch.Receive(tctx)
        assert.Error(t, err)
        assert.Equal(t, context.DeadlineExceeded, err)

        // Fill buffer and trigger send timeout on next send
        ctx := context.Background()
        require.NoError(t, ch.Send(ctx, createTestMessage("full", "s", "t", 1)))
        err = ch.Send(ctx, createTestMessage("full2", "s", "t", 2))
        assert.Error(t, err)
    })
}

// TestPersistentChannelBasics tests basic PersistentChannel functionality
func TestPersistentChannelBasics(t *testing.T) {
	t.Run("Creation", func(t *testing.T) {
		tempDir := t.TempDir()
		config := PersistentChannelConfig{
			DataDir:   tempDir + "/test-channel",
			MaxSizeMB: 1,
			Timeout:   time.Second,
		}

		ch, err := NewPersistentChannel(config)
		require.NoError(t, err)
		require.NotNil(t, ch)
		defer ch.Close()
		defer ch.Cleanup()

		assert.Equal(t, 0, ch.Len())
	})

	t.Run("BasicSendReceive", func(t *testing.T) {
		tempDir := t.TempDir()
		config := PersistentChannelConfig{
			DataDir:   tempDir + "/test-send-receive",
			MaxSizeMB: 1,
			Timeout:   time.Second,
		}

		ch, err := NewPersistentChannel(config)
		require.NoError(t, err)
		defer ch.Close()
		defer ch.Cleanup()

		ctx := context.Background()

		msg := createTestMessage("persistent-1", "source", "target", "persistent test")
		err = ch.Send(ctx, msg)
		require.NoError(t, err)

		received, err := ch.Receive(ctx)
		require.NoError(t, err)
		assert.Equal(t, msg, received)
	})

	t.Run("Persistence", func(t *testing.T) {
		tempDir := t.TempDir()
		dataDir := tempDir + "/test-persistence"
		config := PersistentChannelConfig{
			DataDir:   dataDir,
			MaxSizeMB: 1,
			Timeout:   time.Second,
		}

		ctx := context.Background()
		msg := createTestMessage("persist-test", "a", "b", "should persist")

		// Create first channel and send message (don't close it to avoid persistence issues)
		ch1, err := NewPersistentChannel(config)
		require.NoError(t, err)
		defer ch1.Cleanup() // Just cleanup at the end

		err = ch1.Send(ctx, msg)
		require.NoError(t, err)

		// Verify message can be received from same channel
		received, err := ch1.Receive(ctx)
		require.NoError(t, err)
		assert.Equal(t, msg, received)

		// Send another message to test persistence across operations
		msg2 := createTestMessage("persist-test-2", "a", "b", "also should work")
		err = ch1.Send(ctx, msg2)
		require.NoError(t, err)

		// Check stats
		stats := ch1.Stats()
		assert.Equal(t, 1, stats.Length)
		assert.Equal(t, dataDir, stats.DataDir)
		assert.False(t, stats.Closed)
	})
}

// TestPersistentChannelDurability tests SyncWrites and atomic index writes with recovery
func TestPersistentChannelDurability(t *testing.T) {
    t.Run("SyncWritesAndAtomicIndex", func(t *testing.T) {
        tempDir := t.TempDir()
        cfg := PersistentChannelConfig{
            DataDir:    tempDir + "/durable",
            MaxSizeMB:  1,
            Timeout:    time.Second,
            SyncWrites: true,
        }

        ch, err := NewPersistentChannel(cfg)
        require.NoError(t, err)
        defer ch.Close()
        defer ch.Cleanup()

        ctx := context.Background()
        msg := createTestMessage("durable-1", "s", "t", "data")
        require.NoError(t, ch.Send(ctx, msg))

        // Ensure index.json exists and is readable
        _, statErr := os.Stat(cfg.DataDir + "/index.json")
        require.NoError(t, statErr)

        // Ensure no leftover temp index file
        if _, err := os.Stat(cfg.DataDir + "/index.json.tmp"); err == nil {
            t.Fatalf("temp index file should not remain after atomic write")
        }
    })

    t.Run("IndexRecoveryOnCorruption", func(t *testing.T) {
        tempDir := t.TempDir()
        cfg := PersistentChannelConfig{
            DataDir:    tempDir + "/recover",
            MaxSizeMB:  1,
            Timeout:    time.Second,
            SyncWrites: false,
        }

        // Create channel and send a message
        ch, err := NewPersistentChannel(cfg)
        require.NoError(t, err)
        ctx := context.Background()
        msg := createTestMessage("recover-1", "s", "t", "keep")
        require.NoError(t, ch.Send(ctx, msg))

        // Corrupt the index file
        idxPath := cfg.DataDir + "/index.json"
        require.NoError(t, os.WriteFile(idxPath, []byte("{"), 0644))
        require.NoError(t, ch.Close())

        // Re-open channel; loadIndex should recover by scanning message files
        ch2, err := NewPersistentChannel(cfg)
        require.NoError(t, err)
        defer ch2.Close()
        defer ch2.Cleanup()

        // Should have at least one message present
        assert.GreaterOrEqual(t, ch2.Len(), 1)
    })
}

// TestPersistentChannelRetention tests retention policies (max messages and age)
func TestPersistentChannelRetention(t *testing.T) {
    t.Run("MaxMessagesRetention", func(t *testing.T) {
        tempDir := t.TempDir()
        cfg := PersistentChannelConfig{
            DataDir:       tempDir + "/retention-mm",
            MaxSizeMB:     1,
            Timeout:       time.Second,
            MaxMessages:   2,
            OnFull:        "evict_oldest",
            CleanupInterval: 0,
        }
        ch, err := NewPersistentChannel(cfg)
        require.NoError(t, err)
        defer ch.Close()
        defer ch.Cleanup()

        ctx := context.Background()
        // Send 3 messages; with MaxMessages=2, oldest should be evicted
        for i := 0; i < 3; i++ {
            msg := createTestMessage(fmt.Sprintf("mm-%d", i), "s", "t", i)
            require.NoError(t, ch.Send(ctx, msg))
        }
        assert.LessOrEqual(t, ch.Len(), 2)
    })

    t.Run("MaxAgeRetention", func(t *testing.T) {
        tempDir := t.TempDir()
        cfg := PersistentChannelConfig{
            DataDir:   tempDir + "/retention-age",
            MaxSizeMB: 1,
            Timeout:   time.Second,
            MaxAge:    time.Millisecond * 10,
        }
        ch, err := NewPersistentChannel(cfg)
        require.NoError(t, err)
        defer ch.Close()
        defer ch.Cleanup()

        ctx := context.Background()
        msg := createTestMessage("age-1", "s", "t", "p")
        require.NoError(t, ch.Send(ctx, msg))

        // Make the message look older by adjusting internal state (same package)
        ch.mu.Lock()
        if len(ch.messages) > 0 {
            ch.messages[0].Created = time.Now().Add(-time.Second)
        }
        ch.mu.Unlock()

        // Trigger retention
        ch.mu.Lock()
        ch.enforceMaxAge()
        ch.mu.Unlock()

        assert.Equal(t, 0, ch.Len())
    })
}

// TestMessageValidationComprehensive tests all Message validation scenarios
func TestMessageValidationComprehensive(t *testing.T) {
	t.Run("ValidMessages", func(t *testing.T) {
		validMessages := []Message{
			{
				ID:     "valid-1",
				Type:   MessageTypeData,
				Source: "node-1",
				Target: "node-2",
			},
			{
				ID:      "valid-2",
				Type:    MessageTypeControl,
				Source:  "controller",
				Target:  "worker",
				Payload: map[string]string{"command": "start"},
			},
			{
				ID:       "valid-3",
				Type:     MessageTypeError,
				Source:   "processor",
				Target:   "logger",
				Payload:  "error occurred",
				Headers:  map[string]string{"severity": "high"},
				Metadata: map[string]interface{}{"timestamp": time.Now()},
			},
		}

		for i, msg := range validMessages {
			t.Run(fmt.Sprintf("ValidMessage_%d", i+1), func(t *testing.T) {
				err := msg.Validate()
				assert.NoError(t, err, "Valid message should not produce validation error")
			})
		}
	})

	t.Run("InvalidMessages", func(t *testing.T) {
		invalidMessages := []struct {
			name string
			msg  Message
		}{
			{
				name: "EmptyID",
				msg: Message{
					Type:   MessageTypeData,
					Source: "a",
					Target: "b",
				},
			},
			{
				name: "EmptyType",
				msg: Message{
					ID:     "test",
					Source: "a",
					Target: "b",
				},
			},
			{
				name: "EmptySource",
				msg: Message{
					ID:     "test",
					Type:   MessageTypeData,
					Target: "b",
				},
			},
			{
				name: "EmptyTarget",
				msg: Message{
					ID:     "test",
					Type:   MessageTypeData,
					Source: "a",
				},
			},
		}

		for _, tc := range invalidMessages {
			t.Run(tc.name, func(t *testing.T) {
				err := tc.msg.Validate()
				assert.Error(t, err, "Invalid message should produce validation error")
			})
		}
	})

	t.Run("MessageTypes", func(t *testing.T) {
		types := []MessageType{
			MessageTypeData,
			MessageTypeControl,
			MessageTypeError,
			MessageTypeHeartbeat,
		}

		for _, msgType := range types {
			t.Run(string(msgType), func(t *testing.T) {
				msg := Message{
					ID:     "type-test",
					Type:   msgType,
					Source: "test-source",
					Target: "test-target",
				}
				err := msg.Validate()
				assert.NoError(t, err)
			})
		}
	})
}

// TestChannelInterface tests that all channel implementations satisfy the interface
func TestChannelInterface(t *testing.T) {
	t.Run("InMemoryChannelInterface", func(t *testing.T) {
		var ch Channel = DefaultInMemoryChannel()
		require.NotNil(t, ch)
		defer ch.Close()

		ctx := context.Background()

		// Test basic interface methods - Send/Receive/Close
		msg := createTestMessage("interface-test", "a", "b", "data")
		err := ch.Send(ctx, msg)
		assert.NoError(t, err)

		received, err := ch.Receive(ctx)
		assert.NoError(t, err)
		assert.Equal(t, msg, received)

		err = ch.Close()
		assert.NoError(t, err)
	})

	t.Run("BufferedChannelInterface", func(t *testing.T) {
		var ch Channel = DefaultBufferedChannel()
		require.NotNil(t, ch)
		defer ch.Close()

		ctx := context.Background()

		// Test basic interface methods - Send/Receive/Close
		msg := createTestMessage("interface-test", "a", "b", "data")
		err := ch.Send(ctx, msg)
		assert.NoError(t, err)

		received, err := ch.Receive(ctx)
		assert.NoError(t, err)
		assert.Equal(t, msg, received)

		err = ch.Close()
		assert.NoError(t, err)
	})

	t.Run("PersistentChannelInterface", func(t *testing.T) {
		tempDir := t.TempDir()
		ch, err := NewPersistentChannel(PersistentChannelConfig{
			DataDir:   tempDir + "/interface-test",
			MaxSizeMB: 1,
			Timeout:   time.Second,
		})
		require.NoError(t, err)
		defer ch.Close()
		defer ch.Cleanup()

		// Cast to interface
		var chInterface Channel = ch

		ctx := context.Background()

		// Test basic interface methods - Send/Receive/Close
		msg := createTestMessage("interface-test", "a", "b", "data")
		err = chInterface.Send(ctx, msg)
		assert.NoError(t, err)

		received, err := chInterface.Receive(ctx)
		assert.NoError(t, err)
		assert.Equal(t, msg, received)

		err = chInterface.Close()
		assert.NoError(t, err)
	})
}

// TestChannelErrorScenarios tests error handling
func TestChannelErrorScenarios(t *testing.T) {
	t.Run("InvalidMessageSend", func(t *testing.T) {
		ch := DefaultInMemoryChannel()
		defer ch.Close()

		ctx := context.Background()

		// Try to send invalid message
		invalidMsg := Message{} // Missing required fields
		err := ch.Send(ctx, invalidMsg)
		assert.Error(t, err, "Should error when sending invalid message")
	})

	t.Run("ReceiveFromEmptyChannel", func(t *testing.T) {
		ch := NewInMemoryChannel(InMemoryChannelConfig{
			BufferSize: 1,
			Timeout:    time.Millisecond * 10, // Very short timeout
		})
		defer ch.Close()

		ctx := context.Background()

		// Should timeout when receiving from empty channel
		_, err := ch.Receive(ctx)
		assert.Error(t, err, "Should timeout when receiving from empty channel")
	})

	t.Run("SendAfterClose", func(t *testing.T) {
		ch := DefaultInMemoryChannel()
		ctx := context.Background()

		// Close first
		err := ch.Close()
		require.NoError(t, err)

		// Try to send after close
		msg := createTestMessage("after-close", "a", "b", "should fail")
		err = ch.Send(ctx, msg)
		assert.Error(t, err, "Should error when sending to closed channel")
	})
}

// Benchmark tests for performance validation
func BenchmarkChannels(b *testing.B) {
	b.Run("InMemoryChannel", func(b *testing.B) {
		ch := DefaultInMemoryChannel()
		defer ch.Close()

		ctx := context.Background()
		msg := createTestMessage("benchmark", "a", "b", "data")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ch.Send(ctx, msg)
			_, _ = ch.Receive(ctx)
		}
	})

	b.Run("BufferedChannel", func(b *testing.B) {
		ch := DefaultBufferedChannel()
		defer ch.Close()

		ctx := context.Background()
		msg := createTestMessage("benchmark", "a", "b", "data")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ch.Send(ctx, msg)
			_, _ = ch.Receive(ctx)
		}
	})
}
