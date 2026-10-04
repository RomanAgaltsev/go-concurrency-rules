---
name: New rule
about: Propose a golden rule for concurrent Go
title: "Rule: <imperative name>"
---

## The rule

**Statement** (one sentence):

**Why** — the failure it prevents, by mechanism (happens-before, ownership, scheduling…):

## Broken / fixed

A sketch of the broken code and the fix — they become `broken.go` and `fixed.go`.

## Proof

**Kind:** race | test | vet | measure
**What the broken variant's failure prints** (race/test), or **the analyzer** (vet), or
**the benchmark and machine** (measure):

## Since & sources

**Since:** go1.N — **Primary source(s):**

## Before it can merge

- [ ] G1 — the proof holds: fixed passes N times; broken fails N/N in separate processes with its signature
- [ ] G2 — every Go block on the page is embedded from the repository
- [ ] G3 — every number cites a committed measurement artifact
- [ ] G4 — `since` checked against the release notes; a primary source
- [ ] G5 — original code and prose
- [ ] G6 — front matter valid; alternatives resolve
