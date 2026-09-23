package event

import (
	"context"
	"time"
)

type Repository interface {
	Insert(ctx context.Context, e Event) (Event, error)
	// ListRecent returns up to f.Limit events, newest first.
	ListRecent(ctx context.Context, playerID string, f Filter) ([]Event, error)
	// CountByType groups events with ts >= since by type.
	CountByType(ctx context.Context, playerID string, since time.Time) (map[string]int64, error)
}
