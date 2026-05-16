package engines

import "testing"

const nominatimFixture = `[
  {
    "place_id": 12345,
    "osm_type": "relation",
    "osm_id": 7444,
    "lat": "48.8588897",
    "lon": "2.3200410",
    "display_name": "Paris, Île-de-France, France métropolitaine, France",
    "class": "boundary",
    "type": "administrative"
  },
  {
    "place_id": 12346,
    "osm_type": "relation",
    "osm_id": 7445,
    "lat": "33.6617962",
    "lon": "-95.5555130",
    "display_name": "Paris, Lamar County, Texas, United States",
    "class": "boundary",
    "type": "administrative"
  }
]`

func TestParseNominatim(t *testing.T) {
	resp, err := parseNominatim([]byte(nominatimFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("len=%d", len(resp.Results))
	}
	r := resp.Results[0]
	if r.URL != "https://www.openstreetmap.org/relation/7444" {
		t.Errorf("url=%q", r.URL)
	}
	if r.Extras[ExtraLatitude] != "48.8588897" {
		t.Errorf("lat=%q", r.Extras[ExtraLatitude])
	}
	if r.Extras[ExtraLongitude] != "2.3200410" {
		t.Errorf("lon=%q", r.Extras[ExtraLongitude])
	}
}

func TestParseNominatimEmpty(t *testing.T) {
	if _, err := parseNominatim([]byte(`[]`)); err == nil {
		t.Fatal("expected error")
	}
}
