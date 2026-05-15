package engines

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JoakimCarlsson/scour/query"
)

// resetSuspensionForTest wipes the package-level table between tests so
// they don't bleed state into each other.
func resetSuspensionForTest(t *testing.T) {
	t.Helper()
	suspensionMu.Lock()
	suspensionTable = map[string]*suspensionEntry{}
	suspensionMu.Unlock()
	nowFn = time.Now
}

func TestSuspendAndIsSuspended(t *testing.T) {
	resetSuspensionForTest(t)
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	nowFn = func() time.Time { return now }

	if IsSuspended("foo") {
		t.Fatal("fresh engine should not be suspended")
	}
	d := Suspend("foo")
	if d != 60*time.Second {
		t.Fatalf("first suspend = %v, want 60s", d)
	}
	if !IsSuspended("foo") {
		t.Fatal("expected suspended after first block")
	}
	// Advance past cooldown.
	now = now.Add(2 * time.Minute)
	if IsSuspended("foo") {
		t.Fatal("expected unsuspended after cooldown")
	}
}

func TestSuspendBackoff(t *testing.T) {
	resetSuspensionForTest(t)
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	nowFn = func() time.Time { return now }

	for i, want := range []time.Duration{
		60 * time.Second,
		5 * time.Minute,
		30 * time.Minute,
		30 * time.Minute,
	} {
		got := Suspend("foo")
		if got != want {
			t.Fatalf("strike %d: got %v, want %v", i+1, got, want)
		}
	}
}

func TestClearSuspensionResetsCounter(t *testing.T) {
	resetSuspensionForTest(t)
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	nowFn = func() time.Time { return now }

	Suspend("foo") // 60s
	Suspend("foo") // 5m
	ClearSuspension("foo")
	got := Suspend("foo")
	if got != 60*time.Second {
		t.Fatalf("post-clear suspend = %v, want 60s (counter reset)", got)
	}
}

func TestShouldSuspendDetectsBlocks(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"plain error", errors.New("boom"), false},
		{
			"engine blocked",
			&EngineBlockedError{Engine: "x", Reason: BlockReasonCaptcha},
			true,
		},
		{"http 403", &httpError{Status: 403}, true},
		{"http 429", &httpError{Status: 429}, true},
		{"http 503", &httpError{Status: 503}, true},
		{"http 500", &httpError{Status: 500}, false},
		{"http 404", &httpError{Status: 404}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := shouldSuspend(c.err); got != c.want {
				t.Fatalf("shouldSuspend(%v) = %v, want %v", c.err, got, c.want)
			}
		})
	}
}

func TestFanOutSkipsSuspendedEngines(t *testing.T) {
	resetSuspensionForTest(t)
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	nowFn = func() time.Time { return now }

	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	calls := int32(0)
	e := fakeEngine{
		stubEngine: stubEngine{name: "alpha"},
		results:    []Result{{Title: "t", URL: "u"}},
		calls:      &calls,
	}

	// First call: succeeds, clears any prior suspension.
	_, errs := FanOut(context.Background(), query.Query{}, []Engine{e}, time.Second)
	if len(errs) != 0 {
		t.Fatalf("first call errs: %v", errs)
	}

	// Manually suspend then verify the engine is skipped.
	Suspend("alpha")
	before := atomic.LoadInt32(&calls)
	_, errs = FanOut(context.Background(), query.Query{}, []Engine{e}, time.Second)
	if atomic.LoadInt32(&calls) != before {
		t.Fatalf(
			"suspended engine was called: before=%d after=%d",
			before,
			atomic.LoadInt32(&calls),
		)
	}
	if len(errs) != 1 || !errs[0].Suspended {
		t.Fatalf("expected one Suspended FanOutError, got %v", errs)
	}
}

func TestFanOutSuspendsOnBlockError(t *testing.T) {
	resetSuspensionForTest(t)
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	nowFn = func() time.Time { return now }

	calls := int32(0)
	blocker := fakeEngine{
		stubEngine: stubEngine{name: "beta"},
		err: &EngineBlockedError{
			Engine: "beta",
			Reason: BlockReasonCaptcha,
		},
		calls: &calls,
	}
	_, errs := FanOut(context.Background(), query.Query{}, []Engine{blocker}, time.Second)
	if len(errs) != 1 {
		t.Fatalf("want 1 error, got %v", errs)
	}
	if !IsSuspended("beta") {
		t.Fatal("expected beta suspended after block error")
	}
}
