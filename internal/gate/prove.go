package gate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

// DefaultTimeout bounds one test process. A proof that needs longer is
// waiting on something it should not be.
const DefaultTimeout = 2 * time.Minute

// Prover runs rule proofs with the go command.
type Prover struct {
	// Root is the repository root; go commands run there.
	Root string
	// Go is the go command. Empty means "go".
	Go string
	// Jobs bounds how many broken-variant processes run at once. Below 1 means
	// GOMAXPROCS.
	Jobs int
	// Timeout bounds each test process. Zero means DefaultTimeout.
	Timeout time.Duration
}

// Prove runs r's proof (G1) and returns its violations and, when it held, a
// one-line account of what was run.
func (p Prover) Prove(ctx context.Context, r Rule) ([]Violation, string) {
	switch r.Proof.Kind {
	case "race", "test":
		return p.proveTest(ctx, r)
	default:
		return []Violation{{Rule: r.ID, Gate: "G1", Msg: fmt.Sprintf("proof kind %q has no prover", r.Proof.Kind)}}, ""
	}
}

// proveTest proves a race or test rule. The fixed variant must pass N times;
// the broken variant must fail in each of N separate processes, each failure
// printing every signature substring.
//
// Separate processes, because the race detector reports a given race once per
// process: twenty iterations of a racy test in one process produced two
// reports. And the signature, because a broken variant that does not compile
// exits 1 exactly like one whose test fails.
func (p Prover) proveTest(ctx context.Context, r Rule) ([]Violation, string) {
	bad := func(format string, args ...any) []Violation {
		return []Violation{{Rule: r.ID, Gate: "G1", Msg: fmt.Sprintf(format, args...)}}
	}
	pr := r.Proof
	pkg := "./" + r.Dir
	run := "^" + regexp.QuoteMeta(pr.Test) + "$"
	timeout := p.timeout()

	// The fixed variant: N passes in one process. -v so the passes can be
	// counted — a -run pattern that matches nothing also exits 0.
	out, err := p.goCmd(ctx, timeout*time.Duration(pr.Runs), "test", "-race", "-shuffle=on", "-v",
		fmt.Sprintf("-count=%d", pr.Runs), "-timeout", (timeout * time.Duration(pr.Runs)).String(), "-run", run, pkg)
	if err != nil {
		return bad("fixed variant does not pass %s:\n%s", pr.Test, tail(out, 20)), ""
	}
	passRe := regexp.MustCompile(`(?m)^--- PASS: ` + regexp.QuoteMeta(pr.Test) + ` \(`)
	if got := len(passRe.FindAll(out, -1)); got != pr.Runs {
		return bad("fixed variant: %s passed %d times, want %d — does the test exist?", pr.Test, got, pr.Runs), ""
	}

	// The broken variant: build once, run N separate processes.
	tmp, err := os.MkdirTemp("", "gate-"+r.Slug+"-")
	if err != nil {
		return bad("cannot create a temp dir: %v", err), ""
	}
	defer os.RemoveAll(tmp)
	bin := filepath.Join(tmp, "broken.test")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if out, err := p.goCmd(ctx, timeout, "test", "-race", "-tags", "broken", "-c", "-o", bin, pkg); err != nil {
		return bad("broken variant does not build:\n%s", tail(out, 20)), ""
	}

	results := p.runBroken(ctx, bin, r)
	var withSig, passed, unsigned, notRun int
	var first *runResult // the first run that does not count towards N/N
	for i := range results {
		res := &results[i]
		switch {
		case res.startErr != nil:
			notRun++
		case !res.failed:
			passed++
		case res.missing != "":
			unsigned++
		default:
			withSig++
			continue
		}
		if first == nil {
			first = res
		}
	}
	if first != nil {
		msg := fmt.Sprintf("broken variant failed with the signature in %d/%d runs (%d passed, %d failed without it, %d did not start)",
			withSig, pr.Runs, passed, unsigned, notRun)
		switch {
		case first.startErr != nil:
			msg += fmt.Sprintf("; first run did not start: %v", first.startErr)
		case first.failed:
			msg += fmt.Sprintf("; first run without the signature lacks %q:\n%s", first.missing, excerpt(first.out))
		default:
			msg += fmt.Sprintf("; first passing run:\n%s", tail(first.out, 20))
		}
		return bad("%s", msg), ""
	}
	return nil, fmt.Sprintf("%s: fixed passed %d×; broken failed %d/%d separate runs with the signature",
		pr.Kind, pr.Runs, withSig, pr.Runs)
}

type runResult struct {
	failed   bool   // the process ran and exited non-zero
	missing  string // when failed: the first signature substring absent from the output, or ""
	startErr error  // the process never ran, which proves nothing about the rule
	out      []byte
}

// runBroken runs the broken test binary pr.Runs times, at most p.jobs() at once.
func (p Prover) runBroken(ctx context.Context, bin string, r Rule) []runResult {
	results := make([]runResult, r.Proof.Runs)
	sem := make(chan struct{}, p.jobs())
	var wg sync.WaitGroup
	for i := range results {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			results[i] = p.runOnce(ctx, bin, r)
		})
	}
	wg.Wait()
	return results
}

func (p Prover) runOnce(ctx context.Context, bin string, r Rule) runResult {
	timeout := p.timeout()
	ctx, cancel := context.WithTimeout(ctx, timeout+10*time.Second)
	defer cancel()
	//nolint:gosec // G204: bin is the test binary this prover just built from the rule's package
	cmd := exec.CommandContext(ctx, bin,
		"-test.run", "^"+regexp.QuoteMeta(r.Proof.Test)+"$",
		"-test.count", "1", "-test.v", "-test.timeout", timeout.String())
	cmd.Dir = filepath.Join(p.Root, filepath.FromSlash(r.Dir)) // where go test runs a package's tests
	out, err := cmd.CombinedOutput()

	res := runResult{out: out}
	_, exited := errors.AsType[*exec.ExitError](err)
	switch {
	case err == nil:
	case exited:
		res.failed = true
	default:
		res.startErr = err
	}
	if res.failed {
		for _, s := range r.Proof.Signature {
			if !bytes.Contains(out, []byte(s)) {
				res.missing = s
				break
			}
		}
	}
	return res
}

func (p Prover) goCmd(ctx context.Context, timeout time.Duration, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout+time.Minute) // + build time
	defer cancel()
	goBin := p.Go
	if goBin == "" {
		goBin = "go"
	}
	//nolint:gosec // G204: the go command with arguments built from validated front matter
	cmd := exec.CommandContext(ctx, goBin, args...)
	cmd.Dir = p.Root
	return cmd.CombinedOutput()
}

func (p Prover) jobs() int {
	if p.Jobs < 1 {
		return runtime.GOMAXPROCS(0)
	}
	return p.Jobs
}

func (p Prover) timeout() time.Duration {
	if p.Timeout <= 0 {
		return DefaultTimeout
	}
	return p.Timeout
}

// headlineRe matches the lines that say why a test process failed: panics,
// fatal errors, FAIL lines, race warnings, and t.Log/t.Fatal output (indented
// "x_test.go:12: message" — a stack frame has no ": " after its line number).
var headlineRe = regexp.MustCompile(`^(panic: |fatal error: |--- FAIL|WARNING: DATA RACE)|^\s+[\w.-]+_test\.go:\d+: `)

// excerpt shows why a run failed: up to five headline lines, then the last
// lines. A timeout prints its panic first and its goroutine stacks after, so a
// tail alone would show stacks and hide the reason.
func excerpt(out []byte) string {
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	var head []string
	for _, l := range lines {
		if headlineRe.MatchString(l) {
			head = append(head, l)
			if len(head) == 5 {
				break
			}
		}
	}
	if len(head) == 0 {
		return tail(out, 20)
	}
	return "    " + strings.Join(head, "\n    ") + "\n    ⋮\n" + tail(out, 8)
}

// tail returns the last n lines of out, for violation messages.
func tail(out []byte, n int) string {
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(lines) > n {
		lines = append([]string{"…"}, lines[len(lines)-n:]...)
	}
	return "    " + strings.Join(lines, "\n    ")
}
