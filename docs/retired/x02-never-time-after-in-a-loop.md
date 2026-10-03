---
id: X02
title: Never call time.After in a loop
statement: Before Go 1.23, a time.After timer that had not fired could not be collected, so calling it in a loop leaked a timer per iteration.
group: time
tags: [read, write]
since: go1.0
status: retired
retired_in: go1.23
replaced_by: timers that are no longer referenced are collected even if never stopped
proof:
  kind: none
  reason: with Go 1.27 the old timer behaviour cannot be selected at all — not per file, not by the main module's go line, not by GODEBUG — and the leak it caused showed only as memory held over time, not as a pass or a fail.
sources:
  - https://go.dev/doc/go1.23#timer-changes
  - https://go.dev/doc/godebug#go-123
---

# X02 — Never call `time.After` in a loop

> **Retired in Go 1.23.** The old rule: don't call `time.After` in a loop (typically
> in a `select`); every call leaks a timer until it fires.

## What changed

Go 1.23 changed timers in two ways. From the release notes: "Timers and Tickers that
are no longer referred to by the program become eligible for garbage collection
immediately, even if their Stop methods have not been called" — earlier versions kept
an unstopped Timer until it fired, and never collected an unstopped Ticker. And the
timer channel became unbuffered, so after `Reset` or `Stop` returns, no stale value is
received.

## Why there is no proof here

From Go 1.23 to 1.26 the old behaviour was chosen per program, never per file: "When
Go 1.23 builds older programs, the old behaviors remain in effect", and the
`asynctimerchan=1` GODEBUG brought back the old channel behaviour. Go 1.27 — this
repository's floor — removed even that: built with go1.27.1, a module that says
`go 1.22` gets the new unbuffered timer channel (`cap(timer.C)` is 0, where go1.26.8
gives 1), and `GODEBUG=asynctimerchan=1` stops the program with `fatal error: removed
GODEBUG "asynctimerchan"`. There is no broken variant left to build. The leak itself
was never a failing assertion anyway — it was memory held for as long as the timer's
duration.

## What to do now

`time.After` in a `select` loop is no longer a leak. On a hot path it is still one
timer allocation per iteration: reuse a `time.Timer` and `Reset` it there, for cost —
not for correctness.

## Residue

Only on older toolchains: built with Go 1.23–1.26, a program whose main module says
`go 1.22` or earlier keeps the old timers. Built with Go 1.27, such a program gets the
new timer channels (measured above), and the setting that could bring the old ones
back no longer exists.

## Since & sources

The old rule held from `go1.0` until `go1.23`. Sources:
[Go 1.23 release notes, timer changes](https://go.dev/doc/go1.23#timer-changes);
[GODEBUG history](https://go.dev/doc/godebug#go-123) ("This setting will be removed in
Go 1.27"; the Go 1.27 section: "Go 1.27 removed the asynctimerchan setting").
