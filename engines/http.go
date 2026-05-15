package engines

import (
	"bufio"
	"crypto/tls"
	_ "embed"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/http2"
)

//go:embed data/useragents.txt
var userAgentsData string

//go:embed data/gsa_useragents.txt
var gsaUserAgentsData string

var userAgents = loadUserAgents(userAgentsData)
var gsaUserAgents = loadUserAgents(gsaUserAgentsData)

func loadUserAgents(raw string) []string {
	var out []string
	sc := bufio.NewScanner(strings.NewReader(raw))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out
}

// randomUserAgent returns a real-browser UA chosen at random from the
// embedded pool. Returns the empty string if the pool failed to load
// (callers in that case fall back to whatever they explicitly set).
func randomUserAgent() string {
	if len(userAgents) == 0 {
		return ""
	}
	return userAgents[rand.IntN(len(userAgents))]
}

// gsaUserAgent returns a random Android-Chrome UA from the GSA pool with
// the " NSTNWV" suffix that the Google Search App for Android sends.
// Used by the Google engine so each request looks like a fresh GSA client.
func gsaUserAgent() string {
	if len(gsaUserAgents) == 0 {
		return ""
	}
	return gsaUserAgents[rand.IntN(len(gsaUserAgents))] + " NSTNWV"
}

// httpClient is shared by every engine and forces HTTP/2 ALPN when the
// server supports it. Plain http.DefaultTransport also negotiates h2,
// but constructing the Transport explicitly lets us pin TLS / proxy
// settings and gives the h2 wire test something deterministic to assert.
var httpClient = func() *http.Client {
	t := &http.Transport{
		ForceAttemptHTTP2:   true,
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     &tls.Config{NextProtos: []string{"h2", "http/1.1"}},
	}
	// Explicitly register the HTTP/2 transport so ForceAttemptHTTP2 has a
	// matching protocol handler. Without this, a custom Transport
	// negotiates ALPN but falls back to HTTP/1.1 anyway.
	_ = http2.ConfigureTransport(t)
	return &http.Client{
		Transport: t,
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
}()

func fetch(req *http.Request) ([]byte, error) {
	_, body, err := fetchWithHeaders(req)
	return body, err
}

// fetchWithHeaders is fetch() but also returns the response so callers
// that need to inspect response headers (e.g. Yandex's x-yandex-captcha)
// can do so. The response Body has already been read and closed.
func fetchWithHeaders(req *http.Request) (*http.Response, []byte, error) {
	if req.Header.Get("User-Agent") == "" {
		if ua := randomUserAgent(); ua != "" {
			req.Header.Set("User-Agent", ua)
		}
	}
	if req.Header.Get("Accept-Language") == "" {
		req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return resp, nil, err
	}
	if resp.StatusCode >= 400 {
		return resp, body, &httpError{Status: resp.StatusCode}
	}
	return resp, body, nil
}

type httpError struct {
	Status int
}

func (e *httpError) Error() string {
	return http.StatusText(e.Status)
}
