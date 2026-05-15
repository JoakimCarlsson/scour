package engines

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JoakimCarlsson/scour/query"
)

type fakeEngine struct {
	stubEngine
	delay   time.Duration
	results []Result
	err     error
	calls   *int32
}

func (f fakeEngine) Search(ctx context.Context, _ query.Query) (Response, error) {
	if f.calls != nil {
		atomic.AddInt32(f.calls, 1)
	}
	select {
	case <-ctx.Done():
		return Response{}, ctx.Err()
	case <-time.After(f.delay):
	}
	return Response{Results: f.results}, f.err
}

func TestFanOutAggregatesResults(t *testing.T) {
	a := fakeEngine{stubEngine: stubEngine{name: "a"}, results: []Result{{Title: "a1", URL: "u1"}}}
	b := fakeEngine{
		stubEngine: stubEngine{name: "b"},
		results:    []Result{{Title: "b1", URL: "u2"}, {Title: "b2", URL: "u3"}},
	}
	res, errs := FanOut(context.Background(), query.Query{}, []Engine{a, b}, time.Second)
	if len(res) != 3 {
		t.Fatalf("want 3 results, got %d", len(res))
	}
	if len(errs) != 0 {
		t.Fatalf("want 0 errors, got %v", errs)
	}
}

func TestFanOutCollectsErrors(t *testing.T) {
	a := fakeEngine{stubEngine: stubEngine{name: "a"}, err: errors.New("boom")}
	b := fakeEngine{stubEngine: stubEngine{name: "b"}, results: []Result{{Title: "b1"}}}
	res, errs := FanOut(context.Background(), query.Query{}, []Engine{a, b}, time.Second)
	if len(res) != 1 {
		t.Fatalf("want 1 result, got %d", len(res))
	}
	if len(errs) != 1 || errs[0].Engine != "a" {
		t.Fatalf("want 1 error from a, got %v", errs)
	}
}

func TestFanOutRespectsTimeout(t *testing.T) {
	slow := fakeEngine{stubEngine: stubEngine{name: "slow"}, delay: 200 * time.Millisecond}
	start := time.Now()
	_, errs := FanOut(context.Background(), query.Query{}, []Engine{slow}, 30*time.Millisecond)
	if time.Since(start) > 150*time.Millisecond {
		t.Fatalf("FanOut did not respect timeout, took %v", time.Since(start))
	}
	if len(errs) != 1 {
		t.Fatalf("want 1 timeout error, got %v", errs)
	}
}

func TestFanOutEmpty(t *testing.T) {
	res, errs := FanOut(context.Background(), query.Query{}, nil, time.Second)
	if res != nil || errs != nil {
		t.Fatalf("want nil, nil; got %v, %v", res, errs)
	}
}
