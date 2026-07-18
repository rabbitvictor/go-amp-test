# go-amp-test

Small Go web service (Echo v5 + SQLite) with a Cobra-based CLI client.

## Layout

```
cmd/server/   HTTP server entrypoint
cmd/cli/      CLI client entrypoint
internal/
  config/         Viper-based configuration
  server/         Echo router + HTTP handlers
  domain/         data models
  repository/     database/sql repositories (pure SQL, no ORM)
  infrastructure/ DB init + migration runner
  cli/            Cobra commands + HTTP client for the API
```

## Build & verify

```sh
go build ./...
go vet ./...
gofmt -l .        # must print nothing
go run ./cmd/server
go run ./cmd/cli health
```

## Endpoint / CLI parity policy

**Every endpoint added to or changed in the web server MUST be reflected in
the CLI under `internal/cli/`.** The CLI is the documented client surface for
the API; letting it drift makes it useless.

Concretely, when you touch `internal/server/router.go` or any handler:

1. Add or update the matching Cobra command in `internal/cli/` (e.g. a new
   `items delete` subcommand in `internal/cli/items.go`).
2. Add the matching method to the HTTP client in `internal/cli/client.go`.
3. Reuse the `domain` types for request/response bodies so the CLI and server
   cannot silently disagree on shapes.
4. Follow UNIX CLI conventions already established:
   - data → stdout (JSON via `writeOut`), diagnostics → stderr,
   - bad usage returns a `usageError` (exit 2); runtime/API errors exit 1,
   - short + long flags, sensible defaults, env fallback via `GO_AMP_SERVER`.
5. Document the command in the README "CLI" section.

A PR that adds an endpoint without the corresponding CLI command is incomplete.

## Conventions

- Pure SQL via `database/sql` — no ORMs.
- `modernc.org/sqlite` (pure Go, no CGo); keep `CGO_ENABLED=0` builds working.
- Forward-only embedded migrations in `internal/infrastructure/migrations/`.
- Configuration via Viper: env vars > config file > defaults (see
  `internal/config/config.go`).
- No tests yet by project policy; do not add them unless asked.
