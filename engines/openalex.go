package engines

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/JoakimCarlsson/scour/query"
)

var openalexURL = "https://api.openalex.org/works"

type openalexEngine struct{}

func (openalexEngine) Name() string { return "openalex" }

func (openalexEngine) Categories() []query.Category { return []query.Category{query.CategoryScience} }
func (openalexEngine) Languages() LanguageTraits    { return LanguageTraits{All: true} }
func (openalexEngine) Weight() float64              { return 1.0 }

func (e openalexEngine) Search(ctx context.Context, q query.Query) (Response, error) {
	u, _ := url.Parse(openalexURL)
	v := u.Query()
	v.Set("search", q.Filters.Render(q.Terms))
	v.Set("per-page", "20")
	if q.Page > 1 {
		v.Set("page", fmt.Sprintf("%d", q.Page))
	}
	u.RawQuery = v.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("Accept", "application/json")
	body, err := fetch(req)
	if err != nil {
		return Response{}, err
	}
	return parseOpenAlex(body)
}

func parseOpenAlex(body []byte) (Response, error) {
	var payload struct {
		Results []struct {
			Title           string `json:"title"`
			DOI             string `json:"doi"`
			PublicationDate string `json:"publication_date"`
			Abstract        string `json:"abstract"`
			Authorships     []struct {
				Author struct {
					DisplayName string `json:"display_name"`
				} `json:"author"`
			} `json:"authorships"`
			OpenAccess struct {
				OAURL string `json:"oa_url"`
			} `json:"open_access"`
			PrimaryLocation struct {
				LandingPageURL string `json:"landing_page_url"`
			} `json:"primary_location"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Response{}, fmt.Errorf("openalex: %w", err)
	}
	var results []Result
	for i, w := range payload.Results {
		if w.Title == "" {
			continue
		}
		link := w.OpenAccess.OAURL
		if link == "" {
			link = w.PrimaryLocation.LandingPageURL
		}
		if link == "" && w.DOI != "" {
			link = w.DOI
		}
		if link == "" {
			continue
		}
		var authors []string
		for _, a := range w.Authorships {
			if a.Author.DisplayName != "" {
				authors = append(authors, a.Author.DisplayName)
			}
			if len(authors) == 5 {
				break
			}
		}
		extras := map[string]string{}
		if w.PublicationDate != "" {
			extras[ExtraPublishedAt] = w.PublicationDate
		}
		if len(authors) > 0 {
			extras[ExtraAuthor] = strings.Join(authors, ", ")
		}
		results = append(results, Result{
			Title:    w.Title,
			URL:      link,
			Snippet:  w.Abstract,
			Engine:   "openalex",
			Position: i + 1,
			Extras:   extras,
		})
	}
	if len(results) == 0 {
		return Response{}, fmt.Errorf("openalex: no results parsed")
	}
	return Response{Results: results}, nil
}

func init() {
	Register(openalexEngine{})
}
