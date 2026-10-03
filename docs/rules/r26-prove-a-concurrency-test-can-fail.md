---
id: R26
title: Prove a concurrency test can fail
statement: Before trusting a concurrency test, run it against an implementation with a known bug and watch it fail.
group: testing
tags: [test]
since: go1.0
status: active
proof:
  kind: test
  test: TestCheckCanFail
  signature:
    - "check stayed green over a known bug"
  runs: 20
alternatives: []
undergo: []
sources:
  - https://pkg.go.dev/sync#Once.Do
  - https://pkg.go.dev/testing#TB
---

# R26 — Prove a concurrency test can fail

> **Before trusting a concurrency test, run it against an implementation with a known
> bug and watch it fail.**

## Why

A test that passes tells you two things at once — the code works, or the test cannot
see the bug — and nothing in a green run says which. Concurrency makes the second
much more likely: the bug lives in an interleaving, and a test that never produces
that interleaving passes forever. Sequential calls, one goroutine, a single
iteration, a mock that serialises everything: each one quietly turns a concurrency
test into a sequential one.

The only evidence that a test can catch a bug is having seen it catch one. So keep a
known-buggy implementation next to the test, and make "the test fails on it" part of
the suite.

## Broken / Fixed

The unit under test here is a *check*: `CheckRunsOnce` should fail on any `Initer`
that can run its function twice. The known bug is `FlakyOnce` — an atomic flag, so
the race detector stays silent, but check-then-act, so two callers that arrive
together both run `f`:

```go
--8<-- "rules/r26-prove-a-concurrency-test-can-fail/api.go:initer"
```

=== "Broken"

    ```go
    --8<-- "rules/r26-prove-a-concurrency-test-can-fail/broken.go:check"
    ```

=== "Fixed"

    ```go
    --8<-- "rules/r26-prove-a-concurrency-test-can-fail/fixed.go:check"
    ```

The broken check is the test most people write first, and it is green — on
`sync.Once` and on `FlakyOnce` alike. The fixed check makes the dangerous
interleaving happen on purpose: it holds the first call to `f` open until a second
goroutine is calling `Do`. A correct `Initer` keeps that second caller waiting —
`sync.Once` promises that no call to `Do` returns until the one call to `f` returns —
so the first call gives up after 100 ms and the check passes. `FlakyOnce` lets the
second caller in, and the check fails every time.

## How to spot it

**Reading.** For each concurrency test, ask: *what change to the code under test
would make this test fail?* If the honest answer is "none I can name", the test is
decoration. Look for tests whose "concurrent" part is a loop of sequential calls, a
single goroutine, or a `time.Sleep` that happens to order things.

**Tooling.** Mutation testing automates the question, at a cost. The cheap version is
this rule: one known-buggy implementation per check, kept in the test package and run
on every build.

## Proof

```go
--8<-- "rules/r26-prove-a-concurrency-test-can-fail/rule_test.go:test"
```

`runCheck` runs the check through a `recorder`, a `testing.TB` that records failures
instead of reporting them. It embeds the real `t` — `testing.TB` has unexported
methods, so embedding is the only way to implement it — and runs the check in its
own goroutine, so a `Fatal` inside the check ends the check, not the test.

```console
$ go test -race ./rules/r26-prove-a-concurrency-test-can-fail/
ok  	…/rules/r26-prove-a-concurrency-test-can-fail
$ go test -race -tags broken ./rules/r26-prove-a-concurrency-test-can-fail/
--- FAIL: TestCheckCanFail
    rule_test.go:17: check stayed green over a known bug: FlakyOnce runs f twice when two callers race
```

The test also runs the check against `sync.Once` and requires it to pass: a check that
fails on everything would pass the first half for the wrong reason.

## Trade-offs & when to break it

A known-buggy implementation is code to maintain, and a forced interleaving couples
the check to *how* the bug happens. Keep both small and keep them next to the check.
The 100 ms bound is a cost paid on every run of a correct implementation, and it is
still a bound: the second goroutine has to get from announcing itself to calling `Do`
— a few instructions — inside it. A machine stalled for longer at exactly that point
would let `FlakyOnce` through. The gate keeps that margin honest: it requires the
fixed variant to pass 20 times in a row under `-race`, and every one of those passes
includes the strong check catching `FlakyOnce`.

## Alternatives

Deterministic schedulers (`testing/synctest`, where the code's blocking is durable)
and linearizability checkers each get their own rule as they land.

## Since & sources

`since: go1.0`. Sources: [`sync.Once.Do`](https://pkg.go.dev/sync#Once.Do),
[`testing.TB`](https://pkg.go.dev/testing#TB).
