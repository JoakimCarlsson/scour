package engines

import "testing"

const bingFixture = `<html><body>
<ol id="b_results">
  <li class="b_algo">
    <h2><a href="https://go.dev/">The Go Programming Language</a></h2>
    <div class="b_caption"><p>Go is open source.</p></div>
  </li>
  <li class="b_algo">
    <h2><a href="https://golang.org/doc/">Documentation - Go</a></h2>
    <div class="b_caption"><p>Get started.</p></div>
  </li>
</ol>
</body></html>`

func TestParseBing(t *testing.T) {
	results, err := parseBing([]byte(bingFixture))
	if err != nil {
		t.Fatalf("parseBing: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].URL != "https://go.dev/" || results[0].Title != "The Go Programming Language" {
		t.Errorf("first result wrong: %+v", results[0])
	}
	if results[1].Position != 2 {
		t.Errorf("position: %d", results[1].Position)
	}
}

func TestParseBingEmpty(t *testing.T) {
	if _, err := parseBing([]byte("<html></html>")); err == nil {
		t.Fatal("expected error")
	}
}
