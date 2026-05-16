package engines

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/JoakimCarlsson/scour/query"
)

var redditURL = "https://www.reddit.com/search.json"

// redditUA is the deliberate UA Reddit's API policy requires. Random
// pool UAs trip Reddit's 429.
const redditUA = "scour/0.x (https://github.com/JoakimCarlsson/scour)"

type redditEngine struct{}

func (redditEngine) Name() string                 { return "reddit" }
func (redditEngine) Categories() []query.Category { return []query.Category{query.CategorySocial} }
func (redditEngine) Languages() LanguageTraits    { return LanguageTraits{All: true} }
func (redditEngine) Weight() float64              { return 1.0 }

func (e redditEngine) Search(ctx context.Context, q query.Query) (Response, error) {
	u, _ := url.Parse(redditURL)
	v := u.Query()
	v.Set("q", q.Filters.Render(q.Terms))
	v.Set("limit", "20")
	v.Set("type", "link")
	u.RawQuery = v.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("User-Agent", redditUA)
	req.Header.Set("Accept", "application/json")
	body, err := fetch(req)
	if err != nil {
		return Response{}, err
	}
	return parseReddit(body)
}

func parseReddit(body []byte) (Response, error) {
	var payload struct {
		Data struct {
			Children []struct {
				Data struct {
					Title             string  `json:"title"`
					Permalink         string  `json:"permalink"`
					URL               string  `json:"url"`
					SubredditPrefixed string  `json:"subreddit_name_prefixed"`
					Author            string  `json:"author"`
					Score             int     `json:"score"`
					NumComments       int     `json:"num_comments"`
					CreatedUTC        float64 `json:"created_utc"`
					Selftext          string  `json:"selftext"`
					IsSelf            bool    `json:"is_self"`
					Thumbnail         string  `json:"thumbnail"`
				} `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Response{}, fmt.Errorf("reddit: %w", err)
	}
	var results []Result
	for i, c := range payload.Data.Children {
		p := c.Data
		if p.Title == "" || p.Permalink == "" {
			continue
		}
		// Permalink is the canonical comment-thread URL on reddit.com;
		// p.URL is the linked-to external article for link posts. Prefer
		// permalink so users land in the Reddit thread (matches user
		// expectation of a "Reddit result").
		link := "https://www.reddit.com" + p.Permalink
		extras := map[string]string{}
		if p.CreatedUTC > 0 {
			extras[ExtraPublishedAt] = time.Unix(int64(p.CreatedUTC), 0).
				UTC().
				Format(time.RFC3339)
		}
		if p.Author != "" && p.Author != "[deleted]" {
			extras[ExtraAuthor] = p.Author
		}
		if isHTTPThumb(p.Thumbnail) {
			extras[ExtraThumbnailURL] = p.Thumbnail
		}
		snippet := p.Selftext
		if snippet == "" {
			snippet = fmt.Sprintf("%s · %d upvotes · %d comments",
				p.SubredditPrefixed, p.Score, p.NumComments)
		}
		results = append(results, Result{
			Title:    p.Title,
			URL:      link,
			Snippet:  snippet,
			Engine:   "reddit",
			Position: i + 1,
			Extras:   extras,
		})
	}
	if len(results) == 0 {
		return Response{}, fmt.Errorf("reddit: no results parsed")
	}
	return Response{Results: results}, nil
}

func isHTTPThumb(s string) bool {
	return len(s) > 4 && (s[:5] == "http:" || (len(s) > 5 && s[:6] == "https:"))
}

func init() {
	Register(redditEngine{})
}
