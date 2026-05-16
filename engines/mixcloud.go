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

var mixcloudURL = "https://api.mixcloud.com/search/"

type mixcloudEngine struct{}

func (mixcloudEngine) Name() string                 { return "mixcloud" }
func (mixcloudEngine) Categories() []query.Category { return []query.Category{query.CategoryMusic} }
func (mixcloudEngine) Languages() LanguageTraits    { return LanguageTraits{All: true} }
func (mixcloudEngine) Weight() float64              { return 1.0 }

func (e mixcloudEngine) Search(ctx context.Context, q query.Query) (Response, error) {
	u, _ := url.Parse(mixcloudURL)
	v := u.Query()
	v.Set("q", q.Filters.Render(q.Terms))
	v.Set("type", "cloudcast")
	v.Set("limit", "20")
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
	return parseMixcloud(body)
}

func parseMixcloud(body []byte) (Response, error) {
	var payload struct {
		Data []struct {
			Name        string `json:"name"`
			URL         string `json:"url"`
			Slug        string `json:"slug"`
			AudioLength int    `json:"audio_length"`
			CreatedTime string `json:"created_time"`
			Pictures    struct {
				Large      string `json:"large"`
				Medium     string `json:"medium"`
				Thumbnail  string `json:"thumbnail"`
				ExtraLarge string `json:"extra_large"`
			} `json:"pictures"`
			User struct {
				Name string `json:"name"`
				Key  string `json:"key"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Response{}, fmt.Errorf("mixcloud: %w", err)
	}
	var results []Result
	for i, c := range payload.Data {
		if c.Name == "" || c.URL == "" {
			continue
		}
		extras := map[string]string{}
		if c.AudioLength > 0 {
			extras[ExtraDuration] = strconv.Itoa(c.AudioLength)
		}
		if c.CreatedTime != "" {
			extras[ExtraPublishedAt] = c.CreatedTime
		}
		if c.User.Name != "" {
			extras[ExtraAuthor] = c.User.Name
		}
		thumb := c.Pictures.Large
		if thumb == "" {
			thumb = c.Pictures.ExtraLarge
		}
		if thumb == "" {
			thumb = c.Pictures.Medium
		}
		if thumb == "" {
			thumb = c.Pictures.Thumbnail
		}
		if thumb != "" {
			extras[ExtraThumbnailURL] = thumb
		}
		results = append(results, Result{
			Title:    c.Name,
			URL:      c.URL,
			Engine:   "mixcloud",
			Position: i + 1,
			Extras:   extras,
		})
	}
	if len(results) == 0 {
		return Response{}, fmt.Errorf("mixcloud: no results parsed")
	}
	return Response{Results: results}, nil
}

func init() {
	Register(mixcloudEngine{})
}
