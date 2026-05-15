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

// googleURL is the canonical desktop SERP. The Google Search App UA pool
// gives a class of requests Google's anti-bot treats as first-class.
var googleURL = "https://www.google.com/search"

// googleConsentCookie short-circuits Google's consent interstitial.
const googleConsentCookie = "CONSENT=YES+"

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
	v.Set("filter", "0")
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
	if ua := gsaUserAgent(); ua != "" {
		req.Header.Set("User-Agent", ua)
	}
	req.Header.Set("Accept", "*/*")
	body, err := fetch(req)
	if err != nil {
		return Response{}, err
	}
	if isGoogleConsent(body) {
		return Response{}, fmt.Errorf("google: served consent page")
	}
	if isGoogleSorry(body) {
		return Response{}, fmt.Errorf("google: served sorry/CAPTCHA page")
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

// isGoogleConsent matches the consent interstitial Google still serves to
// some clients even with CONSENT=YES+. Distinct from the sorry/CAPTCHA
// page handled below.
func isGoogleConsent(body []byte) bool {
	return bytes.Contains(body, []byte(`id="consent-bump"`)) ||
		bytes.Contains(body, []byte(`action="https://consent.google.com/save"`))
}

// isGoogleSorry matches the CAPTCHA / sorry page. Anything <2KB containing
// '/sorry/' counts; full pages are matched on host markers.
func isGoogleSorry(body []byte) bool {
	if len(body) < 2000 && bytes.Contains(body, []byte("/sorry/")) {
		return true
	}
	return bytes.Contains(body, []byte("sorry.google.com"))
}

func parseGoogle(body []byte) ([]Result, error) {
	if isGoogleConsent(body) {
		return nil, fmt.Errorf("google: served consent page")
	}
	if isGoogleSorry(body) {
		return nil, fmt.Errorf("google: served sorry/CAPTCHA page")
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	// Primary path: anchors that carry a data-ved attribute and no class
	// are Google's organic result links. Title is the nested div[style]
	// inside the anchor; URL is /url?q=<dest>&sa=U which we unwrap.
	if r := extractGoogleDataVed(doc); len(r) > 0 {
		return r, nil
	}
	// Fallback for older / classic layouts that group results in known
	// container divs.
	for _, sel := range []string{
		"div.g", "div.MjjYud", "div[jscontroller][data-hveid]", "div.tF2Cxc", "div.Gx5Zad",
	} {
		if r := extractGoogleContainer(doc, sel); len(r) > 0 {
			return r, nil
		}
	}
	return nil, fmt.Errorf("google: no results parsed")
}

func extractGoogleDataVed(doc *goquery.Document) []Result {
	var results []Result
	pos := 0
	seen := map[string]struct{}{}
	doc.Find("a[data-ved]").Each(func(_ int, a *goquery.Selection) {
		if cls, has := a.Attr("class"); has && cls != "" {
			return
		}
		href, _ := a.Attr("href")
		real := unwrapGoogleRedirect(href)
		if !strings.HasPrefix(real, "http") ||
			strings.HasPrefix(real, "https://webcache.googleusercontent.com") ||
			strings.Contains(real, "google.com/search") ||
			strings.Contains(real, "support.google.com") ||
			strings.Contains(real, "accounts.google.com") {
			return
		}
		if _, dup := seen[real]; dup {
			return
		}
		title := strings.TrimSpace(a.Find("div[style]").First().Text())
		if title == "" {
			title = strings.TrimSpace(a.Find("h3").First().Text())
		}
		if title == "" {
			return
		}
		snippet := strings.TrimSpace(
			a.Parent().Parent().
				Find("div.ilUpNd, div[data-sncf], div.VwiC3b, div.s3v9rd").
				First().Text(),
		)
		seen[real] = struct{}{}
		pos++
		results = append(results, Result{
			Title:    title,
			URL:      real,
			Snippet:  snippet,
			Engine:   "google",
			Position: pos,
		})
	})
	return results
}

func extractGoogleContainer(doc *goquery.Document, sel string) []Result {
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
