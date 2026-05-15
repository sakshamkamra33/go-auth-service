package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sakshamkamra33/secure-auth/api"
	"github.com/sakshamkamra33/secure-auth/internal/auth"
	"github.com/sakshamkamra33/secure-auth/internal/config"
	"github.com/sakshamkamra33/secure-auth/internal/email"
	"github.com/sakshamkamra33/secure-auth/internal/logger"
	"github.com/sakshamkamra33/secure-auth/internal/middleware"
	"github.com/sakshamkamra33/secure-auth/internal/session"
	"github.com/sakshamkamra33/secure-auth/internal/storage"
)

func main() {
	cfg := config.Load()
	logger.Init(os.Getenv("DEBUG") == "true")
	slog.Info("starting secure-auth", "port", cfg.Port, "data_dir", cfg.DataDir)

	// Storage.
	store, err := storage.NewJSONStore(cfg.DataDir)
	if err != nil {
		slog.Error("storage init failed", "err", err)
		os.Exit(1)
	}

	// Audit log store.
	auditStore, err := storage.NewFileAuditStore(cfg.DataDir)
	if err != nil {
		slog.Error("audit store init failed", "err", err)
		os.Exit(1)
	}

	// Session manager (JWT).
	sessionMgr := session.NewManager(cfg.JWTSecret, cfg.AccessTokenExpiry, cfg.RefreshTokenExpiry)

	// Email mailer (console fallback in dev).
	mailer := email.New()

	// Auth service.
	authSvc := auth.NewService(store, auditStore, sessionMgr, mailer, cfg)

	// Router.
	rl := middleware.NewRateLimiter(cfg.RateLimitRequests, cfg.RateLimitWindow)
	h := api.NewHandler(authSvc, sessionMgr, store, auditStore, cfg)
	router := api.NewRouter(h, rl)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("shutting down gracefully…")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("forced shutdown", "err", err)
	}
	slog.Info("server stopped")
}
