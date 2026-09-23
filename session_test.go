package main

import (
	"net/http"
	"testing"
	"time"
)

type playerResponse struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"display_name"`
	Country     string    `json:"country"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type eventResponse struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	TS      time.Time      `json:"ts"`
	Payload map[string]any `json:"payload"`
}

type sessionResponse struct {
	Profile      playerResponse   `json:"profile"`
	RecentEvents []eventResponse  `json:"recent_events"`
	CountsByType map[string]int64 `json:"counts_by_type"`
	Window       struct {
		Since time.Time `json:"since"`
	} `json:"window"`
}

func TestSessionSummary_EndToEnd(t *testing.T) {
	id := uniqueID(t)
	t.Cleanup(func() { testServer.deletePlayerData(t, id) })

	testServer.get(t, "/v1/players/"+id+"/session", http.StatusNotFound)

	testServer.put(t, "/v1/players/"+id, "", http.StatusCreated)
	testServer.put(t, "/v1/players/"+id, "", http.StatusOK) // idempotent

	testServer.post(t, "/v1/players/"+id+"/events",
		`{"type":"level_start","timestamp":"2026-09-23T10:00:00Z","payload":{"level":3}}`, http.StatusCreated)
	testServer.post(t, "/v1/players/"+id+"/events",
		`{"type":"level_end","timestamp":"2026-09-23T10:05:00Z","payload":{"level":3,"won":true}}`, http.StatusCreated)
	testServer.post(t, "/v1/players/"+id+"/events",
		`{"type":"level_start","timestamp":"2020-01-01T00:00:00Z","payload":{}}`, http.StatusCreated)

	testServer.patch(t, "/v1/players/"+id, `{"display_name":"Ada","country":"FR"}`, http.StatusOK)

	sum := decode[sessionResponse](t, testServer.get(t, "/v1/players/"+id+"/session", http.StatusOK))

	if sum.Profile.ID != id || sum.Profile.DisplayName != "Ada" {
		t.Fatalf("profile = %+v", sum.Profile)
	}
	if len(sum.RecentEvents) != 3 || sum.RecentEvents[0].Type != "level_end" {
		t.Fatalf("recent events = %+v", sum.RecentEvents)
	}
	if sum.CountsByType["level_start"] != 1 || sum.CountsByType["level_end"] != 1 {
		t.Fatalf("counts = %v; the 2020 event must fall outside the 7d window", sum.CountsByType)
	}
}
