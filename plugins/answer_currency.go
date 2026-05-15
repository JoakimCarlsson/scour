package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type AnswerCurrency struct {
	// Endpoint is optional; defaults to the Frankfurter API.
	Endpoint string
	// Client is optional; defaults to a short-timeout http.Client.
	Client *http.Client
}

func (AnswerCurrency) Name() string { return "answer_currency" }

var currencyRe = regexp.MustCompile(
	`^(-?\d+(?:\.\d+)?)\s*([a-z]{3})\s+(?:in|to)\s+([a-z]{3})$`,
)

var currencyCodes = map[string]struct{}{}

func init() {
	for _, c := range []string{
		"USD", "EUR", "GBP", "JPY", "CHF", "CAD", "AUD", "NZD", "SEK", "NOK",
		"DKK", "CNY", "HKD", "SGD", "INR", "BRL", "MXN", "ZAR", "PLN", "TRY",
		"KRW", "RUB", "IDR", "THB", "PHP", "MYR", "CZK", "HUF", "ILS", "AED",
	} {
		currencyCodes[c] = struct{}{}
	}
}

func (a AnswerCurrency) Apply(ctx context.Context, c *Context) error {
	terms := strings.ToLower(strings.TrimSpace(c.Query.Terms))
	m := currencyRe.FindStringSubmatch(terms)
	if m == nil {
		return nil
	}
	amt, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return nil
	}
	from := strings.ToUpper(m[2])
	to := strings.ToUpper(m[3])
	if _, ok := currencyCodes[from]; !ok {
		return nil
	}
	if _, ok := currencyCodes[to]; !ok {
		return nil
	}
	endpoint := a.Endpoint
	if endpoint == "" {
		endpoint = "https://api.frankfurter.app/latest"
	}
	client := a.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil
	}
	q := u.Query()
	q.Set("from", from)
	q.Set("to", to)
	q.Set("amount", strconv.FormatFloat(amt, 'f', -1, 64))
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return nil
	}
	var payload struct {
		Rates map[string]float64 `json:"rates"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	v, ok := payload.Rates[to]
	if !ok {
		return nil
	}
	c.Answer = &Answer{
		Text:   fmt.Sprintf("%g %s", v, to),
		Source: "answer_currency",
	}
	return nil
}
