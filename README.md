# go-concurrency-rules

Golden rules for concurrent Go — each one with broken and fixed code, and a test
that **proves** it: the test fails on the broken code, passes on the fixed code, and
CI checks both directions on every change.

> Not to be confused with a concurrency *exercise* gym: this is a reference. You
> consult a rule, run its proof, and change the code to see what breaks.

**Read the rules:** https://romanagaltsev.github.io/go-concurrency-rules/

[![Open in GitHub Codespaces](https://github.com/codespaces/badge.svg)](https://codespaces.new/RomanAgaltsev/go-concurrency-rules)

## Run a rule

```console
$ task rule -- R07
```

runs the rule's fixed and broken code once each and explains what happened. Every
rule is a Go package under `rules/`; the build tag `broken` selects the broken
variant, and the same test runs against either:

```console
$ go test -race ./rules/r07-race-is-a-missing-edge/                # fixed: ok
$ go test -race -tags broken ./rules/r07-race-is-a-missing-edge/   # broken: WARNING: DATA RACE
```

`-race` needs cgo and a C compiler. Without one (the Windows default), `task rule`
says so and shows the container command; GitHub Codespaces has everything.

## The site

The pages under `docs/` are built with [Zensical](https://zensical.org), pinned in
`requirements-docs.txt`. The group, activity and Retired index pages are generated
from the rules' front matter by `task gen`; CI fails if they are stale.

```console
$ task docs:serve          # with Python: pip install -r requirements-docs.txt
$ task docs:serve:docker   # without Python, in a container
```

## The admission gate

A rule merges only when the gate admits it:

| Gate | Checks |
|---|---|
| G1 | The rule's proof holds, by kind — **race / test:** the fixed variant passes N times; the broken variant fails in **N of N separate processes**, each printing the rule's declared failure signature. **vet:** `go vet -json` finds nothing in the fixed variant and the declared analyzer in the broken one. **measure:** both variants are correct, their benchmarks run, and every measurement on record names its regime and has ≥ 10 samples per side. **none:** retired entries only, with the reason no proof is possible |
| G2 | Every Go code block on the rule's page is embedded from a `.go` file the repository compiles, and every embed resolves |
| G6 | The page's front matter is valid and its references resolve |

```console
$ task gate            # in the golang container; needs Docker
$ task measure -- rules/r01-start-sequential 1,4,12   # record a measure rule's numbers
```

Rules live under `rules/`; rules that **were** golden and expired live under
`retired/`, each saying which Go version retired it and what replaced it.

## Licences

Code: [Apache-2.0](LICENSE). Prose under `docs/`: [CC BY 4.0](LICENSE-docs).
