package gate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// Options configures a gate run.
type Options struct {
	// Root is the repository root.
	Root string
	// IDs restricts the run to these rules; empty means every rule.
	IDs []string
	// Prover runs the proofs. Its Root is set from Options.Root.
	Prover Prover
}

// Admitted is a rule that passed every gate.
type Admitted struct {
	ID      string
	Summary string // what the proof ran, in one line
}

// Report is the outcome of a gate run.
type Report struct {
	Admitted   []Admitted
	Violations []Violation
}

// Check runs every gate over the selected rules.
//
// It returns an error, rather than violations, when the gate itself cannot do
// its job — no rules to check, an unknown ID, or a host that cannot build with
// -race. A gate that checked nothing must not look like a gate that passed.
func Check(ctx context.Context, opt Options) (Report, error) {
	rules, unparsed, err := Load(opt.Root)
	if err != nil {
		return Report{}, err
	}
	if len(rules)+len(unparsed) == 0 {
		return Report{}, fmt.Errorf("no rule pages under %s", filepath.Join(opt.Root, "docs", "rules"))
	}

	selected := rules
	if len(opt.IDs) > 0 {
		selected = nil
		for _, id := range opt.IDs {
			i := slices.IndexFunc(rules, func(r Rule) bool { return strings.EqualFold(r.ID, id) })
			if i < 0 {
				return Report{}, fmt.Errorf("no rule with id %s", id)
			}
			selected = append(selected, rules[i])
		}
	}

	if err := RaceAvailable(ctx, opt.Prover.Go); err != nil {
		return Report{}, err
	}

	// A page that does not parse is reported whatever was selected: it leaves
	// the set of IDs incomplete, and alternatives resolve against that set.
	rep := Report{Violations: unparsed}
	known := countIDs(rules)
	p := opt.Prover
	p.Root = opt.Root
	for _, r := range selected {
		own := validateRule(opt.Root, r, known)
		own = append(own, CheckPage(opt.Root, r)...)
		if slices.ContainsFunc(own, func(v Violation) bool { return v.Gate == "G6" }) {
			// The proof reads the front matter; with the front matter wrong,
			// running it would only bury the real finding in noise.
			rep.Violations = append(rep.Violations, own...)
			continue
		}
		pvs, summary := p.Prove(ctx, r)
		if err := ctx.Err(); err != nil {
			// Interrupted: the killed processes say nothing about the rule, and
			// reporting them as failures would be a verdict nobody reached.
			return Report{}, fmt.Errorf("interrupted while proving %s: %w", r.ID, err)
		}
		own = append(own, pvs...)
		if len(own) == 0 {
			rep.Admitted = append(rep.Admitted, Admitted{ID: r.ID, Summary: summary})
		}
		rep.Violations = append(rep.Violations, own...)
	}
	return rep, nil
}

// dockerGate is the gate run in the golang container, for a reader without
// Task. exec makes the gate the process that receives Ctrl-C.
const dockerGate = `docker run --rm --init -v "$PWD:/src" -w /src golang:1.27 ` +
	`sh -c 'go build -o /tmp/gate ./internal/cmd/gate && exec /tmp/gate check'`

// RaceAvailable reports whether this host can build with -race.
//
// It builds a trivial program instead of reading `go env`: a Windows host can
// report CGO_ENABLED=1 with no C compiler installed, and only the build tells.
func RaceAvailable(ctx context.Context, goBin string) error {
	if goBin == "" {
		goBin = "go"
	}
	dir, err := os.MkdirTemp("", "gate-race-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		return err
	}
	//nolint:gosec // G204: goBin is the go command the gate was told to use; running it is the point
	cmd := exec.CommandContext(ctx, goBin, "build", "-race", "-o", filepath.Join(dir, "racecheck.bin"), "main.go")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err() // interrupted, not incapable
		}
		if _, ok := errors.AsType[*exec.ExitError](err); !ok {
			return fmt.Errorf("cannot run %s: %w", goBin, err)
		}
		return fmt.Errorf("this host cannot build with -race, which needs cgo and a C compiler:\n%s\n"+
			"run the gate in a container instead, from the repository root — `task gate`, or:\n    %s",
			tail(out, 5), dockerGate)
	}
	return nil
}
