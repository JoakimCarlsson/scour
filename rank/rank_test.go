package rank

import (
	"testing"

	"github.com/JoakimCarlsson/scour/merge"
)

func TestRankMultiSourceOutranksSingle(t *testing.T) {
	in := []merge.Merged{
		{URL: "https://a/", Sources: []merge.Source{{Engine: "google", Position: 1}}},
		{URL: "https://b/", Sources: []merge.Source{
			{Engine: "google", Position: 1}, {Engine: "bing", Position: 1},
		}},
	}
	out := Rank(in, nil)
	if out[0].URL != "https://b/" {
		t.Fatalf("expected b/ first, got %q (score %v)", out[0].URL, out[0].Score)
	}
}

func TestRankPositionMatters(t *testing.T) {
	in := []merge.Merged{
		{URL: "https://a/", Sources: []merge.Source{{Engine: "g", Position: 5}}},
		{URL: "https://b/", Sources: []merge.Source{{Engine: "g", Position: 1}}},
	}
	out := Rank(in, nil)
	if out[0].URL != "https://b/" {
		t.Fatalf("expected b/ first (better position), got %q", out[0].URL)
	}
}

func TestRankTiebreakByURL(t *testing.T) {
	in := []merge.Merged{
		{URL: "https://b/", Sources: []merge.Source{{Engine: "g", Position: 1}}},
		{URL: "https://a/", Sources: []merge.Source{{Engine: "g", Position: 1}}},
	}
	out := Rank(in, nil)
	if out[0].URL != "https://a/" || out[1].URL != "https://b/" {
		t.Fatalf("tiebreak: got %q, %q", out[0].URL, out[1].URL)
	}
}

func TestRankMissingWeightDefaultsToOne(t *testing.T) {
	in := []merge.Merged{
		{URL: "https://a/", Sources: []merge.Source{{Engine: "unknown", Position: 1}}},
	}
	out := Rank(in, map[string]float64{"google": 2.0})
	if out[0].Score != 1.0 {
		t.Fatalf("expected score 1.0, got %v", out[0].Score)
	}
}

func TestRankWeightAffectsOrder(t *testing.T) {
	in := []merge.Merged{
		{URL: "https://a/", Sources: []merge.Source{{Engine: "low", Position: 1}}},
		{URL: "https://b/", Sources: []merge.Source{{Engine: "high", Position: 1}}},
	}
	w := map[string]float64{"low": 0.1, "high": 2.0}
	out := Rank(in, w)
	if out[0].URL != "https://b/" {
		t.Fatalf("expected b/ first, got %q", out[0].URL)
	}
}
