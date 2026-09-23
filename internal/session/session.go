package session

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

	"madbox-player-profile/internal/event"
	"madbox-player-profile/internal/player"
)

// ProfileGetter only needs to get player profiles
type ProfileGetter interface {
	Get(ctx context.Context, id string) (player.Player, error)
}

// EventReader only needs to list and count recent events
type EventReader interface {
	ListRecent(ctx context.Context, playerID string, f event.Filter) ([]event.Event, error)
	CountByType(ctx context.Context, playerID string, since time.Time) (map[string]int64, error)
}

type Summary struct {
	Profile      player.Player
	RecentEvents []event.Event
	CountsByType map[string]int64
	WindowStart  time.Time
}

type Config struct {
	RecentLimit int           // default 20
	Window      time.Duration // default 7d
}

type Service struct {
	profiles ProfileGetter
	events   EventReader
	cfg      Config
	now      func() time.Time
}

func NewService(p ProfileGetter, e EventReader, cfg Config) *Service {
	if cfg.RecentLimit <= 0 {
		cfg.RecentLimit = 20
	}
	if cfg.Window <= 0 {
		cfg.Window = 7 * 24 * time.Hour
	}
	return &Service{profiles: p, events: e, cfg: cfg, now: time.Now}
}

// Summarize fans out the three reads concurrently. The profile lookup
// decides the status (404 vs 200); the two event reads only fail on
// infrastructure errors.
func (s *Service) Summarize(ctx context.Context, playerID string) (Summary, error) {
	if err := player.ValidateID(playerID); err != nil {
		return Summary{}, err
	}

	since := s.now().UTC().Add(-s.cfg.Window)
	out := Summary{WindowStart: since}

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		p, err := s.profiles.Get(gctx, playerID)
		if err != nil {
			return err // player.ErrNotFound passes through unwrapped
		}
		out.Profile = p
		return nil
	})
	g.Go(func() error {
		evs, err := s.events.ListRecent(gctx, playerID, event.Filter{Limit: s.cfg.RecentLimit})
		if err != nil {
			return fmt.Errorf("recent events: %w", err)
		}
		out.RecentEvents = evs
		return nil
	})
	g.Go(func() error {
		counts, err := s.events.CountByType(gctx, playerID, since)
		if err != nil {
			return fmt.Errorf("event counts: %w", err)
		}
		out.CountsByType = counts
		return nil
	})

	if err := g.Wait(); err != nil {
		if errors.Is(err, player.ErrNotFound) {
			return Summary{}, player.ErrNotFound
		}
		return Summary{}, err
	}
	return out, nil
}
