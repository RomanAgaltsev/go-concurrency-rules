package r26

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
)

// --8<-- [start:test]

// TestCheckCanFail is the proof that CheckRunsOnce can fail: run against
// FlakyOnce, whose bug is known, it must report a failure. Run against
// sync.Once it must pass, or it would only prove that it fails on everything.
func TestCheckCanFail(t *testing.T) {
	if r := runCheck(t, func() Initer { return new(FlakyOnce) }); !r.Failed() {
		t.Fatal("check stayed green over a known bug: FlakyOnce runs f twice when two callers race")
	}
	CheckRunsOnce(t, func() Initer { return new(sync.Once) })
}

// --8<-- [end:test]

// recorder is a testing.TB that records failures instead of reporting them,
// so a check can be run against a known-buggy implementation.
type recorder struct {
	testing.TB // the real t, for everything not overridden below

	mu     sync.Mutex
	failed bool
	msgs   []string
}

func (r *recorder) Errorf(format string, args ...any) { r.fail(fmt.Sprintf(format, args...)) }
func (r *recorder) Error(args ...any)                 { r.fail(fmt.Sprint(args...)) }
func (r *recorder) Fail()                             { r.fail("") }
func (r *recorder) FailNow()                          { r.fail(""); runtime.Goexit() }
func (r *recorder) Fatal(args ...any)                 { r.fail(fmt.Sprint(args...)); runtime.Goexit() }

func (r *recorder) Fatalf(format string, args ...any) {
	r.fail(fmt.Sprintf(format, args...))
	runtime.Goexit()
}

func (r *recorder) Failed() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.failed
}

func (r *recorder) fail(msg string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.failed = true
	if msg != "" {
		r.msgs = append(r.msgs, msg)
	}
}

// runCheck runs CheckRunsOnce against newIniter's Initer through a recorder,
// in its own goroutine, so that a Fatal inside the check ends only the check.
func runCheck(t *testing.T, newIniter func() Initer) *recorder {
	r := &recorder{TB: t}
	done := make(chan struct{})
	go func() {
		defer close(done)
		CheckRunsOnce(r, newIniter)
	}()
	<-done
	return r
}
