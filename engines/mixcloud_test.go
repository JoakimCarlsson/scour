package engines

import "testing"

const mixcloudFixture = `{
  "data": [
    {
      "name": "Jazz Only - Dawson",
      "url": "https://www.mixcloud.com/face/jazz-only-dawson/",
      "audio_length": 10785,
      "created_time": "2024-11-25T10:00:03Z",
      "pictures": {
        "large": "https://thumbnailer.mixcloud.com/unsafe/600x600/extaudio/a/9/d/b/489d.jpg",
        "medium": "https://thumbnailer.mixcloud.com/unsafe/100x100/extaudio/a/9/d/b/489d.jpg"
      },
      "user": {"name": "The Face Radio", "key": "/face/"}
    },
    {
      "name": "Jazz Day",
      "url": "https://www.mixcloud.com/face/jazz-day/",
      "audio_length": 3584,
      "created_time": "2025-04-30T19:11:01Z",
      "pictures": {"medium": "https://thumbnailer.mixcloud.com/unsafe/100x100/extaudio/2/d/2/d/27e1.jpg"},
      "user": {"name": "The Face Radio"}
    }
  ]
}`

func TestParseMixcloud(t *testing.T) {
	resp, err := parseMixcloud([]byte(mixcloudFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("len=%d", len(resp.Results))
	}
	r := resp.Results[0]
	if r.URL != "https://www.mixcloud.com/face/jazz-only-dawson/" {
		t.Errorf("url=%q", r.URL)
	}
	if r.Extras[ExtraDuration] != "10785" {
		t.Errorf("duration=%q", r.Extras[ExtraDuration])
	}
	if r.Extras[ExtraAuthor] != "The Face Radio" {
		t.Errorf("author=%q", r.Extras[ExtraAuthor])
	}
	// Large should be preferred over medium.
	if r.Extras[ExtraThumbnailURL] != "https://thumbnailer.mixcloud.com/unsafe/600x600/extaudio/a/9/d/b/489d.jpg" {
		t.Errorf("thumb=%q (should prefer large)", r.Extras[ExtraThumbnailURL])
	}
	// Second result has only medium - that should be selected.
	if resp.Results[1].Extras[ExtraThumbnailURL] != "https://thumbnailer.mixcloud.com/unsafe/100x100/extaudio/2/d/2/d/27e1.jpg" {
		t.Errorf("fallback to medium broken")
	}
}

func TestParseMixcloudEmpty(t *testing.T) {
	if _, err := parseMixcloud([]byte(`{"data":[]}`)); err == nil {
		t.Fatal("expected error")
	}
}
