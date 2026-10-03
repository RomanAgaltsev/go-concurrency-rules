---
id: X03
title: Add, go, defer Done for every goroutine
statement: Before Go 1.25, start each WaitGroup goroutine with wg.Add(1) before go, and defer wg.Done() inside it.
group: coordination
tags: [write, refactor]
since: go1.0
status: retired
retired_in: go1.25
replaced_by: sync.WaitGroup.Go
proof:
  kind: none
  reason: nothing was ever broken — Add before go with defer Done still works; WaitGroup.Go retires the boilerplate, not a failure.
sources:
  - https://go.dev/doc/go1.25
  - https://pkg.go.dev/sync#WaitGroup.Go
---

# X03 — `Add`, `go`, `defer Done` for every goroutine

> **Retired in Go 1.25.** The old rule: for each goroutine a `WaitGroup` waits for,
> call `wg.Add(1)` before the `go` statement and `defer wg.Done()` as the goroutine's
> first line.

## What changed

Go 1.25 added `WaitGroup.Go`, which "makes the common pattern of creating and counting
goroutines more convenient" (release notes): `wg.Go(f)` does the `Add`, starts the
goroutine and calls `Done` when `f` returns.

## Why there is no proof here

There is no broken variant to fail: the three-line pattern was correct and still is.
What retired is the chance to get it wrong — `Add` inside the goroutine, where `Wait`
may run before it, or a forgotten `Done`.

## What to do now

Use `wg.Go(f)` for goroutines that only need to be waited for. When the work returns an
error, needs cancellation or a concurrency limit, use `errgroup` instead.

## Residue

Modules that must build with Go 1.24 or earlier keep the old pattern — and its rule
that `Add` happens before the `go` statement, never inside the goroutine.

## Since & sources

The old rule held from `go1.0` until `go1.25`. Sources:
[Go 1.25 release notes](https://go.dev/doc/go1.25),
[`sync.WaitGroup.Go`](https://pkg.go.dev/sync#WaitGroup.Go).
