package engines

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JoakimCarlsson/scour/query"
)

func TestEngineSafeSearchForwarding(t *testing.T) {
	type spec struct {
		name   string
		engine Engine
		urlVar *string
		level  query.SafeLevel
		param  string
		want   string
		body   string
	}
	body := `<html><body>
<li class="b_algo"><h2><a href="https://example.com">t</a></h2><div class="b_caption"><p>s</p></div></li>
<div class="snippet" data-type="web"><a class="heading-serpresult" href="https://example.com"><span class="title">t</span></a><div class="snippet-description">s</div></div>
<div class="g"><a href="https://example.com"><h3>t</h3></a><div class="VwiC3b">s</div></div>
</body></html>`
	specs := []spec{
		{"bing strict", bingEngine{}, &bingURL, query.SafeStrict, "adlt", "strict", body},
		{"bing off", bingEngine{}, &bingURL, query.SafeOff, "adlt", "off", body},
		{
			"brave moderate",
			braveEngine{},
			&braveURL,
			query.SafeModerate,
			"safesearch",
			"moderate",
			body,
		},
		{"brave strict", braveEngine{}, &braveURL, query.SafeStrict, "safesearch", "strict", body},
		{"google strict", googleEngine{}, &googleURL, query.SafeStrict, "safe", "active", body},
		{"google off", googleEngine{}, &googleURL, query.SafeOff, "safe", "off", body},
	}
	for _, sp := range specs {
		t.Run(sp.name, func(t *testing.T) {
			var got string
			srv := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					got = r.URL.Query().Get(sp.param)
					_, _ = w.Write([]byte(sp.body))
				}),
			)
			defer srv.Close()
			orig := *sp.urlVar
			*sp.urlVar = srv.URL
			defer func() { *sp.urlVar = orig }()
			_, _ = sp.engine.Search(
				context.Background(),
				query.Query{Terms: "x", SafeSearch: sp.level},
			)
			if got != sp.want {
				t.Fatalf("%s: %s=%q, want %q", sp.name, sp.param, got, sp.want)
			}
		})
	}
}

func TestDuckDuckGoSafeSearchForwarding(t *testing.T) {
	cases := []struct {
		level query.SafeLevel
		kp    string
	}{
		{query.SafeOff, "-2"},
		{query.SafeModerate, "-1"},
		{query.SafeStrict, "1"},
	}
	for _, c := range cases {
		t.Run(c.kp, func(t *testing.T) {
			var got string
			srv := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_ = r.ParseForm()
					got = r.PostForm.Get("kp")
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
				query.Query{Terms: "x", SafeSearch: c.level},
			)
			if got != c.kp {
				t.Fatalf("kp=%q, want %q", got, c.kp)
			}
		})
	}
}
