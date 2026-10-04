package gate

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"go/build/constraint"
	"io/fs"
	"maps"
	"os"
	"path"
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

	idRe       = regexp.MustCompile(`^[RX]\d{2}$`)
	versionRe  = regexp.MustCompile(`^go1\.\d+$`)
	testRe     = regexp.MustCompile(`^Test[A-Z_0-9]\w*$`)
	analyzerRe = regexp.MustCompile(`^[a-z][a-z0-9]*$`)
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

// CheckCodeDirs refuses (G6) every code directory under rules/ or retired/
// without a page of the same name: nothing proves it and nothing publishes it.
// A renamed or deleted page leaves one behind. Violations are keyed by the
// directory's path.
func CheckCodeDirs(root string) ([]Violation, error) {
	var vs []Violation
	for _, c := range collections {
		entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(c.code)))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			page := path.Join(c.docs, e.Name()+".md")
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(page))); errors.Is(err, fs.ErrNotExist) {
				vs = append(vs, Violation{
					Rule: path.Join(c.code, e.Name()),
					Gate: "G6",
					Msg:  fmt.Sprintf("has no page %s, so nothing proves or publishes it", page),
				})
			} else if err != nil {
				return nil, err
			}
		}
	}
	return vs, nil
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
	if r.Proof.Kind == "none" && r.Status != "retired" {
		bad("kind none is only for retired entries: an active rule must prove itself")
	}
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
	switch {
	case r.Proof.Kind != "none":
		validateTwin(root, r, bad)
	case r.Status == "retired":
		// An entry without a proof has no twin. Code under its name would sit in
		// the repository looking like a proof that nothing runs.
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(r.Dir))); err == nil {
			bad("kind none, but code directory %s exists: nothing proves it — give the entry a proof or delete the directory", r.Dir)
		}
	}
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
	inRetired := strings.HasPrefix(r.Page, "docs/retired/")
	switch r.Status {
	case "active":
		if strings.HasPrefix(r.ID, "X") {
			bad("an X id is for retired entries; an active rule takes an R id")
		}
		if r.RetiredIn != "" || r.ReplacedBy != "" {
			bad("an active rule has no retired_in or replaced_by")
		}
		if inRetired {
			bad("an active rule's page lives in docs/rules/, not %s", r.Page)
		}
	case "retired":
		if !inRetired {
			bad("a retired entry's page lives in docs/retired/, not %s", r.Page)
		}
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
	takes, ok := kindFields[p.Kind]
	if !ok {
		bad("proof.kind %q is not one of %s", p.Kind, strings.Join(slices.Sorted(maps.Keys(kindFields)), ", "))
		return
	}
	// A field another kind uses is a sign the page declares a different proof
	// than its author meant — refuse it rather than ignore it.
	for _, f := range []struct {
		name string
		set  bool
	}{
		{"test", p.Test != ""},
		{"signature", len(p.Signature) > 0},
		{"analyzer", p.Analyzer != ""},
		{"runs", p.Runs != 0},
		{"reason", p.Reason != ""},
	} {
		if f.set && !slices.Contains(takes, f.name) {
			bad("proof.%s does not apply to kind %s", f.name, p.Kind)
		}
	}
	switch p.Kind {
	case "race", "test":
		validateTestProof(p, bad)
	case "vet":
		if !analyzerRe.MatchString(p.Analyzer) {
			bad("proof.analyzer %q is not a go vet analyzer name (see `go tool vet help`)", p.Analyzer)
		}
	case "measure":
		validateTestName(p.Test, bad)
		validateRuns(p.Runs, bad)
	case "none":
		if strings.TrimSpace(p.Reason) == "" {
			bad("proof.reason is empty: an entry without a proof says why it cannot carry one")
		}
	}
}

// kindFields lists, per proof kind, the proof fields it uses.
var kindFields = map[string][]string{
	"race":    {"test", "signature", "runs"},
	"test":    {"test", "signature", "runs"},
	"vet":     {"analyzer"},
	"measure": {"test", "runs"},
	"none":    {"reason"},
}

func validateTestName(test string, bad func(string, ...any)) {
	if !testRe.MatchString(test) {
		bad("proof.test %q is not a test function name", test)
	}
}

func validateRuns(runs int, bad func(string, ...any)) {
	if runs < MinRuns {
		bad("proof.runs is %d; at least %d are required", runs, MinRuns)
	}
}

// validateTestProof checks a race or test proof: what to run, what its
// failure must print, and how many separate processes must print it.
func validateTestProof(p Proof, bad func(string, ...any)) {
	validateTestName(p.Test, bad)
	if len(p.Signature) == 0 {
		bad("proof.signature is empty: the gate must know what the broken variant's failure prints")
	}
	specific := false
	for _, s := range p.Signature {
		switch t := strings.TrimSpace(s); {
		case t == "":
			bad("proof.signature has an empty entry, which every output contains")
		case !slices.ContainsFunc(harnessLines(p.Test), func(h string) bool { return strings.Contains(h, t) }):
			specific = true
		}
	}
	if len(p.Signature) > 0 && !specific {
		bad("proof.signature matches any failure at all; add what THIS rule's failure prints")
	}
	if p.Kind == "race" && len(p.Signature) > 0 && !slices.ContainsFunc(p.Signature, func(s string) bool { return strings.Contains(s, raceWarning) }) {
		bad("kind race must list %q in proof.signature: the detector, not an assertion, is a race proof", raceWarning)
	}
	validateRuns(p.Runs, bad)
}

// raceWarning is what the race detector prints for every race it reports.
const raceWarning = "WARNING: DATA RACE"

// harnessLines is text the testing package prints for test on any run —
// passing, failing or hanging. A signature entry found inside one of them
// says nothing about why the broken variant failed: the gate itself runs every
// process with -test.v, so "=== RUN   TestX" is in every output, and a
// -test.timeout panic prints "running tests:" and the test's name.
func harnessLines(test string) []string {
	return []string{
		"=== RUN   " + test,
		"--- FAIL: " + test + " (",
		"--- PASS: " + test + " (",
		"panic: test timed out after",
		"running tests:",
		"goroutine ",
		"ok  ", "PASS", "FAIL",
		"exit status 1", "exit status 2",
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
		if inBuild(expr, true) != f.withBroken || inBuild(expr, false) == f.withBroken {
			want := "//go:build !broken"
			if f.withBroken {
				want = "//go:build broken"
			}
			bad("%s/%s: build constraint %q does not select it like %q", r.Dir, f.name, expr.String(), want)
		}
	}

	// The proof is one test run against both variants. A test file only one
	// build sees would fail later as "no tests to run" or a pass count of zero.
	entries, err := os.ReadDir(dir)
	if err != nil {
		bad("code directory %s: %v", r.Dir, err)
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		expr, err := buildConstraint(filepath.Join(dir, e.Name()))
		switch {
		case errors.Is(err, errNoBuildLine):
		case err != nil:
			bad("%s/%s: %v", r.Dir, e.Name(), err)
		case inBuild(expr, true) != inBuild(expr, false):
			bad("%s/%s: build constraint %q — a test file must build with both variants", r.Dir, e.Name(), expr.String())
		}
	}
}

// inBuild reports whether a file with build constraint expr is in the build,
// with or without -tags broken.
func inBuild(expr constraint.Expr, broken bool) bool {
	return expr.Eval(func(tag string) bool {
		if tag == "broken" {
			return broken
		}
		// A go1.N term selects a language version. The toolchain running the
		// gate satisfies it, so it never decides which variant builds.
		return strings.HasPrefix(tag, "go1.")
	})
}

// errNoBuildLine: the file has no //go:build line before its package clause.
var errNoBuildLine = errors.New("has no //go:build line")

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
	return nil, errNoBuildLine
}
