---
id: X04
title: synctest is experimental — use synctest.Run
statement: In Go 1.24, testing/synctest existed only behind GOEXPERIMENT=synctest, with synctest.Run as its entry point.
group: testing
tags: [test]
since: go1.24
status: retired
retired_in: go1.25
replaced_by: testing/synctest.Test, generally available
proof:
  kind: none
  reason: the old API no longer exists in any toolchain this repository supports — removed in Go 1.26 — so there is nothing left to run.
sources:
  - https://go.dev/doc/go1.25
  - https://pkg.go.dev/testing/synctest
---

# X04 — `synctest` is experimental; use `synctest.Run`

> **Retired in Go 1.25.** The old rule: `testing/synctest` is an experiment — enable it
> with `GOEXPERIMENT=synctest` and enter a bubble with `synctest.Run`, or use a fake
> clock library.

## What changed

Go 1.25's release notes: the package "was first available in Go 1.24 under
GOEXPERIMENT=synctest, with a slightly different API. The experiment has now graduated
to general availability. The old API is still present if GOEXPERIMENT=synctest is set,
but will be removed in Go 1.26." The entry point is now `synctest.Test(t, f)`, which
gives the bubble its own `*testing.T`.

## Why there is no proof here

`synctest.Run` is gone: it is present in Go 1.25's source and absent from 1.26 and
1.27 (checked in this project's spec, probe P12). Code that calls it does not build
with any toolchain this repository supports, and a build failure is not a proof.

## What to do now

Use `synctest.Test`. Note what a bubble can see: a goroutine blocked on a channel
created inside the bubble is durably blocked; one waiting for a `sync.Mutex` is not, so
mutex deadlocks hang a bubble rather than show up in it (see R09).

## Residue

Nothing in supported toolchains; old blog posts and code samples still show
`synctest.Run`.

## Since & sources

The old rule held from `go1.24` until `go1.25`. Sources:
[Go 1.25 release notes](https://go.dev/doc/go1.25),
[`testing/synctest`](https://pkg.go.dev/testing/synctest).
