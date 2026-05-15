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
	key := cache.KeyFor(q)
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
	results, errs := p.FanOut(ctx, q, engs, timeout)
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
