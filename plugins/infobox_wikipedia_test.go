package plugins

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JoakimCarlsson/scour/query"
)

func TestInfoboxWikipediaSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Go_programming_language" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(
			[]byte(
				`{"title":"Go (programming language)","extract":"Go is a statically typed language.","content_urls":{"desktop":{"page":"https://en.wikipedia.org/wiki/Go_(programming_language)"}}}`,
			),
		)
	}))
	defer srv.Close()

	c := &Context{
		Query: query.Query{Terms: "Go programming language", Category: query.CategoryGeneral},
	}
	p := InfoboxWikipedia{Endpoint: srv.URL, HTTPClient: srv.Client()}
	if err := p.Apply(context.Background(), c); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if c.Infobox == nil {
		t.Fatal("expected infobox, got nil")
	}
	if c.Infobox.Title != "Go (programming language)" {
		t.Errorf("title: %q", c.Infobox.Title)
	}
	if c.Infobox.Summary == "" {
		t.Error("summary empty")
	}
}

func TestInfoboxWikipedia404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.NotFound(w, nil)
	}))
	defer srv.Close()

	c := &Context{Query: query.Query{Terms: "asdf", Category: query.CategoryGeneral}}
	p := InfoboxWikipedia{Endpoint: srv.URL, HTTPClient: srv.Client()}
	if err := p.Apply(context.Background(), c); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if c.Infobox != nil {
		t.Fatalf("expected nil infobox on 404, got %+v", c.Infobox)
	}
}

func TestInfoboxWikipediaSkipsNonGeneral(t *testing.T) {
	c := &Context{Query: query.Query{Terms: "go", Category: query.CategoryImages}}
	p := InfoboxWikipedia{Endpoint: "http://should.not.be.called"}
	if err := p.Apply(context.Background(), c); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if c.Infobox != nil {
		t.Fatalf("expected nil infobox, got %+v", c.Infobox)
	}
}

func TestInfoboxWikipediaSkipsLongQuery(t *testing.T) {
	c := &Context{Query: query.Query{
		Terms:    "this is a very long multi word query unlikely to be a single subject",
		Category: query.CategoryGeneral,
	}}
	p := InfoboxWikipedia{Endpoint: "http://should.not.be.called"}
	if err := p.Apply(context.Background(), c); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if c.Infobox != nil {
		t.Fatalf("expected nil infobox, got %+v", c.Infobox)
	}
}
