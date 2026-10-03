---
id: R12
title: Never copy a sync value
statement: Hold mutexes, wait groups and other sync values by pointer; a copy is a second, unrelated lock.
group: shared-memory
tags: [read]
since: go1.0
status: active
proof:
  kind: vet
  analyzer: copylocks
alternatives: []
undergo: []
sources:
  - https://pkg.go.dev/sync#Mutex
  - https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/copylock
---

# R12 — Never copy a sync value

> **Hold mutexes, wait groups and other sync values by pointer; a copy is a second,
> unrelated lock.**

## Why

A `sync.Mutex` is a small struct whose state *is* the lock. Copy it and you get a
second lock with the same state at the moment of the copy and no connection
afterwards: locking one does nothing to the other. The `sync` package says so in each
type's documentation — "A Mutex must not be copied after first use", and the same for
`WaitGroup`, `Once`, `Cond`, `RWMutex` and `Map`.

Go copies values silently: assignment, passing an argument, returning a result,
ranging over a slice of structs, and — the easiest to miss — calling a method with a
**value receiver**. Every one of those copies a `sync` field along with the struct
that holds it.

## Broken / Fixed

=== "Broken"

    ```go
    --8<-- "rules/r12-never-copy-a-sync-value/broken.go:stats"
    ```

=== "Fixed"

    ```go
    --8<-- "rules/r12-never-copy-a-sync-value/fixed.go:stats"
    ```

One character apart: `Hits` takes `s Stats` instead of `s *Stats`. The broken
`Hits` locks its private copy, so the lock protects nothing, and copying the struct
reads `hits` while another goroutine's `Hit` may be writing it — a data race.

## How to spot it

**Reading.** A struct with a `sync` field must only travel by pointer. Look for value
receivers on such a type, functions that take or return it by value, `for _, v :=
range` over a slice of it, and `*p` dereferences that copy it.

**Tooling.** `go vet`'s `copylocks` analyzer reports all of these. Note what does
*not* run it: `go test` runs only a high-confidence subset of vet (`atomic`, `bools`,
`buildtag`, `directive`, `errorsas`, `ifaceassert`, `nilfunc`, `printf`,
`stdversion`, `stringintconv`, `tests`), so the broken variant here builds and passes
its tests. Run `go vet` — or golangci-lint, whose `govet` includes `copylocks` —
explicitly.

## Proof

```console
$ go vet ./rules/r12-never-copy-a-sync-value/
$ go vet -tags broken ./rules/r12-never-copy-a-sync-value/
…/broken.go:25:9: Hits passes lock by value: …Stats contains sync.Mutex
```

The admission gate runs both with `go vet -json` and reads the JSON: the fixed
variant must have no diagnostics at all, the broken one at least one from
`copylocks`. It never trusts vet's exit code — with `-json`, vet exits 0 even when it
reports.

## Trade-offs & when to break it

A zero value that has never been used may be copied: `var s Stats; t := s` before
either is touched is legal, and vet still reports it, because it cannot see "before
first use". Construct such values in place instead (`t := &Stats{}`), and the warning
goes away along with the doubt.

## Alternatives

Keep the lock out of the value entirely: share a `*Stats`, or let one goroutine own
the state and talk to it over a channel — each gets its own rule as it lands.

## Since & sources

`since: go1.0`. Sources: [`sync.Mutex`](https://pkg.go.dev/sync#Mutex) ("must not be
copied after first use"), the
[copylock analyzer](https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/copylock).
