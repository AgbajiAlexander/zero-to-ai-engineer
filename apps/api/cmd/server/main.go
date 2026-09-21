package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zero-to-ai-engineer/api/internal/config"
	"github.com/zero-to-ai-engineer/api/internal/database"
)

func readyHandlerWithPing(pingFn func(context.Context) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := pingFn(ctx); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"not_ready"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	}
}

func readyHandler(db *database.DB) http.HandlerFunc {
	return readyHandlerWithPing(func(ctx context.Context) error {
		if db == nil || db.Pool == nil {
			return errors.New("database unavailable")
		}
		return db.Pool.Ping(ctx)
	})
}

func main() {
	cfg := config.Load()
	port := cfg.AppPort

	startupCtx, startupCancel := context.WithTimeout(context.Background(), 5*time.Second)
	db, err := database.Connect(startupCtx, cfg)
	startupCancel()
	if err != nil {
		log.Fatalf("database startup failed: %v", err)
	}

	mux := http.NewServeMux()

	// Go 1.22+ supports method routing in the standard library
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("GET /ready", readyHandler(db))

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
		// Reasonable HTTP server timeouts to prevent slow-client attacks
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Channel to listen for errors coming from the listener.
	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("Starting server on port %s", port)
		serverErrors <- srv.ListenAndServe()
	}()

	// Channel to listen for an interrupt or terminate signal from the OS.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Blocking main and waiting for shutdown signal or server error.
	select {
	case err := <-serverErrors:
		if db != nil {
			db.Close()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server startup failed: %v", err)
		}
	case <-shutdown:
		log.Println("Initiating graceful shutdown...")

		// Create a context with a timeout for the shutdown process.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Attempt to gracefully shutdown the server.
		if err := srv.Shutdown(ctx); err != nil {
			if db != nil {
				db.Close()
			}
			log.Fatalf("Graceful shutdown failed: %v", err)
		}

		if db != nil {
			db.Close()
		}
		log.Println("Server stopped cleanly")
	}
}
