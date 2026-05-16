package engines

import "testing"

const sepiaSearchFixture = `{
  "total": 2,
  "data": [
    {
      "name": "Spring - Blender Open Movie",
      "url": "https://video.blender.org/videos/watch/3d95fb3d",
      "description": "A short film by Blender Studio.",
      "duration": 464,
      "thumbnailUrl": "https://video.blender.org/thumbs/spring.jpg",
      "publishedAt": "2019-06-26T14:54:03.643Z",
      "channel": {"displayName": "Official Blender Open Movies"}
    },
    {
      "name": "Charge",
      "url": "https://video.blender.org/videos/watch/04da454b",
      "duration": 263,
      "thumbnailUrl": "https://video.blender.org/thumbs/charge.jpg",
      "publishedAt": "2022-12-15T17:46:29.626Z",
      "channel": {"displayName": "Blender Studio"}
    }
  ]
}`

func TestParseSepiaSearch(t *testing.T) {
	resp, err := parseSepiaSearch([]byte(sepiaSearchFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("len=%d", len(resp.Results))
	}
	r := resp.Results[0]
	if r.URL != "https://video.blender.org/videos/watch/3d95fb3d" {
		t.Errorf("url=%q", r.URL)
	}
	if r.Extras[ExtraDuration] != "464" {
		t.Errorf("duration=%q", r.Extras[ExtraDuration])
	}
	if r.Extras[ExtraThumbnailURL] != "https://video.blender.org/thumbs/spring.jpg" {
		t.Errorf("thumb=%q", r.Extras[ExtraThumbnailURL])
	}
	if r.Extras[ExtraAuthor] != "Official Blender Open Movies" {
		t.Errorf("author=%q", r.Extras[ExtraAuthor])
	}
}

func TestParseSepiaSearchEmpty(t *testing.T) {
	if _, err := parseSepiaSearch([]byte(`{"data":[]}`)); err == nil {
		t.Fatal("expected error")
	}
}
