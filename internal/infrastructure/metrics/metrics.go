package metrics

import (
    "expvar"
)

// Channel metrics (counters/gauges) using expvar maps keyed by implementation type.
var (
    channelSent     = expvar.NewMap("flowgraph_channel_sent_total")
    channelReceived = expvar.NewMap("flowgraph_channel_received_total")
    channelEvicted  = expvar.NewMap("flowgraph_channel_evicted_total")
    channelSize     = expvar.NewMap("flowgraph_channel_size_bytes")
)

// Engine / Scheduler metrics.
var (
    superstepsTotal      = new(expvar.Int)
    vertexExecsTotal     = new(expvar.Int)
    schedulerWorkers     = new(expvar.Int)
    schedulerQueuedTotal = new(expvar.Int)
)

func init() {
    expvar.Publish("flowgraph_supersteps_total", superstepsTotal)
    expvar.Publish("flowgraph_vertex_executions_total", vertexExecsTotal)
    expvar.Publish("flowgraph_scheduler_workers", schedulerWorkers)
    expvar.Publish("flowgraph_scheduler_queued_total", schedulerQueuedTotal)
}

// Channel helpers
func ChannelSent(kind string, n int64) { channelSent.Add(kind, n) }
func ChannelReceived(kind string, n int64) { channelReceived.Add(kind, n) }
func ChannelEvicted(kind string, n int64) { channelEvicted.Add(kind, n) }
func ChannelSizeBytes(kind string, size int64) { setMapInt(channelSize, kind, size) }

// Engine/Scheduler helpers
func IncSupersteps() { superstepsTotal.Add(1) }
func IncVertexExecs(n int64) { vertexExecsTotal.Add(n) }
func SetSchedulerWorkers(n int) { schedulerWorkers.Set(int64(n)) }
func AddSchedulerQueued(n int) { schedulerQueuedTotal.Add(int64(n)) }

// setMapInt replaces value for a key in an expvar.Map with an *expvar.Int set to v.
func setMapInt(m *expvar.Map, key string, v int64) {
    x := new(expvar.Int)
    x.Set(v)
    m.Set(key, x)
}
