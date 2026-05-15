package cache

import (
	"sync"
	"testing"
	"time"

	"github.com/JoakimCarlsson/scour/query"
	"github.com/JoakimCarlsson/scour/rank"
)

func TestKeyForDeterministic(t *testing.T) {
	a := query.Query{
		Terms:    "golang",
		Category: query.CategoryGeneral,
		Language: "en",
		Engines:  []string{"google", "bing"},
	}
	b := query.Query{
		Terms:    "golang",
		Category: query.CategoryGeneral,
		Language: "en",
		Engines:  []string{"bing", "google"},
	}
	if KeyFor(a) != KeyFor(b) {
		t.Fatal("KeyFor not order-independent for Engines")
	}
}

func TestKeyForDiffersOnContent(t *testing.T) {
	a := query.Query{Terms: "golang"}
	b := query.Query{Terms: "rust"}
	if KeyFor(a) == KeyFor(b) {
		t.Fatal("KeyFor collided on different terms")
	}
}

func TestKeyForIncludesFilters(t *testing.T) {
	a := query.Query{Terms: "x"}
	b := query.Query{Terms: "x", Filters: query.Filters{Sites: []string{"reddit.com"}}}
	if KeyFor(a) == KeyFor(b) {
		t.Fatal("KeyFor collided across Filters")
	}
}

func TestKeyForIncludesTimeRange(t *testing.T) {
	a := query.Query{Terms: "news", TimeRange: query.TimeRangeDay}
	b := query.Query{Terms: "news", TimeRange: query.TimeRangeWeek}
	if KeyFor(a) == KeyFor(b) {
		t.Fatal("KeyFor collided across TimeRange")
	}
}

func TestMemoryCacheGetSet(t *testing.T) {
	c := NewMemory(0)
	defer c.Close()
	val := []rank.Ranked{{Score: 1.5}}
	c.Set("k", val, time.Minute)
	got, ok := c.Get("k")
	if !ok || len(got) != 1 || got[0].Score != 1.5 {
		t.Fatalf("Get: ok=%v got=%v", ok, got)
	}
}

func TestMemoryCacheExpiry(t *testing.T) {
	c := NewMemory(0)
	defer c.Close()
	c.Set("k", []rank.Ranked{{Score: 1.0}}, 10*time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	if _, ok := c.Get("k"); ok {
		t.Fatal("expected expiry, still present")
	}
}

func TestMemoryCacheConcurrent(t *testing.T) {
	c := NewMemory(0)
	defer c.Close()
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(2)
		go func() { defer wg.Done(); c.Set("k", nil, time.Minute) }()
		go func() { defer wg.Done(); _, _ = c.Get("k") }()
	}
	wg.Wait()
}

func TestMemoryCacheClose(t *testing.T) {
	c := NewMemory(time.Millisecond)
	c.Close()
	c.Close()
}

func TestMemoryCacheSweep(t *testing.T) {
	c := NewMemory(5 * time.Millisecond)
	defer c.Close()
	c.Set("k", []rank.Ranked{{Score: 1}}, 5*time.Millisecond)
	time.Sleep(30 * time.Millisecond)
	c.mu.RLock()
	_, present := c.data["k"]
	c.mu.RUnlock()
	if present {
		t.Fatal("sweep did not remove expired entry")
	}
}
