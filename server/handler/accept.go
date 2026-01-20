package handler

import (
	"log/slog"
	"net/http"

	"github.com/coder/websocket"
	"github.com/touka-aoi/paralle-vs-single/server/adapter/websocket"
	"github.com/touka-aoi/paralle-vs-single/server/domain"
)

type AcceptHandler struct {
	dispatcher domain.Dispatcher
}

func NewAcceptHandler(dispatcher domain.Dispatcher) *AcceptHandler {
	return &AcceptHandler{dispatcher: dispatcher}
}

func (h *AcceptHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		slog.ErrorContext(ctx, "failed to accept", "err", err)
		return
	}
	transport := adapterwebsocker.NewTransportFrom(conn)
	connection := domain.NewConnection(transport)
	session := domain.NewSession()
	endpoint, err := domain.NewSessionEndpoint(session, connection, h.dispatcher)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create session endpoint", "err", err)
		return
	}
	err = endpoint.Run()
	if err != nil {
		slog.ErrorContext(ctx, "failed to run session endpoint", "err", err)
		return
	}
}
