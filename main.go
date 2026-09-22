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

	mongocli "madbox-player-profile/internal/mongo"
	"madbox-player-profile/internal/player"
	"madbox-player-profile/internal/player/mongorepo"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	addr := getenv("HTTP_ADDR", ":8080")
	mongoURI := getenv("MONGO_URI", "mongodb://localhost:27017")

	connectCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongocli.Connect(connectCtx, mongoURI)
	if err != nil {
		slog.Error("mongo connect", "err", err)
	}
	defer client.Disconnect(context.Background())

	server := &http.Server{
		Addr:              addr,
		Handler:           newMux(client),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Graceful shutdown
	go func() {
		slog.Info("server listening", "addr", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server forced shutdown", "err", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	slog.Info("graceful server shutdown...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server forced to suhtdown with active connections", "err", err)
	}

	slog.Info("server shutdown finished")
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

	db := client.Database(getenv("MONGO_DB", "profiles"))
	playerSvc := player.NewService(mongorepo.NewRepo(db))
	player.NewHandler(playerSvc).Routes(mux)

	return mux
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
