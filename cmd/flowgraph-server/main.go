// Package main provides a minimal HTTP server exposing debug endpoints.
package main

import (
	"expvar"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof" // register /debug/pprof
	"os"
	"sort"
	"strings"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, "FlowGraph server is running. See /healthz, /debug/vars, /debug/pprof/")
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})

	// Prometheus-compatible metrics endpoint (no external deps)
	mux.HandleFunc("/metrics", promMetricsHandler)

	// Workload endpoints to generate metrics load
	mux.HandleFunc("/workload/engine/start", wm.startEngine)
	mux.HandleFunc("/workload/engine/stop", wm.stopEngine)
	mux.HandleFunc("/workload/channel/start", wm.startChannel)
	mux.HandleFunc("/workload/channel/stop", wm.stopChannel)

	addr := ":8080"
	if v := os.Getenv("FLOWGRAPH_ADDR"); v != "" {
		addr = v
	}
	log.Printf("Starting FlowGraph server on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// promMetricsHandler renders expvar-published metrics in Prometheus text exposition format.
// It supports known FlowGraph metrics and falls back to a minimal conversion for other expvar vars.
// promMetricsHandler renders known expvar metrics in Prometheus text format.
// nolint:funlen,gocognit,gocyclo // Straightforward formatter; long but simple
func promMetricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	// Define metadata for known metrics
	type meta struct {
		typ, help string
		isMap     bool
		label     string
	}
	metas := map[string]meta{
		"flowgraph_channel_sent_total":      {typ: "counter", help: "Channel messages sent", isMap: true, label: "kind"},
		"flowgraph_channel_received_total":  {typ: "counter", help: "Channel messages received", isMap: true, label: "kind"},
		"flowgraph_channel_evicted_total":   {typ: "counter", help: "Persistent channel evicted messages", isMap: true, label: "kind"},
		"flowgraph_channel_size_bytes":      {typ: "gauge", help: "Persistent channel current size in bytes", isMap: true, label: "kind"},
		"flowgraph_supersteps_total":        {typ: "counter", help: "Number of supersteps executed", isMap: false},
		"flowgraph_vertex_executions_total": {typ: "counter", help: "Number of vertex computations executed", isMap: false},
		"flowgraph_scheduler_workers":       {typ: "gauge", help: "Number of scheduler workers", isMap: false},
		"flowgraph_scheduler_queued_total":  {typ: "counter", help: "Number of tasks scheduled", isMap: false},
	}

	// Collect variable names deterministically
	varNames := make([]string, 0, 64)
	expvar.Do(func(kv expvar.KeyValue) {
		varNames = append(varNames, kv.Key)
	})
	sort.Strings(varNames)

	printed := make(map[string]bool)

	writeHeader := func(name string, m meta) {
		if printed[name] {
			return
		}
		_, _ = fmt.Fprintf(w, "# HELP %s %s\n", name, sanitizeHelp(m.help))
		_, _ = fmt.Fprintf(w, "# TYPE %s %s\n", name, m.typ)
		printed[name] = true
	}

	for _, name := range varNames {
		v := expvar.Get(name)
		m, known := metas[name]
		if !known {
			// Minimal rendering: publish as an untyped gauge if numeric
			if iv, ok := v.(*expvar.Int); ok {
				_, _ = fmt.Fprintf(w, "# TYPE %s gauge\n", name)
				_, _ = fmt.Fprintf(w, "%s %s\n", name, iv.String())
			}
			continue
		}
		writeHeader(name, m)
		if m.isMap {
			if mp, ok := v.(*expvar.Map); ok {
				// Collect subkeys deterministically
				sub := make([]expvar.KeyValue, 0, 8)
				mp.Do(func(kv expvar.KeyValue) { sub = append(sub, kv) })
				sort.Slice(sub, func(i, j int) bool { return sub[i].Key < sub[j].Key })
				for _, kv := range sub {
					// Expect numeric string; emit sample with label
					fmt.Fprintf(w, "%s{%s=\"%s\"} %s\n", name, m.label, escapeLabel(kv.Key), kv.Value.String())
				}
			}
		} else {
			// Scalar metrics
			fmt.Fprintf(w, "%s %s\n", name, v.String())
		}
	}
}

func sanitizeHelp(s string) string {
	// Replace newlines with spaces to satisfy Prometheus text format
	return strings.ReplaceAll(s, "\n", " ")
}

func escapeLabel(s string) string {
	// Escape backslash, double-quote, and newline per Prometheus format
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
