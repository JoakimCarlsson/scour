package engines

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JoakimCarlsson/scour/query"
)

func TestEngineForwardsFilters(t *testing.T) {
	body := `<html><body>
<li class="b_algo"><h2><a href="https://example.com">t</a></h2><div class="b_caption"><p>s</p></div></li>
<div class="snippet" data-type="web"><a class="heading-serpresult" href="https://example.com"><span class="title">t</span></a><div class="snippet-description">s</div></div>
</body></html>`
	type spec struct {
		name        string
		engine      Engine
		urlVar      *string
		wantInQuery []string
	}
	specs := []spec{
		{
			"bing",
			bingEngine{},
			&bingURL,
			[]string{"site:reddit.com", "filetype:pdf", "-windows"},
		},
		{
			"brave",
			braveEngine{},
			&braveURL,
			[]string{"site:reddit.com", "filetype:pdf", "-windows"},
		},
	}
	for _, sp := range specs {
		t.Run(sp.name, func(t *testing.T) {
			var gotQ string
			srv := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					gotQ = r.URL.Query().Get("q")
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
			for _, want := range sp.wantInQuery {
				if !strings.Contains(gotQ, want) {
					t.Errorf("%s: q=%q missing %q", sp.name, gotQ, want)
				}
			}
		})
	}
}
