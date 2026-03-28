package handler

import (
	"log/slog"
	"net/http"

	"withered/server/adapter/websocket"
	"withered/server/domain"

	"github.com/coder/websocket"
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
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // 開発用: Origin チェックをスキップ
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to accept", "err", err)
		return
	}

	session := domain.NewSession()
	transport := adapterwebsocker.NewTransportFrom(conn)
	connection := domain.NewConnection(session.ID(), transport)
	endpoint, err := domain.NewSessionEndpoint(ctx, session, connection, h.pubsub, h.roomManager)
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
