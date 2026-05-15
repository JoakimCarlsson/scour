package engines

import "testing"

func TestParseBingImages(t *testing.T) {
	body := []byte(`<html><body>
<a class="iusc" m='{"murl":"https://example.com/cat.jpg","turl":"https://example.com/cat-thumb.jpg","t":"cat photo","desc":"a cat"}'></a>
<a class="iusc" m='{"murl":"https://example.com/dog.jpg","turl":"https://example.com/dog-thumb.jpg","t":"dog photo","desc":"a dog"}'></a>
</body></html>`)
	resp, err := parseBingImages(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("len=%d", len(resp.Results))
	}
	if resp.Results[0].Extras[ExtraThumbnailURL] != "https://example.com/cat-thumb.jpg" {
		t.Errorf("thumb=%q", resp.Results[0].Extras[ExtraThumbnailURL])
	}
}

func TestParseBingNews(t *testing.T) {
	body := []byte(`<html><body>
<div class="news-card">
  <a class="title" href="https://example.com/story-1">Story 1</a>
  <div class="snippet">A summary.</div>
  <div class="source"><span>1 hour ago</span></div>
</div>
</body></html>`)
	resp, err := parseBingNews(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("len=%d", len(resp.Results))
	}
	if resp.Results[0].Extras[ExtraPublishedAt] != "1 hour ago" {
		t.Errorf("published=%q", resp.Results[0].Extras[ExtraPublishedAt])
	}
}

func TestParseBraveNews(t *testing.T) {
	body := []byte(`<html><body>
<div class="snippet" data-type="news">
  <a href="https://example.com/story"><h3>Story</h3></a>
  <div class="snippet-description">Summary.</div>
  <time>2 hours ago</time>
</div>
</body></html>`)
	resp, err := parseBraveNews(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("len=%d", len(resp.Results))
	}
	if resp.Results[0].Extras[ExtraPublishedAt] != "2 hours ago" {
		t.Errorf("published=%q", resp.Results[0].Extras[ExtraPublishedAt])
	}
}
