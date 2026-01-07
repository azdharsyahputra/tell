package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tell/internal/auth"
	"tell/internal/config"
	"tell/internal/db"
	httpx "tell/internal/http"
	"tell/internal/jobs"
)

func main() {
	cfg, _ := config.Load()

	gdb, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	if err := db.AutoMigrateAndIndexes(gdb); err != nil {
		log.Fatal(err)
	}

	jwtSvc := auth.NewJWT(cfg.JWTSecret)
	r := httpx.NewRouter(cfg, gdb, jwtSvc)

	// worker
	jobsRepo := &jobs.Repo{DB: gdb}
	worker := &jobs.Worker{ID: "worker-1", Repo: jobsRepo, DB: gdb}

	ctx, cancel := context.WithCancel(context.Background())
	go worker.Run(ctx)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("listening on %s\n", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// graceful shutdown
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch

	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
}
