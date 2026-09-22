package player

import (
	"context"
	"time"
)

type Repository interface {
	// Returns true if a new player was created, must be fetched later.
	Create(ctx context.Context, p Player) (created bool, err error)
	// Returns the player associated to the given ID.
	Get(ctx context.Context, ID string) (Player, error)
	Exists(ctx context.Context, ID string) (bool, error)
	// Returns the player after the write.
	Update(ctx context.Context, ID string, patch Patch, updatedAt time.Time) (Player, error)
}
