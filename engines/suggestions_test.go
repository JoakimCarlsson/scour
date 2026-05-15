package engines

import (
	"reflect"
	"testing"
)

func TestParseBingSuggestions(t *testing.T) {
	body := []byte(
		`<html><body><a class="sa_qs">what is golang</a><a class="sa_qs">golang tutorial</a></body></html>`,
	)
	got := parseBingSuggestions(body)
	want := []string{"what is golang", "golang tutorial"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestParseBraveSuggestions(t *testing.T) {
	body := []byte(
		`<html><body><a class="suggestion">what is rust</a><a class="suggestion">rust tutorial</a></body></html>`,
	)
	got := parseBraveSuggestions(body)
	want := []string{"what is rust", "rust tutorial"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestParseDuckDuckGoSuggestions(t *testing.T) {
	body := []byte(
		`<html><body><a class="js-spelling-suggestion-link">what is golang</a></body></html>`,
	)
	got := parseDuckDuckGoSuggestions(body)
	want := []string{"what is golang"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestParseSuggestionsEmpty(t *testing.T) {
	if got := parseBingSuggestions([]byte("<html></html>")); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}
