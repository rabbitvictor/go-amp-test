package main

import (
	"context"
	"fmt"
	"os"

	"github.com/rabbitvictor/go-amp-test/internal/config"
	"github.com/rabbitvictor/go-amp-test/internal/infrastructure"
	"github.com/rabbitvictor/go-amp-test/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to load config:", err)
		os.Exit(1)
	}

	ctx := context.Background()

	db, err := infrastructure.OpenDB(ctx, infrastructure.DBConfig{
		Path:         cfg.DB.Path,
		MaxOpenConns: cfg.DB.MaxOpenConns,
		DSN:          cfg.DB.DSN(),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to open database:", err)
		os.Exit(1)
	}
	defer db.Close()

	e := server.New(server.Config{
		Service: cfg.Server.ServiceName,
		Version: cfg.Server.Version,
		DB:      db,
	})

	addr := cfg.Server.Addr()
	e.Logger.Info(fmt.Sprintf("starting %s on %s", cfg.Server.ServiceName, addr),
		"version", cfg.Server.Version,
	)
	// e.Start handles SIGINT/SIGTERM and shuts down gracefully.
	if err := e.Start(addr); err != nil {
		e.Logger.Error(err.Error())
		os.Exit(1)
	}
}
