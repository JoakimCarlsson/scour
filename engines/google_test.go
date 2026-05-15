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

const googleNewLayoutFixture = `<html><body>
<div jscontroller="abc" data-hveid="1">
  <a href="https://example.com/a"><h3>Example A</h3></a>
  <div data-sncf="x">Summary A.</div>
</div>
<div jscontroller="abc" data-hveid="2">
  <a href="https://example.com/b"><h3>Example B</h3></a>
  <div class="s3v9rd">Summary B.</div>
</div>
</body></html>`

func TestParseGoogleNewLayout(t *testing.T) {
	results, err := parseGoogle([]byte(googleNewLayoutFixture))
	if err != nil {
		t.Fatalf("parseGoogle: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].URL != "https://example.com/a" {
		t.Errorf("url: %q", results[0].URL)
	}
}

const googleConsentFixture = `<html><body><form action="https://consent.google.com/save">consent</form></body></html>`

func TestParseGoogleConsent(t *testing.T) {
	_, err := parseGoogle([]byte(googleConsentFixture))
	if err == nil {
		t.Fatal("expected consent error")
	}
}

const googleSorryFixture = `<html><body><meta http-equiv="refresh" content="0; url=/sorry/index?continue=..."></body></html>`

func TestParseGoogleSorry(t *testing.T) {
	if _, err := parseGoogle([]byte(googleSorryFixture)); err == nil {
		t.Fatal("expected sorry/CAPTCHA error")
	}
}

const googleDataVedFixture = `<html><body>
<div>
  <a data-ved="ab1" href="/url?q=https://go.dev/&amp;sa=U&amp;ved=xyz">
    <div style="font-weight:500">The Go Programming Language</div>
  </a>
  <div><div><div class="ilUpNd">Go is an open source language.</div></div></div>
</div>
<div>
  <a data-ved="ab2" href="/url?q=https://golang.org/doc/&amp;sa=U">
    <div style="font-weight:500">Documentation - Go</div>
  </a>
  <div><div><div class="ilUpNd">Get started.</div></div></div>
</div>
<a data-ved="cls1" class="some-ui-link" href="https://google.com/preferences"><div style>Settings</div></a>
</body></html>`

func TestParseGoogleDataVed(t *testing.T) {
	results, err := parseGoogle([]byte(googleDataVedFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].URL != "https://go.dev/" {
		t.Errorf("url: %q", results[0].URL)
	}
	if results[0].Title != "The Go Programming Language" {
		t.Errorf("title: %q", results[0].Title)
	}
}
