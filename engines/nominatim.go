package engines

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/JoakimCarlsson/scour/query"
)

var nominatimURL = "https://nominatim.openstreetmap.org/search"

// nominatimUA is the deliberate UA Nominatim's etiquette policy requires:
// identify the application, include a contact / homepage. Random pool
// UAs would look like scrapers and get blocked.
const nominatimUA = "scour/0.x (https://github.com/JoakimCarlsson/scour)"

type nominatimEngine struct{}

func (nominatimEngine) Name() string                 { return "nominatim" }
func (nominatimEngine) Categories() []query.Category { return []query.Category{query.CategoryMap} }
func (nominatimEngine) Languages() LanguageTraits    { return LanguageTraits{All: true} }
func (nominatimEngine) Weight() float64              { return 1.0 }

func (e nominatimEngine) Search(ctx context.Context, q query.Query) (Response, error) {
	u, _ := url.Parse(nominatimURL)
	v := u.Query()
	v.Set("q", q.Filters.Render(q.Terms))
	v.Set("format", "json")
	v.Set("limit", "20")
	u.RawQuery = v.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("User-Agent", nominatimUA)
	req.Header.Set("Accept", "application/json")
	body, err := fetch(req)
	if err != nil {
		return Response{}, err
	}
	return parseNominatim(body)
}

func parseNominatim(body []byte) (Response, error) {
	var places []struct {
		DisplayName string `json:"display_name"`
		Lat         string `json:"lat"`
		Lon         string `json:"lon"`
		PlaceID     int64  `json:"place_id"`
		OSMType     string `json:"osm_type"`
		OSMID       int64  `json:"osm_id"`
		Class       string `json:"class"`
		Type        string `json:"type"`
	}
	if err := json.Unmarshal(body, &places); err != nil {
		return Response{}, fmt.Errorf("nominatim: %w", err)
	}
	var results []Result
	for i, p := range places {
		if p.DisplayName == "" {
			continue
		}
		// Link to OpenStreetMap browse page for this object.
		link := fmt.Sprintf("https://www.openstreetmap.org/%s/%d", p.OSMType, p.OSMID)
		extras := map[string]string{}
		if p.Lat != "" {
			extras[ExtraLatitude] = p.Lat
		}
		if p.Lon != "" {
			extras[ExtraLongitude] = p.Lon
		}
		snippet := p.Class
		if p.Type != "" {
			snippet = p.Class + " · " + p.Type
		}
		results = append(results, Result{
			Title:    p.DisplayName,
			URL:      link,
			Snippet:  snippet,
			Engine:   "nominatim",
			Position: i + 1,
			Extras:   extras,
		})
	}
	if len(results) == 0 {
		return Response{}, fmt.Errorf("nominatim: no places parsed")
	}
	return Response{Results: results}, nil
}

func init() {
	Register(nominatimEngine{})
}
