package player

import (
	"context"
	"errors"
	"testing"
	"time"
)

type FakeRepo struct {
	createFn func(context.Context, Player) (bool, error)
	updateFn func(context.Context, string, Patch, time.Time) (Player, error)
	calls    int
}

func (f *FakeRepo) Create(ctx context.Context, p Player) (bool, error) {
	f.calls++
	return f.createFn(ctx, p)
}

func (f *FakeRepo) Get(ctx context.Context, ID string) (Player, error) {
	ts := time.Now().UTC().Truncate(time.Millisecond)
	return Player{
		ID:        ID,
		CreatedAt: ts,
		UpdatedAt: ts,
	}, nil
}

func (f *FakeRepo) Exists(ctx context.Context, ID string) (bool, error) {
	return true, nil
}

func (f *FakeRepo) Update(ctx context.Context, ID string, patch Patch, updatedAt time.Time) (Player, error) {
	f.calls++
	return f.updateFn(ctx, ID, patch, updatedAt)
}

func TestService_CreateWithBadId(t *testing.T) {
	repo := &FakeRepo{createFn: func(context.Context, Player) (bool, error) {
		t.Fatal("repo must not be called while having an invalid id")
		return false, nil
	}}
	svc := NewService(repo)

	_, err := svc.Create(context.Background(), "!!!")

	if !errors.Is(err, ErrInvalidID) {
		t.Fatalf("got %v, want ErrInvalidID", err)
	}
	if repo.calls != 0 {
		t.Fatalf("repo create called %d time(s), want 0", repo.calls)
	}
}

func TestService_GetWithBadId(t *testing.T) {
	repo := &FakeRepo{createFn: func(context.Context, Player) (bool, error) {
		t.Fatal("repo must not be called while having an invalid id")
		return false, nil
	}}
	svc := NewService(repo)

	_, err := svc.Get(context.Background(), "!!!")

	if !errors.Is(err, ErrInvalidID) {
		t.Fatalf("got %v, want ErrInvalidID", err)
	}
	if repo.calls != 0 {
		t.Fatalf("repo create called %d time(s), want 0", repo.calls)
	}
}

func TestService_UpdateWithInvalidPatch(t *testing.T) {
	repo := &FakeRepo{updateFn: func(context.Context, string, Patch, time.Time) (Player, error) {
		t.Fatal("repo must not be called while having an invalid patch")
		return Player{}, nil
	}}
	svc := NewService(repo)

	_, err := svc.Update(context.Background(), "123456", Patch{})

	if !errors.Is(err, ErrInvalidPatch) {
		t.Fatalf("got %v, want ErrInvalidID", err)
	}
	if repo.calls != 0 {
		t.Fatalf("repo update called %d time(s), want 0", repo.calls)
	}
}

func TestService_UpdateWithValidPatch(t *testing.T) {
	repo := &FakeRepo{updateFn: func(ctx context.Context, ID string, p Patch, ts time.Time) (Player, error) {
		if p.validate() == nil {
			return Player{DisplayName: *p.DisplayName, UpdatedAt: ts}, nil
		}
		return Player{}, ErrInvalidPatch
	}}
	svc := NewService(repo)
	ts := time.Now().UTC()
	svc.now = func() time.Time {
		return ts
	}

	patchDisplayName := "hello"
	p, err := svc.Update(context.Background(), "123456", Patch{DisplayName: &patchDisplayName})
	if err != nil {
		t.Fatalf("expected no error, received: %v", err)
	}
	if !p.UpdatedAt.Equal(ts.Truncate(time.Millisecond)) {
		t.Fatalf("expected updatedAt: %v, got %v", ts.Truncate(time.Millisecond), p.UpdatedAt)
	}
}
