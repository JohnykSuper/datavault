package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/your-org/datavault/internal/api"
	"github.com/your-org/datavault/internal/auth"
	"github.com/your-org/datavault/internal/cache"
	"github.com/your-org/datavault/internal/config"
	"github.com/your-org/datavault/internal/domain/service"
	"github.com/your-org/datavault/internal/hsm"
	"github.com/your-org/datavault/internal/logger"
	"github.com/your-org/datavault/internal/repository"
)

func init() {
	// Загружаем .env если файл существует (dev-режим).
	// В production переменные задаются снаружи — ошибка игнорируется намеренно.
	_ = godotenv.Load()
}

func main() {
	cfg := config.Load()
	log := logger.New(cfg.LogLevel)

	hsmClient, err := hsm.New(cfg)
	if err != nil {
		log.Fatal("failed to init HSM", "error", err)
	}

	dekCache := cache.NewDEKCache(cfg.DEKCacheTTL)

	repos, err := repository.New(cfg)
	if err != nil {
		log.Fatal("failed to init repository", "error", err)
	}

	svc := service.New(hsmClient, dekCache, repos.Records, repos.Audit, log, cfg)

	keyValidator, err := auth.NewStaticValidator(map[string]string{
		cfg.APIKey: "service",
	})
	if err != nil {
		log.Fatal("failed to init key validator", "error", err)
	}

	router := api.NewRouter(svc, log, repos.Pinger, hsmClient, keyValidator)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("starting DataVault", "port", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("server shutdown error", "error", err)
	}
	log.Info("DataVault stopped")
}
