package engines

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JoakimCarlsson/scour/query"
)

func TestEngineTimeRangeForwarding(t *testing.T) {
	body := `<html><body>
<li class="b_algo"><h2><a href="https://example.com">t</a></h2><div class="b_caption"><p>s</p></div></li>
<div class="snippet" data-type="web"><a class="heading-serpresult" href="https://example.com"><span class="title">t</span></a><div class="snippet-description">s</div></div>
<div class="g"><a href="https://example.com"><h3>t</h3></a><div class="VwiC3b">s</div></div>
</body></html>`
	type spec struct {
		name   string
		engine Engine
		urlVar *string
		tr     query.TimeRange
		param  string
		want   string
	}
	specs := []spec{
		{"google day", googleEngine{}, &googleURL, query.TimeRangeDay, "tbs", "qdr:d"},
		{"google year", googleEngine{}, &googleURL, query.TimeRangeYear, "tbs", "qdr:y"},
		{"bing day", bingEngine{}, &bingURL, query.TimeRangeDay, "filters", `ex1:"ez1"`},
		{"brave week", braveEngine{}, &braveURL, query.TimeRangeWeek, "tf", "pw"},
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
				query.Query{Terms: "x", TimeRange: sp.tr},
			)
			if got != sp.want {
				t.Fatalf("%s: %s=%q, want %q", sp.name, sp.param, got, sp.want)
			}
		})
	}
}

func TestDuckDuckGoTimeRangeForwarding(t *testing.T) {
	cases := []struct {
		tr   query.TimeRange
		want string
	}{
		{query.TimeRangeDay, "d"},
		{query.TimeRangeWeek, "w"},
		{query.TimeRangeMonth, "m"},
		{query.TimeRangeYear, "y"},
	}
	for _, c := range cases {
		t.Run(string(c.tr), func(t *testing.T) {
			var got string
			srv := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_ = r.ParseForm()
					got = r.PostForm.Get("df")
					_, _ = w.Write(
						[]byte(
							`<html><body><div class="result"><a class="result__a" href="https://example.com">t</a></div></body></html>`,
						),
					)
				}),
			)
			defer srv.Close()
			orig := duckduckgoURL
			duckduckgoURL = srv.URL
			defer func() { duckduckgoURL = orig }()
			_, _ = duckduckgoEngine{}.Search(
				context.Background(),
				query.Query{Terms: "x", TimeRange: c.tr},
			)
			if got != c.want {
				t.Fatalf("df=%q, want %q", got, c.want)
			}
		})
	}
}
