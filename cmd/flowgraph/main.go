// Package main provides the FlowGraph CLI application
package main

import (
	"fmt"
	"os"
)

// Version information set during build
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("FlowGraph %s (commit: %s, built: %s)\n", Version, Commit, BuildTime)
		return
	}

	fmt.Println("🌊 FlowGraph - Graph-based Workflow Execution")
	fmt.Println("Project structure initialized successfully!")
	fmt.Println("Run 'make help' to see available commands")
}
