package engines

import "testing"

const lemmyFixture = `{
  "posts": [
    {
      "post": {
        "name": "Mad Men and Satisfying Switches",
        "url": "https://example.com/switches",
        "ap_id": "https://lemmy.world/post/46096822",
        "body": "Talking about switches...",
        "published": "2026-04-26T19:35:02.931472Z",
        "thumbnail_url": "https://lemmy.world/pictrs/image/abc.jpg"
      },
      "community": {"name": "mechanicalkeyboards"},
      "creator": {"name": "iconic_admin"},
      "counts": {"score": 42, "comments": 18}
    },
    {
      "post": {
        "name": "Keychron source release",
        "url": "https://www.pcgamer.com/...",
        "ap_id": "https://ibbit.at/post/223841",
        "published": "2026-04-10T14:41:10.254240Z"
      },
      "community": {"name": "pcgamer"},
      "creator": {"name": "rss"},
      "counts": {"score": 7, "comments": 3}
    }
  ]
}`

func TestParseLemmy(t *testing.T) {
	resp, err := parseLemmy([]byte(lemmyFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("len=%d", len(resp.Results))
	}
	r := resp.Results[0]
	if r.URL != "https://lemmy.world/post/46096822" {
		t.Errorf("url=%q (should use ap_id)", r.URL)
	}
	if r.Extras[ExtraAuthor] != "iconic_admin" {
		t.Errorf("author=%q", r.Extras[ExtraAuthor])
	}
	if r.Extras[ExtraThumbnailURL] != "https://lemmy.world/pictrs/image/abc.jpg" {
		t.Errorf("thumb=%q", r.Extras[ExtraThumbnailURL])
	}
	// Second result has no body - synthesize a c/community snippet.
	r2 := resp.Results[1]
	if r2.Snippet == "" {
		t.Errorf("expected synthesized snippet")
	}
}

func TestParseLemmyEmpty(t *testing.T) {
	if _, err := parseLemmy([]byte(`{"posts":[]}`)); err == nil {
		t.Fatal("expected error")
	}
}
