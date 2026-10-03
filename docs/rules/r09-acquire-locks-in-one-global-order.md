---
id: R09
title: Acquire locks in one global order
statement: When code holds two locks at once, every path takes them in the same global order.
group: shared-memory
tags: [read, write]
since: go1.0
status: active
proof:
  kind: test
  test: TestOpposingTransfers
  signature:
    - "watchdog: deadlock"
  runs: 20
alternatives: []
undergo: []
sources:
  - https://doi.org/10.1145/356586.356588
  - https://pkg.go.dev/testing/synctest
---

# R09 — Acquire locks in one global order

> **When code holds two locks at once, every path takes them in the same global
> order.**

## Why

A deadlock needs four things at once (Coffman et al., 1971): resources held
exclusively, held while waiting for more, never taken away, and a **cycle** of
waiters — A holds what B wants while B holds what A wants. Mutexes supply the first
three by design, so the cycle is the only one left to remove.

One global order removes it. If every goroutine that needs two locks takes them
lowest-first, a goroutine can only ever wait for a lock *later* in the order than
every lock it holds — and a chain of "waits for a later lock" cannot loop back to
its start.

The trap is an order that comes from the *arguments*. `Transfer(from, to)` locking
`from` then `to` looks ordered, but `Transfer(a, b)` and `Transfer(b, a)` running
together take the same two locks in opposite orders.

## Broken / Fixed

Both variants share the account type:

```go
--8<-- "rules/r09-acquire-locks-in-one-global-order/api.go:account"
```

=== "Broken"

    ```go
    --8<-- "rules/r09-acquire-locks-in-one-global-order/broken.go:transfer"
    ```

=== "Fixed"

    ```go
    --8<-- "rules/r09-acquire-locks-in-one-global-order/fixed.go:transfer"
    ```

The order is by `ID`, a property of the account, not of the call. Any total order
works — an ID, an address you have reason to trust, a level in a hierarchy — as long
as every path that holds two of these locks uses the same one.

## How to spot it

**Reading.** Find every place that holds one lock while taking another. For each
pair, ask what decides which is taken first. If the answer is "whichever argument
comes first", "whichever the caller passed", or "it depends on the path", there is a
cycle waiting for the right timing. Callbacks run under a lock count: they may take
locks you cannot see.

**Tooling.** Little helps. The runtime reports `all goroutines are asleep -
deadlock!` only when *every* goroutine is blocked, which a real program with any
other goroutine running never is. `go test -race` sees no race here — both
goroutines lock correctly; they just lock in different orders. And `testing/synctest`
cannot see it either: its documentation lists locking a `sync.Mutex` among the
operations that are *not* durably blocking, so a bubble whose goroutines are stuck on
mutexes never looks idle and simply hangs.

## Proof

The deadlocking interleaving is rare by timing alone, so the test **forces** it: a
hook between the two locks holds each `Transfer` until both have taken their first
lock. A **watchdog** then turns "hangs forever" into a fast, declared failure — real
time used to detect a failure, never to synchronise.

```go
--8<-- "rules/r09-acquire-locks-in-one-global-order/rule_test.go:test"
```

```console
$ go test -race ./rules/r09-acquire-locks-in-one-global-order/
ok  	…/rules/r09-acquire-locks-in-one-global-order
$ go test -race -tags broken ./rules/r09-acquire-locks-in-one-global-order/
--- FAIL: TestOpposingTransfers
    rule_test.go:52: watchdog: deadlock — Transfer(a, b) and Transfer(b, a) each hold the lock the other needs
```

With the global order, the second `Transfer` waits for the first lock, never reaches
the hook, and the hook's 100 ms limit lets the first one finish. The admission gate
runs the broken variant as 20 separate processes; each must fail with the watchdog's
message.

## Trade-offs & when to break it

A global order is a design constraint every future lock must fit into, and it gets
harder to keep as the number of locks grows. Often the better move is to need only
one lock: a single mutex over both accounts, or one goroutine that owns all balances
and receives transfers over a channel. `TryLock` and back-off can break a cycle at
run time, but trade a deadlock for livelock risk and code that is much harder to
reason about.

## Alternatives

Fewer locks held at once, or ownership by a single goroutine — each gets its own rule
as it lands.

## Since & sources

`since: go1.0`. Sources: E. G. Coffman, M. Elphick, A. Shoshani,
[*System Deadlocks*](https://doi.org/10.1145/356586.356588), Computing Surveys 3(2),
1971; [`testing/synctest`](https://pkg.go.dev/testing/synctest) ("locking a
sync.Mutex or sync.RWMutex" is not durably blocking).
