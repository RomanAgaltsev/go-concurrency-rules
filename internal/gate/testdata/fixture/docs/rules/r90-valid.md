---
id: R90
title: Fixture R90
statement: A fixture rule the gate must judge.
group: shared-memory
tags: [read]
since: go1.0
status: active
proof:
  kind: race
  test: TestCount
  signature: ["WARNING: DATA RACE"]
  runs: 10
sources:
  - https://go.dev/ref/mem
---
# Valid

=== "Broken"

    ```go
    --8<-- "rules/r90-valid/broken.go:count"
    ```

=== "Fixed"

    ```go
    --8<-- "rules/r90-valid/fixed.go:count"
    ```
