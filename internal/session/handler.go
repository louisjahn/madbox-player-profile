package session

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"madbox-player-profile/internal/event"
	"madbox-player-profile/internal/httpx"
	"madbox-player-profile/internal/player"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/players/{id}/session", h.get)
}

type response struct {
	Profile      player.Response  `json:"profile"`
	RecentEvents []event.Response `json:"recent_events"`
	CountsByType map[string]int64 `json:"counts_by_type"`
	Window       struct {
		Since time.Time `json:"since"`
	} `json:"window"`
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	sum, err := h.svc.Summarize(r.Context(), r.PathValue("id"))
	switch {
	case errors.Is(err, player.ErrInvalidID):
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "valid id characters are [a-zA-Z0-9_-]")
		return
	case errors.Is(err, player.ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "player not found")
		return
	case err != nil:
		slog.ErrorContext(r.Context(), "session summary", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	resp := response{
		Profile:      player.ToResponse(sum.Profile),
		RecentEvents: event.ToResponses(sum.RecentEvents),
		CountsByType: sum.CountsByType,
	}
	resp.Window.Since = sum.WindowStart
	httpx.JSON(w, http.StatusOK, resp)
}
