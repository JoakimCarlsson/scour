package pipeline

import (
	"context"
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
	Select      func(q query.Query, prefs engines.Preferences) []engines.Engine
	PluginChain func(prefs Preferences) []plugins.Plugin
}

func New(c cache.Cache) *Pipeline {
	return &Pipeline{
		Cache:       c,
		FanOut:      engines.FanOut,
		Select:      engines.Select,
		PluginChain: defaultPlugins,
	}
}

func defaultPlugins(p Preferences) []plugins.Plugin {
	return []plugins.Plugin{
		plugins.TrackerStrip{},
		plugins.BadDomains{Domains: p.BadDomains},
		plugins.AnswerMath{},
		plugins.InfoboxWikipedia{},
	}
}

type Output struct {
	Query   query.Query
	Ranked  []rank.Ranked
	Infobox *plugins.Infobox
	Answer  *plugins.Answer
	Errors  []engines.FanOutError
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
	enginePos := map[string]int{}
	for page := 1; page <= pages; page++ {
		pq := q
		pq.Page = page
		pageResults, pageErrs := p.FanOut(ctx, pq, engs, timeout)
		errs = append(errs, pageErrs...)
		for _, r := range pageResults {
			enginePos[r.Engine]++
			r.Position = enginePos[r.Engine]
			results = append(results, r)
		}
	}
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
