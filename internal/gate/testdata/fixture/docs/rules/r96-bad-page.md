---
id: R96
title: Fixture R96
statement: A fixture rule the gate must judge.
group: shared-memory
tags: [read]
since: go1.0
status: active
proof:
  kind: test
  test: TestAnswer
  signature: ["Answer() = "]
  runs: 10
alternatives: [R99]
sources:
  - https://go.dev/ref/mem
---
# Bad page

```go
func Answer() int { return 42 }
```
