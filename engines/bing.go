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

var bingURL = "https://www.bing.com/search"

type bingEngine struct{}

func (bingEngine) Name() string { return "bing" }
func (bingEngine) Categories() []query.Category {
	return []query.Category{
		query.CategoryGeneral,
		query.CategoryNews,
		query.CategoryImages,
		query.CategoryVideos,
	}
}
func (bingEngine) Languages() LanguageTraits {
	return LanguageTraits{
		All: true,
		Supported: map[string]string{
			"en":    "en-US",
			"en-us": "en-US",
			"en-gb": "en-GB",
			"de":    "de-DE",
			"fr":    "fr-FR",
			"es":    "es-ES",
			"ja":    "ja-JP",
			"zh-cn": "zh-CN",
		},
	}
}
func (bingEngine) Weight() float64 { return 1.0 }

func (e bingEngine) Search(ctx context.Context, q query.Query) ([]Result, error) {
	u, _ := url.Parse(bingURL)
	v := u.Query()
	v.Set("q", q.Terms)
	v.Set("form", "QBLH")
	if q.Page > 1 {
		v.Set("first", fmt.Sprintf("%d", (q.Page-1)*10+1))
	}
	if loc, ok := e.Languages().Native(q.Language); ok {
		v.Set("setlang", loc)
	}
	switch q.SafeSearch {
	case query.SafeOff:
		v.Set("adlt", "off")
	case query.SafeModerate:
		v.Set("adlt", "moderate")
	case query.SafeStrict:
		v.Set("adlt", "strict")
	}
	switch q.TimeRange {
	case query.TimeRangeDay:
		v.Set("filters", `ex1:"ez1"`)
	case query.TimeRangeWeek:
		v.Set("filters", `ex1:"ez2"`)
	case query.TimeRangeMonth:
		v.Set("filters", `ex1:"ez3"`)
	}
	u.RawQuery = v.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	body, err := fetch(req)
	if err != nil {
		return nil, err
	}
	return parseBing(body)
}

func parseBing(body []byte) ([]Result, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	var results []Result
	pos := 0
	doc.Find("li.b_algo").Each(func(_ int, s *goquery.Selection) {
		titleEl := s.Find("h2 a").First()
		title := strings.TrimSpace(titleEl.Text())
		href, _ := titleEl.Attr("href")
		snippet := strings.TrimSpace(s.Find(".b_caption p").First().Text())
		if title == "" || href == "" {
			return
		}
		pos++
		results = append(results, Result{
			Title:    title,
			URL:      href,
			Snippet:  snippet,
			Engine:   "bing",
			Position: pos,
		})
	})
	if len(results) == 0 {
		return nil, fmt.Errorf("bing: no results parsed")
	}
	return results, nil
}

func init() {
	Register(bingEngine{})
}
