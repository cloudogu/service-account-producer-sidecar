package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloudogu/service-account-producer-sidecar/internal/configuration"
	"github.com/cloudogu/service-account-producer-sidecar/internal/serviceaccount"
)

func main() {
	config, err := configuration.ReadFromEnv()
	if err != nil {
		panic(err)
	}

	configureLogger(config.LogLevel)

	runner := &serviceaccount.HookRunner{
		CreateHook: config.CreateHook,
		DeleteHook: config.DeleteHook,
		ExistsHook: config.ExistsHook,
		Timeout:    config.HookTimeout,
	}

	srv := serviceaccount.CreateServer(config.Addr, config.ApiKey, runner)

	go func() {
		slog.Info("service-account-producer-sidecar started", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("error starting server", "err", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("error stopping server", "err", err)
	}
}

func configureLogger(levelName string) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(levelName)); err != nil {
		slog.Error("error parsing log level, defaulting to INFO", "err", err)
		level = slog.LevelInfo
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)
	slog.Info("configured logger", "level", level.String())
}
