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
<div class="news-card newsitem cardcommon" url="https://example.com/story-1" title="Story 1" data-author="Example News">
  <div class="caption"><div class="t_s"><div class="t_t">
    <div class="source"><a>Example News</a><span><div class="ns_sc_tm">1 hour ago</div></span></div>
    <a class="title" href="https://example.com/story-1"><h2>Story 1</h2></a>
  </div></div></div>
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
	if resp.Results[0].Extras[ExtraAuthor] != "Example News" {
		t.Errorf("author=%q", resp.Results[0].Extras[ExtraAuthor])
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
