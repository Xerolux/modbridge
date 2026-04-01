#!/usr/bin/env bash
set -euo pipefail

# Conservative regression guardrails for CI runners.
# Values are in ns/op and intentionally generous to reduce flakiness.
MAX_PROXY_CONNECTION_NS=2500000
MAX_PROXY_REQUEST_NS=1500000
MAX_PROXY_CONCURRENT_NS=900000

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

if [[ -z "${GOMODCACHE:-}" ]]; then
  export GOMODCACHE="$ROOT_DIR/.gomodcache"
fi
if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$ROOT_DIR/.gocache"
fi
mkdir -p "$GOMODCACHE" "$GOCACHE"

run_bench_and_extract_ns() {
  local bench_name="$1"
  local output

  output="$(
    go test -run '^$' -bench "^${bench_name}$" -benchmem ./pkg/testing/performance
  )"
  echo "$output"

  # Keep the last ns/op token from the benchmark output.
  # shellcheck disable=SC2001
  echo "$output" | sed -nE 's/.*[[:space:]]([0-9]+)[[:space:]]ns\/op.*/\1/p' | tail -n1
}

assert_threshold() {
  local name="$1"
  local value="$2"
  local max="$3"

  if [[ -z "$value" ]]; then
    echo "ERROR: ${name} benchmark not found in output"
    exit 1
  fi

  if (( value > max )); then
    echo "ERROR: ${name} regression detected (${value} ns/op > ${max} ns/op)"
    exit 1
  fi
}

proxy_connection_ns="$(run_bench_and_extract_ns "BenchmarkProxyConnection" | tail -n1)"
proxy_request_ns="$(run_bench_and_extract_ns "BenchmarkProxyRequest" | tail -n1)"
proxy_concurrent_ns="$(run_bench_and_extract_ns "BenchmarkProxyConcurrent" | tail -n1)"

assert_threshold "BenchmarkProxyConnection" "$proxy_connection_ns" "$MAX_PROXY_CONNECTION_NS"
assert_threshold "BenchmarkProxyRequest" "$proxy_request_ns" "$MAX_PROXY_REQUEST_NS"
assert_threshold "BenchmarkProxyConcurrent" "$proxy_concurrent_ns" "$MAX_PROXY_CONCURRENT_NS"

echo "Benchmark guardrails passed."
