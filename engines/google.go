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

var googleURL = "https://www.google.com/search"

type googleEngine struct{}

func (googleEngine) Name() string { return "google" }
func (googleEngine) Categories() []query.Category {
	return []query.Category{
		query.CategoryGeneral,
		query.CategoryNews,
		query.CategoryImages,
		query.CategoryVideos,
		query.CategoryMap,
	}
}
func (googleEngine) Languages() LanguageTraits {
	return LanguageTraits{
		All: true,
		Supported: map[string]string{
			"en":    "en",
			"en-us": "en",
			"en-gb": "en",
			"de":    "de",
			"fr":    "fr",
			"es":    "es",
			"ja":    "ja",
			"zh-cn": "zh-CN",
		},
	}
}
func (googleEngine) Weight() float64 { return 1.0 }

func (e googleEngine) Search(ctx context.Context, q query.Query) (Response, error) {
	u, _ := url.Parse(googleURL)
	v := u.Query()
	v.Set("q", q.Terms)
	if q.Page > 1 {
		v.Set("start", fmt.Sprintf("%d", (q.Page-1)*10))
	}
	hl := "en"
	if loc, ok := e.Languages().Native(q.Language); ok {
		hl = loc
	}
	v.Set("hl", hl)
	v.Set("num", "20")
	switch q.SafeSearch {
	case query.SafeOff:
		v.Set("safe", "off")
	case query.SafeModerate, query.SafeStrict:
		v.Set("safe", "active")
	}
	switch q.TimeRange {
	case query.TimeRangeDay:
		v.Set("tbs", "qdr:d")
	case query.TimeRangeWeek:
		v.Set("tbs", "qdr:w")
	case query.TimeRangeMonth:
		v.Set("tbs", "qdr:m")
	case query.TimeRangeYear:
		v.Set("tbs", "qdr:y")
	}
	u.RawQuery = v.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Response{}, err
	}
	body, err := fetch(req)
	if err != nil {
		return Response{}, err
	}
	results, err := parseGoogle(body)
	if err != nil {
		return Response{}, err
	}
	return Response{Results: results}, nil
}

func parseGoogle(body []byte) ([]Result, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	var results []Result
	pos := 0
	doc.Find("div.g, div.MjjYud").Each(func(_ int, s *goquery.Selection) {
		linkEl := s.Find("a[href]").First()
		href, _ := linkEl.Attr("href")
		if !strings.HasPrefix(href, "http") {
			return
		}
		title := strings.TrimSpace(s.Find("h3").First().Text())
		snippet := strings.TrimSpace(s.Find("div[data-sncf], div.VwiC3b").First().Text())
		if title == "" {
			return
		}
		pos++
		results = append(results, Result{
			Title:    title,
			URL:      href,
			Snippet:  snippet,
			Engine:   "google",
			Position: pos,
		})
	})
	if len(results) == 0 {
		return nil, fmt.Errorf("google: no results parsed")
	}
	return results, nil
}

func init() {
	Register(googleEngine{})
}
