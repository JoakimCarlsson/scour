package engines

import "testing"

const braveFixture = `<html><body>
<div class="snippet" data-type="web">
  <a class="heading-serpresult" href="https://go.dev/"><span class="title">The Go Programming Language</span></a>
  <div class="snippet-description">Go is open source.</div>
</div>
<div class="snippet" data-type="web">
  <a class="heading-serpresult" href="https://golang.org/"><span class="title">Golang docs</span></a>
  <div class="snippet-description">Docs.</div>
</div>
</body></html>`

func TestParseBrave(t *testing.T) {
	results, err := parseBrave([]byte(braveFixture))
	if err != nil {
		t.Fatalf("parseBrave: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].URL != "https://go.dev/" {
		t.Errorf("url: %q", results[0].URL)
	}
}

func TestParseBraveEmpty(t *testing.T) {
	if _, err := parseBrave([]byte("<html></html>")); err == nil {
		t.Fatal("expected error")
	}
}
