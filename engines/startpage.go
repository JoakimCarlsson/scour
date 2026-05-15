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

var startpageURL = "https://www.startpage.com/sp/search"

type startpageEngine struct{}

func (startpageEngine) Name() string { return "startpage" }
func (startpageEngine) Categories() []query.Category {
	return []query.Category{query.CategoryGeneral}
}
func (startpageEngine) Languages() LanguageTraits {
	return LanguageTraits{
		All: true,
		Supported: map[string]string{
			"en": "english",
			"de": "deutsch",
			"fr": "francais",
			"es": "espanol",
		},
	}
}
func (startpageEngine) Weight() float64 { return 1.0 }

func (e startpageEngine) Search(ctx context.Context, q query.Query) (Response, error) {
	u, _ := url.Parse(startpageURL)
	v := u.Query()
	v.Set("query", q.Terms)
	v.Set("cat", "web")
	if loc, ok := e.Languages().Native(q.Language); ok {
		v.Set("language", loc)
	}
	switch q.SafeSearch {
	case query.SafeOff:
		v.Set("qadf", "none")
	case query.SafeModerate:
		v.Set("qadf", "medium")
	case query.SafeStrict:
		v.Set("qadf", "heavy")
	}
	if q.Page > 1 {
		v.Set("startat", fmt.Sprintf("%d", (q.Page-1)*10))
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
	results, err := parseStartpage(body)
	if err != nil {
		return Response{}, err
	}
	return Response{Results: results}, nil
}

func parseStartpage(body []byte) ([]Result, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	var results []Result
	pos := 0
	doc.Find("div.w-gl__result, section.w-gl__result").Each(func(_ int, s *goquery.Selection) {
		linkEl := s.Find("a.w-gl__result-title, a.result-link").First()
		if linkEl.Length() == 0 {
			linkEl = s.Find("a[href]").First()
		}
		href, _ := linkEl.Attr("href")
		title := strings.TrimSpace(s.Find("h3, .w-gl__result-title").First().Text())
		if title == "" {
			title = strings.TrimSpace(linkEl.Text())
		}
		snippet := strings.TrimSpace(
			s.Find("p.w-gl__description, .w-gl__description").First().Text(),
		)
		if title == "" || href == "" {
			return
		}
		pos++
		results = append(results, Result{
			Title:    title,
			URL:      href,
			Snippet:  snippet,
			Engine:   "startpage",
			Position: pos,
		})
	})
	if len(results) == 0 {
		return nil, fmt.Errorf("startpage: no results parsed")
	}
	return results, nil
}

func init() {
	Register(startpageEngine{})
}
