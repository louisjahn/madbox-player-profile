package event

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"madbox-player-profile/internal/httpx"
	"madbox-player-profile/internal/player"
)

const maxBodyBytes = 16 << 10 // 16 KiB

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/players/{id}/events", h.record)
	mux.HandleFunc("GET /v1/players/{id}/events", h.list)
}

func (h *Handler) record(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req RecordRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body",
			"malformed JSON body (payload must be an object, body ≤ 16KiB)")
		return
	}

	e, err := h.svc.Record(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, e)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	f := Filter{Type: r.URL.Query().Get("type")}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			httpx.Error(w, http.StatusBadRequest, "invalid_limit", "limit must be a positive integer")
			return
		}
		f.Limit = n
	}

	events, err := h.svc.List(r.Context(), r.PathValue("id"), f)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	// Wrap in an object to easily introduce cursors later
	httpx.JSON(w, http.StatusOK, map[string]any{"events": events})
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, player.ErrInvalidID):
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "id must be 1-64 chars of [a-zA-Z0-9_-]")
	case errors.Is(err, player.ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "player not found")
	case errors.Is(err, ErrInvalidEvent):
		httpx.Error(w, http.StatusBadRequest, "invalid_event", err.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, "internal", "internal error")
	}
}
