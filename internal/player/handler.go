package player

import (
	"encoding/json"
	"errors"
	"net/http"

	"madbox-player-profile/internal/httpx"
)

const maxBodyBytes = 4096

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("PUT /v1/players/{id}", h.create)
	mux.HandleFunc("GET /v1/players/{id}", h.get)
	mux.HandleFunc("PATCH /v1/players/{id}", h.update)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	ID := r.PathValue("id")
	created, err := h.svc.Create(r.Context(), ID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
		w.Header().Set("Location", "/v1/players/"+ID)
	}
	httpx.JSON(w, status, nil)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	p, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var patch Patch
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&patch); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "malformed JSON body, payload must be an object < 4KB")
		return
	}

	p, err := h.svc.Update(r.Context(), r.PathValue("id"), patch)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidID):
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "valid id characters are [a-zA-Z0-9_-]")
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "player not found")
	case errors.Is(err, ErrInvalidPatch):
		httpx.Error(w, http.StatusBadRequest, "invalid_patch", err.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, "internal", "internal error")
	}
}
