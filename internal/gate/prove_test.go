package gate

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestProveCannotRun: when the go command cannot even start, the gate has
// learnt nothing about the rule. That is the gate failing (exit 2), never a
// violation the rule's author is sent to fix (exit 1).
func TestProveCannotRun(t *testing.T) {
	rules, _, err := Load("testdata/fixture")
	if err != nil {
		t.Fatal(err)
	}
	p := Prover{Root: "testdata/fixture", Go: filepath.Join(t.TempDir(), "no-such-go")}
	for _, id := range []string{"R90", "R80", "R84"} { // race, vet, measure
		t.Run(id, func(t *testing.T) {
			r := rules[slices.IndexFunc(rules, func(r Rule) bool { return r.ID == id })]
			vs, _, err := p.Prove(context.Background(), r)
			if !errors.Is(err, errCannotRun) || len(vs) != 0 {
				t.Fatalf("Prove = %v, %v; want no violations and errCannotRun", vs, err)
			}
		})
	}
}

func TestExcerpt(t *testing.T) {
	// The shape of a -test.timeout panic: the reason first, then pages of stacks.
	stacks := strings.Repeat("goroutine 7 [sleep]:\n\t/src/rules/r97/rule_test.go:6 +0x2d\n", 30)
	timeout := "=== RUN   TestAnswer\npanic: test timed out after 3s\n\trunning tests:\n\t\tTestAnswer (3s)\n\n" + stacks

	tests := []struct {
		name    string
		out     string
		want    []string
		notWant []string
	}{
		{
			name: "timeout keeps its reason",
			out:  timeout,
			want: []string{"panic: test timed out after 3s", "⋮", "rule_test.go:6 +0x2d"},
		},
		{
			name: "test log line, not stack frames",
			out:  "=== RUN   TestAnswer\n    rule_test.go:7: Answer() = 41, want 42\n--- FAIL: TestAnswer (0.00s)\nFAIL\n",
			want: []string{"rule_test.go:7: Answer() = 41, want 42", "--- FAIL: TestAnswer"},
		},
		{
			name:    "no headline falls back to the tail",
			out:     "line 1\nline 2\n",
			want:    []string{"line 1", "line 2"},
			notWant: []string{"⋮"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := excerpt([]byte(tt.out))
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("excerpt lacks %q:\n%s", w, got)
				}
			}
			for _, w := range tt.notWant {
				if strings.Contains(got, w) {
					t.Errorf("excerpt has %q:\n%s", w, got)
				}
			}
		})
	}
}
