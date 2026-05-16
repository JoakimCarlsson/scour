package engines

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/JoakimCarlsson/scour/query"
)

var arxivURL = "https://export.arxiv.org/api/query"

type arxivEngine struct{}

func (arxivEngine) Name() string                 { return "arxiv" }
func (arxivEngine) Categories() []query.Category { return []query.Category{query.CategoryScience} }
func (arxivEngine) Languages() LanguageTraits    { return LanguageTraits{All: true} }
func (arxivEngine) Weight() float64              { return 1.0 }

func (e arxivEngine) Search(ctx context.Context, q query.Query) (Response, error) {
	u, _ := url.Parse(arxivURL)
	v := u.Query()
	v.Set("search_query", "all:"+q.Filters.Render(q.Terms))
	v.Set("max_results", "20")
	if q.Page > 1 {
		v.Set("start", fmt.Sprintf("%d", (q.Page-1)*20))
	}
	u.RawQuery = v.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Response{}, err
	}
	body, err := fetch(req)
	if err != nil {
		return Response{}, err
	}
	return parseArxiv(body)
}

type arxivFeed struct {
	XMLName xml.Name     `xml:"feed"`
	Entries []arxivEntry `xml:"entry"`
}

type arxivEntry struct {
	Title     string `xml:"title"`
	Summary   string `xml:"summary"`
	Published string `xml:"published"`
	Links     []struct {
		Href string `xml:"href,attr"`
		Rel  string `xml:"rel,attr"`
		Type string `xml:"type,attr"`
	} `xml:"link"`
	Authors []struct {
		Name string `xml:"name"`
	} `xml:"author"`
}

func parseArxiv(body []byte) (Response, error) {
	var feed arxivFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return Response{}, fmt.Errorf("arxiv: %w", err)
	}
	var results []Result
	for i, e := range feed.Entries {
		title := strings.TrimSpace(strings.ReplaceAll(e.Title, "\n", " "))
		if title == "" {
			continue
		}
		// Prefer the html alternate link, fall back to first link.
		link := ""
		for _, l := range e.Links {
			if l.Rel == "alternate" && l.Type == "text/html" {
				link = l.Href
				break
			}
		}
		if link == "" && len(e.Links) > 0 {
			link = e.Links[0].Href
		}
		if link == "" {
			continue
		}
		var authors []string
		for _, a := range e.Authors {
			authors = append(authors, a.Name)
		}
		extras := map[string]string{}
		if e.Published != "" {
			extras[ExtraPublishedAt] = e.Published
		}
		if len(authors) > 0 {
			extras[ExtraAuthor] = strings.Join(authors, ", ")
		}
		results = append(results, Result{
			Title:    title,
			URL:      link,
			Snippet:  strings.TrimSpace(strings.ReplaceAll(e.Summary, "\n", " ")),
			Engine:   "arxiv",
			Position: i + 1,
			Extras:   extras,
		})
	}
	if len(results) == 0 {
		return Response{}, fmt.Errorf("arxiv: no entries parsed")
	}
	return Response{Results: results}, nil
}

func init() {
	Register(arxivEngine{})
}
