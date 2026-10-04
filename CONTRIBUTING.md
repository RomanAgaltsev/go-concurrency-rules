# Contributing

Every rule here is held to the same bar, the first and the hundredth: the
**admission gate**. Start a new rule as an issue (there is a template), then open a
pull request.

## A rule, in files

```text
docs/rules/r42-short-name.md        the page — front matter declares the proof
rules/r42-short-name/
    broken.go                       //go:build broken
    fixed.go                        //go:build !broken
    rule_test.go                    one test, run against whichever variant is built
    bench_test.go                   measure rules only
    testdata/measurements/*.txt     measure rules only: `task measure` writes these
```

Every Go block on the page is a snippet embedded from these files (`--8<--`); none
is typed into the page. Copy an existing rule of the same proof kind as a starting
point.

## Before you push

```console
$ task gen          # regenerate the group, activity and Retired index pages
$ task rule -- R42  # run your rule both ways and read what happened
$ task ci           # what CI runs (needs Docker)
```

Without a C compiler, `-race` only works in a container; `task gate` and
`task test:docker` use one. GitHub Codespaces has everything installed.

`site/` and `.cache/` are the docs build's output and belong to whoever built them
last. If a build stops with `site directory could not be cleaned: Permission
denied`, another user made them — the docs container, a dev container, a `sudo`
run. Both are git-ignored scratch: delete them and build again.

## Retiring a rule

A rule that expires moves to `docs/retired/` and `retired/`, keeps its ID, and gains
`status: retired`, `retired_in` and `replaced_by`. Its page URL changes with the
move; when the first active rule is retired, add a redirect from the old URL.
If no proof is possible any more (`proof.kind: none`, with a `reason`), delete its
code instead of moving it: the gate refuses code that nothing proves.

## Licences

Code is Apache-2.0 ([LICENSE](LICENSE)); prose under `docs/` is CC BY 4.0
([LICENSE-docs](LICENSE-docs)). By contributing you agree your contribution is
licensed the same way.
