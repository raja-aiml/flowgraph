package channel

import (
    "context"
    "fmt"
    "testing"
    "time"
)

func BenchmarkInMemoryChannel_SendReceive(b *testing.B) {
    ch := NewInMemoryChannel(InMemoryChannelConfig{BufferSize: 1024, Timeout: time.Second})
    defer ch.Close()
    ctx := context.Background()
    msg := createBenchMsg("inmem")
    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = ch.Send(ctx, msg)
        _, _ = ch.Receive(ctx)
    }
}

func BenchmarkBufferedChannel_SendReceive(b *testing.B) {
    ch := NewBufferedChannel(BufferedChannelConfig{MaxSize: 4096, Timeout: time.Second})
    defer ch.Close()
    ctx := context.Background()
    msg := createBenchMsg("buf")
    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = ch.Send(ctx, msg)
        _, _ = ch.Receive(ctx)
    }
}

func BenchmarkPersistentChannel_SendReceive(b *testing.B) {
    dir := b.TempDir()
    ch, err := NewPersistentChannel(PersistentChannelConfig{DataDir: dir, MaxSizeMB: 10, Timeout: time.Second})
    if err != nil {
        b.Fatalf("failed to create persistent channel: %v", err)
    }
    defer ch.Close()
    defer ch.Cleanup()
    ctx := context.Background()
    msg := createBenchMsg("disk")
    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = ch.Send(ctx, msg)
        _, _ = ch.Receive(ctx)
    }
}

func createBenchMsg(prefix string) Message {
    return Message{
        ID:      fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano()),
        Type:    MessageTypeData,
        Source:  "A",
        Target:  "B",
        Payload: 42,
    }
}

