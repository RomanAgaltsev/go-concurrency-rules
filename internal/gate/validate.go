package gate

import (
	"bufio"
	"bytes"
	"fmt"
	"go/build/constraint"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// MinRuns is the smallest N a race or test proof may declare. The race
// detector reports a race once per process, so each run is a separate process;
// fewer than ten would let a broken variant that fails "usually" through.
const MinRuns = 10

var (
	groups = []string{"ownership", "shared-memory", "coordination", "time", "testing", "measurement"}
	tags   = []string{"read", "write", "refactor", "test", "profile"}
	kinds  = []string{"race", "test"}

	// genericMarkers appear in any failing test's output. A signature made only
	// of these cannot tell this rule's failure from any other — including one
	// the rule's author never intended.
	genericMarkers = []string{"FAIL", "--- FAIL", "panic", "exit status 1", "exit status 2"}

	idRe      = regexp.MustCompile(`^[RX]\d{2}$`)
	versionRe = regexp.MustCompile(`^go1\.\d+$`)
	testRe    = regexp.MustCompile(`^Test[A-Z_0-9]\w*$`)
)

// Validate checks every rule's front matter and the shape of its code
// directory (G6).
func Validate(root string, rules []Rule) []Violation {
	known := countIDs(rules)
	var vs []Violation
	for _, r := range rules {
		vs = append(vs, validateRule(root, r, known)...)
	}
	return vs
}

// countIDs maps each ID to the number of pages that declare it. A rule is
// validated against the whole set: alternatives must resolve, IDs be unique.
func countIDs(rules []Rule) map[string]int {
	known := make(map[string]int, len(rules))
	for _, r := range rules {
		known[r.ID]++
	}
	return known
}

func validateRule(root string, r Rule, known map[string]int) []Violation {
	var vs []Violation
	id := r.ID
	if id == "" {
		id = r.Slug
	}
	bad := func(format string, args ...any) {
		vs = append(vs, Violation{Rule: id, Gate: "G6", Msg: fmt.Sprintf(format, args...)})
	}

	switch {
	case !idRe.MatchString(r.ID):
		bad("id %q is not R or X followed by two digits", r.ID)
	case known[r.ID] > 1:
		bad("id %s is used by %d pages", r.ID, known[r.ID])
	case !strings.HasPrefix(r.Slug, strings.ToLower(r.ID)+"-"):
		bad("page %s does not start with %s-", r.Page, strings.ToLower(r.ID))
	}
	if strings.TrimSpace(r.Title) == "" {
		bad("title is empty")
	}
	if strings.TrimSpace(r.Statement) == "" {
		bad("statement is empty")
	}
	if !slices.Contains(groups, r.Group) {
		bad("group %q is not one of %s", r.Group, strings.Join(groups, ", "))
	}
	validateTags(r.Tags, bad)
	if !versionRe.MatchString(r.Since) {
		bad("since %q is not a go1.N version", r.Since)
	}
	validateStatus(r, bad)
	validateProof(r.Proof, bad)
	for _, alt := range r.Alternatives {
		switch {
		case alt == r.ID:
			bad("alternatives lists the rule itself")
		case known[alt] == 0:
			bad("alternative %s does not exist", alt)
		}
	}
	if len(r.Sources) == 0 {
		bad("sources is empty; at least one primary source is required")
	}
	for _, s := range r.Sources {
		if !strings.HasPrefix(s, "https://") {
			bad("source %q is not an https URL", s)
		}
	}
	validateTwin(root, r, bad)
	return vs
}

func validateTags(ts []string, bad func(string, ...any)) {
	if len(ts) == 0 {
		bad("tags is empty")
	}
	seen := map[string]bool{}
	for _, t := range ts {
		switch {
		case !slices.Contains(tags, t):
			bad("tag %q is not one of %s", t, strings.Join(tags, ", "))
		case seen[t]:
			bad("tag %q is listed twice", t)
		}
		seen[t] = true
	}
}

func validateStatus(r Rule, bad func(string, ...any)) {
	switch r.Status {
	case "active":
		if strings.HasPrefix(r.ID, "X") {
			bad("an X id is for retired entries; an active rule takes an R id")
		}
		if r.RetiredIn != "" || r.ReplacedBy != "" {
			bad("an active rule has no retired_in or replaced_by")
		}
	case "retired":
		if !versionRe.MatchString(r.RetiredIn) {
			bad("retired_in %q is not a go1.N version", r.RetiredIn)
		}
		if strings.TrimSpace(r.ReplacedBy) == "" {
			bad("a retired entry says what replaced it in replaced_by")
		}
	default:
		bad("status %q is not active or retired", r.Status)
	}
}

func validateProof(p Proof, bad func(string, ...any)) {
	if !slices.Contains(kinds, p.Kind) {
		bad("proof.kind %q is not one of %s", p.Kind, strings.Join(kinds, ", "))
		return
	}
	if !testRe.MatchString(p.Test) {
		bad("proof.test %q is not a test function name", p.Test)
	}
	if len(p.Signature) == 0 {
		bad("proof.signature is empty: the gate must know what the broken variant's failure prints")
	}
	specific := false
	for _, s := range p.Signature {
		switch t := strings.TrimSpace(s); {
		case t == "":
			bad("proof.signature has an empty entry, which every output contains")
		case !slices.Contains(genericMarkers, strings.TrimRight(t, ": ")):
			specific = true
		}
	}
	if len(p.Signature) > 0 && !specific {
		bad("proof.signature matches any failure at all; add what THIS rule's failure prints")
	}
	if p.Runs < MinRuns {
		bad("proof.runs is %d; at least %d separate processes are required", p.Runs, MinRuns)
	}
	if p.Analyzer != "" || p.Reason != "" {
		bad("proof.analyzer and proof.reason do not apply to kind %s", p.Kind)
	}
}

// validateTwin checks that the code directory holds the two variants and that
// their build constraints select exactly one of them for each build. A missing
// or wrong tag would otherwise surface later as a confusing build failure — or
// not at all, if both files happened to compile together.
func validateTwin(root string, r Rule, bad func(string, ...any)) {
	dir := filepath.Join(root, filepath.FromSlash(r.Dir))
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		bad("code directory %s does not exist", r.Dir)
		return
	}
	for _, f := range []struct {
		name       string
		withBroken bool // must the file be in the build when -tags broken is set?
	}{
		{"broken.go", true},
		{"fixed.go", false},
	} {
		expr, err := buildConstraint(filepath.Join(dir, f.name))
		if err != nil {
			bad("%s/%s: %v", r.Dir, f.name, err)
			continue
		}
		in := func(broken bool) bool {
			return expr.Eval(func(tag string) bool {
				if tag == "broken" {
					return broken
				}
				// A go1.N term selects a language version. The toolchain running
				// the gate satisfies it, so it never decides which variant builds.
				return strings.HasPrefix(tag, "go1.")
			})
		}
		if in(true) != f.withBroken || in(false) == f.withBroken {
			want := "//go:build !broken"
			if f.withBroken {
				want = "//go:build broken"
			}
			bad("%s/%s: build constraint %q does not select it like %q", r.Dir, f.name, expr.String(), want)
		}
	}
}

// buildConstraint returns the //go:build expression of a Go file.
func buildConstraint(file string) (constraint.Expr, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("cannot read: %w", err)
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if constraint.IsGoBuild(line) {
			return constraint.Parse(line)
		}
		if line != "" && !strings.HasPrefix(line, "//") {
			break // constraints must precede the package clause
		}
	}
	return nil, fmt.Errorf("has no //go:build line")
}
