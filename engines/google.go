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

// googleConsentCookie skips Google's EU/global consent interstitial. The
// SOCS value rotates; this one was captured from a Firefox session on
// 2026-05-15. Replace if Google starts rejecting it.
const googleConsentCookie = "CONSENT=PENDING+987; SOCS=CAESHAgBEhJnd3NfMjAyNTAxMjMtMF9SQzMaAmVuIAEaBgiAjOq8Bg"

type googleEngine struct{}

func (googleEngine) Name() string { return "google" }
func (googleEngine) Categories() []query.Category {
	return []query.Category{query.CategoryGeneral}
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
	v.Set("gl", "us")
	v.Set("pws", "0")
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
	req.Header.Set("Cookie", googleConsentCookie)
	body, err := fetch(req)
	if err != nil {
		return Response{}, err
	}
	if isGoogleConsent(body) {
		return Response{}, fmt.Errorf("google: served consent page")
	}
	results, err := parseGoogle(body)
	if err != nil {
		return Response{}, err
	}
	return Response{Results: results}, nil
}

func isGoogleConsent(body []byte) bool {
	return bytes.Contains(body, []byte(`id="consent-bump"`)) ||
		bytes.Contains(body, []byte(`action="https://consent.google.com/save"`)) ||
		bytes.Contains(body, []byte(`consent.google.com`))
}

// googleSelectors lists result-container selectors in preference order;
// Google rotates these so we try several and stop at the first that yields
// a non-empty result set.
var googleSelectors = []string{
	"div.g",
	"div.MjjYud",
	"div[jscontroller][data-hveid]",
	"div.tF2Cxc",
	"div.Gx5Zad",
}

func parseGoogle(body []byte) ([]Result, error) {
	if isGoogleConsent(body) {
		return nil, fmt.Errorf("google: served consent page")
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	for _, sel := range googleSelectors {
		results := extractGoogleResults(doc, sel)
		if len(results) > 0 {
			return results, nil
		}
	}
	return nil, fmt.Errorf("google: no results parsed")
}

func extractGoogleResults(doc *goquery.Document, sel string) []Result {
	var results []Result
	pos := 0
	seen := map[string]struct{}{}
	doc.Find(sel).Each(func(_ int, s *goquery.Selection) {
		var href string
		s.Find("a[href]").EachWithBreak(func(_ int, a *goquery.Selection) bool {
			h, _ := a.Attr("href")
			if strings.HasPrefix(h, "http") &&
				!strings.HasPrefix(h, "https://webcache.googleusercontent.com") {
				href = h
				return false
			}
			return true
		})
		if href == "" {
			return
		}
		if _, dup := seen[href]; dup {
			return
		}
		title := strings.TrimSpace(s.Find("h3").First().Text())
		if title == "" {
			return
		}
		snippet := strings.TrimSpace(
			s.Find("div[data-sncf], div.VwiC3b, div.s3v9rd, span.aCOpRe").First().Text(),
		)
		seen[href] = struct{}{}
		pos++
		results = append(results, Result{
			Title:    title,
			URL:      href,
			Snippet:  snippet,
			Engine:   "google",
			Position: pos,
		})
	})
	return results
}

func init() {
	Register(googleEngine{})
}
