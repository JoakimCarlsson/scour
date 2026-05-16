package engines

import "testing"

const stackExchangeFixture = `{
  "items": [
    {
      "title": "How does &lt;T&gt; work in Go generics?",
      "link": "https://stackoverflow.com/questions/72419191/golang-generics",
      "score": 12,
      "answer_count": 3,
      "is_answered": true,
      "creation_date": 1653771168,
      "tags": ["go","generics"],
      "owner": {"display_name": "vocalionecho"}
    },
    {
      "title": "Golang Generic+Variadic Function",
      "link": "https://stackoverflow.com/questions/76602347/golang-generic-variadic",
      "score": 3,
      "answer_count": 1,
      "is_answered": false,
      "creation_date": 1688366138,
      "owner": {"display_name": "404"}
    }
  ]
}`

func TestParseStackExchange(t *testing.T) {
	resp, err := parseStackExchange([]byte(stackExchangeFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("len=%d", len(resp.Results))
	}
	r := resp.Results[0]
	if r.Title != "How does <T> work in Go generics?" {
		t.Errorf("title html-unescape broken: %q", r.Title)
	}
	if r.URL != "https://stackoverflow.com/questions/72419191/golang-generics" {
		t.Errorf("url=%q", r.URL)
	}
	if r.Extras[ExtraAuthor] != "vocalionecho" {
		t.Errorf("author=%q", r.Extras[ExtraAuthor])
	}
	if r.Extras[ExtraPublishedAt] == "" {
		t.Errorf("published empty")
	}
}

func TestParseStackExchangeEmpty(t *testing.T) {
	if _, err := parseStackExchange([]byte(`{"items":[]}`)); err == nil {
		t.Fatal("expected error")
	}
}
