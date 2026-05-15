package engines

import "testing"

const qwantFixture = `{"data":{"result":{"items":{"mainline":[
{"type":"web","items":[
  {"title":"Go Programming Language","url":"https://go.dev/","desc":"An open source language."},
  {"title":"Go - GitHub","url":"https://github.com/golang/go","desc":"The Go source tree."}
]}
]}}}}`

func TestParseQwant(t *testing.T) {
	resp, err := parseQwant([]byte(qwantFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("len=%d", len(resp.Results))
	}
	if resp.Results[0].URL != "https://go.dev/" {
		t.Errorf("url=%q", resp.Results[0].URL)
	}
	if resp.Results[1].Position != 2 {
		t.Errorf("pos=%d", resp.Results[1].Position)
	}
}

func TestParseQwantEmpty(t *testing.T) {
	if _, err := parseQwant([]byte(`{"data":{}}`)); err == nil {
		t.Fatal("expected error")
	}
}
