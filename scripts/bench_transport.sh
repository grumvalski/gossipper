#!/usr/bin/env bash
# UDP transport microbenchmarks (SharedUDP / reuseport).
# Usage:
#   ./scripts/bench_transport.sh
#   BENCHTIME=3s BENCH=BenchmarkSharedUDP_RoundTrip ./scripts/bench_transport.sh
#   CPUPROFILE=/tmp/transport.cpu ./scripts/bench_transport.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

BENCHTIME="${BENCHTIME:-1s}"
BENCH="${BENCH:-.}"
PKG="./internal/transport/..."

cmd=(go test "$PKG" -run='^$' -bench="$BENCH" -benchmem -benchtime="$BENCHTIME" -count=1)

if [[ -n "${CPUPROFILE:-}" ]]; then
	cmd+=(-cpuprofile="$CPUPROFILE")
fi

exec "${cmd[@]}"
