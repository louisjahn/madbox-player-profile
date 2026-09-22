package player

import (
	"context"
	"strings"
	"time"
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

func (s *Service) Create(ctx context.Context, id string) (bool, error) {
	if err := ValidateID(id); err != nil {
		return false, err
	}

	ts := s.now().UTC().Truncate(time.Millisecond)
	p := Player{ID: id, CreatedAt: ts, UpdatedAt: ts}
	return s.repo.Create(ctx, p)
}

func (s *Service) Get(ctx context.Context, ID string) (Player, error) {
	if err := ValidateID(ID); err != nil {
		return Player{}, err
	}
	return s.repo.Get(ctx, ID)
}

func (s *Service) Exists(ctx context.Context, ID string) (bool, error) {
	if err := ValidateID(ID); err != nil {
		return false, err
	}
	return s.repo.Exists(ctx, ID)
}

func (s *Service) Update(ctx context.Context, ID string, patch Patch) (Player, error) {
	if err := ValidateID(ID); err != nil {
		return Player{}, err
	}
	if err := patch.validate(); err != nil {
		return Player{}, err
	}
	if patch.DisplayName != nil {
		trimmed := strings.TrimSpace(*patch.DisplayName)
		patch.DisplayName = &trimmed
	}
	return s.repo.Update(ctx, ID, patch, s.now().UTC().Truncate(time.Millisecond))
}
