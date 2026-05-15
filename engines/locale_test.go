package engines

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JoakimCarlsson/scour/query"
)

func TestLanguageTraitsAccepts(t *testing.T) {
	t.Run("all true accepts any", func(t *testing.T) {
		tr := LanguageTraits{All: true}
		if !tr.Accepts("ja") {
			t.Fatal("expected Accepts(ja) when All=true")
		}
	})
	t.Run("supported accepts known", func(t *testing.T) {
		tr := LanguageTraits{Supported: map[string]string{"ja": "jp"}}
		if !tr.Accepts("ja") {
			t.Fatal("expected Accepts(ja) for supported")
		}
		if tr.Accepts("de") {
			t.Fatal("did not expect Accepts(de) for unsupported non-All")
		}
	})
	t.Run("empty accepts empty query", func(t *testing.T) {
		tr := LanguageTraits{}
		if !tr.Accepts("") {
			t.Fatal("expected Accepts(empty) on empty traits")
		}
		if tr.Accepts("en") {
			t.Fatal("did not expect Accepts(en) on empty traits")
		}
	})
}

func TestEngineLocaleForwarding(t *testing.T) {
	type spec struct {
		name   string
		engine Engine
		urlVar *string
		bcp47  string
		param  string
		want   string
		body   string
	}
	specs := []spec{
		{
			name:   "bing setlang",
			engine: bingEngine{},
			urlVar: &bingURL,
			bcp47:  "ja",
			param:  "setlang",
			want:   "ja-JP",
			body:   `<html><body><li class="b_algo"><h2><a href="https://example.com">t</a></h2><div class="b_caption"><p>s</p></div></li></body></html>`,
		},
		{
			name:   "brave country",
			engine: braveEngine{},
			urlVar: &braveURL,
			bcp47:  "de",
			param:  "country",
			want:   "de",
			body:   `<html><body><div class="snippet" data-type="web"><a class="heading-serpresult" href="https://example.com"><span class="title">t</span></a><div class="snippet-description">s</div></div></body></html>`,
		},
		{
			name:   "google hl",
			engine: googleEngine{},
			urlVar: &googleURL,
			bcp47:  "fr",
			param:  "hl",
			want:   "fr",
			body:   `<html><body><div class="g"><a href="https://example.com"><h3>t</h3></a><div class="VwiC3b">s</div></div></body></html>`,
		},
	}
	for _, sp := range specs {
		t.Run(sp.name, func(t *testing.T) {
			var gotParam string
			srv := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_ = r.ParseForm()
					gotParam = r.URL.Query().Get(sp.param)
					_, _ = w.Write([]byte(sp.body))
				}),
			)
			defer srv.Close()
			orig := *sp.urlVar
			*sp.urlVar = srv.URL
			defer func() { *sp.urlVar = orig }()

			_, _ = sp.engine.Search(
				context.Background(),
				query.Query{Terms: "x", Language: sp.bcp47},
			)
			if gotParam != sp.want {
				t.Fatalf("%s: got %s=%q, want %q", sp.name, sp.param, gotParam, sp.want)
			}
		})
	}
}

func TestDuckDuckGoLocaleForwarding(t *testing.T) {
	var gotKL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotKL = r.PostForm.Get("kl")
		_, _ = w.Write(
			[]byte(
				`<html><body><div class="result"><a class="result__a" href="https://example.com">t</a></div></body></html>`,
			),
		)
	}))
	defer srv.Close()
	orig := duckduckgoURL
	duckduckgoURL = srv.URL
	defer func() { duckduckgoURL = orig }()

	_, _ = duckduckgoEngine{}.Search(
		context.Background(),
		query.Query{Terms: "x", Language: "ja"},
	)
	if gotKL != "jp-jp" {
		t.Fatalf("ddg kl: got %q, want jp-jp", gotKL)
	}
}
