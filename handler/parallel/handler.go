package parallelhandler

import (
	"encoding/json"
	"net/http"

	"github.com/touka-aoi/paralle-vs-single/application/request"
	"github.com/touka-aoi/paralle-vs-single/application/service"
)

type Handler struct {
	interactionService *service.InteractionService
}

func NewHandler(svc *service.InteractionService) *Handler {
	return &Handler{
		interactionService: svc,
	}
}

func (h *Handler) HandleMove(w http.ResponseWriter, r *http.Request) {
	var payload request.Move
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	result, err := h.interactionService.Move(r.Context(), payload)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *Handler) HandleBuff(w http.ResponseWriter, r *http.Request) {
	var payload request.Buff
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	result, err := h.interactionService.Buff(r.Context(), payload)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *Handler) HandleAttack(w http.ResponseWriter, r *http.Request) {
	var payload request.Attack
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	result, err := h.interactionService.Attack(r.Context(), payload)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *Handler) HandleTrade(w http.ResponseWriter, r *http.Request) {
	var payload request.Trade
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	result, err := h.interactionService.Trade(r.Context(), payload)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func httpError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
