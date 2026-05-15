# scour

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
