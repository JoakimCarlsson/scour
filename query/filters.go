package query

import (
	"sort"
	"strings"
)

// Filters is the structured representation of search operators. The
// library accepts this directly; the text parser (query.Parse) and any
// future HTTP/UI front-end produces the same struct. Engines read it
// and either rebuild a native operator-aware query string, or forward
// the fields they support to upstream API params.
type Filters struct {
	Sites     []string // site:reddit.com
	NotSites  []string // -site:pinterest.com
	FileTypes []string // filetype:pdf
	InTitle   []string // intitle:tutorial
	InURL     []string // inurl:docs
	Excluded  []string // -windows (plain negative term)
}

// IsZero reports whether the filter struct has no entries. Useful for
// cache-key construction so the absent case canonicalises to one shape.
func (f Filters) IsZero() bool {
	return len(f.Sites) == 0 && len(f.NotSites) == 0 && len(f.FileTypes) == 0 &&
		len(f.InTitle) == 0 && len(f.InURL) == 0 && len(f.Excluded) == 0
}

// Canonical returns a deterministic string for cache-key purposes. Order
// of multi-value fields is normalised so {site:a site:b} and
// {site:b site:a} produce the same key.
func (f Filters) Canonical() string {
	var parts []string
	for _, g := range [][2]any{
		{"site", f.Sites},
		{"-site", f.NotSites},
		{"filetype", f.FileTypes},
		{"intitle", f.InTitle},
		{"inurl", f.InURL},
		{"-", f.Excluded},
	} {
		k := g[0].(string)
		vs := g[1].([]string)
		if len(vs) == 0 {
			continue
		}
		sorted := append([]string(nil), vs...)
		sort.Strings(sorted)
		for _, v := range sorted {
			parts = append(parts, k+":"+v)
		}
	}
	return strings.Join(parts, " ")
}

// Render rebuilds an operator-aware query string from terms plus the
// filters. This is what engines that natively understand site:/filetype:
// etc. send upstream. Engines that don't simply ignore the operators
// they don't recognise - cheaper than a per-engine support matrix.
func (f Filters) Render(terms string) string {
	if f.IsZero() {
		return terms
	}
	var b strings.Builder
	b.WriteString(strings.TrimSpace(terms))
	for _, s := range f.Sites {
		b.WriteString(" site:")
		b.WriteString(s)
	}
	for _, s := range f.NotSites {
		b.WriteString(" -site:")
		b.WriteString(s)
	}
	for _, s := range f.FileTypes {
		b.WriteString(" filetype:")
		b.WriteString(s)
	}
	for _, s := range f.InTitle {
		b.WriteString(" intitle:")
		b.WriteString(s)
	}
	for _, s := range f.InURL {
		b.WriteString(" inurl:")
		b.WriteString(s)
	}
	for _, s := range f.Excluded {
		b.WriteString(" -")
		b.WriteString(s)
	}
	return strings.TrimSpace(b.String())
}

// parseOperatorToken classifies a single raw token. Returns ok=true if
// the token consumed something filter-shaped; the caller drops it from
// the term list in that case. Unknown tokens (e.g. plain words, colon
// prefixes for language/category/time-range) are left for the rest of
// the parser to handle.
func parseOperatorToken(tok string, f *Filters) bool {
	if tok == "" {
		return false
	}

	// Negative plain term: "-windows". Don't claim "-site:x" here -
	// that's handled below.
	if tok[0] == '-' && len(tok) > 1 && !strings.HasPrefix(tok, "-site:") {
		// Negated operator?
		if i := strings.Index(tok, ":"); i > 1 {
			op := strings.ToLower(tok[1:i])
			val := tok[i+1:]
			if val == "" {
				return false
			}
			switch op {
			case "site":
				f.NotSites = append(f.NotSites, val)
				return true
			}
			// Unknown negated operator - leave it for the term list.
			return false
		}
		f.Excluded = append(f.Excluded, tok[1:])
		return true
	}

	// -site:x explicitly
	if strings.HasPrefix(strings.ToLower(tok), "-site:") {
		val := tok[len("-site:"):]
		if val == "" {
			return false
		}
		f.NotSites = append(f.NotSites, val)
		return true
	}

	// Positive op:value
	if i := strings.Index(tok, ":"); i > 0 && i < len(tok)-1 {
		op := strings.ToLower(tok[:i])
		val := tok[i+1:]
		switch op {
		case "site":
			f.Sites = append(f.Sites, val)
			return true
		case "filetype", "ext":
			f.FileTypes = append(f.FileTypes, val)
			return true
		case "intitle":
			f.InTitle = append(f.InTitle, val)
			return true
		case "inurl":
			f.InURL = append(f.InURL, val)
			return true
		}
	}
	return false
}
