package gate

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

// requireRace skips a test that runs proofs when this host cannot build with
// -race. CI and `task test:docker` run it; a skip here is never a pass there.
func requireRace(t *testing.T) {
	t.Helper()
	if err := RaceAvailable(t.Context(), ""); err != nil {
		t.Skipf("needs -race; run `task test:docker`: %v", err)
	}
}

// TestCheckFixture runs the whole gate over a fixture repository holding one
// valid rule and one rule per way a proof can be wrong. The gate must admit
// the first and refuse each of the others for the right reason — otherwise the
// gate is just one more check nobody has seen fail.
func TestCheckFixture(t *testing.T) {
	requireRace(t)
	t.Setenv("FIXTURE_MARKER_DIR", t.TempDir()) // R94: one process passes

	// A short per-process timeout, so R97's hanging broken variant costs
	// seconds, not the default minutes.
	rep, err := Check(context.Background(), Options{
		Root:   "testdata/fixture",
		Prover: Prover{Jobs: 4, Timeout: 3 * time.Second},
	})
	if err != nil {
		t.Fatal(err)
	}

	admitted := map[string]bool{}
	for _, a := range rep.Admitted {
		admitted[a.ID] = true
	}
	got := map[string][]Violation{}
	for _, v := range rep.Violations {
		got[v.Rule] = append(got[v.Rule], v)
	}

	tests := []struct {
		id   string
		want []string // "G1: substring" per expected violation; nil = admitted
	}{
		{id: "R90"},
		{id: "R91", want: []string{"G1: 0/10 runs (10 passed"}},
		{id: "R92", want: []string{"G1: broken variant does not build"}},
		{id: "R93", want: []string{`G1: 0/10 runs (0 passed, 10 failed without it`}},
		{id: "R94", want: []string{"G1: 9/10 runs (1 passed"}},
		{id: "R95", want: []string{"G1: TestMissing passed 0 times, want 10"}},
		{id: "R96", want: []string{"G6: alternative R99 does not exist", "G2: hand-typed Go"}},
		{id: "R97", want: []string{"G1: 0/10 runs (0 passed, 0 failed without it, 10 timed out"}},
		// A hang is never a signed failure, even when the signature is text
		// the timeout's stack dump happens to print.
		{id: "R89", want: []string{"G1: 0/10 runs (0 passed, 0 failed without it, 10 timed out"}},
		// A signature the fixed run prints cannot tell broken from fixed.
		{id: "R88", want: []string{`G1: signature "checking the answer" also appears in the fixed variant's output`}},
		// A fixed variant that hangs is stopped by one process timeout, not N.
		{id: "R87", want: []string{"G1: fixed variant does not pass TestAnswer"}},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			vs := got[tt.id]
			if tt.want == nil {
				if !admitted[tt.id] || len(vs) != 0 {
					t.Fatalf("want %s admitted, got admitted=%v violations=%v", tt.id, admitted[tt.id], vs)
				}
				return
			}
			if admitted[tt.id] {
				t.Fatalf("%s was admitted; want it refused", tt.id)
			}
			if len(vs) != len(tt.want) {
				t.Fatalf("want %d violations, got %d: %v", len(tt.want), len(vs), vs)
			}
			for i, w := range tt.want {
				gate, sub, _ := strings.Cut(w, ": ")
				if vs[i].Gate != gate || !strings.Contains(vs[i].Msg, sub) {
					t.Errorf("violation %d = %v, want %s mentioning %q", i, vs[i], gate, sub)
				}
			}
		})
	}
	if len(rep.Admitted)+len(got) != len(tests) {
		t.Errorf("the fixture holds %d rules but the report covers %d", len(tests), len(rep.Admitted)+len(got))
	}
	// A hang is refused, and the message says why — for either variant.
	for _, id := range []string{"R97", "R89", "R87"} {
		if vs := got[id]; len(vs) == 1 && !strings.Contains(vs[0].Msg, "test timed out") {
			t.Errorf("%s's violation should show the timeout, got:\n%s", id, vs[0].Msg)
		}
	}
}

// TestCheckInterrupted: when the run is cancelled, the processes it killed say
// nothing about the rule. Check must report the interruption, not violations.
func TestCheckInterrupted(t *testing.T) {
	t.Run("before it starts", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := Check(ctx, Options{Root: "testdata/fixture"})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want context.Canceled", err)
		}
	})
	t.Run("while proving", func(t *testing.T) {
		requireRace(t)
		// R97's broken processes sleep for an hour; the deadline lands while
		// they run (or, on a cold build cache, while the fixed run builds).
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		rep, err := Check(ctx, Options{Root: "testdata/fixture", IDs: []string{"R97"}})
		if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "interrupted while proving R97") {
			t.Fatalf("err = %v (report %+v), want an interruption while proving R97", err, rep)
		}
	})
}

func TestCheckSelect(t *testing.T) {
	requireRace(t)
	rep, err := Check(context.Background(), Options{Root: "testdata/fixture", IDs: []string{"r90"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Admitted) != 1 || rep.Admitted[0].ID != "R90" || len(rep.Violations) != 0 {
		t.Fatalf("want only R90 checked and admitted, got %+v", rep)
	}
}

func TestCheckRefusesToCheckNothing(t *testing.T) {
	tests := []struct {
		name string
		opt  Options
		want string
	}{
		{name: "no rules", opt: Options{Root: t.TempDir()}, want: "no rule pages"},
		{name: "unknown id", opt: Options{Root: "testdata/fixture", IDs: []string{"R42"}}, want: "no rule with id R42"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Check(context.Background(), tt.opt)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want it to mention %q", err, tt.want)
			}
		})
	}
}

// TestMain lets the test binary stand in for a go command that never finishes,
// so a cancel can be made to land mid-build without depending on timing.
func TestMain(m *testing.M) {
	if os.Getenv("GATE_TEST_FAKE_GO") == "hang" {
		time.Sleep(time.Minute)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// TestRaceAvailableInterrupted: a build killed by cancellation says nothing
// about the host. Reporting "this host cannot build with -race" would send the
// reader to install a C compiler they may already have.
func TestRaceAvailableInterrupted(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GATE_TEST_FAKE_GO", "hang")
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if err := RaceAvailable(ctx, self); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
}
