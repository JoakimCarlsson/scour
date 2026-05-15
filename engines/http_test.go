package engines

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
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

func TestChromeClientHintsForUA(t *testing.T) {
	cases := []struct {
		ua            string
		wantOK        bool
		wantPlatform  string
		wantMobile    string
		wantBrandHint string
	}{
		{
			ua:            "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
			wantOK:        true,
			wantPlatform:  `"Windows"`,
			wantMobile:    "?0",
			wantBrandHint: `"Chromium";v="131"`,
		},
		{
			ua:            "Mozilla/5.0 (Linux; Android 14; Pixel 9) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.6778.81 Mobile Safari/537.36",
			wantOK:        true,
			wantPlatform:  `"Android"`,
			wantMobile:    "?1",
			wantBrandHint: `"Chromium";v="131"`,
		},
		{
			ua:            "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36 Edg/130.0.0.0",
			wantOK:        true,
			wantPlatform:  `"macOS"`,
			wantMobile:    "?0",
			wantBrandHint: `"Microsoft Edge";v="130"`,
		},
		{
			ua:     "Mozilla/5.0 (X11; Linux x86_64; rv:130.0) Gecko/20100101 Firefox/130.0",
			wantOK: false,
		},
		{
			ua:     "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Safari/605.1.15",
			wantOK: false,
		},
	}
	for _, c := range cases {
		t.Run(c.ua[:30], func(t *testing.T) {
			chua, mobile, platform, ok := chromeClientHintsForUA(c.ua)
			if ok != c.wantOK {
				t.Fatalf("ok=%v want %v", ok, c.wantOK)
			}
			if !c.wantOK {
				return
			}
			if platform != c.wantPlatform {
				t.Errorf("platform=%q want %q", platform, c.wantPlatform)
			}
			if mobile != c.wantMobile {
				t.Errorf("mobile=%q want %q", mobile, c.wantMobile)
			}
			if !strings.Contains(chua, c.wantBrandHint) {
				t.Errorf("chua=%q missing %q", chua, c.wantBrandHint)
			}
		})
	}
}

func TestFetchSendsClientHintsForChromiumUA(t *testing.T) {
	var ua, chua, plat, mob, fetchSite string
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ua = r.Header.Get("User-Agent")
		chua = r.Header.Get("Sec-CH-UA")
		plat = r.Header.Get("Sec-CH-UA-Platform")
		mob = r.Header.Get("Sec-CH-UA-Mobile")
		fetchSite = r.Header.Get("Sec-Fetch-Site")
	}))
	defer srv.Close()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	req.Header.Set(
		"User-Agent",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	)
	_, _ = fetch(req)
	if !strings.Contains(chua, `"Chromium";v="131"`) {
		t.Errorf("Sec-CH-UA=%q (UA=%q)", chua, ua)
	}
	if plat != `"Windows"` {
		t.Errorf("Sec-CH-UA-Platform=%q", plat)
	}
	if mob != "?0" {
		t.Errorf("Sec-CH-UA-Mobile=%q", mob)
	}
	if fetchSite != "none" {
		t.Errorf("Sec-Fetch-Site=%q", fetchSite)
	}
}

func TestFetchSkipsClientHintsForFirefox(t *testing.T) {
	var chua, fetchSite string
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		chua = r.Header.Get("Sec-CH-UA")
		fetchSite = r.Header.Get("Sec-Fetch-Site")
	}))
	defer srv.Close()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	req.Header.Set(
		"User-Agent",
		"Mozilla/5.0 (X11; Linux x86_64; rv:130.0) Gecko/20100101 Firefox/130.0",
	)
	_, _ = fetch(req)
	if chua != "" {
		t.Errorf("Sec-CH-UA should be empty for Firefox, got %q", chua)
	}
	// Sec-Fetch-* are sent by all modern browsers including Firefox.
	if fetchSite != "none" {
		t.Errorf("Sec-Fetch-Site=%q", fetchSite)
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

// TestClientHelloIsChromeShaped sniffs the first bytes of the outgoing
// TLS handshake and asserts the cipher suite list begins with a GREASE
// value - a Chrome convention not present in Go's crypto/tls. This is a
// crude but decisive check that uTLS is on the dial path: a stock Go
// client would never produce a GREASE-prefixed cipher list.
func TestClientHelloIsChromeShaped(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	type capture struct {
		hello []byte
		err   error
	}
	ch := make(chan capture, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			ch <- capture{err: err}
			return
		}
		defer conn.Close()
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		ch <- capture{hello: buf[:n], err: err}
	}()

	// Dial via the uTLS path. We don't care about the response -
	// connection will fail because the listener doesn't speak TLS - we
	// only care about what bytes went out.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, _ = utlsDialTLS(ctx, "tcp", ln.Addr().String())

	cap := <-ch
	if cap.err != nil && len(cap.hello) == 0 {
		t.Fatalf("no bytes captured: %v", cap.err)
	}

	// TLS record layer: type(1) + version(2) + length(2) + ClientHello.
	// ClientHello body: msg_type(1) + length(3) + version(2) + random(32)
	// + session_id_len(1) + session_id(N) + cipher_suites_len(2)
	// + cipher_suites(M)
	hello := cap.hello
	if len(hello) < 5+38 {
		t.Fatalf("record too short: %d bytes", len(hello))
	}
	if hello[0] != 0x16 {
		t.Fatalf("not a TLS handshake record: type=0x%x", hello[0])
	}
	off := 5 + 4 + 2 + 32 // skip to session_id_len
	if off >= len(hello) {
		t.Fatalf("record truncated at session_id_len")
	}
	sidLen := int(hello[off])
	off += 1 + sidLen
	if off+2 >= len(hello) {
		t.Fatalf("record truncated at cipher_suites_len")
	}
	ciphersLen := int(hello[off])<<8 | int(hello[off+1])
	off += 2
	if off+ciphersLen > len(hello) || ciphersLen < 2 {
		t.Fatalf("cipher_suites length %d exceeds record", ciphersLen)
	}
	firstCipher := uint16(hello[off])<<8 | uint16(hello[off+1])
	// GREASE values are 0x0A0A, 0x1A1A, 0x2A2A, ..., 0xFAFA - both bytes
	// equal and low nibble == 0xA. Chrome puts one at the start of its
	// cipher list; Go's crypto/tls does not.
	if firstCipher&0x0F0F != 0x0A0A || (firstCipher>>8) != (firstCipher&0xFF) {
		t.Fatalf(
			"first cipher 0x%04x is not a GREASE value - uTLS not on dial path",
			firstCipher,
		)
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
