package engines

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/JoakimCarlsson/scour/query"
)

var searxPublicURL = "https://searx.be/search"

type searxPublicEngine struct{}

func (searxPublicEngine) Name() string { return "searx_public" }
func (searxPublicEngine) Categories() []query.Category {
	return []query.Category{query.CategoryGeneral}
}
func (searxPublicEngine) Languages() LanguageTraits { return LanguageTraits{All: true} }
func (searxPublicEngine) Weight() float64           { return 0.7 }

func (e searxPublicEngine) Search(ctx context.Context, q query.Query) (Response, error) {
	u, _ := url.Parse(searxPublicURL)
	v := u.Query()
	v.Set("q", q.Terms)
	v.Set("format", "json")
	v.Set("categories", "general")
	if q.Page > 1 {
		v.Set("pageno", fmt.Sprintf("%d", q.Page))
	}
	if q.Language != "" {
		v.Set("language", q.Language)
	}
	switch q.SafeSearch {
	case query.SafeOff:
		v.Set("safesearch", "0")
	case query.SafeModerate:
		v.Set("safesearch", "1")
	case query.SafeStrict:
		v.Set("safesearch", "2")
	}
	u.RawQuery = v.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("Accept", "application/json")
	body, err := fetch(req)
	if err != nil {
		return Response{}, err
	}
	return parseSearxPublic(body)
}

func parseSearxPublic(body []byte) (Response, error) {
	var payload struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
		Suggestions []string `json:"suggestions"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Response{}, fmt.Errorf("searx_public: %w", err)
	}
	var out []Result
	for i, r := range payload.Results {
		if r.Title == "" || r.URL == "" {
			continue
		}
		out = append(out, Result{
			Title:    r.Title,
			URL:      r.URL,
			Snippet:  r.Content,
			Engine:   "searx_public",
			Position: i + 1,
		})
	}
	if len(out) == 0 {
		return Response{}, fmt.Errorf("searx_public: no results parsed")
	}
	return Response{Results: out, Suggestions: payload.Suggestions}, nil
}

func init() {
	Register(searxPublicEngine{})
}
