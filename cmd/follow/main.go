package main

import (
	"context"
	"log"
	"os"
	"net/http"
	"os/signal"
	"syscall"
	"time"
	"instant/internal/follow"
)

func main() {
	// --- Create server ---
	srv := follow.NewServer()

	// --- Run server in goroutine ---
	go func() {
		log.Printf("Follow Service listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
    		log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// --- Graceful shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down Follow Service...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Forced shutdown failed: %v", err)
	}

	log.Println("Follow Service stopped gracefully.")
}
