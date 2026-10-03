package main

import (
	"bytes"
	"context"
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
	code := run(context.Background(), []string{"check", "-root", "../../gate/testdata/fixture", "R90", "R91"}, &stdout, &stderr)
	if code != exitViolation {
		t.Fatalf("exit code = %d, want %d\nstdout:\n%s\nstderr:\n%s", code, exitViolation, &stdout, &stderr)
	}
	out := stdout.String()
	for _, want := range []string{"R90 ok  race: fixed passed 10×", "R91 G1: broken variant failed", "gate: 1 violation(s); 1 rule(s) admitted"} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout lacks %q:\n%s", want, out)
		}
	}
}
