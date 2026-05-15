package merge

import (
	"testing"

	"github.com/JoakimCarlsson/scour/engines"
)

func TestMergePreservesExtras(t *testing.T) {
	in := []engines.Result{
		{
			URL:    "https://example.com/cat.jpg",
			Title:  "Cat",
			Engine: "a", Position: 1,
			Extras: map[string]string{engines.ExtraThumbnailURL: "https://x/cat1.jpg"},
		},
		{
			URL:    "https://example.com/cat.jpg",
			Title:  "Cat photo",
			Engine: "b", Position: 1,
			Extras: map[string]string{
				engines.ExtraThumbnailURL:    "https://x/cat-larger-thumb.jpg",
				engines.ExtraThumbnailWidth:  "200",
				engines.ExtraThumbnailHeight: "150",
			},
		},
	}
	out := Merge(in)
	if len(out) != 1 {
		t.Fatalf("len=%d", len(out))
	}
	got := out[0].Extras
	if got[engines.ExtraThumbnailURL] != "https://x/cat-larger-thumb.jpg" {
		t.Errorf("thumb_url = %q", got[engines.ExtraThumbnailURL])
	}
	if got[engines.ExtraThumbnailWidth] != "200" {
		t.Errorf("width = %q", got[engines.ExtraThumbnailWidth])
	}
}
