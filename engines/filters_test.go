package engines

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JoakimCarlsson/scour/query"
)

// TestEngineForwardsFilters asserts every engine that takes a free-form
// query param actually appends the filter operators when q.Filters is
// non-empty. Bare q.Terms (the bug fixed here) silently drops the
// filters and leaves the user wondering why site:reddit.com didn't
// narrow anything.
func TestEngineForwardsFilters(t *testing.T) {
	body := `<html><body>
<li class="b_algo"><h2><a href="https://example.com">t</a></h2><div class="b_caption"><p>s</p></div></li>
<div class="snippet" data-type="web"><a class="heading-serpresult" href="https://example.com"><span class="title">t</span></a><div class="snippet-description">s</div></div>
<div class="result"><a class="result__a" href="https://example.com">t</a></div>
<ul class="results-standard"><li><h2><a href="https://example.com">t</a></h2><p class="s">s</p></li></ul>
<li class="serp-item"><h2><a class="OrganicTitle-Link" href="https://example.com">t</a></h2></li>
</body></html>`

	type spec struct {
		name       string
		engine     Engine
		urlVar     *string
		paramName  string // GET query-string key or POST form key
		isPost     bool   // true if engine sends form data
		wantSubstr []string
	}
	specs := []spec{
		{
			name:       "bing",
			engine:     bingEngine{},
			urlVar:     &bingURL,
			paramName:  "q",
			wantSubstr: []string{"golang", "site:reddit.com", "filetype:pdf", "-windows"},
		},
		{
			name:       "brave",
			engine:     braveEngine{},
			urlVar:     &braveURL,
			paramName:  "q",
			wantSubstr: []string{"golang", "site:reddit.com", "filetype:pdf", "-windows"},
		},
		{
			name:       "duckduckgo",
			engine:     duckduckgoEngine{},
			urlVar:     &duckduckgoURL,
			paramName:  "q",
			isPost:     true,
			wantSubstr: []string{"golang", "site:reddit.com", "filetype:pdf", "-windows"},
		},
		{
			name:       "mojeek",
			engine:     mojeekEngine{},
			urlVar:     &mojeekURL,
			paramName:  "q",
			wantSubstr: []string{"golang", "site:reddit.com", "filetype:pdf", "-windows"},
		},
		{
			name:       "qwant",
			engine:     qwantEngine{},
			urlVar:     &qwantURL,
			paramName:  "q",
			wantSubstr: []string{"golang", "site:reddit.com", "filetype:pdf", "-windows"},
		},
		{
			name:       "yandex",
			engine:     yandexEngine{},
			urlVar:     &yandexURL,
			paramName:  "text",
			wantSubstr: []string{"golang", "site:reddit.com", "filetype:pdf", "-windows"},
		},
	}
	for _, sp := range specs {
		t.Run(sp.name, func(t *testing.T) {
			var got string
			srv := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if sp.isPost {
						_ = r.ParseForm()
						got = r.PostForm.Get(sp.paramName)
					} else {
						got = r.URL.Query().Get(sp.paramName)
					}
					_, _ = w.Write([]byte(body))
				}),
			)
			defer srv.Close()
			orig := *sp.urlVar
			*sp.urlVar = srv.URL
			defer func() { *sp.urlVar = orig }()

			_, _ = sp.engine.Search(context.Background(), query.Query{
				Terms: "golang",
				Filters: query.Filters{
					Sites:     []string{"reddit.com"},
					FileTypes: []string{"pdf"},
					Excluded:  []string{"windows"},
				},
			})
			for _, want := range sp.wantSubstr {
				if !strings.Contains(got, want) {
					t.Errorf("%s: %s=%q missing %q", sp.name, sp.paramName, got, want)
				}
			}
		})
	}
}
