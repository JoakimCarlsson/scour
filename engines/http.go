package engines

import (
	"io"
	"net/http"
)

const userAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Firefox/128.0"

var httpClient = &http.Client{
	Timeout: 0,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return http.ErrUseLastResponse
		}
		return nil
	},
}

func fetch(req *http.Request) ([]byte, error) {
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, &httpError{Status: resp.StatusCode}
	}
	return body, nil
}

type httpError struct {
	Status int
}

func (e *httpError) Error() string {
	return http.StatusText(e.Status)
}
