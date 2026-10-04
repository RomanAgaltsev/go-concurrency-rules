# Go concurrency rules

Golden rules for concurrent Go — for reading it, writing it, refactoring it,
testing it and profiling it. Each rule says what to do and *why*, by mechanism
rather than by authority, shows broken and fixed code side by side, and
**proves itself**: the repository holds a test that fails on the broken code
and passes on the fixed code, and CI checks both directions on every change.

## What makes a rule here different

- **Every snippet compiles and is tested.** The code on these pages is embedded
  from the repository; none of it is typed into the page.
- **Every broken version is proven broken** — in N separate processes out of N,
  each failing *for the declared reason*, not merely with a red exit code.
- **Every rule is dated.** Its `since` is the Go version it needs, and rules that
  expire move to [Retired rules](retired/index.md) instead of quietly going stale.
- **Measured claims carry their regime** — machine, Go version, sample count —
  and the artifact the numbers come from.

## Where to start

- [How to read a rule](how-to-read-a-rule.md) — the parts of a page, and what
  each kind of proof means.
- **By group:** [Ownership & lifetime](groups/ownership.md) ·
  [Shared memory](groups/shared-memory.md) · [Coordination](groups/coordination.md) ·
  [Time](groups/time.md) · [Testing](groups/testing.md) ·
  [Measurement](groups/measurement.md)
- **By what you are doing:** [reading](activity/read.md) ·
  [writing](activity/write.md) · [refactoring](activity/refactor.md) ·
  [testing](activity/test.md) · [profiling](activity/profile.md)

## Run a rule yourself

Clone the repository — or open it in GitHub Codespaces, where everything below
works out of the box — and run:

```console
$ task rule -- R07
```

That runs the rule's fixed and broken code once each and tells you what
happened. Then change the code and run it again.
