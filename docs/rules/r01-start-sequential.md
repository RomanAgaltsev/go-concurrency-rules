---
id: R01
title: Start sequential; concurrency is a cost you must measure
statement: Write the sequential version first, and add goroutines only where a measurement shows they pay for themselves.
group: ownership
tags: [write, profile]
since: go1.0
status: active
proof:
  kind: measure
  test: TestSumSquares
  runs: 20
alternatives: []
undergo: []
sources:
  - https://go.dev/blog/waza-talk
  - https://pkg.go.dev/testing#hdr-Benchmarks
  - https://pkg.go.dev/golang.org/x/perf/cmd/benchstat
---

# R01 — Start sequential; concurrency is a cost you must measure

> **Write the sequential version first, and add goroutines only where a measurement
> shows they pay for themselves.**

## Why

Goroutines are cheap compared with threads, not free. Each one is a stack, a
scheduling decision, usually a closure on the heap, and a synchronisation point where
its result is collected. That cost is fixed per goroutine; what it buys depends on
the work inside. When the work per goroutine is smaller than the overhead, adding
goroutines makes the program slower — and, as the measurement below shows, adding
CPUs can make it slower still.

Concurrency is a way to *structure* a program; parallel speed-up is a separate claim
that only a measurement can support. A sequential version is also the reference the
concurrent one is tested against, and the baseline it is measured against: without
it, neither "correct" nor "faster" has a meaning.

## Broken / Fixed

=== "Broken"

    ```go
    --8<-- "rules/r01-start-sequential/broken.go:sum"
    ```

=== "Fixed"

    ```go
    --8<-- "rules/r01-start-sequential/fixed.go:sum"
    ```

Both are correct — the broken version even avoids a lock by giving each goroutine its
own slot. What it gets wrong is the price: one multiplication per goroutine.

## Measurement

```go
--8<-- "rules/r01-start-sequential/bench_test.go:bench"
```

Measured on 2026-10-03 (AMD Ryzen 5 3600, 12 logical CPUs, linux/amd64 in the
`golang:1.27` container, go1.27.1), 10 samples per side, for `xs` of 1000 elements:

| GOMAXPROCS | broken (goroutine per element) | fixed (loop) | broken costs |
|---|---|---|---|
| 1 | 290.9 µs | 520.9 ns | ~560× |
| 4 | 354.8 µs | 520.2 ns | ~680× |
| 12 | 394.6 µs | 519.7 ns | ~760× |

The broken version also allocates 2,002 times and 78.33 KiB per call; the loop
allocates nothing. Every difference is significant (benchstat, p=0.000, n=10). Note
the direction across rows: more CPUs make the goroutine version *slower*.

```text
--8<-- "rules/r01-start-sequential/testdata/measurements/2026-10-03-linux-12cpu.txt"
```

Reproduce with `task measure -- rules/r01-start-sequential 1,4,12`, which writes a new
dated artifact; compare yours with this one by regime, not by absolute numbers.

## How to spot it

**Reading.** A `go` statement inside a loop over data, with a few lines of work in its
body, is the shape. Ask how long the body takes compared with starting a goroutine,
and whether anyone measured.

**Tooling.** Benchmarks, with `-cpu` swept and enough samples for `benchstat` to call
the difference significant. A CPU profile of the concurrent version that shows the
time in the scheduler, `newproc` and `WaitGroup` rather than in the work says the same
thing from the other side.

## Proof

This is a `measure` rule: nothing fails. The admission gate checks that both versions
pass `TestSumSquares`, that the benchmark runs in both, and that every measurement on
record names its regime and rests on at least 10 samples per side. The numbers above
are checked in review against the artifact they come from — CI runners are too noisy
to judge them.

## Trade-offs & when to break it

Concurrency pays when the work per goroutine is large compared with the overhead —
which this measurement puts at roughly 0.3–0.4 µs per goroutine (290.9–394.6 µs for
1000 of them, against 0.52 µs of actual work) — and when the pieces are independent:
I/O that waits, or computation many times that cost. Even then, bound it: a fixed number of workers over chunks of
the input, not one goroutine per element. Some programs are concurrent for structure
rather than speed — a server handling independent requests — and there the question is
correctness first, cost second.

## Alternatives

Bounded worker pools and per-chunk goroutines each get their own rule as they land.

## Since & sources

`since: go1.0`. Sources: Rob Pike, [*Concurrency is not
parallelism*](https://go.dev/blog/waza-talk);
[benchmarks in `testing`](https://pkg.go.dev/testing#hdr-Benchmarks);
[`benchstat`](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat).
