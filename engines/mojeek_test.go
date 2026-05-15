package engines

import "testing"

const mojeekFixture = `<html><body>
<ul class="results-standard">
  <li><h2><a href="https://go.dev/">The Go Programming Language</a></h2>
      <p class="s">An open source programming language.</p></li>
  <li><h2><a href="https://github.com/golang/go">Go on GitHub</a></h2>
      <p class="s">Source tree.</p></li>
</ul></body></html>`

func TestParseMojeek(t *testing.T) {
	results, err := parseMojeek([]byte(mojeekFixture))
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

func TestParseMojeekEmpty(t *testing.T) {
	if _, err := parseMojeek([]byte(`<html></html>`)); err == nil {
		t.Fatal("expected error")
	}
}
