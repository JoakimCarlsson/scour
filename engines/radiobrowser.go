package engines

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/JoakimCarlsson/scour/query"
)

// radioBrowserURL is one of several Radio Browser mirrors. The de1 host
// is a long-standing primary; the official advice is to either query
// all.api.radio-browser.info for SRV records or pick any specific
// mirror. We pin de1 to keep the engine simple.
var radioBrowserURL = "https://de1.api.radio-browser.info/json/stations/search"

type radioBrowserEngine struct{}

func (radioBrowserEngine) Name() string { return "radiobrowser" }

func (radioBrowserEngine) Categories() []query.Category { return []query.Category{query.CategoryMusic} }
func (radioBrowserEngine) Languages() LanguageTraits    { return LanguageTraits{All: true} }
func (radioBrowserEngine) Weight() float64              { return 1.0 }

func (e radioBrowserEngine) Search(ctx context.Context, q query.Query) (Response, error) {
	u, _ := url.Parse(radioBrowserURL)
	v := u.Query()
	v.Set("name", q.Filters.Render(q.Terms))
	v.Set("limit", "20")
	v.Set("hidebroken", "true")
	v.Set("order", "votes")
	v.Set("reverse", "true")
	u.RawQuery = v.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("Accept", "application/json")
	body, err := fetch(req)
	if err != nil {
		return Response{}, err
	}
	return parseRadioBrowser(body)
}

func parseRadioBrowser(body []byte) (Response, error) {
	var stations []struct {
		Name        string `json:"name"`
		URL         string `json:"url"`
		URLResolved string `json:"url_resolved"`
		Homepage    string `json:"homepage"`
		Favicon     string `json:"favicon"`
		Country     string `json:"country"`
		Codec       string `json:"codec"`
		Bitrate     int    `json:"bitrate"`
		Tags        string `json:"tags"`
		Votes       int    `json:"votes"`
	}
	if err := json.Unmarshal(body, &stations); err != nil {
		return Response{}, fmt.Errorf("radiobrowser: %w", err)
	}
	var results []Result
	for i, s := range stations {
		title := strings.TrimSpace(s.Name)
		if title == "" {
			continue
		}
		// Prefer the homepage URL (where a user would go to learn more);
		// fall back to the stream URL if no homepage is set.
		link := s.Homepage
		if link == "" {
			link = s.URLResolved
		}
		if link == "" {
			link = s.URL
		}
		if link == "" {
			continue
		}
		var snippetParts []string
		if s.Country != "" {
			snippetParts = append(snippetParts, s.Country)
		}
		if s.Codec != "" && s.Bitrate > 0 {
			snippetParts = append(snippetParts, fmt.Sprintf("%s @ %d kbps", s.Codec, s.Bitrate))
		}
		if s.Tags != "" {
			snippetParts = append(snippetParts, s.Tags)
		}
		extras := map[string]string{}
		if s.Favicon != "" &&
			(strings.HasPrefix(s.Favicon, "http://") || strings.HasPrefix(s.Favicon, "https://")) {
			extras[ExtraThumbnailURL] = s.Favicon
		}
		results = append(results, Result{
			Title:    title,
			URL:      link,
			Snippet:  strings.Join(snippetParts, " · "),
			Engine:   "radiobrowser",
			Position: i + 1,
			Extras:   extras,
		})
	}
	if len(results) == 0 {
		return Response{}, fmt.Errorf("radiobrowser: no stations parsed")
	}
	return Response{Results: results}, nil
}

func init() {
	Register(radioBrowserEngine{})
}
