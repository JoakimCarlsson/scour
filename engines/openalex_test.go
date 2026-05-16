package engines

import "testing"

const openalexFixture = `{
  "meta": {"count": 2},
  "results": [
    {
      "title": "Attention Is All You Need",
      "doi": "https://doi.org/10.5555/3295222.3295349",
      "publication_date": "2017-12-04",
      "abstract": "We propose a new simple network architecture.",
      "authorships": [
        {"author": {"display_name": "Ashish Vaswani"}},
        {"author": {"display_name": "Noam Shazeer"}}
      ],
      "open_access": {"oa_url": "https://arxiv.org/pdf/1706.03762"},
      "primary_location": {"landing_page_url": "https://doi.org/10.5555/3295222.3295349"}
    },
    {
      "title": "BERT: Pre-training of Deep Bidirectional Transformers",
      "doi": "https://doi.org/10.18653/v1/n19-1423",
      "publication_date": "2019-06-02",
      "authorships": [
        {"author": {"display_name": "Jacob Devlin"}}
      ],
      "primary_location": {"landing_page_url": "https://aclanthology.org/N19-1423"}
    }
  ]
}`

func TestParseOpenAlex(t *testing.T) {
	resp, err := parseOpenAlex([]byte(openalexFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("len=%d", len(resp.Results))
	}
	if resp.Results[0].URL != "https://arxiv.org/pdf/1706.03762" {
		t.Errorf("url=%q (should prefer oa_url)", resp.Results[0].URL)
	}
	if resp.Results[0].Extras[ExtraAuthor] != "Ashish Vaswani, Noam Shazeer" {
		t.Errorf("author=%q", resp.Results[0].Extras[ExtraAuthor])
	}
	if resp.Results[1].URL != "https://aclanthology.org/N19-1423" {
		t.Errorf("fallback to landing_page_url broken: %q", resp.Results[1].URL)
	}
}

func TestParseOpenAlexEmpty(t *testing.T) {
	if _, err := parseOpenAlex([]byte(`{"results":[]}`)); err == nil {
		t.Fatal("expected error")
	}
}
