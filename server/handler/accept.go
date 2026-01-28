package handler

import (
	"log/slog"
	"net/http"

	"github.com/coder/websocket"
	"withered/server/adapter/websocket"
	"withered/server/domain"
)

type AcceptHandler struct {
	pubsub      domain.PubSub
	roomManager domain.RoomManager
}

func NewAcceptHandler(pubsub domain.PubSub, roomManager domain.RoomManager) *AcceptHandler {
	return &AcceptHandler{pubsub: pubsub, roomManager: roomManager}
}

func (h *AcceptHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		slog.ErrorContext(ctx, "failed to accept", "err", err)
		return
	}
	transport := adapterwebsocker.NewTransportFrom(conn)
	session := domain.NewSession()
	connection := domain.NewConnection(session.ID(), transport)
	endpoint, err := domain.NewSessionEndpoint(session, connection, h.pubsub, h.roomManager)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create session endpoint", "err", err)
		return
	}
	slog.DebugContext(ctx, "accepted new connection", "session_id", session.ID())
	err = endpoint.Run()
	if err != nil {
		slog.ErrorContext(ctx, "failed to run session endpoint", "err", err)
		return
	}
}
