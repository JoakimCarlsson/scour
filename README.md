# scour

## Quick start

```sh
make build
./scour "golang tutorials"
```

Useful flags:

| Flag         | Default | Purpose                                  |
| ------------ | ------- | ---------------------------------------- |
| `--timeout`  | `5s`    | Per-engine HTTP timeout                  |
| `--limit`    | `20`    | Max results printed                      |
| `--json`     | `false` | Emit results as JSON                     |
| `--engines`  | `""`    | Comma-separated engine allowlist         |

Examples:

```sh
./scour --json "golang tutorials" | jq '.[0]'
./scour --engines duckduckgo,bing "rust async"
```

## Development

Install required tools (golangci-lint v2, goimports, golines):

```sh
make install
```

Common commands:

| Command          | What it does                                          |
| ---------------- | ----------------------------------------------------- |
| `make build`     | Build the `scour` binary (or `go build ./...`)        |
| `make test`      | Run unit tests with `-race -short`                    |
| `make fmt`       | Format with `goimports` + `golines`                   |
| `make check`     | `fmt-check` + `vet` + `lint` + `test` — run before pushing |

`make check` is the single command CI runs; pass it locally before opening a PR.
