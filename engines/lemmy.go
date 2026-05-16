package engines

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strings"

	"github.com/JoakimCarlsson/scour/query"
)

// lemmyInstances are well-known public Lemmy servers. Each one indexes
// federated content, so any of them gives a reasonable cross-instance
// view. Hard-coded list with random-offset rotation; admin override
// via config is a future concern.
var lemmyInstances = []string{
	"https://lemmy.world",
	"https://lemmy.ml",
	"https://programming.dev",
	"https://sopuli.xyz",
}

type lemmyEngine struct{}

func (lemmyEngine) Name() string                 { return "lemmy" }
func (lemmyEngine) Categories() []query.Category { return []query.Category{query.CategorySocial} }
func (lemmyEngine) Languages() LanguageTraits    { return LanguageTraits{All: true} }
func (lemmyEngine) Weight() float64              { return 1.0 }

func (e lemmyEngine) Search(ctx context.Context, q query.Query) (Response, error) {
	if len(lemmyInstances) == 0 {
		return Response{}, errors.New("lemmy: no instances configured")
	}
	start := rand.IntN(len(lemmyInstances))
	var lastErr error
	for i := range lemmyInstances {
		inst := lemmyInstances[(start+i)%len(lemmyInstances)]
		resp, err := lemmyQuery(ctx, inst, q)
		if err == nil {
			return resp, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("lemmy: all instances failed")
	}
	return Response{}, lastErr
}

func lemmyQuery(ctx context.Context, instance string, q query.Query) (Response, error) {
	u, _ := url.Parse(instance + "/api/v3/search")
	v := u.Query()
	v.Set("q", q.Filters.Render(q.Terms))
	v.Set("type_", "Posts")
	v.Set("limit", "20")
	v.Set("sort", "TopAll")
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
	return parseLemmy(body)
}

func parseLemmy(body []byte) (Response, error) {
	var payload struct {
		Posts []struct {
			Post struct {
				Name         string `json:"name"`
				URL          string `json:"url"`
				APID         string `json:"ap_id"`
				Body         string `json:"body"`
				Published    string `json:"published"`
				ThumbnailURL string `json:"thumbnail_url"`
			} `json:"post"`
			Community struct {
				Name string `json:"name"`
			} `json:"community"`
			Creator struct {
				Name string `json:"name"`
			} `json:"creator"`
			Counts struct {
				Score    int `json:"score"`
				Upvotes  int `json:"upvotes"`
				Comments int `json:"comments"`
			} `json:"counts"`
		} `json:"posts"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Response{}, fmt.Errorf("lemmy: %w", err)
	}
	var results []Result
	for i, p := range payload.Posts {
		if p.Post.Name == "" {
			continue
		}
		// ap_id is the canonical activity-pub URL for the post on its
		// home instance (so federated dedupe works across instances).
		// Fall back to post.url (the linked-to external article) if no
		// ap_id; that's rare.
		link := p.Post.APID
		if link == "" {
			link = p.Post.URL
		}
		if link == "" {
			continue
		}
		extras := map[string]string{}
		if p.Post.Published != "" {
			extras[ExtraPublishedAt] = p.Post.Published
		}
		if p.Creator.Name != "" {
			extras[ExtraAuthor] = p.Creator.Name
		}
		if p.Post.ThumbnailURL != "" {
			extras[ExtraThumbnailURL] = p.Post.ThumbnailURL
		}
		snippet := p.Post.Body
		if snippet == "" {
			parts := []string{}
			if p.Community.Name != "" {
				parts = append(parts, "c/"+p.Community.Name)
			}
			parts = append(
				parts,
				fmt.Sprintf("%d points · %d comments", p.Counts.Score, p.Counts.Comments),
			)
			snippet = strings.Join(parts, " · ")
		}
		results = append(results, Result{
			Title:    p.Post.Name,
			URL:      link,
			Snippet:  snippet,
			Engine:   "lemmy",
			Position: i + 1,
			Extras:   extras,
		})
	}
	if len(results) == 0 {
		return Response{}, fmt.Errorf("lemmy: no posts parsed")
	}
	return Response{Results: results}, nil
}

func init() {
	Register(lemmyEngine{})
}
