package engines

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/JoakimCarlsson/scour/query"
)

var mojeekURL = "https://www.mojeek.com/search"

// mojeekSinceDate returns a YYYYMMDD anchor N units in the past for a
// given TimeRange ("" if none). Mojeek's UI filters results to docs
// crawled after this date.
func mojeekSinceDate(tr query.TimeRange) string {
	now := time.Now()
	var t time.Time
	switch tr {
	case query.TimeRangeDay:
		t = now.AddDate(0, 0, -1)
	case query.TimeRangeWeek:
		t = now.AddDate(0, 0, -7)
	case query.TimeRangeMonth:
		t = now.AddDate(0, -1, 0)
	case query.TimeRangeYear:
		t = now.AddDate(-1, 0, 0)
	default:
		return ""
	}
	return t.Format("20060102")
}

type mojeekEngine struct{}

func (mojeekEngine) Name() string                 { return "mojeek" }
func (mojeekEngine) Categories() []query.Category { return []query.Category{query.CategoryGeneral} }
func (mojeekEngine) Languages() LanguageTraits    { return LanguageTraits{All: true} }
func (mojeekEngine) Weight() float64              { return 1.0 }

func (e mojeekEngine) Search(ctx context.Context, q query.Query) (Response, error) {
	u, _ := url.Parse(mojeekURL)
	v := u.Query()
	v.Set("q", q.Filters.Render(q.Terms))
	if q.Page > 1 {
		v.Set("s", fmt.Sprintf("%d", (q.Page-1)*10+1))
	}
	// Mojeek: safe=1 = on, omit = off. No moderate level upstream.
	if q.SafeSearch == query.SafeStrict || q.SafeSearch == query.SafeModerate {
		v.Set("safe", "1")
	}
	// Mojeek timerange: `since=YYYYMMDD` filters to docs after that date.
	if since := mojeekSinceDate(q.TimeRange); since != "" {
		v.Set("since", since)
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
	results, err := parseMojeek(body)
	if err != nil {
		return Response{}, err
	}
	return Response{Results: results}, nil
}

func parseMojeek(body []byte) ([]Result, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	var results []Result
	pos := 0
	doc.Find("ul.results-standard > li, a.ob").Each(func(_ int, s *goquery.Selection) {
		titleEl := s.Find("h2 a, a.ob").First()
		title := strings.TrimSpace(titleEl.Text())
		href, _ := titleEl.Attr("href")
		snippet := strings.TrimSpace(s.Find("p.s").First().Text())
		if title == "" || href == "" {
			return
		}
		pos++
		results = append(results, Result{
			Title:    title,
			URL:      href,
			Snippet:  snippet,
			Engine:   "mojeek",
			Position: pos,
		})
	})
	if len(results) == 0 {
		return nil, fmt.Errorf("mojeek: no results parsed")
	}
	return results, nil
}

func init() {
	Register(mojeekEngine{})
}
