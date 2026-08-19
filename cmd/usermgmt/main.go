package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"github.com/npbtrac/demo-go-user-management/internal/config"
	"github.com/npbtrac/demo-go-user-management/internal/db"
	grpcsvc "github.com/npbtrac/demo-go-user-management/internal/grpc"
	httpserver "github.com/npbtrac/demo-go-user-management/internal/http"
	"github.com/npbtrac/demo-go-user-management/internal/user"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("usermgmt exited", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := db.Migrate(cfg.DatabaseURL); err != nil {
		return err
	}
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return err
	}

	svc := user.NewService(user.NewPostgresRepository(pool))
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return httpserver.NewServer("public", cfg.PublicHTTPAddr, httpserver.PublicMux(svc)).Serve(ctx)
	})
	g.Go(func() error {
		return httpserver.NewServer("internal", cfg.InternalHTTPAddr, httpserver.InternalMux(svc)).Serve(ctx)
	})
	g.Go(func() error {
		slog.Info("grpc listening", "addr", cfg.GRPCAddr)
		return grpcsvc.ListenAndServe(ctx, cfg.GRPCAddr, svc)
	})
	return g.Wait()
}
