#!/usr/bin/env bash
set -euo pipefail

# Simple benchmark runner with profiles
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

pushd "$ROOT_DIR" >/dev/null

echo "==> Channel benchmarks"
go test ./internal/core/channel -run ^$ -bench . -benchmem -cpuprofile channel_cpu.out -memprofile channel_mem.out

echo "==> Pregel benchmarks"
go test ./internal/core/pregel -run ^$ -bench . -benchmem -cpuprofile pregel_cpu.out -memprofile pregel_mem.out

echo "Profiles generated: channel_cpu.out, channel_mem.out, pregel_cpu.out, pregel_mem.out"

popd >/dev/null

