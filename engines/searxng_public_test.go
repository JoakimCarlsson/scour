package engines

import "testing"

const searxFixture = `{
  "results": [
    {"title":"Go","url":"https://go.dev/","content":"open source language"},
    {"title":"Go on GitHub","url":"https://github.com/golang/go","content":"source"}
  ],
  "suggestions": ["golang tutorial"]
}`

func TestParseSearxPublic(t *testing.T) {
	resp, err := parseSearxPublic([]byte(searxFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("len=%d", len(resp.Results))
	}
	if len(resp.Suggestions) != 1 || resp.Suggestions[0] != "golang tutorial" {
		t.Errorf("suggestions=%v", resp.Suggestions)
	}
}

func TestParseSearxPublicEmpty(t *testing.T) {
	if _, err := parseSearxPublic([]byte(`{"results":[]}`)); err == nil {
		t.Fatal("expected error")
	}
}
