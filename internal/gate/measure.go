package gate

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// MinSamples is the fewest benchmark samples per side a measurement may rest
// on (spec §5.3): single runs on a shared machine vary by tens of percent.
const MinSamples = 10

// measureHeader is what a measurement artifact must say about its regime
// before its numbers: who measured what, where, with which Go, and how.
var measureHeader = []string{"rule", "date", "go", "os", "cpu", "gomaxprocs", "command"}

var (
	benchLineRe = regexp.MustCompile(`(?m)^Benchmark\S*\s+\d+\s`)
	samplesRe   = regexp.MustCompile(`\bn=(\d+)(?:\+(\d+))?\)`)
	dateRe      = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	pkgRe       = regexp.MustCompile(`(?m)^pkg: (\S+)\s*$`)
)

// proveMeasure proves a measure rule. Its claim is about cost, so nothing is
// gated on failure; what the gate checks is that the claim can be trusted:
// both variants are correct (comparing the speed of a wrong program proves
// nothing), the benchmarks run in both, and every measurement on record names
// its regime and rests on at least MinSamples samples per side. The numbers
// themselves are judged in review (G3) — CI runners are too noisy to gate on.
func (p Prover) proveMeasure(ctx context.Context, r Rule) ([]Violation, string, error) {
	bad := func(format string, args ...any) []Violation {
		return []Violation{{Rule: r.ID, Gate: "G1", Msg: fmt.Sprintf(format, args...)}}
	}
	if _, vs, err := p.passes(ctx, r, "fixed", r.Proof.Runs); vs != nil || err != nil {
		return vs, "", err
	}
	if _, vs, err := p.passes(ctx, r, "broken", 1); vs != nil || err != nil {
		return vs, "", err
	}
	for _, variant := range []string{"fixed", "broken"} {
		args := []string{"test", "-run", "^$", "-bench", ".", "-benchtime", "1x"}
		if variant == "broken" {
			args = append(args, "-tags", "broken")
		}
		out, err := p.goCmd(ctx, p.timeout(), append(args, "./"+r.Dir)...)
		if err := cannotRun(err); err != nil {
			return nil, "", err
		}
		if err != nil {
			return bad("%s variant: benchmarks fail:\n%s", variant, excerpt(out)), "", nil
		}
		if !benchLineRe.Match(out) {
			return bad("%s variant: no benchmark ran — a measure rule needs a Benchmark function in both variants", variant), "", nil
		}
	}
	files, problems, err := checkMeasurements(p.Root, r)
	if err != nil {
		return nil, "", err
	}
	if len(problems) > 0 {
		return bad("%s", strings.Join(problems, "; ")), "", nil
	}
	return nil, fmt.Sprintf("measure: fixed passed %d×, broken is correct, benchmarks run in both; %d measurement(s) on record",
		r.Proof.Runs, files), nil
}

// checkMeasurements validates every artifact in the rule's
// testdata/measurements directory and returns how many there are. An artifact
// it cannot read is the gate's failure (errCannotRun), not the rule's.
func checkMeasurements(root string, r Rule) (int, []string, error) {
	rel := path.Join(r.Dir, "testdata", "measurements")
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(rel)))
	if errors.Is(err, fs.ErrNotExist) {
		return 0, []string{fmt.Sprintf("no measurement artifact in %s: a measure rule ships the numbers its page quotes", rel)}, nil
	}
	if err != nil {
		return 0, nil, fmt.Errorf("%w: %w", errCannotRun, err)
	}
	var problems []string
	n := 0
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".txt" {
			continue
		}
		n++
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel), e.Name()))
		if err != nil {
			return 0, nil, fmt.Errorf("%w: %w", errCannotRun, err)
		}
		for _, msg := range checkMeasurement(r.ID, r.Dir, data) {
			problems = append(problems, fmt.Sprintf("measurement %s: %s", e.Name(), msg))
		}
	}
	if n == 0 && len(problems) == 0 {
		problems = append(problems, fmt.Sprintf("no measurement artifact in %s: a measure rule ships the numbers its page quotes", rel))
	}
	return n, problems, nil
}

// checkMeasurement checks one artifact: a "# key: value" header naming the
// regime, then benchstat output comparing broken with fixed.
func checkMeasurement(id, dir string, data []byte) []string {
	header := map[string]string{}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line, ok := strings.CutPrefix(sc.Text(), "# ")
		if !ok {
			break
		}
		if k, v, ok := strings.Cut(line, ": "); ok {
			header[k] = strings.TrimSpace(v)
		}
	}
	var problems []string
	for _, k := range measureHeader {
		if header[k] == "" {
			problems = append(problems, fmt.Sprintf("header lacks %q", "# "+k+":"))
		}
	}
	if v := header["rule"]; v != "" && v != id {
		problems = append(problems, fmt.Sprintf("header says rule %s, but it belongs to %s", v, id))
	}
	if v := header["date"]; v != "" && !dateRe.MatchString(v) {
		problems = append(problems, fmt.Sprintf("header date %q is not YYYY-MM-DD", v))
	}
	// benchstat copies go test's "pkg:" line into its output: a witness the
	// header cannot fake. An artifact copied from another rule, header edited,
	// still names the package it measured.
	if m := pkgRe.FindSubmatch(data); m == nil {
		problems = append(problems, "no pkg: line — benchstat prints the measured package; was this produced by scripts/measure.sh?")
	} else if pkg := string(m[1]); pkg != dir && !strings.HasSuffix(pkg, "/"+dir) {
		problems = append(problems, fmt.Sprintf("measured package %q, not %s", pkg, dir))
	}
	if !bytes.Contains(data, []byte("vs base")) {
		problems = append(problems, "no benchstat comparison (benchstat broken.txt fixed.txt)")
	}
	counts, oneSided := scanComparisons(data)
	for _, name := range oneSided {
		problems = append(problems, fmt.Sprintf("row %q has no comparison (p=… n=…): it was measured on one side only", name))
	}
	if bytes.Contains(data, []byte("vs base")) && len(counts) == 0 && len(oneSided) == 0 {
		problems = append(problems, "no sample counts (n=…) in the comparison")
	}
	for _, k := range counts {
		if k < MinSamples {
			problems = append(problems, fmt.Sprintf("a comparison has n=%d; at least %d samples per side are required", k, MinSamples))
			break
		}
	}
	return problems
}

// scanComparisons walks benchstat's comparison tables — each starts at its
// "vs base" header and ends at a blank line — and returns the sample count
// each data row rests on, plus the names of rows that carry no comparison.
// benchstat prints n=10 when both sides have 10 samples and n=10+2 when they
// differ, so the smaller side is the count; a benchmark measured on one side
// only prints no (p=… n=…) at all.
func scanComparisons(data []byte) (counts []int, oneSided []string) {
	inTable := false
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.Contains(line, "vs base"):
			inTable = true
			continue
		case strings.TrimSpace(line) == "":
			inTable = false
			continue
		case !inTable, strings.Contains(line, "│"), strings.HasPrefix(line, "geomean"), isFootnote(line):
			continue // not a data row: headers, summaries, footnotes
		}
		m := samplesRe.FindStringSubmatch(line)
		if m == nil {
			oneSided = append(oneSided, strings.Fields(line)[0])
			continue
		}
		n, _ := strconv.Atoi(m[1])
		if m[2] != "" {
			if k, _ := strconv.Atoi(m[2]); k < n {
				n = k
			}
		}
		counts = append(counts, n)
	}
	return counts, oneSided
}

// isFootnote reports whether a line is one of benchstat's footnotes, which
// start with a superscript digit — a multi-byte rune, so the first byte alone
// cannot tell.
func isFootnote(line string) bool {
	r, _ := utf8.DecodeRuneInString(line)
	return strings.ContainsRune("¹²³⁴⁵⁶⁷⁸⁹", r)
}
