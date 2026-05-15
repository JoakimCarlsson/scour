package engines

import "testing"

func TestParseDuckDuckGoImages(t *testing.T) {
	body := []byte(
		`{"results":[{"title":"cat","url":"https://example.com/cat","image":"https://example.com/cat.jpg","thumbnail":"https://example.com/cat-thumb.jpg","width":640,"height":480}]}`,
	)
	resp, err := parseDuckDuckGoImages(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("results=%d", len(resp.Results))
	}
	r := resp.Results[0]
	if r.Extras[ExtraThumbnailURL] != "https://example.com/cat-thumb.jpg" {
		t.Errorf("thumbnail = %q", r.Extras[ExtraThumbnailURL])
	}
	if r.Extras[ExtraThumbnailWidth] != "640" {
		t.Errorf("width = %q", r.Extras[ExtraThumbnailWidth])
	}
}
