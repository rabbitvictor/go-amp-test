### Build stage: compile the Go binary.
# Uses the official Go 1.26 image (Debian-based, non-Alpine) because
# modernc.org/sqlite is pure Go (no CGo), so we do not need gcc or musl.
FROM golang:1.26 AS build

WORKDIR /src

# Cache dependencies: copy module files first and download before adding
# the rest of the source. This layer is reused as long as go.mod/go.sum
# are unchanged.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source and build a static binary.
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

### Runtime stage: minimal image with just the binary and CA certs.
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

COPY --from=build /out/server /server

# Default runtime configuration. The SQLite file lives in /data so it can be
# mounted as a volume; the path can be overridden via DB_PATH.
ENV PORT=8080 \
    DB_PATH=/data/app.db \
    SERVICE_NAME=go-amp-test \
    SERVICE_VERSION=0.1.0

EXPOSE 8080

# /data is the volume mount point for the persistent SQLite file.
VOLUME ["/data"]

ENTRYPOINT ["/server"]
