## What this changes

<!-- One or two sentences. For a rule: its ID and statement. -->

## Admission checklist (new or changed rules)

CI's `gate` job checks **G1** (the proof holds both ways), **G2** (every Go block is an
embed that resolves) and **G6** (front matter). A reviewer checks the rest:

- [ ] **G3** — every number on the page cites a committed measurement artifact, with its regime
- [ ] **G4** — `since` is checked against the release notes, and the page has at least one primary source
- [ ] **G5** — nothing is copied from licensed sources: ideas re-derived, code and prose new
- [ ] `task gen` was run and the regenerated index pages are committed
