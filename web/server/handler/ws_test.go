package handler_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"flourish/server/adapter/memory"
	"flourish/server/application"
	"flourish/server/handler"

	"github.com/coder/websocket"
	"github.com/google/uuid"

	"log/slog"
)

func setupWSServer(t *testing.T) (*httptest.Server, *application.SyncService) {
	t.Helper()
	eventStore := memory.NewEventStore()
	syncService := application.NewSyncService(eventStore)
	log := slog.Default()
	wsHandler := handler.NewWS(syncService, log)
	srv := httptest.NewServer(wsHandler)
	t.Cleanup(srv.Close)
	return srv, syncService
}

func dial(t *testing.T, srv *httptest.Server) *websocket.Conn {
	t.Helper()
	url := "ws" + srv.URL[len("http"):]
	ctx := t.Context()
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.CloseNow() })
	return conn
}

func readJSON[T any](t *testing.T, conn *websocket.Conn) T {
	t.Helper()
	ctx := t.Context()
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func writeJSON(t *testing.T, conn *websocket.Conn, v any) {
	t.Helper()
	data, _ := json.Marshal(v)
	if err := conn.Write(t.Context(), websocket.MessageText, data); err != nil {
		t.Fatal(err)
	}
}

func TestWS_OpAndAck(t *testing.T) {
	srv, _ := setupWSServer(t)
	conn := dial(t, srv)

	entryID := uuid.New().String()
	reqID := uuid.New().String()

	writeJSON(t, conn, map[string]any{
		"type":       "op",
		"request_id": reqID,
		"entry_id":   entryID,
		"op_type":    1,
		"node_id":    map[string]any{"site_id": uuid.New().String(), "timestamp": 1},
		"value":      "a",
	})

	// ACKを受信
	ack := readJSON[handler.AckMsg](t, conn)
	if ack.Type != "ack" {
		t.Errorf("typeがackであるべき: got %q", ack.Type)
	}
	if ack.RequestID != reqID {
		t.Errorf("request_idが一致すべき: got %q, want %q", ack.RequestID, reqID)
	}
	if ack.ServerSeq != 1 {
		t.Errorf("server_seqが1であるべき: got %d", ack.ServerSeq)
	}
}

func TestWS_BroadcastToOtherClient(t *testing.T) {
	srv, _ := setupWSServer(t)
	conn1 := dial(t, srv)
	conn2 := dial(t, srv)

	entryID := uuid.New().String()

	// conn2にsync_requestを送信してsubscribeさせる
	writeJSON(t, conn2, map[string]any{
		"type":            "sync_request",
		"request_id":      uuid.New().String(),
		"entry_id":        entryID,
		"last_server_seq": 0,
	})
	// 空syncを受信
	readJSON[handler.SyncMsg](t, conn2)

	// conn1からop送信
	writeJSON(t, conn1, map[string]any{
		"type":       "op",
		"request_id": uuid.New().String(),
		"entry_id":   entryID,
		"op_type":    1,
		"node_id":    map[string]any{"site_id": uuid.New().String(), "timestamp": 1},
		"value":      "x",
	})

	// conn1: ACK + sync
	readJSON[handler.AckMsg](t, conn1)
	sync1 := readJSON[handler.SyncMsg](t, conn1)
	if sync1.Type != "sync" {
		t.Errorf("conn1がsyncを受信すべき: got %q", sync1.Type)
	}

	// conn2: syncを受信
	sync2 := readJSON[handler.SyncMsg](t, conn2)
	if sync2.Type != "sync" {
		t.Errorf("conn2がsyncを受信すべき: got %q", sync2.Type)
	}
	if len(sync2.Ops) != 1 {
		t.Errorf("opsが1件であるべき: got %d", len(sync2.Ops))
	}
}

func TestWS_SyncRequest_Diff(t *testing.T) {
	srv, syncService := setupWSServer(t)

	entryID := uuid.New()
	siteID := uuid.New()

	// 先にopを3件登録
	for i := range 3 {
		payload, _ := json.Marshal(map[string]any{
			"op_type": 1,
			"node_id": map[string]any{"site_id": siteID.String(), "timestamp": i + 1},
			"value":   string(rune('a' + i)),
		})
		syncService.HandleOp(t.Context(), entryID, siteID, uuid.New(), payload)
	}

	// WebSocket接続してsync_request（last_server_seq=1）
	conn := dial(t, srv)
	writeJSON(t, conn, map[string]any{
		"type":            "sync_request",
		"request_id":      uuid.New().String(),
		"entry_id":        entryID.String(),
		"last_server_seq": 1,
	})

	sync := readJSON[handler.SyncMsg](t, conn)
	if len(sync.Ops) != 2 {
		t.Errorf("差分opsが2件であるべき: got %d", len(sync.Ops))
	}
	if sync.LatestServerSeq != 3 {
		t.Errorf("latest_server_seqが3であるべき: got %d", sync.LatestServerSeq)
	}
}

func TestWS_Dedup(t *testing.T) {
	srv, _ := setupWSServer(t)
	conn := dial(t, srv)

	entryID := uuid.New().String()
	reqID := uuid.New().String()

	msg := map[string]any{
		"type":       "op",
		"request_id": reqID,
		"entry_id":   entryID,
		"op_type":    1,
		"node_id":    map[string]any{"site_id": uuid.New().String(), "timestamp": 1},
		"value":      "a",
	}

	// 1回目
	writeJSON(t, conn, msg)
	ack1 := readJSON[handler.AckMsg](t, conn)
	if ack1.ServerSeq != 1 {
		t.Errorf("1回目のserver_seqが1であるべき: got %d", ack1.ServerSeq)
	}

	// 少し待ってsyncを読み飛ばす
	time.Sleep(10 * time.Millisecond)

	// 2回目（重複）
	writeJSON(t, conn, msg)
	ack2 := readJSON[handler.AckMsg](t, conn)
	if ack2.ServerSeq != 0 {
		t.Errorf("重複opのserver_seqが0であるべき: got %d", ack2.ServerSeq)
	}
}

func TestWS_InvalidMessage(t *testing.T) {
	srv, _ := setupWSServer(t)
	conn := dial(t, srv)

	// 不正なJSON
	conn.Write(t.Context(), websocket.MessageText, []byte("{invalid"))

	errMsg := readJSON[handler.ErrorMsg](t, conn)
	if errMsg.Type != "error" {
		t.Errorf("typeがerrorであるべき: got %q", errMsg.Type)
	}
	if errMsg.ErrorType != "error:invalid_op" {
		t.Errorf("error_typeがerror:invalid_opであるべき: got %q", errMsg.ErrorType)
	}
}
