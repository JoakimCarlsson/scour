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

const braveURL = "https://search.brave.com/search"

type braveEngine struct{}

func (braveEngine) Name() string { return "brave" }
func (braveEngine) Categories() []query.Category {
	return []query.Category{
		query.CategoryGeneral,
		query.CategoryNews,
		query.CategoryImages,
		query.CategoryVideos,
	}
}
func (braveEngine) Languages() []string { return []string{"*"} }
func (braveEngine) Weight() float64     { return 1.0 }

func (e braveEngine) Search(ctx context.Context, q query.Query) ([]Result, error) {
	u, _ := url.Parse(braveURL)
	v := u.Query()
	v.Set("q", q.Terms)
	v.Set("source", "web")
	u.RawQuery = v.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	body, err := fetch(req)
	if err != nil {
		return nil, err
	}
	return parseBrave(body)
}

func parseBrave(body []byte) ([]Result, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	var results []Result
	pos := 0
	doc.Find("div.snippet[data-type='web']").Each(func(_ int, s *goquery.Selection) {
		titleEl := s.Find("a.heading-serpresult").First()
		if titleEl.Length() == 0 {
			titleEl = s.Find("a").First()
		}
		title := strings.TrimSpace(s.Find(".title").First().Text())
		if title == "" {
			title = strings.TrimSpace(titleEl.Text())
		}
		href, _ := titleEl.Attr("href")
		snippet := strings.TrimSpace(s.Find(".snippet-description").First().Text())
		if title == "" || href == "" {
			return
		}
		pos++
		results = append(results, Result{
			Title:    title,
			URL:      href,
			Snippet:  snippet,
			Engine:   "brave",
			Position: pos,
		})
	})
	if len(results) == 0 {
		return nil, fmt.Errorf("brave: no results parsed")
	}
	return results, nil
}

func init() {
	Register(braveEngine{})
}
