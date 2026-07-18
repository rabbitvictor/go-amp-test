# go-amp-test

A small Go web service using [Echo](https://echo.labstack.com) v5 and SQLite
([modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite), pure Go, no CGo).

## Requirements

- Go 1.26+

## Layout

```
cmd/server/                       application entrypoint (main package)
internal/server/                  Echo router, middleware, and HTTP handlers
internal/domain/                  data models / domain types
internal/repository/              database/sql repositories (pure SQL, no ORM)
internal/infrastructure/          DB init + migration runner
internal/infrastructure/migrations/  forward-only *.sql migrations
```

## Run

```sh
go run ./cmd/server
```

The server listens on `:8080` by default. Configure with env vars:

| Variable          | Default       | Description                          |
|-------------------|---------------|--------------------------------------|
| `PORT`            | `8080`        | Listen port                          |
| `SERVICE_NAME`    | `go-amp-test` | Name reported by `/health`           |
| `SERVICE_VERSION` | `0.1.0`       | Version reported by `/health`        |
| `DB_PATH`         | `app.db`      | SQLite database file path            |

## Endpoints

| Method | Path           | Description              |
|--------|----------------|--------------------------|
| GET    | `/health`      | Service health check     |
| GET    | `/items`       | List all items           |
| GET    | `/items/:id`   | Get a single item        |
| POST   | `/items`       | Create an item           |

### Examples

```sh
curl -X POST http://localhost:8080/items \
  -H 'Content-Type: application/json' \
  -d '{"name":"my item"}'

curl http://localhost:8080/items
```

## Database

SQLite is opened in WAL mode with a busy timeout and foreign keys enabled.
The database is created on first run, and pending migrations are applied
automatically on startup. Each migration runs in its own transaction and is
recorded in the `schema_migrations` table; migrations are forward-only and
applied in lexical filename order.

To add a migration, drop a new `NNNN_description.sql` file into
`internal/infrastructure/migrations/` — it is embedded into the binary via
`go:embed` and applied on the next start.

## Build

```sh
go build ./...
```
