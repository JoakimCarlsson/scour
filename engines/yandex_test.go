package engines

import "testing"

const yandexFixture = `<html><body>
<li class="serp-item">
  <h2><a class="OrganicTitle-Link" href="https://go.dev/">Go Programming Language</a></h2>
  <div class="OrganicTextContentSpan">An open source programming language.</div>
</li>
<li class="serp-item">
  <h2><a class="OrganicTitle-Link" href="https://github.com/golang/go">Go on GitHub</a></h2>
  <div class="OrganicTextContentSpan">Source tree.</div>
</li>
</body></html>`

func TestParseYandex(t *testing.T) {
	results, err := parseYandex([]byte(yandexFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("len=%d", len(results))
	}
	if results[0].URL != "https://go.dev/" {
		t.Errorf("url=%q", results[0].URL)
	}
}

func TestParseYandexEmpty(t *testing.T) {
	if _, err := parseYandex([]byte(`<html></html>`)); err == nil {
		t.Fatal("expected error")
	}
}
