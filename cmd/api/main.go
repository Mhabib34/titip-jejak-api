package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"titip-jejak-api/cmd/wire"
	"titip-jejak-api/internal/logger"

	"time"
)

func main() {
	// ── 0. Init logger ────────────────────────────────────────────────────────
	logger.Init()
	log := logger.Get()

	// ── 1. Wire — inisialisasi semua dependency ───────────────────────────────
	engine, cleanup, err := wire.InitializeApp()
	if err != nil {
		logger.Fatal("failed to initialize app", "error", err)
	}

	// ── 2. Port dari env, default 8080 ────────────────────────────────────────
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// ── 3. HTTP server ────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      engine,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("server starting", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server listen error", "error", err)
		}
	}()

	// ── 4. Graceful shutdown ──────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("server shutting down...")

	cleanup()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Fatal("server forced shutdown", "error", err)
	}

	log.Info("server exited cleanly")
}