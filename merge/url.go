package merge

import (
	"fmt"
	"net/url"
	"strings"
)

var trackerParams = map[string]struct{}{
	"fbclid":  {},
	"gclid":   {},
	"mc_eid":  {},
	"yclid":   {},
	"msclkid": {},
}

func isTrackerParam(name string) bool {
	if _, ok := trackerParams[name]; ok {
		return true
	}
	return strings.HasPrefix(name, "utm_")
}

func Normalize(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("merge: missing scheme or host in %q", raw)
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""
	u.RawFragment = ""
	q := u.Query()
	for k := range q {
		if isTrackerParam(k) {
			q.Del(k)
		}
	}
	u.RawQuery = q.Encode()
	if len(u.Path) > 1 && strings.HasSuffix(u.Path, "/") {
		u.Path = strings.TrimRight(u.Path, "/")
	}
	return u.String(), nil
}
