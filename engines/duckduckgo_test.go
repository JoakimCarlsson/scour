package engines

import "testing"

const ddgFixture = `<html><body>
<div class="result">
  <a class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fgo.dev%2F&rut=abc">The Go Programming Language</a>
  <a class="result__snippet">Go is an open source programming language.</a>
</div>
<div class="result">
  <a class="result__a" href="https://golang.org/doc/tutorial/">Tutorial - Go</a>
  <a class="result__snippet">Get started with Go.</a>
</div>
<div class="result">
  <a class="result__a" href=""></a>
</div>
</body></html>`

func TestParseDuckDuckGo(t *testing.T) {
	results, err := parseDuckDuckGo([]byte(ddgFixture))
	if err != nil {
		t.Fatalf("parseDuckDuckGo: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].URL != "https://go.dev/" {
		t.Errorf("redirect not decoded: %q", results[0].URL)
	}
	if results[0].Position != 1 || results[1].Position != 2 {
		t.Errorf("positions wrong: %d, %d", results[0].Position, results[1].Position)
	}
	if results[0].Engine != "duckduckgo" {
		t.Errorf("engine: %q", results[0].Engine)
	}
}

func TestParseDuckDuckGoEmpty(t *testing.T) {
	if _, err := parseDuckDuckGo([]byte("<html></html>")); err == nil {
		t.Fatal("expected error on empty doc")
	}
}
