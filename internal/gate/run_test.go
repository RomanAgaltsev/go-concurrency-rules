package gate

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

// TestExplain runs `gate run` against fixture rules. Every case passes
// race=false, so it runs on any host: what a reader without a C compiler gets
// is exactly what is under test.
func TestExplain(t *testing.T) {
	tests := []struct {
		id      string
		wantOK  bool
		wantErr error
		want    []string // substrings of the narration
	}{
		// A retired entry without a proof says why, and that is all.
		{id: "X90", wantOK: true, want: []string{"X90 — Fixture X90", "No proof:", "selected per module"}},
		// vet runs without -race.
		{id: "R80", wantOK: true, want: []string{"$ go vet -json ./rules/r80-vet-valid", "fixed: go vet finds nothing", "broken: copylocks —"}},
		{id: "R82", wantOK: false, want: []string{"fixed: go vet reports copylocks"}},
		// A test proof runs without -race and says so.
		{id: "X91", wantOK: true, want: []string{"without -race", "$ go test -count=1 -v -run ^TestCapture$ -tags broken ./retired/x91-retired-twin", "broken: failed, printing \"closures saw [3 3 3]\""}},
		{id: "R91", wantOK: false, want: []string{"broken: passed — the broken variant should fail"}},
		{id: "R93", wantOK: false, want: []string{"broken: failed, but without \"kaboom\""}},
		// A race proof needs the race detector: no verdict without it.
		{id: "R90", wantErr: ErrNeedsRace, want: []string{"needs the race detector", "docker run", "gate run R90"}},
		// measure: correctness both ways, then the numbers on record.
		{id: "R84", wantOK: true, want: []string{"fixed: passed", "broken: passed (it is correct, only slower)", "vs base", "task measure -- rules/r84-measure-valid 1,4"}},
		{id: "R77", wantOK: false, want: []string{"broken: failed — a measure rule's broken variant must be correct"}},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			var out bytes.Buffer
			ok, err := Explain(context.Background(), Options{Root: "testdata/fixture"}, tt.id, false, &out)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v\n%s", err, tt.wantErr, &out)
			}
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			for _, w := range tt.want {
				if !strings.Contains(out.String(), w) {
					t.Errorf("narration lacks %q:\n%s", w, &out)
				}
			}
		})
	}
}

func TestExplainUnknownID(t *testing.T) {
	_, err := Explain(context.Background(), Options{Root: "testdata/fixture"}, "R42", false, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "no rule with id R42") {
		t.Fatalf("err = %v, want it to name the missing id", err)
	}
}

// TestExplainInterrupted: a process killed by cancellation says nothing about
// the rule. Explain must report the interruption, not a verdict.
func TestExplainInterrupted(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ok, err := Explain(ctx, Options{Root: "testdata/fixture"}, "R91", false, &bytes.Buffer{})
	if ok || !errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), "interrupted while running R91") {
		t.Fatalf("ok = %v, err = %v; want an interruption while running R91", ok, err)
	}
}
