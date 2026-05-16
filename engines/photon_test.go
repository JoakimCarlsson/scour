package engines

import "testing"

const photonFixture = `{
  "features": [
    {
      "geometry": {"coordinates": [2.3483915, 48.8534951]},
      "properties": {
        "name": "Paris",
        "country": "France",
        "state": "Île-de-France",
        "type": "city",
        "osm_type": "R",
        "osm_id": 71525
      }
    },
    {
      "geometry": {"coordinates": [-95.555513, 33.6617962]},
      "properties": {
        "name": "Paris",
        "country": "United States",
        "state": "Texas",
        "city": "Paris",
        "type": "city",
        "osm_type": "R",
        "osm_id": 115357
      }
    }
  ]
}`

func TestParsePhoton(t *testing.T) {
	resp, err := parsePhoton([]byte(photonFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("len=%d", len(resp.Results))
	}
	r := resp.Results[0]
	if r.URL != "https://www.openstreetmap.org/relation/71525" {
		t.Errorf("url=%q", r.URL)
	}
	if r.Extras[ExtraLatitude] != "48.8534951" {
		t.Errorf("lat=%q", r.Extras[ExtraLatitude])
	}
	if r.Extras[ExtraLongitude] != "2.3483915" {
		t.Errorf("lon=%q", r.Extras[ExtraLongitude])
	}
	// State + country appended; city omitted because it equals name.
	if r.Title != "Paris, Île-de-France, France" {
		t.Errorf("title=%q", r.Title)
	}
}

func TestParsePhotonEmpty(t *testing.T) {
	if _, err := parsePhoton([]byte(`{"features":[]}`)); err == nil {
		t.Fatal("expected error")
	}
}
