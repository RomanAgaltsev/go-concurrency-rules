package gate

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// ErrNeedsRace means a rule's proof is the race detector and this host cannot
// build with -race. Explain reports no verdict rather than a misleading one.
var ErrNeedsRace = errors.New("this rule needs the race detector")

// Explain runs one rule both ways — once each — and narrates what happened,
// for a reader rather than for the gate: the commands it ran, what each
// variant did, and whether that is what the rule predicts. It returns whether
// the rule behaved as declared.
//
// race says whether this host can build with -race. Without it, test and
// measure rules still run (their proofs do not depend on the detector) and say
// so; a race rule is not run at all (ErrNeedsRace): a race proof without the
// race detector would be the "assertion alone", which spec probe P3 measured
// as nearly vacuous.
func Explain(ctx context.Context, opt Options, id string, race bool, w io.Writer) (bool, error) {
	rules, _, err := Load(opt.Root)
	if err != nil {
		return false, err
	}
	i := slices.IndexFunc(rules, func(r Rule) bool { return strings.EqualFold(r.ID, id) })
	if i < 0 {
		return false, fmt.Errorf("no rule with id %s", id)
	}
	r := rules[i]
	if vs := validateRule(opt.Root, r, countIDs(rules)); len(vs) > 0 {
		return false, fmt.Errorf("%s has invalid front matter — run `gate check %s`: %s", r.ID, r.ID, vs[0].Msg)
	}
	p := opt.Prover
	p.Root = opt.Root

	fmt.Fprintf(w, "%s — %s\n  %s\n\n", r.ID, r.Title, r.Statement)
	var ok bool
	switch r.Proof.Kind {
	case "race":
		if !race {
			fmt.Fprintf(w, "This rule's proof needs the race detector, and this host cannot build with -race\n"+
				"(it needs cgo and a C compiler). Run it in the golang container instead:\n    %s\n",
				dockerGateCmd("run "+r.ID))
			return false, ErrNeedsRace
		}
		ok, err = p.explainTest(ctx, r, true, w)
	case "test":
		if !race {
			fmt.Fprint(w, "(without -race: this host has no C compiler, and this rule's proof does not need it)\n\n")
		}
		ok, err = p.explainTest(ctx, r, race, w)
	case "vet":
		ok, err = p.explainVet(ctx, r, w)
	case "measure":
		ok, err = p.explainMeasure(ctx, r, race, w)
	case "none":
		fmt.Fprintf(w, "No proof: %s\n", r.Proof.Reason)
		return true, nil
	default:
		return false, fmt.Errorf("proof kind %q has no explanation", r.Proof.Kind)
	}
	// A process killed by cancellation says nothing about the rule: report the
	// interruption, not the verdict it would otherwise read as (as Check does).
	if err := ctx.Err(); err != nil {
		return false, fmt.Errorf("interrupted while running %s: %w", r.ID, err)
	}
	if err != nil { // errCannotRun: no verdict either way
		return false, fmt.Errorf("running %s: %w", r.ID, err)
	}
	return ok, nil
}

// testArgs builds `go test` for one variant's proof test.
func testArgs(r Rule, race, broken bool) []string {
	args := []string{"test"}
	if race {
		args = append(args, "-race")
	}
	args = append(args, "-count=1", "-v", "-run", "^"+regexp.QuoteMeta(r.Proof.Test)+"$")
	if broken {
		args = append(args, "-tags", "broken")
	}
	return append(args, "./"+r.Dir)
}

// runShown prints the command, runs it, and returns its output and error. An
// error wrapping errCannotRun means the command never ran.
func (p Prover) runShown(ctx context.Context, w io.Writer, args []string) ([]byte, error) {
	fmt.Fprintf(w, "$ go %s\n", strings.Join(args, " "))
	out, err := p.goCmd(ctx, p.timeout(), args...)
	if cerr := cannotRun(err); cerr != nil {
		return out, cerr
	}
	return out, err
}

func (p Prover) explainTest(ctx context.Context, r Rule, race bool, w io.Writer) (bool, error) {
	ok := true
	out, err := p.runShown(ctx, w, testArgs(r, race, false))
	switch {
	case errors.Is(err, errCannotRun):
		return false, err
	case err != nil:
		fmt.Fprintf(w, "fixed: FAILED — the fixed variant should pass:\n%s\n\n", excerpt(out))
		ok = false
	default:
		fmt.Fprint(w, "fixed: passed\n\n")
	}

	out, err = p.runShown(ctx, w, testArgs(r, race, true))
	if errors.Is(err, errCannotRun) {
		return false, err
	}
	missing := ""
	for _, s := range r.Proof.Signature {
		if !bytes.Contains(out, []byte(s)) {
			missing = s
			break
		}
	}
	switch {
	case err == nil:
		fmt.Fprint(w, "broken: passed — the broken variant should fail\n")
		return false, nil
	case missing != "":
		fmt.Fprintf(w, "broken: failed, but without %q — not the failure this rule predicts:\n%s\n", missing, excerpt(out))
		return false, nil
	default:
		fmt.Fprintf(w, "broken: failed, printing %q — the failure this rule predicts\n", r.Proof.Signature[0])
	}
	fmt.Fprint(w, "\nThe admission gate runs the broken variant as separate processes and requires every one to fail like this.\n")
	return ok, nil
}

func (p Prover) explainVet(ctx context.Context, r Rule, w io.Writer) (bool, error) {
	pkg := "./" + r.Dir
	an := r.Proof.Analyzer
	ok := true

	fmt.Fprintf(w, "$ go vet -json %s\n", pkg)
	switch fixed, err := p.vet(ctx, pkg); {
	case errors.Is(err, errCannotRun):
		return false, err
	case err != nil:
		fmt.Fprintf(w, "fixed: %v\n\n", err)
		ok = false
	case len(fixed) > 0:
		fmt.Fprintf(w, "fixed: go vet reports %s — the fixed variant should be clean\n\n", describeVet(fixed))
		ok = false
	default:
		fmt.Fprint(w, "fixed: go vet finds nothing\n\n")
	}

	fmt.Fprintf(w, "$ go vet -json -tags broken %s\n", pkg)
	broken, err := p.vet(ctx, "-tags", "broken", pkg)
	if errors.Is(err, errCannotRun) {
		return false, err
	}
	if err != nil {
		fmt.Fprintf(w, "broken: %v\n", err)
		return false, nil
	}
	if len(broken[an]) == 0 {
		fmt.Fprintf(w, "broken: go vet reports no %s — the broken variant should trip it\n", an)
		return false, nil
	}
	for _, d := range broken[an] {
		fmt.Fprintf(w, "broken: %s — %s%s: %s\n", an, posnFile(d.Posn), posnRe.FindString(d.Posn), d.Message)
	}
	return ok, nil
}

func (p Prover) explainMeasure(ctx context.Context, r Rule, race bool, w io.Writer) (bool, error) {
	ok := true
	out, err := p.runShown(ctx, w, testArgs(r, race, false))
	switch {
	case errors.Is(err, errCannotRun):
		return false, err
	case err != nil:
		fmt.Fprintf(w, "fixed: FAILED — the fixed variant should pass:\n%s\n\n", excerpt(out))
		ok = false
	default:
		fmt.Fprint(w, "fixed: passed\n\n")
	}
	out, err = p.runShown(ctx, w, testArgs(r, race, true))
	switch {
	case errors.Is(err, errCannotRun):
		return false, err
	case err != nil:
		fmt.Fprintf(w, "broken: failed — a measure rule's broken variant must be correct, only slower:\n%s\n\n", excerpt(out))
		ok = false
	default:
		fmt.Fprint(w, "broken: passed (it is correct, only slower)\n\n")
	}

	dir := path.Join(r.Dir, "testdata", "measurements")
	entries, _ := os.ReadDir(filepath.Join(p.Root, filepath.FromSlash(dir)))
	var newest string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".txt" {
			newest = e.Name() // ReadDir sorts by name; the date leads the name
		}
	}
	if newest == "" {
		fmt.Fprintf(w, "No measurement on record in %s.\n", dir)
		return false, nil
	}
	data, err := os.ReadFile(filepath.Join(p.Root, filepath.FromSlash(dir), newest))
	if err != nil {
		return false, fmt.Errorf("%w: %w", errCannotRun, err)
	}
	fmt.Fprintf(w, "The cost, as measured (%s/%s):\n\n%s\n", dir, newest, firstComparison(data))
	cpus := "1,4"
	if m := regexp.MustCompile(`(?m)^# gomaxprocs: (\S+)`).FindSubmatch(data); m != nil {
		cpus = string(m[1])
	}
	fmt.Fprintf(w, "Re-measure on this machine: task measure -- %s %s\n", r.Dir, cpus)
	return ok, nil
}

// firstComparison returns benchstat's first comparison table: its two header
// lines and the rows up to the first blank line.
func firstComparison(data []byte) string {
	var lines []string
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := sc.Text()
		if len(lines) == 0 && !strings.Contains(line, "│") {
			continue
		}
		if strings.TrimSpace(line) == "" {
			break
		}
		lines = append(lines, "    "+line)
	}
	return strings.Join(lines, "\n")
}
