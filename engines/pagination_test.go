package engines

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JoakimCarlsson/scour/query"
)

func TestEnginePaginationForwarding(t *testing.T) {
	body := `<html><body>
<li class="b_algo"><h2><a href="https://example.com">t</a></h2><div class="b_caption"><p>s</p></div></li>
<div class="snippet" data-type="web"><a class="heading-serpresult" href="https://example.com"><span class="title">t</span></a><div class="snippet-description">s</div></div>
<div class="g"><a href="https://example.com"><h3>t</h3></a><div class="VwiC3b">s</div></div>
</body></html>`
	type spec struct {
		name   string
		engine Engine
		urlVar *string
		page   int
		param  string
		want   string
	}
	specs := []spec{
		{"google p2", googleEngine{}, &googleURL, 2, "start", "10"},
		{"google p3", googleEngine{}, &googleURL, 3, "start", "20"},
		{"bing p2", bingEngine{}, &bingURL, 2, "first", "11"},
		{"brave p2", braveEngine{}, &braveURL, 2, "offset", "1"},
	}
	for _, sp := range specs {
		t.Run(sp.name, func(t *testing.T) {
			var got string
			srv := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					got = r.URL.Query().Get(sp.param)
					_, _ = w.Write([]byte(body))
				}),
			)
			defer srv.Close()
			orig := *sp.urlVar
			*sp.urlVar = srv.URL
			defer func() { *sp.urlVar = orig }()
			_, _ = sp.engine.Search(
				context.Background(),
				query.Query{Terms: "x", Page: sp.page},
			)
			if got != sp.want {
				t.Fatalf("%s: %s=%q, want %q", sp.name, sp.param, got, sp.want)
			}
		})
	}
}

func TestDuckDuckGoPaginationForwarding(t *testing.T) {
	var gotS string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotS = r.PostForm.Get("s")
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
		query.Query{Terms: "x", Page: 2},
	)
	if gotS != "30" {
		t.Fatalf("ddg s: got %q, want 30", gotS)
	}
}
