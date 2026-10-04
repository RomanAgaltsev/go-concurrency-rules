package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RomanAgaltsev/go-concurrency-rules/internal/gate"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStderr string
	}{
		{name: "no command", args: nil, wantCode: exitCannot, wantStderr: "usage: gate check"},
		{name: "unknown command", args: []string{"chek"}, wantCode: exitCannot, wantStderr: `unknown command "chek"`},
		{name: "bad flag", args: []string{"check", "-nope"}, wantCode: exitCannot, wantStderr: "flag provided but not defined"},
		// Checking nothing is not passing.
		{name: "no rules", args: []string{"check", "-root", t.TempDir()}, wantCode: exitCannot, wantStderr: "no rule pages"},
		{name: "help", args: []string{"help"}, wantCode: exitOK},
		// Asking for help is not an error, for any subcommand.
		{name: "check -h", args: []string{"check", "-h"}, wantCode: exitOK, wantStderr: "Usage of check"},
		{name: "gen no rules", args: []string{"gen", "-root", t.TempDir()}, wantCode: exitCannot, wantStderr: "no rule pages"},
		{name: "run without id", args: []string{"run"}, wantCode: exitCannot, wantStderr: "usage: gate run"},
		{name: "run unknown id", args: []string{"run", "-root", "../../gate/testdata/fixture", "R42"}, wantCode: exitCannot, wantStderr: "no rule with id R42"},
		// A rule that cannot carry a proof is explained, not failed.
		{name: "run none", args: []string{"run", "-root", "../../gate/testdata/fixture", "X90"}, wantCode: exitOK},
		// The fixture holds rules with invalid front matter: generating
		// indexes from it would publish them, so gen refuses.
		{name: "gen invalid rules", args: []string{"gen", "-root", "../../gate/testdata/fixture"}, wantCode: exitCannot, wantStderr: "fix the front matter first"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(context.Background(), tt.args, &stdout, &stderr)
			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d (stderr: %s)", code, tt.wantCode, stderr.String())
			}
			if !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("stderr = %q, want it to contain %q", stderr.String(), tt.wantStderr)
			}
		})
	}
}

// TestRunFixture checks the exit status against the gate's fixture repository:
// it holds refused rules, so check must exit 1 and say which.
func TestRunFixture(t *testing.T) {
	if err := gate.RaceAvailable(t.Context(), ""); err != nil {
		t.Skipf("needs -race; run `task test:docker`: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"check", "-root", "../../gate/testdata/fixture", "R90", "R91", "X90"}, &stdout, &stderr)
	if code != exitViolation {
		t.Fatalf("exit code = %d, want %d\nstdout:\n%s\nstderr:\n%s", code, exitViolation, &stdout, &stderr)
	}
	out := stdout.String()
	for _, want := range []string{"R90 ok  race: fixed passed 10×", "R91 G1: broken variant failed", "X90 ok  none: no proof", "gate: 1 violation(s); 2 admitted: 1 proven, 1 retired without a proof"} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout lacks %q:\n%s", want, out)
		}
	}
}

// TestAdmitted: an entry admitted without a proof is counted apart from the
// proven ones — "9 admitted" must not read as nine proofs when three are
// retired entries with nothing to run.
func TestAdmitted(t *testing.T) {
	tests := []struct {
		name string
		as   []gate.Admitted
		want string
	}{
		{name: "none", as: nil, want: "0 admitted: 0 proven, 0 retired without a proof"},
		{
			name: "mixed",
			as:   []gate.Admitted{{ID: "R07", Proven: true}, {ID: "X02"}, {ID: "R12", Proven: true}},
			want: "3 admitted: 2 proven, 1 retired without a proof",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := admitted(tt.as); got != tt.want {
				t.Errorf("admitted() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestGen writes the index pages for a one-rule repository, then checks that
// -check passes on them and fails, naming the page, once one is hand-edited.
func TestGen(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("docs/rules/r07-x.md", "---\nid: R07\ntitle: T\nstatement: S.\ngroup: shared-memory\ntags: [read]\nsince: go1.0\nstatus: active\n"+
		"proof:\n  kind: race\n  test: TestX\n  signature: [\"WARNING: DATA RACE\"]\n  runs: 10\nsources:\n  - https://go.dev/ref/mem\n---\n# T\n")
	write("rules/r07-x/broken.go", "//go:build broken\n\npackage r07\n")
	write("rules/r07-x/fixed.go", "//go:build !broken\n\npackage r07\n")

	gen := func(args ...string) (int, string) {
		var stdout, stderr bytes.Buffer
		code := run(context.Background(), append([]string{"gen", "-root", root}, args...), &stdout, &stderr)
		return code, stdout.String() + stderr.String()
	}
	if code, out := gen("-check"); code != exitViolation || !strings.Contains(out, "stale: docs/groups/shared-memory.md") {
		t.Fatalf("before generating, -check = %d, want %d naming the missing pages:\n%s", code, exitViolation, out)
	}
	if code, out := gen(); code != exitOK || !strings.Contains(out, "wrote 12 page(s)") {
		t.Fatalf("gen = %d:\n%s", code, out)
	}
	if code, out := gen("-check"); code != exitOK {
		t.Fatalf("after generating, -check = %d, want %d:\n%s", code, exitOK, out)
	}
	write("docs/activity/read.md", "hand edit\n")
	if code, out := gen("-check"); code != exitViolation || !strings.Contains(out, "stale: docs/activity/read.md") {
		t.Fatalf("after a hand edit, -check = %d, want %d naming the page:\n%s", code, exitViolation, out)
	}
}
