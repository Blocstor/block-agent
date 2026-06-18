package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/blocstor/bloc-agent/internal/api"
)

func main() {
	listen := flag.String("listen", ":8080", "address to listen on")
	logLevel := flag.String("log-level", "info", "log level: debug, info, warn, error")
	thinpool := flag.String("thinpool", "", "LVM thin pool name for thin-provisioned LV creation (optional)")
	flag.Parse()

	level := parseLogLevel(*logLevel)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	srv := api.NewServer(logger, *thinpool)

	httpSrv := &http.Server{
		Addr:    *listen,
		Handler: srv,
	}

	// Start server in background.
	go func() {
		logger.Info("bloc-agent starting", "addr", *listen)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpSrv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "err", err)
		os.Exit(1)
	}
	logger.Info("stopped")
}

func parseLogLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
