# How to read a rule

Every rule page has the same parts, in the same order.

1. **The statement** — the rule in one sentence, under its ID and name. IDs are
   stable forever: R07 will always be this rule.
2. **Why** — the failure the rule prevents, explained by mechanism:
   happens-before edges, ownership, scheduling, what a channel does in each
   state.
3. **Broken / Fixed** — the two versions, in tabs. Both are real files in the
   repository, `broken.go` and `fixed.go`; a build tag picks which one is
   compiled.
4. **How to spot it** — what to look for when reading code, and which tools
   catch it: the race detector, a `go vet` analyzer, a linter — or that nothing
   does, which is often why the rule exists.
5. **Proof** — the test, and the exact command to run it.
6. **Trade-offs & when to break it** — a golden rule without its exceptions is
   folklore.
7. **Alternatives**, and **Since & sources** — the Go version the idiom needs,
   and at least one primary source.

## The kinds of proof

Each rule declares how it proves itself, and the admission gate checks that
proof on every change.

| Kind | The fixed version must | The broken version must |
|---|---|---|
| **race** | pass its test under `-race`, N times | fail in each of N separate processes, each time with `WARNING: DATA RACE` |
| **test** | pass its test, N times | fail in each of N separate processes, each time printing the failure the rule declares |
| **vet** | have no `go vet` findings at all | trip the declared analyzer, in `broken.go` |
| **measure** | be correct | be correct too — just slower; the page's numbers come from a recorded benchmark comparison |

Retired rules may carry **no proof**, when the old behaviour can no longer be
built at all; they say why.

*Separate processes* matter: the race detector reports a given race once per
process, so N iterations in one process prove much less than N processes do.
And *the declared failure* matters: a broken version that fails to compile also
exits with an error — the gate checks *why* it failed, not just *that* it did.

## Running one

```console
$ task rule -- R09
```

runs the rule's two versions once each and explains the result. `-race` needs a
C compiler; without one the command says so and shows how to run it in a
container — or open the repository in GitHub Codespaces, which has one.
