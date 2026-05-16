package engines

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/JoakimCarlsson/scour/query"
)

var photonURL = "https://photon.komoot.io/api/"

type photonEngine struct{}

func (photonEngine) Name() string                 { return "photon" }
func (photonEngine) Categories() []query.Category { return []query.Category{query.CategoryMap} }
func (photonEngine) Languages() LanguageTraits {
	return LanguageTraits{
		All: true,
		Supported: map[string]string{
			"en": "en",
			"de": "de",
			"fr": "fr",
			"it": "it",
		},
	}
}
func (photonEngine) Weight() float64 { return 1.0 }

func (e photonEngine) Search(ctx context.Context, q query.Query) (Response, error) {
	u, _ := url.Parse(photonURL)
	v := u.Query()
	v.Set("q", q.Filters.Render(q.Terms))
	v.Set("limit", "20")
	if loc, ok := e.Languages().Native(q.Language); ok {
		v.Set("lang", loc)
	}
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
	return parsePhoton(body)
}

// osmTypeLong maps Photon's single-letter OSM type to the long form
// openstreetmap.org expects in its browse URLs.
var osmTypeLong = map[string]string{
	"N": "node",
	"W": "way",
	"R": "relation",
}

func parsePhoton(body []byte) (Response, error) {
	var payload struct {
		Features []struct {
			Geometry struct {
				Coordinates []float64 `json:"coordinates"`
			} `json:"geometry"`
			Properties struct {
				Name    string `json:"name"`
				Country string `json:"country"`
				City    string `json:"city"`
				State   string `json:"state"`
				Type    string `json:"type"`
				OSMType string `json:"osm_type"`
				OSMID   int64  `json:"osm_id"`
			} `json:"properties"`
		} `json:"features"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Response{}, fmt.Errorf("photon: %w", err)
	}
	var results []Result
	for i, f := range payload.Features {
		p := f.Properties
		if p.Name == "" {
			continue
		}
		// Build a display name. Photon doesn't pre-concat like Nominatim.
		var parts []string
		parts = append(parts, p.Name)
		if p.City != "" && p.City != p.Name {
			parts = append(parts, p.City)
		}
		if p.State != "" {
			parts = append(parts, p.State)
		}
		if p.Country != "" {
			parts = append(parts, p.Country)
		}
		title := strings.Join(parts, ", ")
		osmType, ok := osmTypeLong[p.OSMType]
		if !ok {
			osmType = "node"
		}
		link := fmt.Sprintf("https://www.openstreetmap.org/%s/%d", osmType, p.OSMID)
		extras := map[string]string{}
		if len(f.Geometry.Coordinates) >= 2 {
			extras[ExtraLongitude] = strconv.FormatFloat(f.Geometry.Coordinates[0], 'f', -1, 64)
			extras[ExtraLatitude] = strconv.FormatFloat(f.Geometry.Coordinates[1], 'f', -1, 64)
		}
		results = append(results, Result{
			Title:    title,
			URL:      link,
			Snippet:  p.Type,
			Engine:   "photon",
			Position: i + 1,
			Extras:   extras,
		})
	}
	if len(results) == 0 {
		return Response{}, fmt.Errorf("photon: no features parsed")
	}
	return Response{Results: results}, nil
}

func init() {
	Register(photonEngine{})
}
