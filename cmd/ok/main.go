package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/AaronKaa/ok/internal/config"
	"github.com/AaronKaa/ok/internal/health"
	"github.com/AaronKaa/ok/internal/scheduler"
	"github.com/AaronKaa/ok/internal/server"
)

type App struct {
	Config    *config.Config
	Service   *health.Service
	Scheduler *scheduler.Scheduler
	Server    *server.Server

	shutdownTimeout time.Duration
}

func NewApp() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	if err := config.Validate(cfg); err != nil {
		return nil, err
	}

	slog.Info("configuration loaded",
		"port", cfg.Port,
		"check_count", len(cfg.Checks),
		"refresh_interval", cfg.RefreshInterval,
	)

	checks := make([]health.Check, len(cfg.Checks))
	for i, def := range cfg.Checks {
		checks[i] = health.NewCheck(def)

		slog.Debug("registered check",
			"id", checks[i].ID,
			"title", checks[i].Title,
			"url", checks[i].URL,
			"interval", checks[i].Interval,
			"critical", checks[i].Critical,
		)
	}

	httpTimeout := 10 * time.Second
	checker := health.NewHTTPChecker(httpTimeout)
	service := health.NewService(checks, checker)

	sched := scheduler.New(service)
	handler := server.NewHandler(service, cfg.RefreshInterval)
	srv := server.New(cfg.Port, handler)

	return &App{
		Config:          cfg,
		Service:         service,
		Scheduler:       sched,
		Server:          srv,
		shutdownTimeout: 10 * time.Second,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	a.Scheduler.Start(ctx)

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		slog.Info("server starting", "port", a.Config.Port)
		return a.Server.Start()
	})

	g.Go(func() error {
		<-ctx.Done()

		slog.Info("shutdown initiated")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
		defer cancel()

		a.Scheduler.Stop()

		return a.Server.Shutdown(shutdownCtx)
	})

	if err := g.Wait(); err != nil {
		slog.Error("application error", "error", err)
		return err
	}

	slog.Info("shutdown complete")
	return nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	app, err := NewApp()
	if err != nil {
		slog.Error("failed to create app", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx); err != nil {
		os.Exit(1)
	}
}
