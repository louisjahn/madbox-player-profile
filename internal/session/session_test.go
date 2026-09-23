package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"madbox-player-profile/internal/event"
	"madbox-player-profile/internal/player"
)

type fakeProfiles struct {
	p   player.Player
	err error
}

func (f fakeProfiles) Get(context.Context, string) (player.Player, error) { return f.p, f.err }

type fakeEvents struct {
	gotSince time.Time
}

func (f *fakeEvents) ListRecent(context.Context, string, event.Filter) ([]event.Event, error) {
	return []event.Event{}, nil
}

func (f *fakeEvents) CountByType(_ context.Context, _ string, since time.Time) (map[string]int64, error) {
	f.gotSince = since
	return map[string]int64{}, nil
}

func TestSummarize_WindowIsComputedFromInjectedClock(t *testing.T) {
	frozen := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	ev := &fakeEvents{}
	s := NewService(fakeProfiles{p: player.Player{ID: "p1"}}, ev, Config{Window: 48 * time.Hour})
	s.now = func() time.Time { return frozen }

	sum, err := s.Summarize(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	want := frozen.Add(-48 * time.Hour)
	if !ev.gotSince.Equal(want) || !sum.WindowStart.Equal(want) {
		t.Fatalf("window start = %v, want %v", ev.gotSince, want)
	}
}

func TestSummarize_UnknownPlayerIsNotFound(t *testing.T) {
	s := NewService(fakeProfiles{err: player.ErrNotFound}, &fakeEvents{}, Config{})
	_, err := s.Summarize(context.Background(), "ghost")
	if !errors.Is(err, player.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
