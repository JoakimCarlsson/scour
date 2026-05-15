package pipeline

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JoakimCarlsson/scour/cache"
	"github.com/JoakimCarlsson/scour/engines"
	"github.com/JoakimCarlsson/scour/plugins"
	"github.com/JoakimCarlsson/scour/query"
)

type countingEngine struct {
	name  string
	calls *int32
}

func (e countingEngine) Name() string { return e.name }

func (e countingEngine) Categories() []query.Category { return []query.Category{query.CategoryGeneral} }
func (e countingEngine) Languages() engines.LanguageTraits {
	return engines.LanguageTraits{All: true}
}
func (e countingEngine) Weight() float64 { return 1.0 }
func (e countingEngine) Search(_ context.Context, _ query.Query) ([]engines.Result, error) {
	atomic.AddInt32(e.calls, 1)
	return []engines.Result{
		{Title: "Go", URL: "https://go.dev/", Engine: e.name, Position: 1},
	}, nil
}

func TestPipelineCacheHitSkipsEngines(t *testing.T) {
	var calls int32
	mem := cache.NewMemory(0)
	defer mem.Close()
	p := New(mem)
	p.Select = func(_ query.Query, _ engines.Preferences) []engines.Engine {
		return []engines.Engine{countingEngine{name: "test", calls: &calls}}
	}
	p.PluginChain = func(_ Preferences) []plugins.Plugin { return nil }

	prefs := Preferences{Timeout: time.Second, CacheTTL: time.Minute}
	if _, err := p.Search(context.Background(), "golang", prefs); err != nil {
		t.Fatalf("first search: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("first call: expected 1 engine call, got %d", got)
	}
	if _, err := p.Search(context.Background(), "golang", prefs); err != nil {
		t.Fatalf("second search: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("second call: cache miss — engine called %d times", got)
	}
}

func TestPipelineEmptyQueryError(t *testing.T) {
	p := New(nil)
	p.Select = func(_ query.Query, _ engines.Preferences) []engines.Engine { return nil }
	p.PluginChain = func(_ Preferences) []plugins.Plugin { return nil }
	if _, err := p.Search(context.Background(), "", Preferences{}); err == nil {
		t.Fatal("expected error on empty query")
	}
}
