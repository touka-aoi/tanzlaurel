package parallel

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/coder/websocket"
	"github.com/google/uuid"

	"github.com/touka-aoi/paralle-vs-single/handler"
	"github.com/touka-aoi/paralle-vs-single/service"
)

const (
	scopeAck       = "ack"
	scopeBroadcast = "broadcast"
)

type RoomID string
type ClientID string
type ClientSet map[*wsClient]struct{}

func (c ClientSet) Add(client *wsClient) {
	c[client] = struct{}{}
}

func (c ClientSet) Remove(client *wsClient) {
	delete(c, client)
}

type Handler struct {
	svc     *service.InteractionService
	mu      sync.RWMutex
	clients map[*wsClient]*clientInfo
	rooms   map[RoomID]ClientSet
}

type clientInfo struct {
	roomID string
}

type wsClient struct {
	id        ClientID
	conn      *websocket.Conn
	send      chan *outboundFrame
	done      chan struct{}
	closeOnce sync.Once
}

func (c *wsClient) closeChannels() {
	c.closeOnce.Do(func() {
		close(c.send)
		close(c.done)
	})
}

// NewHandler は依存するサービスを受け取り、WebSocket ハンドラを構築する。
func NewHandler(svc *service.InteractionService) *Handler {
	h := &Handler{
		svc:     svc,
		clients: make(map[*wsClient]*clientInfo),
		rooms:   make(map[RoomID]ClientSet),
	}
	//NOTE: テスト用のためroom1を作成
	h.rooms[RoomID("1")] = make(ClientSet)
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		slog.ErrorContext(ctx, "failed to accept websocket connection: %v", err)
		return
	}
	client := &wsClient{
		id:   ClientID(uuid.NewString()),
		conn: conn,
		send: make(chan *outboundFrame, 512),
		done: make(chan struct{}),
	}

	h.addClient(client)

	//NOTE: テスト用のためクライアントはすべてルーム1に行く
	h.assignRoom(client, "1")

	wg := sync.WaitGroup{}
	wg.Go(func() { h.writeLoop(ctx, client) })
	wg.Go(func() { h.readLoop(ctx, client) })
	wg.Wait()
}

func (h *Handler) readLoop(ctx context.Context, client *wsClient) {
	defer func() {
		h.removeClient(client)
		_ = client.conn.Close(websocket.StatusNormalClosure, "")
	}()
	for {
		msgType, data, err := client.conn.Read(ctx)
		if err != nil {
			status := websocket.CloseStatus(err)
			if status != websocket.StatusNormalClosure && status != websocket.StatusGoingAway {
				slog.ErrorContext(ctx, "failed to read websocket message: %v", err)
			}
			return
		}
		if msgType != websocket.MessageText {
			h.sendToClient(client, &outboundFrame{Scope: scopeAck, Error: "invalid message type"})
			continue
		}
		resp, roomID, broadcastFrame := h.handleFrame(ctx, data)
		h.sendToClient(client, resp)
		if broadcastFrame != nil {
			h.broadcast(roomID, broadcastFrame, client)
		}
	}
}

func (h *Handler) writeLoop(ctx context.Context, client *wsClient) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-client.send:
			if !ok {
				return
			}
			data, err := json.Marshal(msg)
			if err != nil {
				slog.ErrorContext(ctx, "parallel ws: marshal error", "error", err)
				continue
			}
			if err := client.conn.Write(ctx, websocket.MessageText, data); err != nil {
				slog.ErrorContext(ctx, "parallel ws: write error", "error", err)
				return
			}
		}
	}
}

func (h *Handler) addClient(client *wsClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client] = &clientInfo{}
}

func (h *Handler) removeClient(client *wsClient) {
	h.mu.Lock()
	info := h.clients[client]
	if info != nil && info.roomID != "" {
		h.removeRoomLocked(client, info.roomID)
	}
	delete(h.clients, client)
	h.mu.Unlock()
	client.closeChannels()
}

func (h *Handler) sendToClient(client *wsClient, frame *outboundFrame) {
	select {
	case <-client.done:
		return
	case client.send <- frame:
		return
	}
}

func (h *Handler) broadcast(roomID string, frame *outboundFrame, exclude *wsClient) {
	h.mu.RLock()
	members := h.rooms[RoomID(roomID)]
	targets := make([]*wsClient, 0, len(members))
	for client := range members {
		if client == exclude {
			continue
		}
		targets = append(targets, client)
	}
	h.mu.RUnlock()
	for _, target := range targets {
		select {
		case <-target.done:
			break
		default:
		}
		target.send <- frame
	}
}

type inboundFrame struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type outboundFrame struct {
	Type   string      `json:"type"`
	Scope  string      `json:"scope"`
	RoomID string      `json:"roomId,omitempty"`
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
}

func (h *Handler) handleFrame(ctx context.Context, data []byte) (*outboundFrame, string, *outboundFrame) {
	var frame inboundFrame
	if err := json.Unmarshal(data, &frame); err != nil {
		return &outboundFrame{Scope: scopeAck, Error: fmt.Sprintf("invalid frame: %v", err)}, "", nil
	}
	frameType := strings.ToLower(frame.Type)
	switch frameType {
	case "move":
		var payload handler.MovePayload
		if err := json.Unmarshal(frame.Payload, &payload); err != nil {
			return &outboundFrame{Type: frameType, Scope: scopeAck, Error: fmt.Sprintf("invalid payload: %v", err)}, "", nil
		}
		result, err := h.svc.Move(ctx, &payload)
		return h.makeResponse(frameType, payload.RoomID, result, err, true)
	case "attack":
		var payload handler.AttackPayload
		if err := json.Unmarshal(frame.Payload, &payload); err != nil {
			return &outboundFrame{Type: frameType, Scope: scopeAck, Error: fmt.Sprintf("invalid payload: %v", err)}, "", nil
		}
		result, err := h.svc.Attack(ctx, &payload)
		return h.makeResponse(frameType, payload.RoomID, result, err, true)
	default:
		return &outboundFrame{Type: frameType, Scope: scopeAck, Error: fmt.Sprintf("unsupported type: %s", frame.Type)}, "", nil
	}
}

func (h *Handler) makeResponse(frameType, roomID string, result interface{}, err error, broadcast bool) (*outboundFrame, string, *outboundFrame) {
	resp := &outboundFrame{Type: frameType, Scope: scopeAck, RoomID: roomID}
	if err != nil {
		resp.Error = err.Error()
		return resp, roomID, nil
	}
	resp.Result = result
	if !broadcast {
		return resp, roomID, nil
	}
	return resp, roomID, &outboundFrame{
		Type:   frameType,
		Scope:  scopeBroadcast,
		RoomID: roomID,
		Result: result,
	}
}

func (h *Handler) assignRoom(client *wsClient, roomID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	info, ok := h.clients[client]
	if !ok {
		panic("client not found")
	}
	if info.roomID == roomID {
		panic("room assignment duplicated")
	}
	if info.roomID != "" {
		panic("room assignment conflict")
	}
	info.roomID = roomID
	h.rooms[RoomID(roomID)].Add(client)
}

func (h *Handler) removeRoomLocked(client *wsClient, roomID string) {
	h.rooms[RoomID(roomID)].Remove(client)
}
