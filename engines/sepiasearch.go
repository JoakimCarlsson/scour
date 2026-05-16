package engines

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/JoakimCarlsson/scour/query"
)

var sepiaSearchURL = "https://sepiasearch.org/api/v1/search/videos"

type sepiaSearchEngine struct{}

func (sepiaSearchEngine) Name() string { return "sepiasearch" }
func (sepiaSearchEngine) Categories() []query.Category {
	return []query.Category{query.CategoryVideos}
}
func (sepiaSearchEngine) Languages() LanguageTraits { return LanguageTraits{All: true} }
func (sepiaSearchEngine) Weight() float64           { return 1.0 }

func (e sepiaSearchEngine) Search(ctx context.Context, q query.Query) (Response, error) {
	u, _ := url.Parse(sepiaSearchURL)
	v := u.Query()
	v.Set("search", q.Filters.Render(q.Terms))
	v.Set("count", "20")
	if q.Page > 1 {
		v.Set("start", fmt.Sprintf("%d", (q.Page-1)*20))
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
	return parseSepiaSearch(body)
}

func parseSepiaSearch(body []byte) (Response, error) {
	var payload struct {
		Total int `json:"total"`
		Data  []struct {
			Name         string `json:"name"`
			URL          string `json:"url"`
			Description  string `json:"description"`
			Duration     int    `json:"duration"`
			ThumbnailURL string `json:"thumbnailUrl"`
			PublishedAt  string `json:"publishedAt"`
			Channel      struct {
				DisplayName string `json:"displayName"`
			} `json:"channel"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Response{}, fmt.Errorf("sepiasearch: %w", err)
	}
	var results []Result
	for i, v := range payload.Data {
		if v.Name == "" || v.URL == "" {
			continue
		}
		extras := map[string]string{}
		if v.ThumbnailURL != "" {
			extras[ExtraThumbnailURL] = v.ThumbnailURL
		}
		if v.Duration > 0 {
			extras[ExtraDuration] = strconv.Itoa(v.Duration)
		}
		if v.PublishedAt != "" {
			extras[ExtraPublishedAt] = v.PublishedAt
		}
		if v.Channel.DisplayName != "" {
			extras[ExtraAuthor] = v.Channel.DisplayName
		}
		results = append(results, Result{
			Title:    v.Name,
			URL:      v.URL,
			Snippet:  v.Description,
			Engine:   "sepiasearch",
			Position: i + 1,
			Extras:   extras,
		})
	}
	if len(results) == 0 {
		return Response{}, fmt.Errorf("sepiasearch: no results parsed")
	}
	return Response{Results: results}, nil
}

func init() {
	Register(sepiaSearchEngine{})
}
