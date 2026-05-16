package engines

import "testing"

const redditFixture = `{
  "data": {
    "children": [
      {"data": {
        "title": "New to mechanical keyboards - whats the best one to get?",
        "permalink": "/r/MechKeyboards/comments/1qtho3t/new_to_mechanical_keyboards/",
        "url": "https://www.reddit.com/r/MechKeyboards/comments/1qtho3t/new_to_mechanical_keyboards/",
        "subreddit_name_prefixed": "r/MechKeyboards",
        "author": "user1",
        "score": 6,
        "num_comments": 12,
        "created_utc": 1769996850,
        "selftext": "Hi all..."
      }},
      {"data": {
        "title": "Best Mechanical Keyboards 2025",
        "permalink": "/r/MechanicalKeyboards/comments/1q0c0ls/best_mechanical_keyboards_2025/",
        "url": "https://www.reddit.com/gallery/1q0c0ls",
        "subreddit_name_prefixed": "r/MechanicalKeyboards",
        "author": "[deleted]",
        "score": 207,
        "num_comments": 50,
        "created_utc": 1767184522,
        "thumbnail": "https://b.thumbs.redditmedia.com/abc.jpg"
      }}
    ]
  }
}`

func TestParseReddit(t *testing.T) {
	resp, err := parseReddit([]byte(redditFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("len=%d", len(resp.Results))
	}
	r := resp.Results[0]
	if r.URL != "https://www.reddit.com/r/MechKeyboards/comments/1qtho3t/new_to_mechanical_keyboards/" {
		t.Errorf("url=%q", r.URL)
	}
	if r.Extras[ExtraAuthor] != "user1" {
		t.Errorf("author=%q", r.Extras[ExtraAuthor])
	}
	if r.Extras[ExtraPublishedAt] == "" {
		t.Errorf("published empty")
	}
	// Second result: author [deleted] → not set; thumbnail set.
	r2 := resp.Results[1]
	if _, ok := r2.Extras[ExtraAuthor]; ok {
		t.Errorf("[deleted] author shouldn't be emitted")
	}
	if r2.Extras[ExtraThumbnailURL] != "https://b.thumbs.redditmedia.com/abc.jpg" {
		t.Errorf("thumb=%q", r2.Extras[ExtraThumbnailURL])
	}
}

func TestParseRedditEmpty(t *testing.T) {
	if _, err := parseReddit([]byte(`{"data":{"children":[]}}`)); err == nil {
		t.Fatal("expected error")
	}
}
