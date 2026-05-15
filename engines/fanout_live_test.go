//go:build live

package engines

import (
	"context"
	"testing"
	"time"

	"github.com/JoakimCarlsson/scour/query"
)

func TestFanOutLive(t *testing.T) {
	q := query.Query{Terms: "golang tutorials", Category: query.CategoryGeneral, Language: "en"}
	engs := Select(q, Preferences{})
	res, errs := FanOut(context.Background(), q, engs, 10*time.Second)
	for _, e := range errs {
		t.Logf("engine error: %v", e.Error())
	}
	engineHits := map[string]int{}
	for _, r := range res {
		engineHits[r.Engine]++
	}
	if len(engineHits) < 2 {
		t.Fatalf("expected results from at least 2 engines, got: %v", engineHits)
	}
	t.Logf("results per engine: %v", engineHits)
}
