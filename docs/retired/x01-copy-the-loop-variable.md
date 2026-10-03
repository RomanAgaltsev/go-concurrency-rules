---
id: X01
title: Copy the loop variable before capturing it
statement: Before Go 1.22, copy a for-loop variable (i := i) before a closure or goroutine captures it.
group: ownership
tags: [read, write]
since: go1.0
status: retired
retired_in: go1.22
replaced_by: per-iteration loop variables — each iteration of a for loop creates new variables
proof:
  kind: test
  test: TestCapture
  signature:
    - "closures saw [3 3 3]"
  runs: 10
sources:
  - https://go.dev/doc/go1.22#language
  - https://go.dev/blog/loopvar-preview
---

# X01 — Copy the loop variable before capturing it

> **Retired in Go 1.22.** The old rule: before a closure or goroutine captures a
> `for` loop variable, copy it (`i := i`).

## What changed

Before Go 1.22, the variables a `for` loop declares were created once and updated by
each iteration. A closure that captured `i` captured that one variable, so every
closure saw its final value — and a goroutine started in the loop raced with the
loop's next update. Go 1.22 made each iteration create new variables: in the release
notes' words, "each iteration of the loop creates new variables, to avoid accidental
sharing bugs".

## The proof, both ways

The same code, twice. The broken file carries `//go:build broken && go1.21`, which
sets *that file's* language version to go1.21, so the old semantics come back inside
this Go 1.27 module:

```go
--8<-- "retired/x01-copy-the-loop-variable/broken.go:capture"
```

```go
--8<-- "retired/x01-copy-the-loop-variable/rule_test.go:test"
```

```console
$ go test ./retired/x01-copy-the-loop-variable/
ok  	…/retired/x01-copy-the-loop-variable
$ go test -tags broken ./retired/x01-copy-the-loop-variable/
--- FAIL: TestCapture
    rule_test.go:12: closures saw [3 3 3], want [0 1 2]
```

## What to do now

Nothing: delete `i := i` copies when you touch the code; they are harmless but noise.

## Residue

The new semantics apply per file, by language version: a module whose `go.mod` says
`go 1.21` or earlier — or a file whose `//go:build` line names such a version, as
above — still shares one variable across iterations.

## Since & sources

The old rule held from `go1.0` until `go1.22`. Sources:
[Go 1.22 release notes](https://go.dev/doc/go1.22#language),
[*Fixing For Loops in Go 1.22*](https://go.dev/blog/loopvar-preview).
