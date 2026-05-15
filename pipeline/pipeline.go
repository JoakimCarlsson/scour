package pipeline

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/JoakimCarlsson/scour/cache"
	"github.com/JoakimCarlsson/scour/engines"
	"github.com/JoakimCarlsson/scour/merge"
	"github.com/JoakimCarlsson/scour/plugins"
	"github.com/JoakimCarlsson/scour/query"
	"github.com/JoakimCarlsson/scour/rank"
)

type Preferences struct {
	query.Preferences
	DisabledEngines []string
	EngineWeights   map[string]float64
	BadDomains      []string
	EnginesAllow    []string
	Timeout         time.Duration
	CacheTTL        time.Duration
	Pages           int
}

type Pipeline struct {
	Cache       cache.Cache
	FanOut      func(ctx context.Context, q query.Query, engs []engines.Engine, timeout time.Duration) ([]engines.Result, []engines.FanOutError)
	FanOutResp  func(ctx context.Context, q query.Query, engs []engines.Engine, timeout time.Duration) ([]engines.Result, []string, []engines.FanOutError)
	Select      func(q query.Query, prefs engines.Preferences) []engines.Engine
	PluginChain func(prefs Preferences) []plugins.Plugin
}

func New(c cache.Cache) *Pipeline {
	return &Pipeline{
		Cache:       c,
		FanOut:      engines.FanOut,
		FanOutResp:  engines.FanOutResponse,
		Select:      engines.Select,
		PluginChain: defaultPlugins,
	}
}

func defaultPlugins(p Preferences) []plugins.Plugin {
	return []plugins.Plugin{
		plugins.TrackerStrip{},
		plugins.BadDomains{Domains: p.BadDomains},
		plugins.AnswerMath{},
		plugins.AnswerRandom{},
		plugins.AnswerUnits{},
		plugins.AnswerStats{},
		plugins.AnswerCurrency{},
		plugins.InfoboxWikipedia{},
	}
}

type Output struct {
	Query       query.Query
	Ranked      []rank.Ranked
	Infobox     *plugins.Infobox
	Answer      *plugins.Answer
	Suggestions []string
	Errors      []engines.FanOutError
}

func (p *Pipeline) Search(ctx context.Context, raw string, prefs Preferences) (*Output, error) {
	q, err := query.Parse(raw, prefs.Preferences)
	if err != nil {
		return nil, err
	}
	if len(prefs.EnginesAllow) > 0 && len(q.Engines) == 0 {
		q.Engines = prefs.EnginesAllow
	}
	out := &Output{Query: q}
	pagesForKey := prefs.Pages
	if pagesForKey < 1 {
		pagesForKey = 1
	}
	keyQ := q
	keyQ.Page = pagesForKey
	key := cache.KeyFor(keyQ)
	if p.Cache != nil {
		if cached, ok := p.Cache.Get(key); ok {
			out.Ranked = cached
			return out, nil
		}
	}
	engs := p.Select(q, engines.Preferences{DisabledEngines: prefs.DisabledEngines})
	timeout := prefs.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	pages := prefs.Pages
	if pages < 1 {
		pages = 1
	}
	var results []engines.Result
	var errs []engines.FanOutError
	sugCounts := map[string]int{}
	sugDisplay := map[string]string{}
	enginePos := map[string]int{}
	for page := 1; page <= pages; page++ {
		pq := q
		pq.Page = page
		var pageResults []engines.Result
		var pageSugs []string
		var pageErrs []engines.FanOutError
		if p.FanOutResp != nil {
			pageResults, pageSugs, pageErrs = p.FanOutResp(ctx, pq, engs, timeout)
		} else {
			pageResults, pageErrs = p.FanOut(ctx, pq, engs, timeout)
		}
		errs = append(errs, pageErrs...)
		for _, r := range pageResults {
			enginePos[r.Engine]++
			r.Position = enginePos[r.Engine]
			results = append(results, r)
		}
		for _, s := range pageSugs {
			k := strings.ToLower(s)
			if _, ok := sugDisplay[k]; !ok {
				sugDisplay[k] = s
			}
			sugCounts[k]++
		}
	}
	out.Suggestions = topSuggestions(sugCounts, sugDisplay, 5)
	out.Errors = errs
	merged := merge.Merge(results)
	ranked := rank.Rank(merged, prefs.EngineWeights)
	pctx := &plugins.Context{Query: q, Ranked: ranked}
	if err := plugins.Run(ctx, p.PluginChain(prefs), pctx); err != nil {
		return nil, err
	}
	out.Ranked = pctx.Ranked
	out.Infobox = pctx.Infobox
	out.Answer = pctx.Answer
	if p.Cache != nil {
		ttl := prefs.CacheTTL
		if ttl == 0 {
			ttl = 5 * time.Minute
		}
		p.Cache.Set(key, out.Ranked, ttl)
	}
	return out, nil
}

func topSuggestions(counts map[string]int, display map[string]string, n int) []string {
	if len(counts) == 0 {
		return nil
	}
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if counts[keys[i]] != counts[keys[j]] {
			return counts[keys[i]] > counts[keys[j]]
		}
		return strings.ToLower(display[keys[i]]) < strings.ToLower(display[keys[j]])
	})
	if len(keys) > n {
		keys = keys[:n]
	}
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, display[k])
	}
	return out
}
