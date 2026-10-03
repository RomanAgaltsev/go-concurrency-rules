---
id: X91
title: Fixture X91
statement: A retired fixture entry proven with a file-level go1.21 term.
group: ownership
tags: [read]
since: go1.0
status: retired
retired_in: go1.22
replaced_by: per-iteration loop variables
proof:
  kind: test
  test: TestCapture
  signature: ["closures saw [3 3 3]"]
  runs: 10
sources:
  - https://go.dev/doc/go1.22
---
# Fixture X91
