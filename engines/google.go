package engines

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/JoakimCarlsson/scour/query"
)

// googleURL points at the /m/search endpoint: Google's mobile no-JS SERP.
// The desktop endpoint now serves a JS-required wall, but /m/search still
// renders organic results as plain HTML when paired with a mobile UA and
// a CONSENT=YES+ cookie. Live tests verified on 2026-05-15.
var googleURL = "https://www.google.com/m/search"

// googleConsentCookie short-circuits Google's consent interstitial. The
// 'YES+cb' form is what SearXNG uses (engines/google.py) and what an
// already-accepted real session settles on.
const googleConsentCookie = "CONSENT=YES+cb"

// googleMobileUA is a real Android Chrome UA. Google's mobile SERP only
// serves no-JS HTML when the UA looks like a mobile browser.
const googleMobileUA = "Mozilla/5.0 (Linux; Android 11; SM-S901U) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/99.0.4844.88 Mobile Safari/537.36"

type googleEngine struct{}

func (googleEngine) Name() string { return "google" }
func (googleEngine) Categories() []query.Category {
	return []query.Category{query.CategoryGeneral, query.CategoryNews}
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
	if q.Category == query.CategoryNews {
		return e.searchNews(ctx, q)
	}
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
	req.Header.Set("User-Agent", googleMobileUA)
	req.Header.Set("Accept", "*/*")
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

// unwrapGoogleRedirect pulls the real destination out of Google's
// /url?q=<dest>&... redirect wrapper. Returns the input unchanged if it
// is not a wrapper.
func unwrapGoogleRedirect(href string) string {
	if !strings.HasPrefix(href, "/url?") && !strings.HasPrefix(href, "/url%3F") {
		return href
	}
	u, err := url.Parse(href)
	if err != nil {
		return href
	}
	if q := u.Query().Get("q"); q != "" {
		return q
	}
	return href
}

func isGoogleConsent(body []byte) bool {
	return bytes.Contains(body, []byte(`id="consent-bump"`)) ||
		bytes.Contains(body, []byte(`action="https://consent.google.com/save"`)) ||
		bytes.Contains(body, []byte(`consent.google.com`))
}

// googleSelectors lists result-container selectors in preference order;
// Google rotates these so we try several and stop at the first that yields
// a non-empty result set. Gx5Zad is the mobile /m/search container we
// target first; the desktop ones remain as a fallback in case the mobile
// SERP layout shifts.
var googleSelectors = []string{
	"div.Gx5Zad",
	"div.g",
	"div.MjjYud",
	"div[jscontroller][data-hveid]",
	"div.tF2Cxc",
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
			real := unwrapGoogleRedirect(h)
			if strings.HasPrefix(real, "http") &&
				!strings.HasPrefix(real, "https://webcache.googleusercontent.com") &&
				!strings.Contains(real, "google.com/search") &&
				!strings.Contains(real, "support.google.com") {
				href = real
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

var googleNewsURL = "https://news.google.com/rss/search"

func (googleEngine) searchNews(ctx context.Context, q query.Query) (Response, error) {
	u, _ := url.Parse(googleNewsURL)
	v := u.Query()
	v.Set("q", q.Terms)
	v.Set("hl", "en-US")
	v.Set("gl", "US")
	v.Set("ceid", "US:en")
	u.RawQuery = v.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Response{}, err
	}
	body, err := fetch(req)
	if err != nil {
		return Response{}, err
	}
	return parseGoogleNewsRSS(body)
}

type googleRSSFeed struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct {
		Items []struct {
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			PubDate     string `xml:"pubDate"`
			Description string `xml:"description"`
			Source      struct {
				Name string `xml:",chardata"`
				URL  string `xml:"url,attr"`
			} `xml:"source"`
		} `xml:"item"`
	} `xml:"channel"`
}

func parseGoogleNewsRSS(body []byte) (Response, error) {
	var feed googleRSSFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return Response{}, fmt.Errorf("google news rss: %w", err)
	}
	var results []Result
	for i, it := range feed.Channel.Items {
		if it.Title == "" || it.Link == "" {
			continue
		}
		extras := map[string]string{}
		if it.PubDate != "" {
			extras[ExtraPublishedAt] = it.PubDate
		}
		if it.Source.Name != "" {
			extras[ExtraAuthor] = it.Source.Name
		}
		results = append(results, Result{
			Title:    it.Title,
			URL:      it.Link,
			Snippet:  stripHTML(it.Description),
			Engine:   "google",
			Position: i + 1,
			Extras:   extras,
		})
	}
	if len(results) == 0 {
		return Response{}, fmt.Errorf("google news rss: no items")
	}
	return Response{Results: results}, nil
}

func init() {
	Register(googleEngine{})
}
