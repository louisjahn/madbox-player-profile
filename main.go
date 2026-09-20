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

	mongocli "github.com/MadBox-Games/backend-engineer-hiring/internal/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func main() {
	addr := getenv("HTTP_ADDR", ":8080")
	mongoURI := getenv("MONGO_URI", "mongodb://localhost:27017")

	connectCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongocli.Connect(connectCtx, mongoURI)
	if err != nil {
		log.Fatalf("mongo connect: %v", err)
	}

	server := &http.Server{
		Addr:    addr,
		Handler: newMux(client),
	}

	// Graceful shutdown
	go func() {
		log.Printf("server listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server forced shutdown: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("graceful server shutdown...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to suhdown with active connections: %v", err)
	}

	log.Println("server shutdown finished")
}

// newMux builds the HTTP handler tree with the Mongo client wired in.
// Candidate: register your handlers here. Extracting this from main()
// keeps the handler wiring testable — see main_test.go.
func newMux(client *mongo.Client) *http.ServeMux {
	mux := http.NewServeMux()

	// Example handler showing how to thread `client` into a handler.
	// Keep, replace, or delete — it's a reference, not a requirement.
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := client.Ping(r.Context(), nil); err != nil {
			http.Error(w, "mongo unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	return mux
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
