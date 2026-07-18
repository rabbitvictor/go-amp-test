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
internal/config/                  Viper-based configuration loading
```

## Run

```sh
go run ./cmd/server
```

The server listens on `:8080` by default. Configuration is loaded with
[Viper](https://github.com/spf13/viper) from (in order of precedence):

1. environment variables,
2. a config file (`config.yaml` by default), and
3. built-in defaults.

The config file is optional — a `config.yaml` with documented defaults ships
at the repo root. Set `CONFIG_PATH` to point at a different file. Viper
searches `.`, `./config`, and `/etc/go-amp-test` when `CONFIG_PATH` is unset.

### Settings

| Key (YAML)            | Env var              | Default       | Description                          |
|-----------------------|----------------------|---------------|--------------------------------------|
| `server.port`         | `PORT`               | `8080`        | Listen port                          |
| `server.service_name` | `SERVICE_NAME`       | `go-amp-test` | Name reported by `/health`           |
| `server.version`      | `SERVICE_VERSION`    | `0.1.0`       | Version reported by `/health`        |
| `db.path`             | `DB_PATH`            | `app.db`      | SQLite database file path            |
| `db.max_open_conns`   | `DB_MAX_OPEN_CONNS`  | `1`           | Max open DB connections (SQLite serializes writes) |
| `db.busy_timeout`     | `DB_BUSY_TIMEOUT`    | `5000`        | SQLite busy timeout, in milliseconds |
| `db.journal_mode`     | `DB_JOURNAL_MODE`    | `WAL`         | SQLite journal mode (`WAL`/`MEMORY`/`DELETE`/`OFF`) |
| `db.synchronous`      | `DB_SYNCHRONOUS`     | `NORMAL`      | SQLite synchronous pragma (`NORMAL`/`FULL`/`OFF`) |
| `db.foreign_keys`     | `DB_FOREIGN_KEYS`    | `true`        | Enable SQLite foreign key enforcement |

### Example config.yaml

```yaml
server:
  port: "8080"
  service_name: "go-amp-test"
  version: "0.1.0"

db:
  path: "app.db"
  max_open_conns: 1
  busy_timeout: 5000
  journal_mode: "WAL"
  synchronous: "NORMAL"
  foreign_keys: true
```

Env vars take precedence over the config file, which takes precedence over
the defaults above.

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

## Docker

A multi-stage Dockerfile builds a static binary (`CGO_ENABLED=0`, since
`modernc.org/sqlite` is pure Go) and runs it on a minimal distroless image.

```sh
# Build the image
docker build -t go-amp-test .

# Run it, persisting the SQLite file to a host volume
docker run -p 8080:8080 -v go-amp-test-data:/data go-amp-test
```

The SQLite database is written to `/data/app.db` inside the container by
default; mount `/data` to persist it. Override any env var to configure:

```sh
docker run -p 9000:9000 -e PORT=9000 -v go-amp-test-data:/data go-amp-test
```
