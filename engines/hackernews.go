package engines

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/JoakimCarlsson/scour/query"
)

var hackernewsURL = "https://hn.algolia.com/api/v1/search"

type hackernewsEngine struct{}

func (hackernewsEngine) Name() string                 { return "hackernews" }
func (hackernewsEngine) Categories() []query.Category { return []query.Category{query.CategoryIT} }
func (hackernewsEngine) Languages() LanguageTraits    { return LanguageTraits{All: true} }
func (hackernewsEngine) Weight() float64              { return 1.0 }

func (e hackernewsEngine) Search(ctx context.Context, q query.Query) (Response, error) {
	u, _ := url.Parse(hackernewsURL)
	v := u.Query()
	v.Set("query", q.Filters.Render(q.Terms))
	v.Set("hitsPerPage", "20")
	if q.Page > 1 {
		v.Set("page", fmt.Sprintf("%d", q.Page-1))
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
	return parseHackerNews(body)
}

func parseHackerNews(body []byte) (Response, error) {
	var payload struct {
		Hits []struct {
			Title       string `json:"title"`
			StoryTitle  string `json:"story_title"`
			URL         string `json:"url"`
			StoryURL    string `json:"story_url"`
			ObjectID    string `json:"objectID"`
			CreatedAt   string `json:"created_at"`
			Author      string `json:"author"`
			Points      int    `json:"points"`
			NumComments int    `json:"num_comments"`
			StoryText   string `json:"story_text"`
			CommentText string `json:"comment_text"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Response{}, fmt.Errorf("hackernews: %w", err)
	}
	var results []Result
	for i, h := range payload.Hits {
		title := h.Title
		if title == "" {
			title = h.StoryTitle
		}
		if title == "" {
			continue
		}
		link := h.URL
		if link == "" {
			link = h.StoryURL
		}
		if link == "" && h.ObjectID != "" {
			link = "https://news.ycombinator.com/item?id=" + h.ObjectID
		}
		if link == "" {
			continue
		}
		snippet := h.StoryText
		if snippet == "" {
			snippet = h.CommentText
		}
		extras := map[string]string{}
		if h.CreatedAt != "" {
			extras[ExtraPublishedAt] = h.CreatedAt
		}
		if h.Author != "" {
			extras[ExtraAuthor] = h.Author
		}
		results = append(results, Result{
			Title:    title,
			URL:      link,
			Snippet:  stripHTML(snippet),
			Engine:   "hackernews",
			Position: i + 1,
			Extras:   extras,
		})
	}
	if len(results) == 0 {
		return Response{}, fmt.Errorf("hackernews: no results parsed")
	}
	return Response{Results: results}, nil
}

func init() {
	Register(hackernewsEngine{})
}
