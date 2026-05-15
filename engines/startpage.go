package engines

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/JoakimCarlsson/scour/query"
)

var startpageURL = "https://www.startpage.com/sp/search"
var startpageHomeURL = "https://www.startpage.com/"

type startpageEngine struct{}

func (startpageEngine) Name() string { return "startpage" }
func (startpageEngine) Categories() []query.Category {
	return []query.Category{query.CategoryGeneral}
}
func (startpageEngine) Languages() LanguageTraits {
	return LanguageTraits{
		All: true,
		Supported: map[string]string{
			"en": "english",
			"de": "deutsch",
			"fr": "francais",
			"es": "espanol",
		},
	}
}
func (startpageEngine) Weight() float64 { return 1.0 }

var (
	startpageSCMu      sync.Mutex
	startpageSCValue   string
	startpageSCFetched time.Time
)

const startpageSCTTL = 30 * time.Minute

var startpageSCRe = regexp.MustCompile(`name="sc"[^>]*value="([^"]+)"`)

func startpageSC(ctx context.Context) (string, error) {
	startpageSCMu.Lock()
	defer startpageSCMu.Unlock()
	if startpageSCValue != "" && time.Since(startpageSCFetched) < startpageSCTTL {
		return startpageSCValue, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, startpageHomeURL, nil)
	if err != nil {
		return "", err
	}
	body, err := fetch(req)
	if err != nil {
		return "", err
	}
	m := startpageSCRe.FindSubmatch(body)
	if m == nil {
		return "", fmt.Errorf("startpage: no sc token in homepage")
	}
	startpageSCValue = string(m[1])
	startpageSCFetched = time.Now()
	return startpageSCValue, nil
}

// startpagePreferenceCookie packs Startpage's preferences cookie. The
// cookie format is keyEEEvalue joined by N1N. Critically:
// disable_family_filter takes "1" for OFF and "0" for ON (inverted),
// and the cookie controls the SERP family filter that the URL params
// don't expose.
func startpagePreferenceCookie(s query.SafeLevel, lang string) string {
	filter := "0"
	if s == query.SafeOff {
		filter = "1"
	}
	entries := []string{
		"date_time=world",
		"disable_family_filter=" + filter,
		"disable_open_in_new_window=0",
		"enable_post_method=1",
		"enable_proxy_safety_suggest=1",
		"enable_stay_control=1",
		"instant_answers=1",
		"lang_homepage=s/device/" + lang + "/",
		"num_of_results=10",
		"suggestions=1",
		"wt_unit=celsius",
		"language=" + lang,
		"language_ui=" + lang,
	}
	for i, e := range entries {
		entries[i] = strings.Replace(e, "=", "EEE", 1)
	}
	return "preferences=" + strings.Join(entries, "N1N")
}

func (e startpageEngine) Search(ctx context.Context, q query.Query) (Response, error) {
	sc, err := startpageSC(ctx)
	if err != nil {
		return Response{}, err
	}
	lang := "english"
	if loc, ok := e.Languages().Native(q.Language); ok {
		lang = loc
	}
	form := url.Values{}
	form.Set("query", q.Filters.Render(q.Terms))
	form.Set("cat", "web")
	form.Set("t", "device")
	form.Set("sc", sc)
	form.Set("language", lang)
	form.Set("lui", lang)
	form.Set("abp", "1")
	form.Set("abd", "1")
	form.Set("abe", "1")
	if q.Page > 1 {
		form.Set("page", fmt.Sprintf("%d", q.Page))
		form.Set("segment", "startpage.udog")
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		startpageURL,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://www.startpage.com")
	req.Header.Set("Referer", "https://www.startpage.com/")
	req.Header.Set("Cookie", startpagePreferenceCookie(q.SafeSearch, lang))
	body, err := fetch(req)
	if err != nil {
		return Response{}, err
	}
	results, err := parseStartpage(body)
	if err != nil {
		return Response{}, err
	}
	return Response{Results: results}, nil
}

// parseStartpage extracts the JSON blob embedded in
// React.createElement(UIStartpage.AppSerpWeb, {...}) and reads web results
// from render.presenter.regions.mainline[display_type=web-google].
func parseStartpage(body []byte) ([]Result, error) {
	start := bytesIndex(body, []byte("React.createElement(UIStartpage.AppSerpWeb,"))
	if start < 0 {
		return nil, fmt.Errorf("startpage: no AppSerp marker")
	}
	open := bytesIndex(body[start:], []byte("{"))
	if open < 0 {
		return nil, fmt.Errorf("startpage: no JSON open")
	}
	open += start
	depth := 0
	end := -1
	for i := open; i < len(body); i++ {
		switch body[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				end = i + 1
			}
		}
		if end > 0 {
			break
		}
	}
	if end < 0 {
		return nil, fmt.Errorf("startpage: unbalanced JSON")
	}
	var payload struct {
		Render struct {
			Presenter struct {
				Regions struct {
					Mainline []struct {
						DisplayType string `json:"display_type"`
						Results     []struct {
							Title       string `json:"title"`
							Description string `json:"description"`
							ClickURL    string `json:"clickUrl"`
							URL         string `json:"url"`
							DisplayURL  string `json:"displayUrl"`
						} `json:"results"`
					} `json:"mainline"`
				} `json:"regions"`
			} `json:"presenter"`
		} `json:"render"`
	}
	if err := json.Unmarshal(body[open:end], &payload); err != nil {
		return nil, fmt.Errorf("startpage: %w", err)
	}
	var results []Result
	pos := 0
	for _, group := range payload.Render.Presenter.Regions.Mainline {
		if group.DisplayType != "web-google" {
			continue
		}
		for _, r := range group.Results {
			href := r.ClickURL
			if href == "" {
				href = r.URL
			}
			if href == "" {
				href = r.DisplayURL
			}
			if href == "" || r.Title == "" {
				continue
			}
			pos++
			results = append(results, Result{
				Title:    stripHTML(r.Title),
				URL:      href,
				Snippet:  stripHTML(r.Description),
				Engine:   "startpage",
				Position: pos,
			})
		}
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("startpage: no web-google results")
	}
	return results, nil
}

func bytesIndex(haystack, needle []byte) int {
	return strings.Index(string(haystack), string(needle))
}

var htmlTagRe = regexp.MustCompile(`<[^>]+>`)

func stripHTML(s string) string {
	return strings.TrimSpace(htmlTagRe.ReplaceAllString(s, ""))
}

func init() {
	Register(startpageEngine{})
}
