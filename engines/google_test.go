package engines

import "testing"

const googleFixture = `<html><body>
<div class="g">
  <a href="https://go.dev/"><h3>The Go Programming Language</h3></a>
  <div class="VwiC3b">Go is open source.</div>
</div>
<div class="g">
  <a href="https://golang.org/doc/"><h3>Documentation - Go</h3></a>
  <div class="VwiC3b">Get started.</div>
</div>
</body></html>`

func TestParseGoogle(t *testing.T) {
	results, err := parseGoogle([]byte(googleFixture))
	if err != nil {
		t.Fatalf("parseGoogle: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].URL != "https://go.dev/" {
		t.Errorf("url: %q", results[0].URL)
	}
}

func TestParseGoogleEmpty(t *testing.T) {
	if _, err := parseGoogle([]byte("<html></html>")); err == nil {
		t.Fatal("expected error")
	}
}
