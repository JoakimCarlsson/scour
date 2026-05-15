package engines

import (
	"bytes"
	"context"
	"encoding/json"
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

func (e bingEngine) Search(ctx context.Context, q query.Query) (Response, error) {
	switch q.Category {
	case query.CategoryImages:
		return e.searchImages(ctx, q)
	case query.CategoryNews:
		return e.searchNews(ctx, q)
	}
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
		return Response{}, err
	}
	body, err := fetch(req)
	if err != nil {
		return Response{}, err
	}
	results, err := parseBing(body)
	if err != nil {
		return Response{}, err
	}
	return Response{Results: results, Suggestions: parseBingSuggestions(body)}, nil
}

func parseBingSuggestions(body []byte) []string {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil
	}
	var sugs []string
	seen := map[string]struct{}{}
	doc.Find("div.sp_requery a, a.sc_qs, a.sa_qs, a.sa_tup").
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

var bingImagesURL = "https://www.bing.com/images/search"
var bingNewsURL = "https://www.bing.com/news/search"

func (bingEngine) searchImages(ctx context.Context, q query.Query) (Response, error) {
	u, _ := url.Parse(bingImagesURL)
	v := u.Query()
	v.Set("q", q.Terms)
	v.Set("form", "HDRSC2")
	u.RawQuery = v.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Response{}, err
	}
	body, err := fetch(req)
	if err != nil {
		return Response{}, err
	}
	return parseBingImages(body)
}

func parseBingImages(body []byte) (Response, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return Response{}, err
	}
	var results []Result
	pos := 0
	doc.Find("a.iusc, div.imgpt > a").Each(func(_ int, s *goquery.Selection) {
		meta, _ := s.Attr("m")
		if meta == "" {
			return
		}
		var payload struct {
			MURL string `json:"murl"`
			TURL string `json:"turl"`
			T    string `json:"t"`
			DESC string `json:"desc"`
		}
		if err := json.Unmarshal([]byte(meta), &payload); err != nil {
			return
		}
		if payload.MURL == "" || payload.T == "" {
			return
		}
		pos++
		extras := map[string]string{}
		if payload.TURL != "" {
			extras[ExtraThumbnailURL] = payload.TURL
		}
		results = append(results, Result{
			Title:    payload.T,
			URL:      payload.MURL,
			Snippet:  payload.DESC,
			Engine:   "bing",
			Position: pos,
			Extras:   extras,
		})
	})
	if len(results) == 0 {
		return Response{}, fmt.Errorf("bing: no image results parsed")
	}
	return Response{Results: results}, nil
}

func (bingEngine) searchNews(ctx context.Context, q query.Query) (Response, error) {
	u, _ := url.Parse(bingNewsURL)
	v := u.Query()
	v.Set("q", q.Terms)
	v.Set("form", "QBNH")
	u.RawQuery = v.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Response{}, err
	}
	body, err := fetch(req)
	if err != nil {
		return Response{}, err
	}
	return parseBingNews(body)
}

func parseBingNews(body []byte) (Response, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return Response{}, err
	}
	var results []Result
	pos := 0
	doc.Find("div.news-card, div.t_s, div.newsitem").Each(func(_ int, s *goquery.Selection) {
		linkEl := s.Find("a.title, a[data-href], h2 a, a").First()
		title := strings.TrimSpace(linkEl.Text())
		href, _ := linkEl.Attr("href")
		if href == "" {
			href, _ = linkEl.Attr("data-href")
		}
		snippet := strings.TrimSpace(s.Find(".snippet, .description").First().Text())
		if title == "" || href == "" {
			return
		}
		published := strings.TrimSpace(s.Find(".source span, .t_s_sn").First().Text())
		extras := map[string]string{}
		if published != "" {
			extras[ExtraPublishedAt] = published
		}
		pos++
		results = append(results, Result{
			Title:    title,
			URL:      href,
			Snippet:  snippet,
			Engine:   "bing",
			Position: pos,
			Extras:   extras,
		})
	})
	if len(results) == 0 {
		return Response{}, fmt.Errorf("bing: no news results parsed")
	}
	return Response{Results: results}, nil
}

func init() {
	Register(bingEngine{})
}
