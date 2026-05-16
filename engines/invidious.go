package engines

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/JoakimCarlsson/scour/query"
)

// invidiousInstances is a small hard-coded rotation. Invidious instances
// come and go; an admin should be able to override this from config in
// the future. For v1 we just round-robin and the suspension layer
// retires dead ones.
var invidiousInstances = []string{
	"https://invidious.materialio.us",
	"https://invidious.f5.si",
}

type invidiousEngine struct{}

func (invidiousEngine) Name() string { return "invidious" }

func (invidiousEngine) Categories() []query.Category { return []query.Category{query.CategoryVideos} }
func (invidiousEngine) Languages() LanguageTraits    { return LanguageTraits{All: true} }
func (invidiousEngine) Weight() float64              { return 1.0 }

func (e invidiousEngine) Search(ctx context.Context, q query.Query) (Response, error) {
	if len(invidiousInstances) == 0 {
		return Response{}, errors.New("invidious: no instances configured")
	}
	// Try instances starting from a random offset; first non-error wins.
	// Suspension would cool the whole engine if all instances fail.
	start := rand.IntN(len(invidiousInstances))
	var lastErr error
	for i := range invidiousInstances {
		inst := invidiousInstances[(start+i)%len(invidiousInstances)]
		resp, err := invidiousQuery(ctx, inst, q)
		if err == nil {
			return resp, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("invidious: all instances failed")
	}
	return Response{}, lastErr
}

func invidiousQuery(ctx context.Context, instance string, q query.Query) (Response, error) {
	u, _ := url.Parse(instance + "/api/v1/search")
	v := u.Query()
	v.Set("q", q.Filters.Render(q.Terms))
	v.Set("type", "video")
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
	return parseInvidious(body)
}

func parseInvidious(body []byte) (Response, error) {
	var items []struct {
		Title           string `json:"title"`
		VideoID         string `json:"videoId"`
		Author          string `json:"author"`
		LengthSeconds   int    `json:"lengthSeconds"`
		Published       int64  `json:"published"`
		Description     string `json:"description"`
		VideoThumbnails []struct {
			URL    string `json:"url"`
			Width  int    `json:"width"`
			Height int    `json:"height"`
		} `json:"videoThumbnails"`
	}
	if err := json.Unmarshal(body, &items); err != nil {
		return Response{}, fmt.Errorf("invidious: %w", err)
	}
	var results []Result
	for i, v := range items {
		if v.Title == "" || v.VideoID == "" {
			continue
		}
		// Use canonical youtube.com watch URL so the result merges with
		// other engines that link to the same video.
		link := "https://www.youtube.com/watch?v=" + v.VideoID
		extras := map[string]string{}
		if v.LengthSeconds > 0 {
			extras[ExtraDuration] = strconv.Itoa(v.LengthSeconds)
		}
		if v.Published > 0 {
			extras[ExtraPublishedAt] = time.Unix(v.Published, 0).UTC().Format(time.RFC3339)
		}
		if v.Author != "" {
			extras[ExtraAuthor] = v.Author
		}
		// Pick the largest thumbnail.
		var thumb string
		var thumbArea int
		for _, t := range v.VideoThumbnails {
			a := t.Width * t.Height
			if a > thumbArea {
				thumbArea = a
				thumb = t.URL
			}
		}
		if thumb != "" {
			extras[ExtraThumbnailURL] = thumb
		}
		results = append(results, Result{
			Title:    v.Title,
			URL:      link,
			Snippet:  v.Description,
			Engine:   "invidious",
			Position: i + 1,
			Extras:   extras,
		})
	}
	if len(results) == 0 {
		return Response{}, fmt.Errorf("invidious: no results parsed")
	}
	return Response{Results: results}, nil
}

func init() {
	Register(invidiousEngine{})
}
