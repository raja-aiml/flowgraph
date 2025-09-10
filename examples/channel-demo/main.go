package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/flowgraph/flowgraph/internal/core/channel"
)

// Simulate a distributed notification system with multiple agents
func main() {
	ctx := context.Background()
	fmt.Println("📢 Distributed Notification System (FlowGraph Real-World Example)")
	fmt.Println("==============================================================")

	// 1. Create channels for notifications
	inMemCh := channel.DefaultInMemoryChannel()
	bufCh := channel.NewBufferedChannel(channel.BufferedChannelConfig{MaxSize: 10, Timeout: 3 * time.Second})
	persistCh, err := channel.NewPersistentChannel(channel.PersistentChannelConfig{DataDir: "/tmp/flowgraph-notify", MaxSizeMB: 2, Timeout: 3 * time.Second})
	if err != nil {
		log.Fatalf("Failed to create persistent channel: %v", err)
	}
	defer inMemCh.Close()
	defer bufCh.Close()
	defer persistCh.Close()

	// 2. Simulate agents sending notifications
	agents := []string{"agent-A", "agent-B", "agent-C"}
	for _, agent := range agents {
		msg := channel.Message{
			ID:      fmt.Sprintf("msg-%d", rand.Intn(10000)),
			Type:    channel.MessageTypeData,
			Source:  agent,
			Target:  "all",
			Payload: fmt.Sprintf("Alert from %s at %s", agent, time.Now().Format(time.RFC3339)),
		}
		// Send to all channels
		_ = inMemCh.Send(ctx, msg)
		_ = bufCh.Send(ctx, msg)
		_ = persistCh.Send(ctx, msg)
		fmt.Printf("✓ %s sent notification\n", agent)
	}

	// 3. Simulate receiving notifications
	fmt.Println("\n🔔 Receiving notifications from channels:")
	receiveAndPrint(ctx, "InMemoryChannel", inMemCh)
	receiveAndPrint(ctx, "BufferedChannel", bufCh)
	receiveAndPrint(ctx, "PersistentChannel", persistCh)

	// 4. Aggregate notifications using reducers
	fmt.Println("\n📊 Aggregating notifications with reducers:")
	appendReducer := channel.NewAppendReducer()
	current := map[string]interface{}{"alerts": []interface{}{"init"}}
	for _, agent := range agents {
		update := map[string]interface{}{"alerts": []interface{}{fmt.Sprintf("alert-from-%s", agent)}}
		current = appendReducer.Reduce(current, update)
	}
	fmt.Printf("✓ Aggregated alerts: %v\n", current["alerts"])

	// 5. Simulate error handling (message loss)
	fmt.Println("\n⚠️ Simulating message loss:")
	lostMsg := channel.Message{ID: "lost-1", Type: channel.MessageTypeData, Source: "agent-X", Target: "all", Payload: "Lost alert"}
	if err := inMemCh.Send(ctx, lostMsg); err != nil {
		fmt.Printf("❌ Failed to send lost message: %v\n", err)
	} else {
		// Simulate loss by not receiving
		fmt.Println("(Message sent but not received)")
	}
}

func receiveAndPrint(ctx context.Context, chName string, ch channel.Channel) {
	for {
		msg, err := ch.Receive(ctx)
		if err != nil {
			break
		}
		fmt.Printf("[%s] Received: %s from %s\n", chName, msg.Payload, msg.Source)
	}
}
