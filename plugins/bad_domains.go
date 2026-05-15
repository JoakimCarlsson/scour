package plugins

import (
	"context"
	"net/url"
	"strings"
)

type BadDomains struct {
	Domains []string
	Flag    string
}

func (BadDomains) Name() string { return "bad_domains" }

func (p BadDomains) Apply(_ context.Context, c *Context) error {
	if len(p.Domains) == 0 {
		return nil
	}
	flag := p.Flag
	if flag == "" {
		flag = "bad_domain"
	}
	set := make(map[string]struct{}, len(p.Domains))
	for _, d := range p.Domains {
		set[strings.ToLower(d)] = struct{}{}
	}
	for i := range c.Ranked {
		u, err := url.Parse(c.Ranked[i].URL)
		if err != nil {
			continue
		}
		if _, bad := set[strings.ToLower(u.Host)]; bad {
			c.Ranked[i].Flags = append(c.Ranked[i].Flags, flag)
		}
	}
	return nil
}
