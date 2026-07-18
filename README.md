# go-amp-test

A small Go web service scaffold using [Echo](https://echo.labstack.com) v5.

## Requirements

- Go 1.26+

## Layout

```
cmd/server/      application entrypoint (main package)
internal/server/ Echo router, middleware, and HTTP handlers
internal/domain/ data models / domain types
```

## Run

```sh
go run ./cmd/server
```

The server listens on `:8080` by default. Configure with env vars:

| Variable          | Default       | Description                |
|-------------------|---------------|----------------------------|
| `PORT`            | `8080`        | Listen port                |
| `SERVICE_NAME`    | `go-amp-test` | Name reported by `/health` |
| `SERVICE_VERSION` | `0.1.0`       | Version reported by `/health` |

## Endpoints

| Method | Path     | Description        |
|--------|----------|--------------------|
| GET    | `/health`| Service health check |

## Build

```sh
go build ./...
```
