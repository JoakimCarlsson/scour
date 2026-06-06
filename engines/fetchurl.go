package engines

import (
	"context"
	"net/http"
)

// FetchURL GETs rawURL using the shared uTLS (Chrome-fingerprinted) client and
// returns the response body (capped), the final URL after redirects, and the
// response Content-Type. It is exported so packages outside engines (e.g. the
// page package) can reuse the anti-bot transport for fetching arbitrary pages.
func FetchURL(ctx context.Context, rawURL string) (body []byte, finalURL, contentType string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", "", err
	}
	resp, b, err := fetchWithHeaders(req)
	finalURL = rawURL
	if resp != nil {
		if resp.Request != nil && resp.Request.URL != nil {
			finalURL = resp.Request.URL.String()
		}
		contentType = resp.Header.Get("Content-Type")
	}
	if err != nil {
		return b, finalURL, contentType, err
	}
	return b, finalURL, contentType, nil
}
