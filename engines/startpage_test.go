package engines

import "testing"

const startpageFixture = `<html><body>
<script>
React.createElement(UIStartpage.AppSerpWeb, {"render":{"presenter":{"regions":{"mainline":[
  {"display_type":"web-google","results":[
    {"title":"The Go Programming Language","description":"<b>Go</b> is open source.","clickUrl":"https://go.dev/","url":null,"displayUrl":"https://go.dev/"},
    {"title":"Documentation - Go","description":"Get started.","clickUrl":"https://golang.org/doc/","url":null,"displayUrl":"https://golang.org/doc/"}
  ]},
  {"display_type":"ads","results":[{"title":"Ad","clickUrl":"https://ad.example/"}]}
]}}},"translations":{}});
</script>
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
	if results[0].Title != "The Go Programming Language" {
		t.Errorf("title=%q", results[0].Title)
	}
	if results[0].Snippet != "Go is open source." {
		t.Errorf("snippet=%q", results[0].Snippet)
	}
}

func TestParseStartpageEmpty(t *testing.T) {
	if _, err := parseStartpage([]byte(`<html></html>`)); err == nil {
		t.Fatal("expected error")
	}
}
