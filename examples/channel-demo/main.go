// Example demonstrating channel implementations
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/flowgraph/flowgraph/internal/core/channel"
)

func main() {
	ctx := context.Background()

	// Test InMemoryChannel
	fmt.Println("=== Testing InMemoryChannel ===")
	testInMemoryChannel(ctx)

	// Test BufferedChannel
	fmt.Println("\n=== Testing BufferedChannel ===")
	testBufferedChannel(ctx)

	// Test PersistentChannel
	fmt.Println("\n=== Testing PersistentChannel ===")
	testPersistentChannel(ctx)

	// Test State Reducers
	fmt.Println("\n=== Testing State Reducers ===")
	testStateReducers()
}

func testInMemoryChannel(ctx context.Context) {
	ch := channel.DefaultInMemoryChannel()
	defer ch.Close()

	msg := channel.Message{
		ID:      "test-1",
		Type:    channel.MessageTypeData,
		Source:  "sender",
		Target:  "receiver",
		Payload: "Hello from InMemoryChannel!",
	}

	// Send message
	if err := ch.Send(ctx, msg); err != nil {
		log.Printf("Send error: %v", err)
		return
	}
	fmt.Println("✓ Message sent successfully")

	// Receive message
	received, err := ch.Receive(ctx)
	if err != nil {
		log.Printf("Receive error: %v", err)
		return
	}
	fmt.Printf("✓ Message received: %s\n", received.Payload)
}

func testBufferedChannel(ctx context.Context) {
	ch := channel.NewBufferedChannel(channel.BufferedChannelConfig{
		MaxSize: 5,
		Timeout: 5 * time.Second,
	})
	defer ch.Close()

	msg := channel.Message{
		ID:      "test-2",
		Type:    channel.MessageTypeData,
		Source:  "sender",
		Target:  "receiver",
		Payload: "Hello from BufferedChannel!",
	}

	// Send message
	if err := ch.Send(ctx, msg); err != nil {
		log.Printf("Send error: %v", err)
		return
	}
	fmt.Println("✓ Message sent to buffer")

	// Receive message
	received, err := ch.Receive(ctx)
	if err != nil {
		log.Printf("Receive error: %v", err)
		return
	}
	fmt.Printf("✓ Message received from buffer: %s\n", received.Payload)
}

func testPersistentChannel(ctx context.Context) {
	tmpDir := "/tmp/flowgraph-channel-test"

	ch, err := channel.NewPersistentChannel(channel.PersistentChannelConfig{
		DataDir:   tmpDir,
		MaxSizeMB: 1, // 1MB
		Timeout:   5 * time.Second,
	})
	if err != nil {
		log.Printf("Channel creation error: %v", err)
		return
	}
	defer ch.Close()

	msg := channel.Message{
		ID:      "test-3",
		Type:    channel.MessageTypeData,
		Source:  "sender",
		Target:  "receiver",
		Payload: "Hello from PersistentChannel!",
	}

	// Send message
	if err := ch.Send(ctx, msg); err != nil {
		log.Printf("Send error: %v", err)
		return
	}
	fmt.Println("✓ Message persisted to disk")

	// Receive message
	received, err := ch.Receive(ctx)
	if err != nil {
		log.Printf("Receive error: %v", err)
		return
	}
	fmt.Printf("✓ Message read from disk: %s\n", received.Payload)
}

func testStateReducers() {
	// Test AppendReducer
	appendReducer := channel.NewAppendReducer()
	current := map[string]interface{}{
		"items": []interface{}{"a", "b"},
	}
	update := map[string]interface{}{
		"items": []interface{}{"c", "d"},
	}
	result := appendReducer.Reduce(current, update)
	fmt.Printf("✓ AppendReducer result: %v\n", result["items"])

	// Test MergeReducer
	mergeReducer := channel.NewMergeReducer()
	current = map[string]interface{}{
		"config": map[string]interface{}{
			"timeout": 30,
			"retries": 3,
		},
	}
	update = map[string]interface{}{
		"config": map[string]interface{}{
			"timeout": 60,
			"debug":   true,
		},
	}
	result = mergeReducer.Reduce(current, update)
	fmt.Printf("✓ MergeReducer result: %v\n", result["config"])

	// Test ReplaceReducer
	replaceReducer := channel.NewReplaceReducer()
	current = map[string]interface{}{
		"value": "old",
	}
	update = map[string]interface{}{
		"value": "new",
	}
	result = replaceReducer.Reduce(current, update)
	fmt.Printf("✓ ReplaceReducer result: %v\n", result["value"])
}
