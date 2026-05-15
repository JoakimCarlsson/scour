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

	prefs := pipeline.Preferences{
		Preferences: query.Preferences{
			DefaultLanguage: "en",
			DefaultCategory: query.CategoryGeneral,
		},
		Timeout: *timeout,
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
		emitJSON(out.Ranked, *limit)
		return
	}
	emitPretty(out, *limit)
}

func emitJSON(results []rank.Ranked, limit int) {
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(results)
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
