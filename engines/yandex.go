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

var yandexURL = "https://yandex.com/search/"

type yandexEngine struct{}

func (yandexEngine) Name() string                 { return "yandex" }
func (yandexEngine) Categories() []query.Category { return []query.Category{query.CategoryGeneral} }
func (yandexEngine) Languages() LanguageTraits    { return LanguageTraits{All: true} }
func (yandexEngine) Weight() float64              { return 1.0 }

func (e yandexEngine) Search(ctx context.Context, q query.Query) (Response, error) {
	u, _ := url.Parse(yandexURL)
	v := u.Query()
	v.Set("text", q.Terms)
	if q.Page > 1 {
		v.Set("p", fmt.Sprintf("%d", q.Page-1))
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
	results, err := parseYandex(body)
	if err != nil {
		return Response{}, err
	}
	return Response{Results: results}, nil
}

func parseYandex(body []byte) ([]Result, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	var results []Result
	pos := 0
	doc.Find("li.serp-item, div.serp-item").Each(func(_ int, s *goquery.Selection) {
		linkEl := s.Find("a.OrganicTitle-Link, a.Link.organic__url, h2 a").First()
		href, _ := linkEl.Attr("href")
		title := strings.TrimSpace(s.Find("h2, .OrganicTitle").First().Text())
		if title == "" {
			title = strings.TrimSpace(linkEl.Text())
		}
		snippet := strings.TrimSpace(
			s.Find(".OrganicTextContentSpan, .organic__text").First().Text(),
		)
		if title == "" || href == "" {
			return
		}
		pos++
		results = append(results, Result{
			Title:    title,
			URL:      href,
			Snippet:  snippet,
			Engine:   "yandex",
			Position: pos,
		})
	})
	if len(results) == 0 {
		return nil, fmt.Errorf("yandex: no results parsed")
	}
	return results, nil
}

func init() {
	Register(yandexEngine{})
}
