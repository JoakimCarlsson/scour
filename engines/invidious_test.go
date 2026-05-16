package engines

import "testing"

const invidiousFixture = `[
  {
    "title": "Spring - Blender Open Movie",
    "videoId": "WhWc3b3KhnY",
    "author": "Blender Studio",
    "lengthSeconds": 465,
    "published": 1557991370,
    "description": "Made in Blender 2.8.",
    "videoThumbnails": [
      {"url": "https://invid/vi/WhWc3b3KhnY/maxres.jpg", "width": 1280, "height": 720},
      {"url": "https://invid/vi/WhWc3b3KhnY/sddefault.jpg", "width": 640, "height": 480}
    ]
  },
  {
    "title": "Sprite Fright",
    "videoId": "_cMxraX_5RE",
    "author": "Blender Studio",
    "lengthSeconds": 630,
    "published": 1652685770,
    "videoThumbnails": [
      {"url": "https://invid/vi/x/medium.jpg", "width": 320, "height": 180}
    ]
  }
]`

func TestParseInvidious(t *testing.T) {
	resp, err := parseInvidious([]byte(invidiousFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("len=%d", len(resp.Results))
	}
	r := resp.Results[0]
	if r.URL != "https://www.youtube.com/watch?v=WhWc3b3KhnY" {
		t.Errorf("url=%q (should be canonical youtube)", r.URL)
	}
	if r.Extras[ExtraDuration] != "465" {
		t.Errorf("duration=%q", r.Extras[ExtraDuration])
	}
	if r.Extras[ExtraAuthor] != "Blender Studio" {
		t.Errorf("author=%q", r.Extras[ExtraAuthor])
	}
	// Largest thumbnail should win.
	if r.Extras[ExtraThumbnailURL] != "https://invid/vi/WhWc3b3KhnY/maxres.jpg" {
		t.Errorf("thumb=%q", r.Extras[ExtraThumbnailURL])
	}
}

func TestParseInvidiousEmpty(t *testing.T) {
	if _, err := parseInvidious([]byte(`[]`)); err == nil {
		t.Fatal("expected error")
	}
}
