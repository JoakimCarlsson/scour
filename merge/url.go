package merge

import (
	"fmt"
	"net/url"
	"strings"
)

var trackerParams = map[string]struct{}{
	// click IDs
	"fbclid":  {}, // facebook
	"gclid":   {}, // google ads
	"gclsrc":  {}, // google ads
	"dclid":   {}, // google display
	"msclkid": {}, // bing
	"yclid":   {}, // yandex
	"twclid":  {}, // twitter / x
	"igshid":  {}, // instagram
	"mc_eid":  {}, // mailchimp
	"mc_cid":  {}, // mailchimp

	// hubspot
	"_hsenc":        {},
	"_hsmi":         {},
	"__hstc":        {},
	"__hssc":        {},
	"__hsfp":        {},
	"hsCtaTracking": {},

	// marketo / vero / adobe
	"mkt_tok":   {},
	"vero_id":   {},
	"vero_conv": {},
	"s_cid":     {},
	"s_kwcid":   {},

	// social referral
	"ref_src": {}, // twitter
	"ref_url": {}, // twitter

	// linkedin
	"trk":         {},
	"trkCampaign": {},

	// misc
	"spm":  {}, // alibaba
	"icid": {}, // internal click id
}

var trackerPrefixes = []string{
	"utm_",    // google analytics
	"pk_",     // matomo / piwik
	"piwik_",  // piwik
	"matomo_", // matomo
	"mtm_",    // matomo
}

func isTrackerParam(name string) bool {
	if _, ok := trackerParams[name]; ok {
		return true
	}
	for _, p := range trackerPrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
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
	u.Fragment = ""
	u.RawFragment = ""

	host := strings.ToLower(u.Hostname())
	port := u.Port()
	if strings.HasPrefix(host, "www.") && strings.Count(host, ".") >= 2 {
		host = host[4:]
	}
	if (u.Scheme == "http" && port == "80") || (u.Scheme == "https" && port == "443") {
		port = ""
	}
	if port != "" {
		u.Host = host + ":" + port
	} else {
		u.Host = host
	}

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
