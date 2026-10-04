#!/bin/sh
# measure.sh records a measure rule's benchmark comparison: the same benchmarks
# run against the broken and the fixed variant, ten samples each, compared by
# benchstat, under a header naming the regime (spec §5.3). The gate checks the
# header and the sample counts; review checks the numbers the page quotes.
#
# Run it inside the golang container, from the directory holding the rule —
# the repository root, normally:
#   task measure -- rules/r01-start-sequential 1,4,8
set -eu

usage='usage: measure.sh <rule-dir> <cpu-list> [bench-regexp]'
dir=${1:?$usage}
cpus=${2:?$usage}
bench=${3:-.}

id=$(basename "$dir" | cut -d- -f1 | tr '[:lower:]' '[:upper:]')
# Before the benchmarks, not after: a CPU this script cannot name would cost
# minutes of measuring for an artifact the gate refuses.
cpu=$(sh "$(dirname "$0")/cpu-name.sh")
benchstat=golang.org/x/perf/cmd/benchstat@v0.0.0-20260929162123-406019bb8b68
tmp=$(mktemp -d)

# Serially, never in parallel: two benchmark runs sharing the CPU measure
# each other.
go test -run '^$' -bench "$bench" -benchmem -count 10 -cpu "$cpus" -tags broken "./$dir" >"$tmp/broken.txt"
go test -run '^$' -bench "$bench" -benchmem -count 10 -cpu "$cpus" "./$dir" >"$tmp/fixed.txt"

date=$(date -u +%F)
out="$dir/testdata/measurements/$date-$(go env GOOS)-$(nproc)cpu.txt"
mkdir -p "$(dirname "$out")"
{
	echo "# rule: $id"
	echo "# date: $date"
	echo "# go: $(go env GOVERSION)"
	echo "# os: $(go env GOOS)/$(go env GOARCH)"
	echo "# cpu: $cpu"
	echo "# gomaxprocs: $cpus"
	echo "# command: go test -run '^\$' -bench '$bench' -benchmem -count 10 -cpu $cpus [-tags broken] ./$dir; benchstat broken.txt fixed.txt"
	echo
	(cd "$tmp" && go run "$benchstat" broken.txt fixed.txt)
} >"$out"
echo "wrote $out"
