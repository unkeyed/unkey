#!/usr/bin/env bash
set -euo pipefail

output="${1:?usage: bash benchmarks/streamschema.sh OUTPUT_DIRECTORY}"
mkdir -p "$output"
scratch="$(mktemp -d)"
trap 'rm -rf "$scratch"' EXIT

export GOMAXPROCS=4 GOGC=100 GOMEMLIMIT=off
mise exec -- go test -c -o "$scratch/compare.test" ./pkg/spool/integration
{
  git rev-parse HEAD
  sha256sum pkg/spool/integration/comparison_bench_test.go benchmarks/streamschema.sh
  mise exec -- go version
  uname -a
  uptime
  nproc
  df -T "$scratch"
  printf 'GOMAXPROCS=%s GOGC=%s GOMEMLIMIT=%s\n' "$GOMAXPROCS" "$GOGC" "$GOMEMLIMIT"
} > "$output/environment.txt"

while read -r benchmark fixture concurrency iterations; do
  for repeat in 1 2 3 4 5; do
    modes=(before after)
    if (( repeat % 2 == 0 )); then modes=(after before); fi
    for mode in "${modes[@]}"; do
      label="$benchmark-$fixture-c$concurrency-$mode-$repeat"
      printf '%s\n' "$label"
      mise exec -- "$scratch/compare.test" -test.run='^$' \
        -test.bench="^${benchmark}\$/^${fixture}\$/^c${concurrency}\$/^${mode}\$" \
        -test.benchtime="${iterations}x" -test.benchmem \
        > "$output/$label.log" 2>&1
      grep '^Benchmark' "$output/$label.log"
    done
  done
done <<'CASES'
BenchmarkValidationComparison 1KiB 1 1000
BenchmarkValidationComparison 1MiB 1 16
BenchmarkValidationComparison 8MiB 1 8
BenchmarkValidationComparison 16MiB 1 8
BenchmarkValidationComparison 64MiB 1 4
BenchmarkValidationComparison unicode16MiB 1 4
BenchmarkValidationComparison objects16MiB 1 2
BenchmarkValidationComparison early-invalid16MiB 1 16
BenchmarkValidationComparison late-invalid16MiB 1 8
BenchmarkValidationComparison 16MiB 8 32
BenchmarkHTTPValidationComparison 1KiB 1 256
BenchmarkHTTPValidationComparison 1KiB 8 256
BenchmarkHTTPValidationComparison 16MiB 1 8
BenchmarkHTTPValidationComparison 16MiB 8 32
CASES
