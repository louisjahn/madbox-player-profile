package mongorepo

import (
	"context"
	"testing"
	"time"

	"madbox-player-profile/internal/event"
	mongocli "madbox-player-profile/internal/mongo"
)

func seed(t *testing.T, repo *Repo, playerID string, base time.Time, types ...string) {
	t.Helper()
	for i, typ := range types {
		_, err := repo.Insert(context.Background(), event.Event{
			PlayerID:  playerID,
			Type:      typ,
			Timestamp: base.Add(time.Duration(i) * time.Minute),
			Payload:   map[string]any{"i": i},
		})
		if err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}
}

func TestListRecent_OrderLimitAndTypeFilter(t *testing.T) {
	client, _ := mongocli.Connect(context.Background(), "mongodb://localhost:27017")
	repo := NewRepo(client.Database("events"))
	ctx := context.Background()
	if err := repo.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}

	base := time.Now().UTC().Truncate(time.Millisecond)
	seed(t, repo, "p1", base, "level_start", "level_end", "purchase", "level_start")
	seed(t, repo, "p2", base, "level_start") // must never leak into p1's list

	got, err := repo.ListRecent(ctx, "p1", event.Filter{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("limit not applied: got %d events", len(got))
	}
	if !got[0].Timestamp.After(got[1].Timestamp) {
		t.Fatalf("not newest-first: %v then %v", got[0].Timestamp, got[1].Timestamp)
	}

	got, err = repo.ListRecent(ctx, "p1", event.Filter{Type: "level_start", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("type filter: got %d, want 2", len(got))
	}
	for _, e := range got {
		if e.PlayerID != "p1" || e.Type != "level_start" {
			t.Fatalf("unexpected event in result: %+v", e)
		}
	}
}

func TestCountByType_RespectsWindow(t *testing.T) {
	client, _ := mongocli.Connect(context.Background(), "mongodb://localhost:27017")
	repo := NewRepo(client.Database("events"))
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Millisecond)
	// Two old events outside the window, three inside.
	seed(t, repo, "p1", now.Add(-10*24*time.Hour), "purchase", "purchase")
	seed(t, repo, "p1", now.Add(-time.Hour), "level_start", "level_start", "purchase")

	counts, err := repo.CountByType(ctx, "p1", now.Add(-7*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if counts["level_start"] != 2 || counts["purchase"] != 1 {
		t.Fatalf("counts = %v, want level_start:2 purchase:1", counts)
	}

	empty, err := repo.CountByType(ctx, "nobody", now.Add(-7*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if empty == nil || len(empty) != 0 {
		t.Fatalf("want empty non-nil map, got %v", empty)
	}
}
