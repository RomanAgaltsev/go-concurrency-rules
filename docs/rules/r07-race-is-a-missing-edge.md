---
id: R07
title: A data race is a missing happens-before edge
statement: Fix a race by adding the missing happens-before edge, not by moving code until the detector goes quiet.
group: shared-memory
tags: [read, write]
since: go1.0
status: active
proof:
  kind: race
  test: TestConcurrentInc
  signature:
    - "WARNING: DATA RACE"
  runs: 20
alternatives: []
undergo: []
sources:
  - https://go.dev/ref/mem
  - https://go.dev/doc/articles/race_detector
---

# R07 — A data race is a missing happens-before edge

> **Fix a race by adding the missing happens-before edge, not by moving code until
> the detector goes quiet.**

## Why

Two memory accesses race when they touch the same variable, at least one of them
writes, and **neither happens before the other**. That last clause is the whole
definition. Go's memory model does not promise that the program sees the writes in
the order the source lists them; it promises only what its *happens-before* edges
imply. Within one goroutine, statement order gives you edges for free. Between
goroutines, every edge comes from synchronisation: a channel send and the receive
that takes it, an `Unlock` and the next `Lock` of the same mutex, a `WaitGroup.Done`
and the `Wait` it releases, an atomic store and the load that observes it.

So a race report is not "two goroutines ran at once". It names two accesses that no
chain of edges connects. Reordering statements, adding a `time.Sleep`, or making
the window smaller changes *how often* the bad interleaving occurs; it adds no edge,
so the program is exactly as wrong as before — the detector may just stop seeing it.
The fix is to decide which edge is missing and add that edge.

## Broken / Fixed

Both variants are driven the same way: four goroutines, a thousand increments each,
then a wait.

```go
--8<-- "rules/r07-race-is-a-missing-edge/api.go:drive"
```

=== "Broken"

    ```go
    --8<-- "rules/r07-race-is-a-missing-edge/broken.go:counter"
    ```

=== "Fixed"

    ```go
    --8<-- "rules/r07-race-is-a-missing-edge/fixed.go:counter"
    ```

`wg.Wait()` already supplies an edge from every increment to the final read, which
is why `Value` in the broken version is *not* where the race is. What no code
supplies is an edge between one goroutine's `c.n++` and another's. The mutex adds
exactly that: each `Unlock` happens before the next `Lock`, so every increment is
ordered against every other.

## How to spot it

**Reading.** For every variable a goroutine touches, ask *what orders this access
against the other goroutine's access?* and name the edge. If the answer is "it runs
later", "it's fast" or "it's only a counter", there is no edge.

**Tooling.** The race detector (`go test -race`) reports races in the executions
you run, and only those: an interleaving your tests never produce is never checked
(R22, when it lands, is about that gap). It reports each distinct race once per
process — run a racy test twenty times in one process and you get one or two reports.

## Proof

The same test is compiled against each variant:

```go
--8<-- "rules/r07-race-is-a-missing-edge/rule_test.go:test"
```

```console
$ go test -race ./rules/r07-race-is-a-missing-edge/
ok  	…/rules/r07-race-is-a-missing-edge
$ go test -race -tags broken ./rules/r07-race-is-a-missing-edge/
WARNING: DATA RACE
…
--- FAIL: TestConcurrentInc
```

The admission gate builds the broken variant once and runs it as **20 separate
processes**; each must fail and print `WARNING: DATA RACE`. Without `-race` the same
assertion catches a lost update only occasionally — the detector, not the
assertion, is the proof.

## Trade-offs & when to break it

Do not look for a "benign" race. Go defines more for a racy program than C does — a
read of a word-sized variable observes some value that was actually written — but
that is all it promises: which write you see is up to the race, values larger than a
word can be observed half-updated, and an implementation may report the race and
stop the program. What *is* a choice is which edge to add. A
mutex is the general answer; a single word that is only ever loaded and stored whole
can use an atomic instead; state that one goroutine owns needs no edge at all,
because nothing else touches it.

## Alternatives

Atomics for a single word, and ownership transfer through a channel, each get their
own rule as they land.

## Since & sources

`since: go1.0` — happens-before has defined Go's concurrency since the first memory
model. Sources: [The Go Memory Model](https://go.dev/ref/mem),
[Data Race Detector](https://go.dev/doc/articles/race_detector).
