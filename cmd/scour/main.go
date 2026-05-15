package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/JoakimCarlsson/scour/cache"
	"github.com/JoakimCarlsson/scour/pipeline"
	"github.com/JoakimCarlsson/scour/query"
	"github.com/JoakimCarlsson/scour/rank"
)

func main() {
	timeout := flag.Duration("timeout", 5*time.Second, "per-engine timeout")
	limit := flag.Int("limit", 20, "max results printed")
	jsonOut := flag.Bool("json", false, "emit results as JSON")
	enginesCSV := flag.String("engines", "", "comma-separated engine allowlist")
	safeSearch := flag.String("safesearch", "moderate", "safesearch level: off|moderate|strict")
	timeRange := flag.String("timerange", "", "time range: day|week|month|year (default: any)")
	pages := flag.Int("pages", 1, "number of pages per engine to request")
	flag.Parse()

	raw := strings.TrimSpace(strings.Join(flag.Args(), " "))
	if raw == "" {
		fmt.Fprintln(os.Stderr, "scour: empty query")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	mem := cache.NewMemory(time.Minute)
	defer mem.Close()
	p := pipeline.New(mem)

	safe, ok := query.ParseSafeLevel(*safeSearch)
	if !ok {
		fmt.Fprintf(os.Stderr, "scour: invalid --safesearch %q\n", *safeSearch)
		os.Exit(1)
	}
	tr, ok := query.ParseTimeRange(*timeRange)
	if !ok {
		fmt.Fprintf(os.Stderr, "scour: invalid --timerange %q\n", *timeRange)
		os.Exit(1)
	}
	prefs := pipeline.Preferences{
		Preferences: query.Preferences{
			DefaultLanguage:   "en",
			DefaultCategory:   query.CategoryGeneral,
			DefaultSafeSearch: safe,
			DefaultTimeRange:  tr,
		},
		Timeout: *timeout,
		Pages:   *pages,
	}
	if *enginesCSV != "" {
		for e := range strings.SplitSeq(*enginesCSV, ",") {
			if e = strings.TrimSpace(e); e != "" {
				prefs.EnginesAllow = append(prefs.EnginesAllow, e)
			}
		}
	}

	out, err := p.Search(ctx, raw, prefs)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			os.Exit(130)
		}
		fmt.Fprintf(os.Stderr, "scour: %v\n", err)
		os.Exit(1)
	}

	if *jsonOut {
		emitJSON(out, *limit)
		return
	}
	emitPretty(out, *limit)
}

func emitJSON(out *pipeline.Output, limit int) {
	results := out.Ranked
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	payload := struct {
		Results     []rank.Ranked `json:"results"`
		Suggestions []string      `json:"suggestions,omitempty"`
	}{Results: results, Suggestions: out.Suggestions}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(payload)
}

func emitPretty(out *pipeline.Output, limit int) {
	if out.Answer != nil {
		fmt.Printf("Answer: %s\n\n", out.Answer.Text)
	}
	if out.Infobox != nil {
		fmt.Printf("%s\n%s\n", out.Infobox.Title, truncate(out.Infobox.Summary, 200))
		if out.Infobox.URL != "" {
			fmt.Printf("%s\n", out.Infobox.URL)
		}
		fmt.Println()
	}
	results := out.Ranked
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	if len(results) == 0 {
		fmt.Fprintln(os.Stderr, "no results")
		if len(out.Errors) > 0 {
			for _, e := range out.Errors {
				fmt.Fprintf(os.Stderr, "  %s\n", e.Error())
			}
		}
		return
	}
	for i, r := range results {
		fmt.Printf("%d. [score=%.2f] %s\n", i+1, r.Score, r.Title)
		fmt.Printf("   %s\n", r.URL)
		if r.Snippet != "" {
			fmt.Printf("   %s\n", truncate(r.Snippet, 120))
		}
		fmt.Printf("   sources: %s\n", formatSources(r))
		fmt.Println()
	}
	if len(out.Suggestions) > 0 {
		fmt.Println("Did you mean:")
		for _, s := range out.Suggestions {
			fmt.Printf("  - %s\n", s)
		}
	}
}

func formatSources(r rank.Ranked) string {
	parts := make([]string, 0, len(r.Sources))
	for _, s := range r.Sources {
		parts = append(parts, fmt.Sprintf("%s@%d", s.Engine, s.Position))
	}
	return strings.Join(parts, ", ")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
