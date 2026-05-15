package engines

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/JoakimCarlsson/scour/query"
)

var qwantURL = "https://api.qwant.com/v3/search/web"

type qwantEngine struct{}

func (qwantEngine) Name() string { return "qwant" }
func (qwantEngine) Categories() []query.Category {
	return []query.Category{query.CategoryGeneral, query.CategoryNews}
}

func (qwantEngine) Languages() LanguageTraits {
	return LanguageTraits{
		All: true,
		Supported: map[string]string{
			"en":    "en_us",
			"en-us": "en_us",
			"en-gb": "en_gb",
			"de":    "de_de",
			"fr":    "fr_fr",
			"es":    "es_es",
			"ja":    "ja_jp",
			"zh-cn": "zh_cn",
		},
	}
}

func (qwantEngine) Weight() float64 { return 1.0 }

func (e qwantEngine) Search(ctx context.Context, q query.Query) (Response, error) {
	u, _ := url.Parse(qwantURL)
	v := u.Query()
	v.Set("q", q.Filters.Render(q.Terms))
	v.Set("count", "10")
	if q.Page > 1 {
		v.Set("offset", fmt.Sprintf("%d", (q.Page-1)*10))
	}
	if loc, ok := e.Languages().Native(q.Language); ok {
		v.Set("locale", loc)
	}
	switch q.SafeSearch {
	case query.SafeOff:
		v.Set("safesearch", "0")
	case query.SafeModerate:
		v.Set("safesearch", "1")
	case query.SafeStrict:
		v.Set("safesearch", "2")
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
	return parseQwant(body)
}

func parseQwant(body []byte) (Response, error) {
	var payload struct {
		Data struct {
			Result struct {
				Items struct {
					Mainline []struct {
						Type  string `json:"type"`
						Items []struct {
							Title string `json:"title"`
							URL   string `json:"url"`
							Desc  string `json:"desc"`
						} `json:"items"`
					} `json:"mainline"`
				} `json:"items"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Response{}, fmt.Errorf("qwant: %w", err)
	}
	var results []Result
	pos := 0
	for _, group := range payload.Data.Result.Items.Mainline {
		if group.Type != "web" {
			continue
		}
		for _, it := range group.Items {
			if it.Title == "" || it.URL == "" {
				continue
			}
			pos++
			results = append(results, Result{
				Title:    it.Title,
				URL:      it.URL,
				Snippet:  it.Desc,
				Engine:   "qwant",
				Position: pos,
			})
		}
	}
	if len(results) == 0 {
		return Response{}, fmt.Errorf("qwant: no results parsed")
	}
	return Response{Results: results}, nil
}

func init() {
	Register(qwantEngine{})
}
