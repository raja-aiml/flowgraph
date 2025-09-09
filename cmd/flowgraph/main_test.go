// Package main tests for the FlowGraph CLI application
package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureOutput captures stdout output during test execution
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestMain_VersionFlag(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		version   string
		commit    string
		buildTime string
		want      string
	}{
		{
			name:      "version with dev defaults",
			args:      []string{"flowgraph", "version"},
			version:   "dev",
			commit:    "unknown",
			buildTime: "unknown",
			want:      "FlowGraph dev (commit: unknown, built: unknown)\n",
		},
		{
			name:      "version with custom values",
			args:      []string{"flowgraph", "version"},
			version:   "v1.0.0",
			commit:    "abc123",
			buildTime: "2024-01-01",
			want:      "FlowGraph v1.0.0 (commit: abc123, built: 2024-01-01)\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original values
			oldVersion := Version
			oldCommit := Commit
			oldBuildTime := BuildTime
			oldArgs := os.Args

			// Set test values
			Version = tt.version
			Commit = tt.commit
			BuildTime = tt.buildTime
			os.Args = tt.args

			// Capture output
			output := captureOutput(func() {
				main()
			})

			// Restore original values
			Version = oldVersion
			Commit = oldCommit
			BuildTime = oldBuildTime
			os.Args = oldArgs

			// Assert
			assert.Equal(t, tt.want, output)
		})
	}
}

func TestMain_DefaultOutput(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "no arguments",
			args: []string{"flowgraph"},
			want: []string{
				"🌊 FlowGraph - Graph-based Workflow Execution",
				"Project structure initialized successfully!",
				"Run 'make help' to see available commands",
			},
		},
		{
			name: "with other arguments",
			args: []string{"flowgraph", "help"},
			want: []string{
				"🌊 FlowGraph - Graph-based Workflow Execution",
				"Project structure initialized successfully!",
				"Run 'make help' to see available commands",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original args
			oldArgs := os.Args

			// Set test args
			os.Args = tt.args

			// Capture output
			output := captureOutput(func() {
				main()
			})

			// Restore original args
			os.Args = oldArgs

			// Assert each expected line is present
			for _, expected := range tt.want {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestMain_Integration(t *testing.T) {
	// This test verifies the main function can be called without panicking
	t.Run("main executes without panic", func(t *testing.T) {
		oldArgs := os.Args
		os.Args = []string{"flowgraph"}

		require.NotPanics(t, func() {
			output := captureOutput(func() {
				main()
			})
			assert.NotEmpty(t, output)
		})

		os.Args = oldArgs
	})
}

func TestVersionVariables(t *testing.T) {
	t.Run("version variables have default values", func(t *testing.T) {
		// These should have their default values
		assert.NotEmpty(t, Version)
		assert.NotEmpty(t, Commit)
		assert.NotEmpty(t, BuildTime)
	})
}

func TestOutputFormatting(t *testing.T) {
	t.Run("version output format", func(t *testing.T) {
		oldArgs := os.Args
		os.Args = []string{"flowgraph", "version"}

		output := captureOutput(func() {
			main()
		})

		os.Args = oldArgs

		// Check format includes all required parts
		assert.True(t, strings.HasPrefix(output, "FlowGraph "))
		assert.Contains(t, output, "commit:")
		assert.Contains(t, output, "built:")
	})

	t.Run("default output includes emoji", func(t *testing.T) {
		oldArgs := os.Args
		os.Args = []string{"flowgraph"}

		output := captureOutput(func() {
			main()
		})

		os.Args = oldArgs

		// Check for emoji in output
		assert.Contains(t, output, "🌊")
	})
}
