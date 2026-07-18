package main

import (
	"context"
	"fmt"
	"os"

	"github.com/rabbitvictor/go-amp-test/internal/infrastructure"
	"github.com/rabbitvictor/go-amp-test/internal/server"
)

const (
	defaultPort    = "8080"
	defaultService = "go-amp-test"
	defaultVersion = "0.1.0"
	defaultDBPath  = "app.db"
)

func main() {
	port := envOr("PORT", defaultPort)
	service := envOr("SERVICE_NAME", defaultService)
	version := envOr("SERVICE_VERSION", defaultVersion)
	dbPath := envOr("DB_PATH", defaultDBPath)

	ctx := context.Background()

	db, err := infrastructure.OpenDB(ctx, infrastructure.DBConfig{Path: dbPath})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	e := server.New(server.Config{
		Service: service,
		Version: version,
		DB:      db,
	})

	addr := fmt.Sprintf(":%s", port)
	e.Logger.Info(fmt.Sprintf("starting %s on %s", service, addr),
		"version", version,
	)
	// e.Start handles SIGINT/SIGTERM and shuts down gracefully.
	if err := e.Start(addr); err != nil {
		e.Logger.Error(err.Error())
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
