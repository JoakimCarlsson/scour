package plugins

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/JoakimCarlsson/scour/query"
)

type InfoboxWikipedia struct {
	Endpoint   string
	HTTPClient *http.Client
}

func (InfoboxWikipedia) Name() string { return "infobox_wikipedia" }

func (p InfoboxWikipedia) Apply(ctx context.Context, c *Context) error {
	if c.Query.Category != query.CategoryGeneral {
		return nil
	}
	terms := strings.TrimSpace(c.Query.Terms)
	if terms == "" {
		return nil
	}
	if wordCount(terms) >= 5 {
		return nil
	}
	endpoint := p.Endpoint
	if endpoint == "" {
		endpoint = "https://en.wikipedia.org/api/rest_v1/page/summary"
	}
	client := p.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	target := endpoint + "/" + url.PathEscape(strings.ReplaceAll(terms, " ", "_"))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil
	}
	var payload struct {
		Title       string `json:"title"`
		Extract     string `json:"extract"`
		ContentURLs struct {
			Desktop struct {
				Page string `json:"page"`
			} `json:"desktop"`
		} `json:"content_urls"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	if payload.Title == "" || payload.Extract == "" {
		return nil
	}
	c.Infobox = &Infobox{
		Title:   payload.Title,
		Summary: payload.Extract,
		URL:     payload.ContentURLs.Desktop.Page,
	}
	return nil
}

func wordCount(s string) int {
	return len(strings.Fields(s))
}
