#!/usr/bin/env bash
# cli_smoke.sh — end-to-end smoke test for the go-amp-test API.
#
# Builds the CLI, starts the server, then drives the endpoints through the
# CLI entrypoint:
#   1. health check
#   2. create an item, then GET it back by id and verify the name matches
#   3. list items and confirm the created item is present
#   4. GET a non-existent id and confirm the CLI exits non-zero (404)
#
# The server can be built two ways:
#   * Docker (default when available): builds the server image from the
#     repo's Dockerfile and runs it in an ephemeral container.
#   * Local go build (fallback): builds ./cmd/server and runs it as a
#     background process against an isolated temp SQLite DB.
#
# The CLI is always built locally with go build; the Dockerfile only
# produces the server binary.
#
# Usage:
#   scripts/cli_smoke.sh                  # docker if available, else local
#   USE_DOCKER=1 scripts/cli_smoke.sh     # force docker (error if absent)
#   USE_DOCKER=0 scripts/cli_smoke.sh     # force local go build
#   TEST_PORT=19000 scripts/cli_smoke.sh  # pin the host port
#   IMAGE_TAG=go-amp-test:smoke scripts/cli_smoke.sh  # override image tag
set -euo pipefail

# --- helpers ---------------------------------------------------------------
c_red()   { printf '\033[31m%s\033[0m' "$*"; }
c_green() { printf '\033[32m%s\033[0m' "$*"; }
c_bold()  { printf '\033[1m%s\033[0m' "$*"; }

log()  { printf '%s\n' "$*" >&2; }
ok()   { printf '  [%s] %s\n' "$(c_green PASS)" "$*" >&2; }
fail() { printf '  [%s] %s\n' "$(c_red FAIL)" "$*" >&2; }
step() { printf '\n%s %s\n' "$(c_bold '==>')" "$*" >&2; }

failures=0
assert_eq() {
  local label="$1" expected="$2" actual="$3"
  if [[ "$expected" == "$actual" ]]; then
    ok "$label ($actual)"
  else
    fail "$label: expected '$expected', got '$actual'"
    failures=$((failures + 1))
  fi
}

# --- setup -----------------------------------------------------------------
repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

if ! command -v jq >/dev/null 2>&1; then
  log "error: jq is required (apt-get install jq / brew install jq)"
  exit 2
fi
if ! command -v go >/dev/null 2>&1; then
  log "error: go is required to build the CLI client"
  exit 2
fi

workdir="$(mktemp -d -t go-amp-smoke-XXXXXX)"
cli_bin="$workdir/go-amp-test"
server_bin="$workdir/go-amp-server"
db_path="$workdir/app.db"
server_log="$workdir/server.log"
image_tag="${IMAGE_TAG:-go-amp-test:smoke}"
container_name=""
server_pid=""
# How the server was started: "docker" or "local".
server_kind=""

cleanup() {
  if [[ "$server_kind" == "docker" && -n "$container_name" ]]; then
    docker rm -f "$container_name" >/dev/null 2>&1 || true
  elif [[ -n "$server_pid" ]] && kill -0 "$server_pid" 2>/dev/null; then
    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
  fi
  rm -rf "$workdir"
}
trap cleanup EXIT

# Pick a free TCP port unless TEST_PORT is set.
pick_port() {
  if [[ -n "${TEST_PORT:-}" ]]; then
    echo "$TEST_PORT"
    return
  fi
  # Let the kernel hand us an ephemeral port, then close it. Small race, but
  # fine for a local smoke test.
  python3 - <<'PY' 2>/dev/null || { log "error: need python3 or TEST_PORT set"; exit 2; }
import socket
s = socket.socket()
s.bind(("", 0))
print(s.getsockname()[1])
s.close()
PY
}

# Decide whether to use Docker for the server.
#   USE_DOCKER unset -> auto (docker if available)
#   USE_DOCKER=1     -> force docker (error if absent)
#   USE_DOCKER=0     -> force local go build
# Returns: 0 = use docker, 1 = use local, 2 = forced docker but missing.
use_docker() {
  case "${USE_DOCKER:-}" in
    1) command -v docker >/dev/null 2>&1 && return 0
       log "error: USE_DOCKER=1 but docker not found"
       return 2 ;;
    0) return 1 ;;
    *) command -v docker >/dev/null 2>&1 ;;
  esac
}

port="$(pick_port)"
base_url="http://localhost:${port}"

step "building CLI binary"
if ! CGO_ENABLED=0 go build -o "$cli_bin" ./cmd/cli; then
  fail "go build ./cmd/cli"
  exit 1
fi
ok "built cli ($cli_bin)"

build_server_docker() {
  step "building server image via Docker ($image_tag)"
  if ! docker build -t "$image_tag" "$repo_root"; then
    fail "docker build"
    return 1
  fi
  ok "built image $image_tag"
}

build_server_local() {
  step "building server binary via go build"
  if ! CGO_ENABLED=0 go build -o "$server_bin" ./cmd/server; then
    fail "go build ./cmd/server"
    return 1
  fi
  ok "built server ($server_bin)"
}

start_server_docker() {
  container_name="go-amp-smoke-$$-$RANDOM"
  : > "$server_log"
  # The image's ENTRYPOINT is the server; it listens on 8080 by default
  # (ENV PORT=8080) and writes its SQLite file to /data (isolated per
  # container). We map the host port and tail logs to our log file.
  if ! docker run -d --name "$container_name" \
        -p "${port}:8080" \
        -e DB_JOURNAL_MODE=MEMORY \
        "$image_tag" >>"$server_log" 2>&1; then
    fail "docker run"
    return 1
  fi
  server_kind="docker"
}

start_server_local() {
  : > "$server_log"
  PORT="$port" DB_PATH="$db_path" DB_JOURNAL_MODE=MEMORY "$server_bin" >>"$server_log" 2>&1 &
  server_pid=$!
  server_kind="local"
}

# Wait for the server to accept connections (up to ~30s).
wait_ready() {
  local i
  for ((i = 0; i < 60; i++)); do
    if [[ "$server_kind" == "local" ]] && ! kill -0 "$server_pid" 2>/dev/null; then
      return 1
    fi
    if "$cli_bin" --server "$base_url" health >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.5
  done
  return 1
}

case "$(use_docker; echo $?)" in
  0)
    if ! build_server_docker; then exit 1; fi
    if ! start_server_docker; then
      cat "$server_log" >&2 || true
      exit 1
    fi
    step "starting server container ($container_name) -> :$port"
    ;;
  1)
    if ! build_server_local; then exit 1; fi
    step "starting server on :$port (db: $db_path)"
    start_server_local
    ;;
  *)
    # Forced docker but docker is absent; use_docker already logged.
    exit 2
    ;;
esac

if ! wait_ready; then
  fail "server did not become healthy"
  log "--- server log ---"
  cat "$server_log" >&2 || true
  if [[ "$server_kind" == "docker" && -n "$container_name" ]]; then
    log "--- docker logs (last 40 lines) ---"
    docker logs --tail 40 "$container_name" >&2 || true
  fi
  exit 1
fi
ok "server is up ($server_kind)"

# CLI wrapper that always targets our test server.
cli() { "$cli_bin" --server "$base_url" --format compact "$@"; }

# --- tests -----------------------------------------------------------------

step "GET /health via CLI"
health_json="$(cli health)"
assert_eq "health status"  "up"          "$(printf '%s' "$health_json" | jq -r '.status')"
assert_eq "health service" "go-amp-test" "$(printf '%s' "$health_json" | jq -r '.service')"
assert_eq "health version" "0.1.0"       "$(printf '%s' "$health_json" | jq -r '.version')"

step "POST /items then GET /items/:id"
item_name="smoke-item-$$"
create_json="$(cli items create -n "$item_name")"
created_id="$(printf '%s' "$create_json" | jq -r '.id')"
assert_eq "created item name" "$item_name" "$(printf '%s' "$create_json" | jq -r '.name')"
if [[ -z "$created_id" || "$created_id" == "null" ]]; then
  fail "could not parse created item id"
  failures=$((failures + 1))
fi
log "  created id=$created_id"

get_json="$(cli items get "$created_id")"
assert_eq "get item id"   "$created_id" "$(printf '%s' "$get_json" | jq -r '.id')"
assert_eq "get item name" "$item_name"  "$(printf '%s' "$get_json" | jq -r '.name')"

step "GET /items (list) contains the created item"
list_json="$(cli items list)"
list_count="$(printf '%s' "$list_json" | jq 'length')"
log "  list length=$list_count"
if (( list_count < 1 )); then
  fail "expected at least one item in list"
  failures=$((failures + 1))
else
  ok "list has $list_count item(s)"
fi
found="$(printf '%s' "$list_json" | jq -r --arg id "$created_id" '.[] | select(.id == ($id|tonumber)) | .name')"
assert_eq "created item present in list" "$item_name" "$found"

step "GET /items/:id for non-existent id returns non-zero"
if cli items get 999999 >/dev/null 2>&1; then
  fail "expected cli items get 999999 to exit non-zero"
  failures=$((failures + 1))
else
  ok "cli items get 999999 exited non-zero (404 as expected)"
fi

# --- summary ---------------------------------------------------------------
echo >&2
if (( failures == 0 )); then
  printf '%s all smoke tests passed (%s server)\n' "$(c_green '✓')" "$server_kind" >&2
  exit 0
else
  printf '%s %d smoke test(s) failed (%s server)\n' "$(c_red '✗')" "$failures" "$server_kind" >&2
  exit 1
fi
