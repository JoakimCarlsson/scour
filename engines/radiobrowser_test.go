package engines

import "testing"

const radioBrowserFixture = `[
  {
    "name": "SmoothJazz.com 64k aac+",
    "url": "https://smoothjazz.cdnstream1.com/2585_64.aac",
    "url_resolved": "https://smoothjazz.cdnstream1.com/2585_64.aac",
    "homepage": "https://www.smoothjazz.com/",
    "favicon": "https://www.smoothjazz.com/favicon.png",
    "country": "The United States Of America",
    "codec": "AAC+",
    "bitrate": 64,
    "tags": "jazz,smooth jazz",
    "votes": 1234
  },
  {
    "name": "Jazz Gumbo",
    "url": "https://streaming.smartradio.ch:9502/stream",
    "url_resolved": "https://streaming.smartradio.ch:9502/stream",
    "homepage": "https://jazzgumboradio.com/",
    "favicon": "https://jazzgumboradio.com/favicon.ico",
    "country": "Switzerland",
    "codec": "AAC+",
    "bitrate": 128,
    "tags": "jazz",
    "votes": 88
  }
]`

func TestParseRadioBrowser(t *testing.T) {
	resp, err := parseRadioBrowser([]byte(radioBrowserFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("len=%d", len(resp.Results))
	}
	r := resp.Results[0]
	if r.URL != "https://www.smoothjazz.com/" {
		t.Errorf("url=%q (should prefer homepage)", r.URL)
	}
	if r.Extras[ExtraThumbnailURL] != "https://www.smoothjazz.com/favicon.png" {
		t.Errorf("favicon=%q", r.Extras[ExtraThumbnailURL])
	}
	if r.Snippet == "" {
		t.Errorf("empty snippet")
	}
}

func TestParseRadioBrowserEmpty(t *testing.T) {
	if _, err := parseRadioBrowser([]byte(`[]`)); err == nil {
		t.Fatal("expected error")
	}
}
