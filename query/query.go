package query

import (
	"errors"
	"regexp"
	"strings"
)

type Query struct {
	Raw        string
	Terms      string
	Language   string
	Category   Category
	Engines    []string
	SafeSearch SafeLevel
	TimeRange  TimeRange
	Page       int
	Filters    Filters
}

type Preferences struct {
	DefaultLanguage   string
	DefaultCategory   Category
	DefaultSafeSearch SafeLevel
	DefaultTimeRange  TimeRange
}

var ErrEmptyQuery = errors.New("query: empty terms after parsing")

var bcp47Re = regexp.MustCompile(`^[A-Za-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)

func looksLikeBCP47(s string) bool { return bcp47Re.MatchString(s) }

func Parse(raw string, prefs Preferences) (Query, error) {
	cat := prefs.DefaultCategory
	if cat == "" {
		cat = CategoryGeneral
	}
	q := Query{
		Raw:        raw,
		Category:   cat,
		Language:   prefs.DefaultLanguage,
		SafeSearch: prefs.DefaultSafeSearch,
		TimeRange:  prefs.DefaultTimeRange,
		Page:       1,
	}

	var terms []string
	var engines []string
	seenEngine := map[string]struct{}{}

	for t := range strings.FieldsSeq(raw) {
		// Search operators (site:, filetype:, -term, ...) consume the
		// token entirely - we don't want them to leak into the term
		// list and confuse the engine's q= param.
		if parseOperatorToken(t, &q.Filters) {
			continue
		}
		switch {
		case len(t) > 1 && t[0] == '!':
			name := strings.ToLower(t[1:])
			if c, ok := ParseCategory(name); ok {
				q.Category = c
				continue
			}
			if engine, ok := bangRegistry[name]; ok {
				if _, dup := seenEngine[engine]; !dup {
					seenEngine[engine] = struct{}{}
					engines = append(engines, engine)
				}
				continue
			}
			terms = append(terms, t)
		case len(t) > 1 && t[0] == ':':
			val := t[1:]
			if c, ok := ParseCategory(val); ok {
				q.Category = c
				continue
			}
			if tr, ok := ParseTimeRange(val); ok && tr != TimeRangeAny {
				q.TimeRange = tr
				continue
			}
			if looksLikeBCP47(val) {
				q.Language = val
				continue
			}
			terms = append(terms, t)
		default:
			terms = append(terms, t)
		}
	}

	q.Terms = strings.Join(terms, " ")
	if q.Terms == "" {
		return Query{}, ErrEmptyQuery
	}
	q.Engines = engines
	return q, nil
}
