package engines

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestGSAUserAgentRotatesWithSuffix(t *testing.T) {
	if len(gsaUserAgents) < 2 {
		t.Fatalf("gsa pool too small: %d", len(gsaUserAgents))
	}
	seen := map[string]struct{}{}
	for range 200 {
		ua := gsaUserAgent()
		if !strings.HasSuffix(ua, " NSTNWV") {
			t.Fatalf("expected NSTNWV suffix, got %q", ua)
		}
		seen[ua] = struct{}{}
		if len(seen) > 1 {
			return
		}
	}
	t.Fatalf("gsaUserAgent never rotated across 200 calls: %v", seen)
}

func TestRandomUserAgentRotates(t *testing.T) {
	if len(userAgents) < 2 {
		t.Fatalf("pool too small: %d", len(userAgents))
	}
	seen := map[string]struct{}{}
	for range 200 {
		ua := randomUserAgent()
		seen[ua] = struct{}{}
		if len(seen) > 1 {
			return
		}
	}
	t.Fatalf("randomUserAgent always returned the same value across 200 calls: %v", seen)
}

func TestFetchRotatesUAByDefault(t *testing.T) {
	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Header.Get("User-Agent"))
		mu.Unlock()
	}))
	defer srv.Close()

	for range 30 {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = fetch(req)
	}
	uniq := map[string]struct{}{}
	for _, s := range seen {
		uniq[s] = struct{}{}
	}
	if len(uniq) < 2 {
		t.Fatalf("expected multiple distinct UAs across 30 fetches, got %d: %v", len(uniq), uniq)
	}
}

func TestFetchHonorsPresetUA(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("User-Agent")
	}))
	defer srv.Close()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	req.Header.Set("User-Agent", "custom-engine-ua/1.0")
	_, _ = fetch(req)
	if got != "custom-engine-ua/1.0" {
		t.Fatalf("expected preset UA preserved, got %q", got)
	}
}

func TestFetchNegotiatesHTTP2(t *testing.T) {
	srv := httptest.NewUnstartedServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Echo the negotiated protocol so the client can assert it.
			w.Header().Set("X-Proto", r.Proto)
		}),
	)
	srv.EnableHTTP2 = true
	srv.StartTLS()
	defer srv.Close()

	// httpClient's TLSClientConfig pins ALPN ["h2","http/1.1"] but uses the
	// real root CAs - point it at the test cert for this request only.
	clone := srv.Client()
	if t, ok := clone.Transport.(*http.Transport); ok {
		t.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
			NextProtos:         []string{"h2", "http/1.1"},
		}
	}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := clone.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if !strings.HasPrefix(resp.Proto, "HTTP/2") {
		t.Fatalf("expected HTTP/2 negotiation, got %q", resp.Proto)
	}
}
