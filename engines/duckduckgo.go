package engines

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/JoakimCarlsson/scour/query"
)

var duckduckgoURL = "https://html.duckduckgo.com/html/"

type duckduckgoEngine struct{}

func (duckduckgoEngine) Name() string { return "duckduckgo" }
func (duckduckgoEngine) Categories() []query.Category {
	return []query.Category{query.CategoryGeneral, query.CategoryImages}
}
func (duckduckgoEngine) Languages() LanguageTraits {
	return LanguageTraits{
		All: true,
		Supported: map[string]string{
			"en":    "us-en",
			"en-us": "us-en",
			"en-gb": "uk-en",
			"de":    "de-de",
			"fr":    "fr-fr",
			"es":    "es-es",
			"ja":    "jp-jp",
			"zh-cn": "cn-zh",
		},
	}
}
func (duckduckgoEngine) Weight() float64 { return 1.0 }

func (e duckduckgoEngine) Search(ctx context.Context, q query.Query) (Response, error) {
	if q.Category == query.CategoryImages {
		return e.searchImages(ctx, q)
	}
	form := url.Values{}
	form.Set("q", q.Terms)
	if q.Page > 1 {
		form.Set("s", fmt.Sprintf("%d", (q.Page-1)*30))
		form.Set("dc", fmt.Sprintf("%d", (q.Page-1)*30+1))
	}
	kl := "us-en"
	if loc, ok := e.Languages().Native(q.Language); ok {
		kl = loc
	}
	form.Set("kl", kl)
	switch q.SafeSearch {
	case query.SafeOff:
		form.Set("kp", "-2")
	case query.SafeModerate:
		form.Set("kp", "-1")
	case query.SafeStrict:
		form.Set("kp", "1")
	}
	switch q.TimeRange {
	case query.TimeRangeDay:
		form.Set("df", "d")
	case query.TimeRangeWeek:
		form.Set("df", "w")
	case query.TimeRangeMonth:
		form.Set("df", "m")
	case query.TimeRangeYear:
		form.Set("df", "y")
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		duckduckgoURL,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Referer", "https://html.duckduckgo.com/")
	body, err := fetch(req)
	if err != nil {
		return Response{}, err
	}
	results, err := parseDuckDuckGo(body)
	if err != nil {
		return Response{}, err
	}
	return Response{Results: results, Suggestions: parseDuckDuckGoSuggestions(body)}, nil
}

func parseDuckDuckGoSuggestions(body []byte) []string {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil
	}
	var sugs []string
	seen := map[string]struct{}{}
	doc.Find("div.msg--spelling a, div.zci__suggestion, div.zci__suggestions a, a.js-spelling-suggestion-link").
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

func parseDuckDuckGo(body []byte) ([]Result, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	var results []Result
	pos := 0
	doc.Find("div.result").Each(func(_ int, s *goquery.Selection) {
		titleEl := s.Find("a.result__a").First()
		title := strings.TrimSpace(titleEl.Text())
		href, _ := titleEl.Attr("href")
		link := cleanDDGRedirect(href)
		snippet := strings.TrimSpace(s.Find(".result__snippet").Text())
		if title == "" || link == "" {
			return
		}
		pos++
		results = append(results, Result{
			Title:    title,
			URL:      link,
			Snippet:  snippet,
			Engine:   "duckduckgo",
			Position: pos,
		})
	})
	if len(results) == 0 {
		return nil, fmt.Errorf("duckduckgo: no results parsed")
	}
	return results, nil
}

func cleanDDGRedirect(raw string) string {
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "//duckduckgo.com/l/?") || strings.HasPrefix(raw, "/l/?") {
		u, err := url.Parse(raw)
		if err == nil {
			if real := u.Query().Get("uddg"); real != "" {
				if dec, err := url.QueryUnescape(real); err == nil {
					return dec
				}
			}
		}
	}
	return raw
}

var duckduckgoImagesURL = "https://duckduckgo.com/i.js"

func (duckduckgoEngine) searchImages(ctx context.Context, q query.Query) (Response, error) {
	// DDG's i.js needs a vqd token from a prior hit on the search page. We
	// keep this simple: hit duckduckgo.com first to get the token, then call
	// i.js. If anything fails we return an error and the fan-out drops us.
	tokenReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"https://duckduckgo.com/?"+url.Values{"q": {q.Terms}, "iax": {"images"}, "ia": {"images"}}.
			Encode(),
		nil,
	)
	if err != nil {
		return Response{}, err
	}
	body, err := fetch(tokenReq)
	if err != nil {
		return Response{}, err
	}
	m := vqdRe.FindSubmatch(body)
	if m == nil {
		return Response{}, fmt.Errorf("duckduckgo: no vqd token")
	}
	vqd := string(m[1])
	u, _ := url.Parse(duckduckgoImagesURL)
	v := u.Query()
	v.Set("l", "us-en")
	v.Set("o", "json")
	v.Set("q", q.Terms)
	v.Set("vqd", vqd)
	v.Set("f", ",,,")
	v.Set("p", "1")
	u.RawQuery = v.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("Accept", "application/json")
	imgBody, err := fetch(req)
	if err != nil {
		return Response{}, err
	}
	return parseDuckDuckGoImages(imgBody)
}

var vqdRe = regexp.MustCompile(`vqd=["']?([0-9-]+)["']?`)

func parseDuckDuckGoImages(body []byte) (Response, error) {
	var payload struct {
		Results []struct {
			Title     string `json:"title"`
			Image     string `json:"image"`
			Thumbnail string `json:"thumbnail"`
			URL       string `json:"url"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Response{}, fmt.Errorf("duckduckgo: image json: %w", err)
	}
	if len(payload.Results) == 0 {
		return Response{}, fmt.Errorf("duckduckgo: no image results")
	}
	out := make([]Result, 0, len(payload.Results))
	for i, r := range payload.Results {
		if r.URL == "" || r.Title == "" {
			continue
		}
		extras := map[string]string{}
		if r.Thumbnail != "" {
			extras[ExtraThumbnailURL] = r.Thumbnail
		} else if r.Image != "" {
			extras[ExtraThumbnailURL] = r.Image
		}
		if r.Width > 0 {
			extras[ExtraThumbnailWidth] = strconv.Itoa(r.Width)
		}
		if r.Height > 0 {
			extras[ExtraThumbnailHeight] = strconv.Itoa(r.Height)
		}
		out = append(out, Result{
			Title:    r.Title,
			URL:      r.URL,
			Engine:   "duckduckgo",
			Position: i + 1,
			Extras:   extras,
		})
	}
	if len(out) == 0 {
		return Response{}, fmt.Errorf("duckduckgo: no image results")
	}
	return Response{Results: out}, nil
}

func init() {
	Register(duckduckgoEngine{})
}
