# Testing guide

This document explains how the unit tests in `go-amp-test` are structured and
how the Go testing primitives they rely on (`testing`, `net/http/httptest`,
and an in-memory SQLite database) are used. It is the reference for anyone
adding or modifying tests — the conventions here are enforced by
`AGENTS.md` and the project's test policy.

## Layout

Tests live next to the code they exercise, as `*_test.go` files in the same
package (white-box testing), so they can reach unexported helpers when needed:

```
internal/
  config/         config_test.go            DSN/Addr pure unit tests
  repository/     testutil_test.go          shared newMemoryDB helper
                  item_repository_test.go   repository CRUD against in-memory SQLite
  infrastructure/ db_test.go                OpenDB + Migrate idempotency
  server/         server_test.go            shared newTestEcho + do helpers
                  health_test.go            GET /health
                  item_test.go              POST/GET /items
                  router_test.go            route registration with DB=nil
                  fuzz_test.go              Fuzz targets for POST/GET /items
  cli/            client_test.go            HTTP client against httptest.Server
                  output_test.go            writeOut formatting
                  items_test.go             resolveName parsing
                  fuzz_test.go              Fuzz target for resolveName JSON parsing
```

Run them with:

```sh
go test ./...          # all tests
go test -race ./...    # with the race detector (run before declaring done)
go test -v ./internal/server/...   # a single package, verbose
```

## What counts as a unit test here

The project's `AGENTS.md` policy is explicit: **no integration tests.** That
means tests must NOT:

- start the real `cmd/server` process or bind a real port,
- open real network sockets to external services,
- read or write files on disk outside the test process (no `app.db`, no
  temp dirs),
- depend on wall-clock time or external state.

Everything that would normally require an external dependency is replaced:

| Real thing              | Replaced with                                |
|-------------------------|----------------------------------------------|
| SQLite file on disk     | in-memory SQLite (`:memory:`)                |
| Echo HTTP server        | `echo.Echo` driven via `httptest`            |
| Remote API for the CLI  | `httptest.Server` with a stub handler        |

The tests are therefore fast (the whole suite runs in well under a second),
hermetic, and safe to run in parallel and under `-race`.

## The `testing` package basics

Every test is a function named `TestXxx(t *testing.T)` in a `*_test.go` file.
The `*testing.T` is both the failure reporter and the test lifecycle handle.
The project uses three of its features heavily:

### 1. `t.Helper()` marks test-only helpers

When a helper calls `t.Fatalf`, the failure should point at the line that
*called* the helper, not the line inside it. Marking the helper makes that
happen:

```go
func newMemoryDB(t *testing.T) *sql.DB {
    t.Helper()
    db, err := infrastructure.OpenDB(...)
    if err != nil {
        t.Fatalf("open in-memory db: %v", err)
    }
    return db
}
```

### 2. `t.Cleanup` for guaranteed teardown

`t.Cleanup(func() { _ = db.Close() })` registers a teardown callback that
runs when the test (or subtest) ends, even on failure. This is used for
closing the in-memory `*sql.DB` and shutting down `httptest.Server` instances.
Prefer `t.Cleanup` over `defer` in helpers so the caller doesn't have to
remember to close anything.

### 3. Table-driven subtests with `t.Run`

The validation test for `POST /items` uses a table of cases and runs each as
a named subtest, so a single failure names the offending case:

```go
cases := []struct {
    name string
    body string
    want int
}{
    {"empty name", `{"name":""}`, http.StatusBadRequest},
    {"malformed json", `{not json}`, http.StatusBadRequest},
}
for _, tc := range cases {
    t.Run(tc.name, func(t *testing.T) {
        rec := do(t, srv, http.MethodPost, "/items", tc.body)
        if rec.Code != tc.want { t.Errorf(...) }
    })
}
```

## In-memory SQLite for repository / infrastructure tests

`modernc.org/sqlite` is a pure-Go driver, so `:memory:` works without CGo and
without touching disk. The one subtlety: an in-memory database lives in the
*connection* that created it. If `database/sql` opens a second connection,
that connection sees an empty database. The fix is to pin the pool to a
single connection:

```go
db, err := infrastructure.OpenDB(ctx, infrastructure.DBConfig{
    Path:         ":memory:",
    MaxOpenConns: 1,   // keep the ephemeral schema alive for the *sql.DB
})
```

`OpenDB` also runs the embedded migrations, so each test gets a freshly
migrated schema (the `items` table + `schema_migrations` bookkeeping table)
with no manual `CREATE TABLE` in the test. The shared helper lives in
`internal/repository/testutil_test.go`:

```go
func newMemoryDB(t *testing.T) *sql.DB {
    t.Helper()
    db, err := infrastructure.OpenDB(context.Background(),
        infrastructure.DBConfig{Path: ":memory:", MaxOpenConns: 1})
    if err != nil { t.Fatalf("open in-memory db: %v", err) }
    t.Cleanup(func() { _ = db.Close() })
    return db
}
```

Repository tests then exercise real SQL against a real (in-memory) SQLite,
which catches SQL bugs that mocks would miss — e.g. column order mismatches
in `Scan`, `ErrNotFound` mapping, and `ORDER BY id DESC` ordering:

```go
func TestItemRepository_Get_NotFound(t *testing.T) {
    repo := NewItemRepository(newMemoryDB(t))
    _, err := repo.Get(context.Background(), 9999)
    if !errors.Is(err, ErrNotFound) {
        t.Fatalf("Get missing row returned %v, want ErrNotFound", err)
    }
}
```

## `httptest` for the web layer

`net/http/httptest` provides two complementary tools; the project uses both.

### `httptest.NewRecorder` — capture a response without a server

`httptest.NewRecorder()` returns a `*httptest.ResponseRecorder` that
implements `http.ResponseWriter` but just buffers what gets written. Combined
with `httptest.NewRequest(method, path, body)` to build a request, you can
invoke a handler directly and inspect the recorded status code and body:

```go
rec := httptest.NewRecorder()
req := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(`{"name":"x"}`))
handler(rec, req)
if rec.Code != http.StatusCreated { t.Errorf(...) }
```

This is the cheapest way to test a single `http.HandlerFunc`.

### `httptest.NewServer` — a real loopback server bound to the Echo engine

For the Echo web layer the project wraps the fully configured `*echo.Echo`
in an `httptest.Server` so the full middleware stack and router run on every
request:

```go
func newTestEcho(t *testing.T) (*httptest.Server, *sql.DB) {
    t.Helper()
    db := ... // in-memory SQLite, migrated
    e := New(Config{Service: "go-amp-test", Version: "test", DB: db})
    srv := httptest.NewServer(e)        // e implements http.Handler
    t.Cleanup(srv.Close)
    return srv, db
}
```

Echo's `*Echo` implements `http.Handler` (it has a `ServeHTTP` method), so it
can be passed directly to `httptest.NewServer`. The `do` helper then dispatches
a request through the server's handler and captures the response with a
recorder — no real socket needed, but the full Echo routing + middleware runs:

```go
func do(t *testing.T, srv *httptest.Server, method, path, body string) *httptest.ResponseRecorder {
    t.Helper()
    req := httptest.NewRequest(method, path, strings.NewReader(body))
    rec := httptest.NewRecorder()
    srv.Config.Handler.ServeHTTP(rec, req)   // run Echo's router in-process
    return rec
}
```

Assertions are then made on the recorded status code and the decoded JSON
body:

```go
rec := do(t, srv, http.MethodGet, "/health", "")
if rec.Code != http.StatusOK { t.Fatalf(...) }
var h domain.Health
if err := json.Unmarshal(rec.Body.Bytes(), &h); err != nil { t.Fatalf(...) }
if h.Status != domain.HealthStatusUp { t.Errorf(...) }
```

Why drive `ServeHTTP` through the server rather than calling the handler
methods directly? Because Echo handlers take `*echo.Context`, which is
non-trivial to construct by hand. Going through `e.ServeHTTP` lets Echo build
its own context from the `*http.Request`, exactly as it does in production.
This is the Echo-idiomatic way to test the full request path without spinning
up a real listener.

## `httptest.Server` for the CLI HTTP client

The CLI's `Client` is an HTTP client. Instead of pointing it at a running
server, the tests stand up a tiny `httptest.Server` with a stub
`http.HandlerFunc` and point the client at its URL:

```go
func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
    t.Helper()
    srv := httptest.NewServer(handler)
    t.Cleanup(srv.Close)
    return NewClient(srv.URL, 5*time.Second)
}
```

The stub handler switches on `r.Method + " " + r.URL.Path` to return canned
responses, letting the test verify that the client sends the right method and
path, decodes success bodies into `domain` types, and turns non-2xx responses
into `*APIError`:

```go
c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
    http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
})
_, err := c.Health(context.Background())
apiErr, ok := err.(*APIError)
if !ok || apiErr.Status != http.StatusInternalServerError { t.Errorf(...) }
```

Because `httptest.Server` listens on a real loopback port, this is the one
place a real socket is used — but it is entirely in-process and ephemeral, so
it does not violate the "no integration tests" rule.

## Pure helper unit tests

Helpers with no I/O (`writeOut`, `resolveName`, `DBConfig.DSN`,
`ServerConfig.Addr`) get plain unit tests: call the function with a known
input, assert on the output or side effect. For `writeOut` the side effect is
bytes written to a `bytes.Buffer`; for `resolveName` it is the returned
string or error. These need no `httptest` and no database — they are the
fastest and most boring tests in the suite, which is the point.

## Fuzzing

Native `testing.F` fuzz targets cover the untrusted-input boundaries of the
service. Each target lives in a `fuzz_test.go` file next to the code it
exercises and reuses the same helpers as the unit tests.

| Target | Package | What it fuzzes | Invariant |
|--------|---------|----------------|-----------|
| `FuzzItemHandler_Create` | `internal/server` | `POST /items` JSON body via the full Echo router | always 201/400/500, never panics; on 201 the name round-trips byte-for-byte |
| `FuzzItemHandler_Get` | `internal/server` | `:id` path param of `GET /items/:id` | always 200/404/400, never 500, never panics |
| `FuzzItemRepository_Create` | `internal/repository` | arbitrary name strings through `Create` + `Get` against in-memory SQLite | never panics; accepted values round-trip byte-for-byte |
| `FuzzResolveName` | `internal/cli` | `--data` JSON parsing in `resolveName` | never returns both a name and an error, never panics |

A target has two parts: a seed corpus added with `f.Add`, and the fuzz
function passed to `f.Fuzz`. Seeds cover the edge cases worth pinning (empty,
null bytes, invalid UTF-8, non-object JSON roots, very large strings); the
mutator then explores around them.

```go
func FuzzItemHandler_Create(f *testing.F) {
    f.Add(`{"name":"alpha"}`)
    f.Add(`{not json}`)
    f.Add(`{"name":"\xff\xfe\xfd"}`)
    f.Fuzz(func(t *testing.T, body string) {
        srv, _ := newTestEcho(t)
        rec := do(t, srv, http.MethodPost, "/items", body)
        switch rec.Code {
        case http.StatusCreated, http.StatusBadRequest, http.StatusInternalServerError:
        default:
            t.Fatalf("unexpected status %d", rec.Code)
        }
    })
}
```

Two harness-level subtleties:

- **`url.PathEscape` the path input.** `httptest.NewRequest` panics on
  malformed URL paths, so `FuzzItemHandler_Get` escapes the fuzzed id before
  building the path. This keeps the harness itself from panicking before the
  handler runs, while still exercising `strconv.ParseInt` on the raw param.
- **`t.Skip` non-hermetic branches.** `resolveName` reads stdin when
  `data == "-"`. `FuzzResolveName` skips that input so the target stays
  deterministic and does not block on stdin.

Run the seed corpus (no mutation) with the normal test command:

```sh
go test ./...                              # runs every f.Add seed as a subtest
```

Run a short mutation pass against a single target:

```sh
go test -run=NONE -fuzz=FuzzItemHandler_Create -fuzztime=10s ./internal/server/
```

When the fuzzer finds a failing input it writes it to
`testdata/fuzz/<TargetName>/` under the package. Commit that file so the seed
corpus permanently guards against the regression. `-fuzz=...` without
`-fuzztime` runs until interrupted.

### When to add a fuzz target

Fuzz targets are for *untrusted byte inputs* — request bodies, path/query
params, raw CLI flags parsed as structured data. A pure helper that takes a
typed argument and returns a typed result is better served by a table test.
When you add a new handler, CLI parser, or repository method that ingests
arbitrary bytes, add a `FuzzXxx` target alongside the unit tests.

## Conventions checklist

When adding or changing a test:

- [ ] Same package as the code under test (white-box) unless there is a
      reason to be black-box.
- [ ] No real server process, no on-disk database, no external network.
- [ ] Reuse the shared helpers (`newMemoryDB`, `newTestEcho`, `do`,
      `newTestClient`) instead of re-rolling setup.
- [ ] Use `t.Helper()` in test helpers and `t.Cleanup` for teardown.
- [ ] Assert on status codes, decoded JSON bodies, and error types
      (`errors.Is`, type assertions for `*APIError`) — not on raw strings.
- [ ] Prefer table-driven subtests when there are several cases of the same
      shape.
- [ ] `go vet ./...`, `gofmt -l -s .` (clean), and `go test -race ./...`
      (green) before declaring done.
- [ ] Any function ingesting untrusted bytes (request bodies, path
      params, raw CLI `--data`) has a `FuzzXxx` target covering it.
