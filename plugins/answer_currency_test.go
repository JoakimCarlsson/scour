package plugins

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JoakimCarlsson/scour/query"
)

func TestAnswerCurrency(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("from") != "USD" || r.URL.Query().Get("to") != "EUR" {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		_, _ = w.Write(
			[]byte(`{"amount":100,"base":"USD","date":"2026-05-15","rates":{"EUR":92.5}}`),
		)
	}))
	defer srv.Close()

	a := AnswerCurrency{Endpoint: srv.URL}
	c := &Context{Query: query.Query{Terms: "100 usd in eur"}}
	if err := a.Apply(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	if c.Answer == nil || c.Answer.Text != "92.5 EUR" {
		t.Fatalf("got %+v", c.Answer)
	}
}

func TestAnswerCurrencyNoMatch(t *testing.T) {
	a := AnswerCurrency{Endpoint: "http://invalid.invalid"}
	c := &Context{Query: query.Query{Terms: "weather in london"}}
	_ = a.Apply(context.Background(), c)
	if c.Answer != nil {
		t.Fatalf("expected nil, got %+v", c.Answer)
	}
}

func TestAnswerCurrencyNetworkFailure(t *testing.T) {
	a := AnswerCurrency{Endpoint: "http://127.0.0.1:1"}
	c := &Context{Query: query.Query{Terms: "100 usd in eur"}}
	if err := a.Apply(context.Background(), c); err != nil {
		t.Fatalf("plugin should swallow network failure: %v", err)
	}
	if c.Answer != nil {
		t.Fatalf("expected nil on failure, got %+v", c.Answer)
	}
}
