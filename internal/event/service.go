package event

import (
	"context"
	"time"

	"madbox-player-profile/internal/player"
)

// PlayerChecker only uses the Exists method in player package, satisfied by *player.Service
type PlayerChecker interface {
	Exists(ctx context.Context, id string) (bool, error)
}

type Service struct {
	repo    Repository
	players PlayerChecker
	now     func() time.Time
}

func NewService(repo Repository, players PlayerChecker) *Service {
	return &Service{repo: repo, players: players, now: time.Now}
}

func (s *Service) Record(ctx context.Context, playerID string, req RecordRequest) (Event, error) {
	if err := player.ValidateID(playerID); err != nil {
		return Event{}, err
	}
	now := s.now().UTC().Truncate(time.Millisecond)
	if err := req.validate(now); err != nil {
		return Event{}, err
	}

	// One cheap round-trip so we never store events for a player that doesn't exist.
	// Introduce caching when traffic gets too large.
	ok, err := s.players.Exists(ctx, playerID)
	if err != nil {
		return Event{}, err
	}
	if !ok {
		return Event{}, player.ErrNotFound
	}

	ts := now
	if req.Timestamp != nil {
		ts = req.Timestamp.UTC().Truncate(time.Millisecond)
	}
	return s.repo.Insert(ctx, Event{
		PlayerID:   playerID,
		Type:       req.Type,
		Timestamp:  ts,
		RecordedAt: now,
		Payload:    req.Payload,
	})
}

func (s *Service) List(ctx context.Context, playerID string, f Filter) ([]Event, error) {
	if err := player.ValidateID(playerID); err != nil {
		return nil, err
	}
	return s.repo.ListRecent(ctx, playerID, f.normalized())
}

func (s *Service) CountByType(ctx context.Context, playerID string, since time.Time) (map[string]int64, error) {
	if err := player.ValidateID(playerID); err != nil {
		return nil, err
	}
	return s.repo.CountByType(ctx, playerID, since)
}
