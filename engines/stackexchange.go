package engines

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"time"

	"github.com/JoakimCarlsson/scour/query"
)

var stackExchangeURL = "https://api.stackexchange.com/2.3/search/advanced"

type stackExchangeEngine struct{}

func (stackExchangeEngine) Name() string { return "stackexchange" }

func (stackExchangeEngine) Categories() []query.Category { return []query.Category{query.CategoryIT} }
func (stackExchangeEngine) Languages() LanguageTraits    { return LanguageTraits{All: true} }
func (stackExchangeEngine) Weight() float64              { return 1.0 }

func (e stackExchangeEngine) Search(ctx context.Context, q query.Query) (Response, error) {
	u, _ := url.Parse(stackExchangeURL)
	v := u.Query()
	v.Set("q", q.Filters.Render(q.Terms))
	v.Set("site", "stackoverflow")
	v.Set("pagesize", "20")
	v.Set("order", "desc")
	v.Set("sort", "relevance")
	if q.Page > 1 {
		v.Set("page", fmt.Sprintf("%d", q.Page))
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
	return parseStackExchange(body)
}

func parseStackExchange(body []byte) (Response, error) {
	var payload struct {
		Items []struct {
			Title        string   `json:"title"`
			Link         string   `json:"link"`
			Score        int      `json:"score"`
			AnswerCount  int      `json:"answer_count"`
			IsAnswered   bool     `json:"is_answered"`
			CreationDate int64    `json:"creation_date"`
			Tags         []string `json:"tags"`
			Owner        struct {
				DisplayName string `json:"display_name"`
			} `json:"owner"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Response{}, fmt.Errorf("stackexchange: %w", err)
	}
	var results []Result
	for i, q := range payload.Items {
		if q.Title == "" || q.Link == "" {
			continue
		}
		extras := map[string]string{}
		if q.CreationDate > 0 {
			extras[ExtraPublishedAt] = time.Unix(q.CreationDate, 0).UTC().Format(time.RFC3339)
		}
		if q.Owner.DisplayName != "" {
			extras[ExtraAuthor] = q.Owner.DisplayName
		}
		var status string
		if q.IsAnswered {
			status = fmt.Sprintf("✓ answered (%d answers, score %d)", q.AnswerCount, q.Score)
		} else {
			status = fmt.Sprintf("%d answers, score %d", q.AnswerCount, q.Score)
		}
		results = append(results, Result{
			Title:    html.UnescapeString(q.Title),
			URL:      q.Link,
			Snippet:  status,
			Engine:   "stackexchange",
			Position: i + 1,
			Extras:   extras,
		})
	}
	if len(results) == 0 {
		return Response{}, fmt.Errorf("stackexchange: no results parsed")
	}
	return Response{Results: results}, nil
}

func init() {
	Register(stackExchangeEngine{})
}
