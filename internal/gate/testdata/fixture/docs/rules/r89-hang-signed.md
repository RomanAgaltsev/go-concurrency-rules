---
id: R89
title: Fixture R89
statement: A fixture rule the gate must judge.
group: shared-memory
tags: [read]
since: go1.0
status: active
proof:
  kind: test
  test: TestAnswer
  signature: ["r89-hang-signed/broken.go"]
  runs: 10
sources:
  - https://go.dev/ref/mem
---
# Fixture R89
