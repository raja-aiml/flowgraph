// Package metrics exposes expvar-published counters and gauges used by the
// FlowGraph runtime (channels, scheduler, and engine). It intentionally avoids
// external dependencies and is consumed by the optional flowgraph-server for
// /debug/vars and /metrics endpoints.
package metrics

