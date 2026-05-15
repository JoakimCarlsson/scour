package engines

import "testing"

const startpageFixture = `<html><body>
<section class="w-gl__result">
  <a class="w-gl__result-title" href="https://go.dev/"><h3 class="w-gl__result-title">The Go Programming Language</h3></a>
  <p class="w-gl__description">An open source programming language.</p>
</section>
<section class="w-gl__result">
  <a class="w-gl__result-title" href="https://github.com/golang/go"><h3 class="w-gl__result-title">Go on GitHub</h3></a>
  <p class="w-gl__description">Source tree.</p>
</section>
</body></html>`

func TestParseStartpage(t *testing.T) {
	results, err := parseStartpage([]byte(startpageFixture))
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

func TestParseStartpageEmpty(t *testing.T) {
	if _, err := parseStartpage([]byte(`<html></html>`)); err == nil {
		t.Fatal("expected error")
	}
}
