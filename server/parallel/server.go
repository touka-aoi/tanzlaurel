package parallel

import (
	"encoding/json"
	stdhttp "net/http"

	"github.com/touka-aoi/paralle-vs-single/application/request"
	"github.com/touka-aoi/paralle-vs-single/application/service"
	appstate "github.com/touka-aoi/paralle-vs-single/application/state"
)

// Dependencies bundles the collaborators required by the HTTP transport.
type Dependencies struct {
	Service *service.InteractionService
	State   appstate.InteractionState
}

// NewMux wires HTTP handlers for the parallel implementation.
func NewMux(deps Dependencies) *stdhttp.ServeMux {
	mux := stdhttp.NewServeMux()
	handler := &Handler{svc: deps.Service}

	mux.HandleFunc("/move", handler.handleMove)
	mux.HandleFunc("/buff", handler.handleBuff)
	mux.HandleFunc("/attack", handler.handleAttack)
	mux.HandleFunc("/trade", handler.handleTrade)

	return mux
}

// Handler translates HTTP requests into service calls.
type Handler struct {
	svc *service.InteractionService
}

func (h *Handler) handleMove(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if r.Method != stdhttp.MethodPost {
		w.WriteHeader(stdhttp.StatusMethodNotAllowed)
		return
	}
	var payload request.Move
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httpError(w, stdhttp.StatusBadRequest, err)
		return
	}
	result, err := h.svc.Move(r.Context(), payload)
	if err != nil {
		httpError(w, stdhttp.StatusInternalServerError, err)
		return
	}
	writeJSON(w, result)
}

func (h *Handler) handleBuff(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if r.Method != stdhttp.MethodPost {
		w.WriteHeader(stdhttp.StatusMethodNotAllowed)
		return
	}
	var payload request.Buff
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httpError(w, stdhttp.StatusBadRequest, err)
		return
	}
	result, err := h.svc.Buff(r.Context(), payload)
	if err != nil {
		httpError(w, stdhttp.StatusInternalServerError, err)
		return
	}
	writeJSON(w, result)
}

func (h *Handler) handleAttack(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if r.Method != stdhttp.MethodPost {
		w.WriteHeader(stdhttp.StatusMethodNotAllowed)
		return
	}
	var payload request.Attack
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httpError(w, stdhttp.StatusBadRequest, err)
		return
	}
	result, err := h.svc.Attack(r.Context(), payload)
	if err != nil {
		httpError(w, stdhttp.StatusInternalServerError, err)
		return
	}
	writeJSON(w, result)
}

func (h *Handler) handleTrade(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if r.Method != stdhttp.MethodPost {
		w.WriteHeader(stdhttp.StatusMethodNotAllowed)
		return
	}
	var payload request.Trade
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httpError(w, stdhttp.StatusBadRequest, err)
		return
	}
	result, err := h.svc.Trade(r.Context(), payload)
	if err != nil {
		httpError(w, stdhttp.StatusInternalServerError, err)
		return
	}
	writeJSON(w, result)
}

func httpError(w stdhttp.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func writeJSON(w stdhttp.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
