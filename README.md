# go-concurrency-rules

Golden rules for concurrent Go — each one with broken and fixed code, and a test
that **proves** it: the test fails on the broken code, passes on the fixed code, and
CI checks both directions on every change.

> Not to be confused with a concurrency *exercise* gym: this is a reference. You
> consult a rule, run its proof, and change the code to see what breaks.

## Run a rule's proof

Every rule is a Go package under `rules/`. The build tag `broken` selects the
broken variant; the same test runs against either.

```console
$ go test -race ./rules/r07-race-is-a-missing-edge/                # fixed: ok
$ go test -race -tags broken ./rules/r07-race-is-a-missing-edge/   # broken: WARNING: DATA RACE
```

`-race` needs cgo and a C compiler. Without one (the Windows default), run it in a
container: `task test:docker`.

## The admission gate

A rule merges only when the gate admits it:

| Gate | Checks |
|---|---|
| G1 | The fixed variant passes N times; the broken variant fails in **N of N separate processes**, each printing the rule's declared failure signature |
| G2 | Every Go code block on the rule's page is embedded from the repository, and every embed resolves |
| G6 | The page's front matter is valid and its references resolve |

```console
$ task gate            # in the golang container; needs Docker
```

## Licences

Code: [Apache-2.0](LICENSE). Prose under `docs/`: [CC BY 4.0](LICENSE-docs).
