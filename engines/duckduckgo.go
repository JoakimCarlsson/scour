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

var duckduckgoURL = "https://html.duckduckgo.com/html/"

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
func (duckduckgoEngine) Languages() LanguageTraits {
	return LanguageTraits{
		All: true,
		Supported: map[string]string{
			"en":    "us-en",
			"en-us": "us-en",
			"en-gb": "uk-en",
			"de":    "de-de",
			"fr":    "fr-fr",
			"es":    "es-es",
			"ja":    "jp-jp",
			"zh-cn": "cn-zh",
		},
	}
}
func (duckduckgoEngine) Weight() float64 { return 1.0 }

func (e duckduckgoEngine) Search(ctx context.Context, q query.Query) (Response, error) {
	form := url.Values{}
	form.Set("q", q.Terms)
	if q.Page > 1 {
		form.Set("s", fmt.Sprintf("%d", (q.Page-1)*30))
		form.Set("dc", fmt.Sprintf("%d", (q.Page-1)*30+1))
	}
	kl := "us-en"
	if loc, ok := e.Languages().Native(q.Language); ok {
		kl = loc
	}
	form.Set("kl", kl)
	switch q.SafeSearch {
	case query.SafeOff:
		form.Set("kp", "-2")
	case query.SafeModerate:
		form.Set("kp", "-1")
	case query.SafeStrict:
		form.Set("kp", "1")
	}
	switch q.TimeRange {
	case query.TimeRangeDay:
		form.Set("df", "d")
	case query.TimeRangeWeek:
		form.Set("df", "w")
	case query.TimeRangeMonth:
		form.Set("df", "m")
	case query.TimeRangeYear:
		form.Set("df", "y")
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		duckduckgoURL,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	body, err := fetch(req)
	if err != nil {
		return Response{}, err
	}
	results, err := parseDuckDuckGo(body)
	if err != nil {
		return Response{}, err
	}
	return Response{Results: results, Suggestions: parseDuckDuckGoSuggestions(body)}, nil
}

func parseDuckDuckGoSuggestions(body []byte) []string {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil
	}
	var sugs []string
	seen := map[string]struct{}{}
	doc.Find("div.zci__suggestion, div.zci__suggestions a, a.js-spelling-suggestion-link").
		Each(func(_ int, s *goquery.Selection) {
			t := strings.TrimSpace(s.Text())
			if t == "" {
				return
			}
			k := strings.ToLower(t)
			if _, dup := seen[k]; dup {
				return
			}
			seen[k] = struct{}{}
			sugs = append(sugs, t)
		})
	return sugs
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
