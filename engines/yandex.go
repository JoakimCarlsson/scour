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

// yandexURL points at the iframe-friendly site-search frame instead of
// the regular /search/. /search/ is gated by Yandex's full anti-bot
// (verification page + tmgrdfrend.fp.js challenge); /search/site/ is
// meant to be embedded by site owners and is behind a much lighter
// gate, so it serves real HTML to non-browser clients.
var yandexURL = "https://yandex.com/search/site/"

// yandexCookie carries a fake viewport + family-filter timestamp. Yandex
// expects clients to have visited the search page before; this cookie
// asserts a prior session without actually having one.
const yandexCookie = "yp=1716337604.sp.family%3A0#1685406411.szm.1:1920x1080:1920x999"

type yandexEngine struct{}

func (yandexEngine) Name() string                 { return "yandex" }
func (yandexEngine) Categories() []query.Category { return []query.Category{query.CategoryGeneral} }
func (yandexEngine) Languages() LanguageTraits    { return LanguageTraits{All: true} }
func (yandexEngine) Weight() float64              { return 1.0 }

func (e yandexEngine) Search(ctx context.Context, q query.Query) (Response, error) {
	u, _ := url.Parse(yandexURL)
	v := u.Query()
	v.Set("text", q.Terms)
	v.Set("tmpl_version", "releases")
	v.Set("web", "1")
	v.Set("frame", "1")
	v.Set("searchid", "3131712")
	if loc, ok := e.Languages().Native(q.Language); ok {
		v.Set("lang", loc)
	}
	if q.Page > 1 {
		v.Set("p", fmt.Sprintf("%d", q.Page-1))
	}
	u.RawQuery = v.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("Cookie", yandexCookie)
	resp, body, err := fetchWithHeaders(req)
	if err != nil {
		return Response{}, err
	}
	if resp != nil && resp.Header.Get("x-yandex-captcha") == "captcha" {
		return Response{}, fmt.Errorf("yandex: captcha challenge")
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
	doc.Find("li.b-serp-item, li.serp-item, div.serp-item").
		Each(func(_ int, s *goquery.Selection) {
			linkEl := s.Find("a.b-serp-item__title-link, a.OrganicTitle-Link, a.Link.organic__url, h2 a").
				First()
			href, _ := linkEl.Attr("href")
			title := strings.TrimSpace(
				s.Find("h3.b-serp-item__title, h2, .OrganicTitle").First().Text(),
			)
			if title == "" {
				title = strings.TrimSpace(linkEl.Text())
			}
			snippet := strings.TrimSpace(
				s.Find(".b-serp-item__text, .OrganicTextContentSpan, .organic__text").
					First().
					Text(),
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
