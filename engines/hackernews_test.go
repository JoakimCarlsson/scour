package engines

import "testing"

const hnFixture = `{
  "nbHits": 2,
  "hits": [
    {"title":"Go generics proposal accepted","url":"https://github.com/golang/go/issues/43651","objectID":"26101471","created_at":"2021-02-10T19:48:30Z","author":"alice","points":1234,"num_comments":420},
    {"story_title":"Generics Diaries","story_url":"https://tbray.org/x","objectID":"31374322","created_at":"2022-05-15T13:04:17Z","author":"bob"}
  ]
}`

func TestParseHackerNews(t *testing.T) {
	resp, err := parseHackerNews([]byte(hnFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("len=%d", len(resp.Results))
	}
	if resp.Results[0].URL != "https://github.com/golang/go/issues/43651" {
		t.Errorf("url=%q", resp.Results[0].URL)
	}
	if resp.Results[0].Extras[ExtraAuthor] != "alice" {
		t.Errorf("author=%q", resp.Results[0].Extras[ExtraAuthor])
	}
	if resp.Results[1].Title != "Generics Diaries" {
		t.Errorf("fallback to story_title broken: %q", resp.Results[1].Title)
	}
}

func TestParseHackerNewsEmpty(t *testing.T) {
	if _, err := parseHackerNews([]byte(`{"hits":[]}`)); err == nil {
		t.Fatal("expected error")
	}
}
