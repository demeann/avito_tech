package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/demeann/avito-pr-reviewer/internal/config"
	"github.com/demeann/avito-pr-reviewer/internal/httpserver"
	"github.com/demeann/avito-pr-reviewer/internal/repository/postgres"
	"github.com/demeann/avito-pr-reviewer/internal/service"
)

func main() {
	ctx := context.Background()

	cfg := config.Load()

	pool, err := postgres.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to init db: %v", err)
	}
	defer pool.Close()

	if err := postgres.AutoMigrate(ctx, pool); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	repo := postgres.NewRepository(pool)
	svc := service.NewService(repo)

	handler := httpserver.NewHandler(svc)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler.Router(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("starting server on :%s", cfg.Port)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("server error: %v", err)
		os.Exit(1)
	}
}
