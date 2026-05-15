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

var braveURL = "https://search.brave.com/search"

// braveCookie builds Brave's session cookie. safesearch is cookie-driven
// (the URL param is silently ignored); the others are required to look
// like a normal session.
func braveCookie(s query.SafeLevel) string {
	val := "off"
	switch s {
	case query.SafeModerate:
		val = "moderate"
	case query.SafeStrict:
		val = "strict"
	}
	return "safesearch=" + val + "; useLocation=0; summarizer=0"
}

type braveEngine struct{}

func (braveEngine) Name() string { return "brave" }
func (braveEngine) Categories() []query.Category {
	return []query.Category{
		query.CategoryGeneral,
		query.CategoryNews,
	}
}
func (braveEngine) Languages() LanguageTraits {
	return LanguageTraits{
		All: true,
		Supported: map[string]string{
			"en":    "us",
			"en-us": "us",
			"en-gb": "gb",
			"de":    "de",
			"fr":    "fr",
			"es":    "es",
			"ja":    "jp",
			"zh-cn": "cn",
		},
	}
}
func (braveEngine) Weight() float64 { return 1.0 }

func (e braveEngine) Search(ctx context.Context, q query.Query) (Response, error) {
	if q.Category == query.CategoryNews {
		return e.searchNews(ctx, q)
	}
	u, _ := url.Parse(braveURL)
	v := u.Query()
	v.Set("q", q.Filters.Render(q.Terms))
	v.Set("source", "web")
	if q.Page > 1 {
		v.Set("offset", fmt.Sprintf("%d", q.Page-1))
	}
	if loc, ok := e.Languages().Native(q.Language); ok {
		v.Set("country", loc)
	}
	switch q.TimeRange {
	case query.TimeRangeDay:
		v.Set("tf", "pd")
	case query.TimeRangeWeek:
		v.Set("tf", "pw")
	case query.TimeRangeMonth:
		v.Set("tf", "pm")
	case query.TimeRangeYear:
		v.Set("tf", "py")
	}
	u.RawQuery = v.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Response{}, err
	}
	// Brave's safesearch is cookie-driven, not URL. The `safesearch` query
	// param is silently ignored; the cookie is what the backend reads.
	req.Header.Set("Cookie", braveCookie(q.SafeSearch))
	body, err := fetch(req)
	if err != nil {
		return Response{}, err
	}
	results, err := parseBrave(body)
	if err != nil {
		return Response{}, err
	}
	return Response{Results: results, Suggestions: parseBraveSuggestions(body)}, nil
}

func parseBraveSuggestions(body []byte) []string {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil
	}
	var sugs []string
	seen := map[string]struct{}{}
	doc.Find("a.suggestion, a.related-searches a").Each(func(_ int, s *goquery.Selection) {
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

var braveNewsURL = "https://search.brave.com/news"

func (braveEngine) searchNews(ctx context.Context, q query.Query) (Response, error) {
	u, _ := url.Parse(braveNewsURL)
	v := u.Query()
	v.Set("q", q.Filters.Render(q.Terms))
	v.Set("source", "news")
	u.RawQuery = v.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("Cookie", braveCookie(q.SafeSearch))
	body, err := fetch(req)
	if err != nil {
		return Response{}, err
	}
	return parseBraveNews(body)
}

func parseBraveNews(body []byte) (Response, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return Response{}, err
	}
	var results []Result
	pos := 0
	seen := map[string]struct{}{}
	doc.Find("a.news-card, div.snippet[data-type='news'], div[data-type='news']").
		Each(func(_ int, s *goquery.Selection) {
			href, _ := s.Attr("href")
			if href == "" {
				href, _ = s.Find("a[href]").First().Attr("href")
			}
			if href == "" {
				return
			}
			if _, dup := seen[href]; dup {
				return
			}
			title := strings.TrimSpace(
				s.Find(".line-clamp-2, .title, h3, .news-card-title").First().Text(),
			)
			if title == "" {
				title = strings.TrimSpace(s.Find("a").First().Text())
			}
			source := strings.TrimSpace(s.Find(".news-card-site span").First().Text())
			snippet := strings.TrimSpace(
				s.Find(".snippet-description, .description").First().Text(),
			)
			if snippet == "" {
				snippet = source
			}
			published := strings.TrimSpace(
				s.Find(".news-card-metadata span, time, .time, .date").First().Text(),
			)
			if title == "" {
				return
			}
			pos++
			extras := map[string]string{}
			if published != "" {
				extras[ExtraPublishedAt] = published
			}
			if source != "" {
				extras[ExtraAuthor] = source
			}
			seen[href] = struct{}{}
			results = append(results, Result{
				Title:    title,
				URL:      href,
				Snippet:  snippet,
				Engine:   "brave",
				Position: pos,
				Extras:   extras,
			})
		})
	if len(results) == 0 {
		return Response{}, fmt.Errorf("brave: no news results parsed")
	}
	return Response{Results: results}, nil
}

func init() {
	Register(braveEngine{})
}
