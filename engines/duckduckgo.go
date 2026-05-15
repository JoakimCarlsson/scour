package engines

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/JoakimCarlsson/scour/query"
)

const duckduckgoURL = "https://html.duckduckgo.com/html/"

type duckduckgoEngine struct{}

func (duckduckgoEngine) Name() string { return "duckduckgo" }
func (duckduckgoEngine) Categories() []query.Category {
	return []query.Category{
		query.CategoryGeneral,
		query.CategoryNews,
		query.CategoryImages,
		query.CategoryVideos,
	}
}
func (duckduckgoEngine) Languages() []string { return []string{"*"} }
func (duckduckgoEngine) Weight() float64     { return 1.0 }

func (e duckduckgoEngine) Search(ctx context.Context, q query.Query) ([]Result, error) {
	form := url.Values{}
	form.Set("q", q.Terms)
	form.Set("kl", "us-en")
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		duckduckgoURL,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	body, err := fetch(req)
	if err != nil {
		return nil, err
	}
	return parseDuckDuckGo(body)
}

func parseDuckDuckGo(body []byte) ([]Result, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	var results []Result
	pos := 0
	doc.Find("div.result").Each(func(_ int, s *goquery.Selection) {
		titleEl := s.Find("a.result__a").First()
		title := strings.TrimSpace(titleEl.Text())
		href, _ := titleEl.Attr("href")
		link := cleanDDGRedirect(href)
		snippet := strings.TrimSpace(s.Find(".result__snippet").Text())
		if title == "" || link == "" {
			return
		}
		pos++
		results = append(results, Result{
			Title:    title,
			URL:      link,
			Snippet:  snippet,
			Engine:   "duckduckgo",
			Position: pos,
		})
	})
	if len(results) == 0 {
		return nil, fmt.Errorf("duckduckgo: no results parsed")
	}
	return results, nil
}

func cleanDDGRedirect(raw string) string {
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "//duckduckgo.com/l/?") || strings.HasPrefix(raw, "/l/?") {
		u, err := url.Parse(raw)
		if err == nil {
			if real := u.Query().Get("uddg"); real != "" {
				if dec, err := url.QueryUnescape(real); err == nil {
					return dec
				}
			}
		}
	}
	return raw
}

func init() {
	Register(duckduckgoEngine{})
}
