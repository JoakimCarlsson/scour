# scour

A Go metasearch library and CLI. Fans out a single query across many search engines in parallel, merges results, deduplicates URLs, and ranks by cross-engine consensus.

## Quick start

```sh
make build
./scour "golang tutorials"
```

A query without any flags fans out to all `general`-category engines, merges duplicates, and prints the top 20 ranked results.

## CLI flags

| Flag | Default | Purpose |
|---|---|---|
| `--timeout` | `5s` | Per-engine HTTP timeout. Slow engines that exceed it are dropped from the result set, fast ones still contribute. |
| `--limit` | `20` | Maximum results printed (pretty or JSON). Set high to inspect the long tail. |
| `--json` | `false` | Emit results as JSON. Includes the parsed `query`, the ranked `results`, and any aggregated `suggestions`. |
| `--engines` | `""` | Comma-separated engine allowlist. Overrides default selection. Example: `--engines bing,brave`. |
| `--safesearch` | `moderate` | `off`, `moderate`, or `strict`. Forwarded as the engine-specific param/cookie (see [Safesearch & timerange](#safesearch--timerange)). |
| `--timerange` | `""` | `day`, `week`, `month`, `year`. Empty means no time filter. Forwarded per-engine. |
| `--pages` | `1` | Number of pages per engine to request. Results are concatenated; positions remain monotonic per engine. |

## Query syntax

scour parses several lightweight directives out of the raw query string. They are stripped from the term list and applied as structured filters on the request.

### Bangs (`!`)

| Bang | Effect |
|---|---|
| `!g`, `!b`, `!ddg`, `!br`, `!qw`, `!mj`, `!sp`, `!ya` | Pin the fan-out to a single engine (google, bing, duckduckgo, brave, qwant, mojeek, startpage, yandex respectively). |
| `!general`, `!news`, `!images`, `!videos`, `!map`, `!music`, `!it`, `!science`, `!social`, `!files` | Switch to that category — selects only engines that implement it. |
| Unknown `!foo` | Passes through as a literal term. |

```sh
./scour "!news ukraine"           # news category, news-capable engines only
./scour "!g python decorators"    # google only
./scour "!videos blender"         # videos category
```

### Colons (`:`)

| Token | Effect |
|---|---|
| `:day`, `:week`, `:month`, `:year` | Same as `--timerange`. |
| `:en`, `:de`, `:ja`, `:en-US`, ... (BCP-47) | Set the request language. Engines without a mapping for that language are filtered out. |
| `:news`, `:images`, etc. | Same as the `!`-bang category form. |

```sh
./scour ":de :day golang generics"  # German results, last 24h
./scour ":ja プログラミング"        # Japanese
```

### Search operators

| Operator | Effect |
|---|---|
| `site:reddit.com` | Restrict to a site. |
| `-site:pinterest.com` | Exclude a site. |
| `filetype:pdf` (also `ext:pdf`) | Restrict to a file type. |
| `intitle:tutorial` | Term must appear in the title. |
| `inurl:docs` | Term must appear in the URL. |
| `-windows` | Exclude a term. |

All operators are case-insensitive. They land on `query.Query.Filters` as structured fields; the `Render()` helper rebuilds them into an engine-native query string before each engine sends its request. Engines that don't natively understand a given operator forward it anyway — upstream silently ignores unknowns.

```sh
./scour "golang site:reddit.com -windows"
./scour "transformers filetype:pdf"
```

## Engines

20 engines registered, organized by category. Each lives in `engines/<name>.go`. Adding a new engine means implementing the `Engine` interface (`Name`, `Categories`, `Languages`, `Weight`, `Search`) and calling `Register` from `init()`.

| Engine | Category(ies) | Notes |
|---|---|---|
| `bing` | general, news, images | HTML scrape, ck/a redirect URLs unwrapped |
| `brave` | general, news | HTML scrape |
| `duckduckgo` | general, images | POST to `html.duckduckgo.com`; images via `i.js` with vqd token |
| `google` | general, news | Old Android Chrome UA pool triggers server-rendered SERP; news via RSS |
| `mojeek` | general | Independent crawler |
| `qwant` | general, news | JSON API (often DataDome-blocked) |
| `startpage` | general | POST flow with `sc` token from homepage |
| `yandex` | general | `/search/site/` frame endpoint |
| `hackernews` | it | Algolia API (`hn.algolia.com`) |
| `stackexchange` | it | Stack Overflow API |
| `arxiv` | science | Atom XML feed at `export.arxiv.org` |
| `openalex` | science | Open scholarly graph |
| `nominatim` | map | OSM geocoder, populates lat/lon Extras |
| `photon` | map | Komoot's OSM-based geocoder |
| `radiobrowser` | music | Internet radio stations |
| `mixcloud` | music | DJ sets and podcasts |
| `sepiasearch` | videos | PeerTube federation |
| `invidious` | videos | YouTube via Invidious instances; canonical youtube.com URLs |
| `reddit` | social | search.json |
| `lemmy` | social | Federated Reddit alt; rotates across known instances |

Live status varies — some engines (Google, Qwant, Yandex) periodically rate-limit. The pipeline's suspension layer drops a blocked engine for a cooldown window so it doesn't waste the fan-out timeout on subsequent queries.

## Output

### Pretty (default)

```
1. [score=2.50] The Go Programming Language
   https://go.dev/
   sources: bing@1, brave@2, duckduckgo@1

2. [score=1.67] Go (programming language) - Wikipedia
   https://en.wikipedia.org/wiki/Go_(programming_language)
   sources: bing@3, brave@1, duckduckgo@3
...
Did you mean:
  - golang tutorial
```

The `score` is the sum of `1 / position` across each engine that returned the result (weighted by engine weight). `sources: engine@pos` lists the rank each engine assigned. A page seen at #1 by three engines outranks any page only one engine returned.

Category-specific output is added below the URL:
- **Images**: `thumbnail`, `dimensions`
- **Videos**: `duration`, `thumbnail`
- **News**: `published`, `by`

### JSON (`--json`)

```jsonc
{
  "query": {
    "Raw": "golang :de",
    "Terms": "golang",
    "Language": "de",
    "Filters": { ... }
  },
  "results": [
    {
      "Title": "...",
      "URL": "...",
      "Snippet": "...",
      "Sources": [{"Engine": "bing", "Position": 1}],
      "Extras": {"thumbnail_url": "...", "published_at": "..."},
      "Score": 1.5
    }
  ],
  "suggestions": ["..."]
}
```

`Extras` keys are stable: `thumbnail_url`, `thumbnail_width`, `thumbnail_height`, `duration`, `published_at`, `author`, `latitude`, `longitude` — defined as constants in the `engines` package.

## Examples

```sh
# Search news from the last day in German
./scour --timerange day ":de !news klimawandel"

# Restrict to GitHub PDFs, exclude Windows mentions
./scour "filetype:pdf site:github.com -windows performance tuning"

# Map lookup
./scour --json "!map santa monica" | jq '.results[0].Extras'

# Tech Q&A across HN + Stack Overflow
./scour "!it goroutine deadlock"

# Federated video search
./scour "!videos blender open movie"

# Multi-page fan-out
./scour --pages 3 --limit 100 "rust async runtime"
```

## Safesearch & timerange

Both are forwarded engine-specifically:

| Engine | Safesearch | Timerange |
|---|---|---|
| Bing | `adlt` URL param + `SRCHHPGUSR=ADLT=...` cookie | `filters=ex1:"ezN"` |
| Brave | `safesearch=` cookie | `tf=pd\|pw\|pm\|py` |
| DuckDuckGo | `kp` form data + cookie | `df=d\|w\|m\|y` |
| Google | `safe=off\|active` | `tbs=qdr:d\|w\|m\|y` |
| Mojeek | `safe=1` | `since=YYYYMMDD` |
| Startpage | `disable_family_filter` in preferences cookie | `with_date=d\|w\|m\|y` |

Other engines either accept the same params or ignore them gracefully.

## Development

Install required tools (golangci-lint v2, goimports, golines):

```sh
make install
```

| Command | What it does |
|---|---|
| `make build` | Build the `scour` binary (or `go build ./...`) |
| `make test` | Run unit tests with `-race -short` |
| `make fmt` | Format with `goimports` + `golines` |
| `make check` | `fmt-check` + `vet` + `lint` + `test` — single gate for "done" |

`make check` is what CI runs; pass it locally before opening a PR.

## Library use

scour is importable. The CLI in `cmd/scour/main.go` is a thin frontend over `pipeline.Pipeline`. Minimal embed:

```go
import (
    "context"
    "time"
    "github.com/JoakimCarlsson/scour/cache"
    "github.com/JoakimCarlsson/scour/pipeline"
    "github.com/JoakimCarlsson/scour/query"
)

mem := cache.NewMemory(time.Minute)
defer mem.Close()
p := pipeline.New(mem)

prefs := pipeline.Preferences{
    Preferences: query.Preferences{
        DefaultLanguage:   "en",
        DefaultCategory:   query.CategoryGeneral,
        DefaultSafeSearch: query.SafeModerate,
    },
    Timeout: 5 * time.Second,
}
out, err := p.Search(context.Background(), "golang generics", prefs)
```

`out.Ranked` is the merged, ranked result list. `out.Query` is the parsed query (post bang / colon / operator extraction). `out.Suggestions` carries aggregated "did you mean" entries from engines that returned any.

For a structured-input front-end (HTTP API / MCP / UI), bypass the text parser and construct a `query.Query` directly — `query.Filters`, `query.SafeLevel`, `query.TimeRange`, `query.Category` are all exported and ready to populate.
