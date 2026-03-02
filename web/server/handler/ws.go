package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"
	"sync"

	"flourish/server/application"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

// WS はWebSocketハンドラー。
type WS struct {
	syncService *application.SyncService
	projector   *application.EntryProjector
	log         *slog.Logger
}

func NewWS(syncService *application.SyncService, projector *application.EntryProjector, log *slog.Logger) *WS {
	return &WS{
		syncService: syncService,
		projector:   projector,
		log:         log,
	}
}

// wsSubscriber はWebSocket接続のSubscriber実装。
type wsSubscriber struct {
	conn *websocket.Conn
	mu   sync.Mutex
	log  *slog.Logger
}

func (s *wsSubscriber) Send(msg application.SyncMessage) {
	syncMsg := convertSyncMessage(msg)
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.Marshal(syncMsg)
	if err != nil {
		s.log.Error("sync message marshal error", "error", err)
		return
	}
	if err := s.conn.Write(context.Background(), websocket.MessageText, data); err != nil {
		s.log.Debug("sync message write error", "error", err)
	}
}

func (h *WS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		h.log.Error("websocket accept error", "error", err)
		return
	}
	defer conn.CloseNow()

	h.log.Info("websocket connected", "remoteAddr", r.RemoteAddr)

	sub := &wsSubscriber{conn: conn, log: h.log}
	var subscribedEntries []uuid.UUID
	defer func() {
		for _, entryID := range subscribedEntries {
			h.syncService.Unsubscribe(entryID, sub)
		}
		h.log.Info("websocket disconnected", "remoteAddr", r.RemoteAddr)
	}()

	for {
		_, data, err := conn.Read(r.Context())
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				return
			}
			h.log.Debug("websocket read error", "error", err)
			return
		}

		var msg IncomingMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			h.writeError(conn, nil, "error:invalid_op", "Invalid Operation")
			continue
		}

		switch msg.Type {
		case MsgTypeOp:
			h.handleOp(r.Context(), conn, sub, msg, &subscribedEntries)
		case MsgTypeSyncRequest:
			h.handleSyncRequest(r.Context(), conn, sub, msg, &subscribedEntries)
		default:
			h.writeError(conn, &msg.RequestID, "error:invalid_op", "Invalid Operation")
		}
	}
}

func (h *WS) handleOp(ctx context.Context, conn *websocket.Conn, sub *wsSubscriber, msg IncomingMessage, subscribedEntries *[]uuid.UUID) {
	entryID, err := uuid.Parse(msg.EntryID)
	if err != nil {
		h.writeError(conn, &msg.RequestID, "error:invalid_op", "Invalid Operation")
		return
	}
	requestID, err := uuid.Parse(msg.RequestID)
	if err != nil {
		h.writeError(conn, &msg.RequestID, "error:invalid_op", "Invalid Operation")
		return
	}

	// opのpayloadをそのまま永続化
	payload, _ := json.Marshal(msg)

	// Subscribe if not already
	h.ensureSubscribed(entryID, sub, subscribedEntries)

	siteID := uuid.Nil
	if msg.NodeID != nil {
		siteID, _ = uuid.Parse(msg.NodeID.SiteID)
	}

	ack, err := h.syncService.HandleOp(ctx, entryID, siteID, requestID, payload)
	if err != nil {
		h.writeError(conn, &msg.RequestID, "error:internal", "Internal Error")
		return
	}

	// ACKを先に送信
	ackMsg := AckMsg{
		Type:      MsgTypeAck,
		RequestID: msg.RequestID,
		EntryID:   msg.EntryID,
		ServerSeq: ack.ServerSeq,
	}
	data, _ := json.Marshal(ackMsg)
	sub.mu.Lock()
	conn.Write(ctx, websocket.MessageText, data)
	sub.mu.Unlock()

	// 重複でなければprojector適用 → broadcast
	if ack.ServerSeq > 0 {
		if h.projector != nil {
			h.projector.Apply(ctx, entryID, payload)
		}

		h.syncService.Broadcast(entryID, application.SyncMessage{
			EntryID: entryID,
			Ops: []application.SyncOp{
				{
					RequestID: requestID,
					ServerSeq: ack.ServerSeq,
					Payload:   payload,
				},
			},
			LatestServerSeq: ack.ServerSeq,
		})
	}
}

func (h *WS) handleSyncRequest(ctx context.Context, conn *websocket.Conn, sub *wsSubscriber, msg IncomingMessage, subscribedEntries *[]uuid.UUID) {
	entryID, err := uuid.Parse(msg.EntryID)
	if err != nil {
		h.writeError(conn, &msg.RequestID, "error:invalid_op", "Invalid Operation")
		return
	}

	h.ensureSubscribed(entryID, sub, subscribedEntries)

	diff, err := h.syncService.GetDiff(ctx, entryID, msg.LastServerSeq)
	if err != nil {
		h.writeError(conn, &msg.RequestID, "error:internal", "Internal Error")
		return
	}

	syncMsg := convertSyncMessage(diff)
	data, _ := json.Marshal(syncMsg)
	sub.mu.Lock()
	conn.Write(ctx, websocket.MessageText, data)
	sub.mu.Unlock()
}

func (h *WS) ensureSubscribed(entryID uuid.UUID, sub *wsSubscriber, subscribedEntries *[]uuid.UUID) {
	if slices.Contains(*subscribedEntries, entryID) {
		return
	}
	h.syncService.Subscribe(entryID, sub)
	*subscribedEntries = append(*subscribedEntries, entryID)
}

func (h *WS) writeError(conn *websocket.Conn, requestID *string, errorType, title string) {
	msg := newErrorMsg(requestID, errorType, title)
	data, _ := json.Marshal(msg)
	conn.Write(context.Background(), websocket.MessageText, data)
}

func convertSyncMessage(msg application.SyncMessage) SyncMsg {
	ops := make([]SyncOpMsg, len(msg.Ops))
	for i, op := range msg.Ops {
		// payloadからop情報を復元
		var incoming IncomingMessage
		json.Unmarshal(op.Payload, &incoming)

		ops[i] = SyncOpMsg{
			RequestID: op.RequestID.String(),
			ServerSeq: op.ServerSeq,
			OpType:    incoming.OpType,
			NodeID:    incoming.NodeID,
			After:     incoming.After,
			Value:     incoming.Value,
		}
	}

	return SyncMsg{
		Type:            MsgTypeSync,
		EntryID:         msg.EntryID.String(),
		Ops:             ops,
		LatestServerSeq: msg.LatestServerSeq,
	}
}
