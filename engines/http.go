package engines

import (
	"bufio"
	"context"
	"crypto/tls"
	_ "embed"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	utls "github.com/refraction-networking/utls"
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

// utlsDialTLS performs a uTLS handshake against addr using a Chrome
// ClientHello with ALPN h2+http/1.1. Go's crypto/tls produces a handshake
// shape that anti-bot systems (Google's JS wall, DataDome on Qwant) flag
// instantly; mimicking Chrome's exact cipher suite order, extensions,
// GREASE values, and ALPN list defeats the fingerprint check at the
// transport layer so every engine inherits the bypass without per-engine
// wiring.
func utlsDialTLS(ctx context.Context, network, addr string) (net.Conn, error) {
	return utlsDialTLSWithALPN(ctx, network, addr, []string{"h2", "http/1.1"})
}

// httpClient is shared by every engine. uTLS is used for outgoing TLS
// so handshakes look like Chrome instead of Go's crypto/tls defaults;
// a small switching RoundTripper picks http2.Transport when the host's
// ALPN negotiated h2, falling back to a plain http.Transport (still
// uTLS-dialed but speaking HTTP/1.1) for hosts that don't.
var httpClient = &http.Client{
	Transport: &utlsRoundTripper{
		h2: &http2.Transport{
			AllowHTTP:          false,
			DisableCompression: false,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *utlsTLSConfig) (net.Conn, error) {
				return dialUTLSForALPN(ctx, network, addr, "h2")
			},
		},
		h1: &http.Transport{
			MaxIdleConns:        100,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
			DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialUTLSForALPN(ctx, network, addr, "http/1.1")
			},
		},
	},
	CheckRedirect: func(_ *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return http.ErrUseLastResponse
		}
		return nil
	},
}

// utlsTLSConfig is just *tls.Config under a different name. http2's
// DialTLSContext signature wants a *tls.Config but we ignore it: the
// real config lives inside dialUTLSForALPN. The type alias keeps the
// signature happy without dragging crypto/tls into our top-level imports.
type utlsTLSConfig = tls.Config

// utlsRoundTripper picks an inner RoundTripper based on the request's
// host. h2 is preferred; on a confirmed-h1 host (per the h2 cache) it
// drops through to the plain h1 transport. The host cache is populated
// lazily as we observe ALPN outcomes.
type utlsRoundTripper struct {
	h2 *http2.Transport
	h1 *http.Transport

	mu sync.RWMutex
	// h1Hosts records hosts where h2 failed and we should skip it next time.
	h1Hosts map[string]struct{}
}

func (rt *utlsRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	host := req.URL.Host
	rt.mu.RLock()
	_, isH1 := rt.h1Hosts[host]
	rt.mu.RUnlock()
	if isH1 || req.URL.Scheme == "http" {
		return rt.h1.RoundTrip(req)
	}
	resp, err := rt.h2.RoundTrip(req)
	if err == nil {
		return resp, nil
	}
	// Distinguish protocol-fallback failures from real errors. A host
	// that doesn't speak h2 surfaces as either a uTLS handshake that
	// negotiates http/1.1 (caught in dialUTLSForALPN) or http2.Transport
	// rejecting the conn. Cache the result and retry on h1.
	if isProtocolFallback(err) {
		rt.mu.Lock()
		if rt.h1Hosts == nil {
			rt.h1Hosts = map[string]struct{}{}
		}
		rt.h1Hosts[host] = struct{}{}
		rt.mu.Unlock()
		return rt.h1.RoundTrip(req)
	}
	return resp, err
}

func isProtocolFallback(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "http2: unsupported scheme") ||
		strings.Contains(s, "alpn protocol mismatch") ||
		strings.Contains(s, "expected h2 alpn")
}

// dialUTLSForALPN dials with uTLS and only returns a conn if the
// negotiated ALPN matches wantProto. Lets the caller (h2 transport vs
// h1 transport) refuse a conn that doesn't speak its protocol.
func dialUTLSForALPN(ctx context.Context, network, addr, wantProto string) (net.Conn, error) {
	conn, err := utlsDialTLSWithALPN(ctx, network, addr, []string{"h2", "http/1.1"})
	if err != nil {
		return nil, err
	}
	got := conn.ConnectionState().NegotiatedProtocol
	if got == "" {
		// No ALPN at all - treat as http/1.1
		got = "http/1.1"
	}
	if got != wantProto {
		conn.Close()
		return nil, fmt.Errorf("alpn protocol mismatch: got %q, want %q", got, wantProto)
	}
	return conn, nil
}

// utlsDialTLSWithALPN is utlsDialTLS but with a caller-chosen ALPN list.
func utlsDialTLSWithALPN(
	ctx context.Context,
	network, addr string,
	alpn []string,
) (*utls.UConn, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	raw, err := dialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}
	cfg := &utls.Config{ServerName: host, NextProtos: alpn}
	conn := utls.UClient(raw, cfg, utls.HelloChrome_Auto)
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	if err := conn.HandshakeContext(ctx); err != nil {
		raw.Close()
		return nil, fmt.Errorf("utls handshake: %w", err)
	}
	_ = conn.SetDeadline(time.Time{})
	return conn, nil
}

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
