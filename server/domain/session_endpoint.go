package domain

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"time"

	"withered/server/domain/protocol"

	"golang.org/x/sync/errgroup"
)

var (
	// ErrSessionAlreadyAttached はセッションに既に接続が紐付けられている場合に返されるエラーです。
	ErrSessionAlreadyAttached = errors.New("session already has an attached connection")
	// ErrSessionNotAttached はセッションに接続が紐付けられていない場合に返されるエラーです。
	ErrSessionNotAttached = errors.New("session has no attached connection")
	// ErrBackpressure は書き込みチャネルが満杯の場合に返されるエラーです。
	ErrBackpressure = errors.New("write channel is full, apply backpressure")
	// ErrInitializationFailed はセッションエンドポイントの初期化に失敗した場合に返されるエラーです。
	ErrInitializationFailed = errors.New("failed to initialize session endpoint")
)

type SessionEndpoint struct {
	ctx    context.Context
	cancel context.CancelFunc

	session     *Session
	connection  *Connection
	pubsub      PubSub
	roomManager RoomManager
	roomID      string // 現在参加中のRoomID（空文字列=未参加）

	ctrlCh  chan endpointEvent // 制御用チャネル
	writeCh chan []byte        // 書き込み用チャネル

	// lifecycle
	closed atomic.Bool
}

func NewSessionEndpoint(ctx context.Context, session *Session, connection *Connection, pubsub PubSub, roomManager RoomManager) (*SessionEndpoint, error) {
	if session == nil {
		return nil, ErrInitializationFailed
	}
	if connection == nil {
		return nil, ErrInitializationFailed
	}
	if pubsub == nil {
		return nil, ErrInitializationFailed
	}
	if roomManager == nil {
		return nil, ErrInitializationFailed
	}
	ctx, cancel := context.WithCancel(ctx)
	se := &SessionEndpoint{
		ctx:         ctx,
		cancel:      cancel,
		session:     session,
		connection:  connection,
		pubsub:      pubsub,
		roomManager: roomManager,
		ctrlCh:      make(chan endpointEvent, 16),
		writeCh:     make(chan []byte, 1024),
	}
	return se, nil
}

func (se *SessionEndpoint) Run() error {
	// 自分宛のメッセージを購読
	sessionTopic := Topic("session:" + se.session.ID().String())
	msgCh := se.pubsub.Subscribe(sessionTopic)
	defer se.pubsub.Unsubscribe(sessionTopic, msgCh)

	eg, ctx := errgroup.WithContext(se.ctx)
	eg.Go(func() error {
		se.ownerLoop(ctx)
		return nil
	})
	eg.Go(func() error {
		se.readLoop(ctx)
		return nil
	})
	eg.Go(func() error {
		se.writeLoop(ctx)
		return nil
	})
	eg.Go(func() error {
		se.subscribeLoop(ctx, msgCh)
		return nil
	})

	// セッションID通知を送信
	assignMsg := protocol.EncodeAssign(se.session.ID().Bytes())
	if err := se.Send(assignMsg); err != nil {
		return err
	}

	// Assign後にHeartbeatServiceを起動（クライアントがsessionIdを持った状態でpongを返せるようにする）
	heartbeat := NewHeartbeatService(5*time.Second, se.session, se.writeCh)
	eg.Go(func() error {
		heartbeat.Run(ctx)
		return nil
	})

	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}

func (se *SessionEndpoint) Send(data []byte) error {
	select {
	case se.writeCh <- data:
		return nil
	default:
		return ErrBackpressure
	}
}

func (se *SessionEndpoint) Close(ctx context.Context) {
	se.sendCtrlEvent(ctx, endpointEvent{kind: evClose, err: nil})
}

// ownerLoop は論理セッションの状態を監視し、必要に応じて接続の管理を行います。
func (se *SessionEndpoint) ownerLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-se.ctrlCh:
			se.handleControlEvent(ctx, ev)
		case <-ticker.C:
			ok, reason := se.session.IsIdle(30 * time.Second)
			if ok {
				se.handleControlEvent(ctx, endpointEvent{
					kind: evClose,
					err:  errors.New(reason.String()),
				})
			}
		}
	}
}

func (se *SessionEndpoint) readLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			data, err := se.connection.Read(ctx)
			if err != nil {
				se.sendCtrlEvent(ctx, endpointEvent{kind: evClose, err: err})
				return
			}
			se.session.TouchRead()
			se.handleData(ctx, data)
		}
	}
}

func (se *SessionEndpoint) writeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case data := <-se.writeCh:
			err := se.connection.Write(ctx, data)
			if err != nil {
				se.sendCtrlEvent(ctx, endpointEvent{kind: evClose, err: err})
				return
			}
			se.session.TouchWrite()
		}
	}
}

// subscribeLoop はpubsubからのメッセージをwriteChに転送します。
func (se *SessionEndpoint) subscribeLoop(ctx context.Context, msgCh <-chan Message) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-msgCh:
			if !ok {
				return
			}
			select {
			case se.writeCh <- msg.Data:
				// 送信成功
			default:
				slog.Warn("subscribeLoop: writeCh full, message dropped", "sessionID", se.session.ID())
			}
		}
	}
}

func (se *SessionEndpoint) close() {
	if !se.closed.CompareAndSwap(false, true) {
		return
	}
	// Roomからの離脱（cancel前に実行）
	if se.roomID != "" {
		roomTopic := Topic("room:" + se.roomID)
		leavePayload := protocol.EncodeRoomMessage(se.roomID, protocol.RoomMsgTypeLeave, nil)
		// Room.HandleMessageに渡すのはTransportHeaderを除いたペイロード部分
		_, _, hdrLen, _ := protocol.ParseTransportHeader(leavePayload)
		se.pubsub.Publish(se.ctx, roomTopic, Message{
			SessionID: se.session.ID(),
			Data:      leavePayload[hdrLen:],
		})
		se.roomID = ""
	}
	se.cancel()
	se.session.Close()
	se.connection.Close()
}

func (se *SessionEndpoint) handleData(ctx context.Context, data []byte) {
	msgType, totalLen, hdrLen, err := protocol.ParseTransportHeader(data)
	if err != nil {
		slog.WarnContext(ctx, "failed to parse transport header", "err", err)
		return
	}
	if hdrLen+int(totalLen) > len(data) {
		slog.WarnContext(ctx, "message truncated", "sessionID", se.session.ID())
		return
	}
	payload := data[hdrLen : hdrLen+int(totalLen)]

	switch msgType {
	case protocol.MsgTypePong:
		slog.DebugContext(ctx, "received pong", "sessionID", se.session.ID())
		se.sendCtrlEvent(ctx, endpointEvent{kind: evPong})

	case protocol.MsgTypeRoomMessage:
		// RoomIDを解析してroom topicにpublish
		roomID, n, err := protocol.ParseRoomID(payload)
		if err != nil {
			slog.WarnContext(ctx, "failed to parse room ID", "err", err)
			return
		}

		// RoomMsgTypeを先読みしてJoin/Leave/AppDataを判定
		roomMsgType, _, err := protocol.ParseRoomHeader(payload[n:])
		if err != nil {
			slog.WarnContext(ctx, "failed to parse room header", "err", err)
			return
		}

		switch roomMsgType {
		case protocol.RoomMsgTypeJoin:
			se.roomID = roomID
			slog.InfoContext(ctx, "session joined room", "sessionID", se.session.ID(), "roomID", roomID)
		case protocol.RoomMsgTypeLeave:
			slog.InfoContext(ctx, "session left room", "sessionID", se.session.ID(), "roomID", roomID)
			defer func() { se.roomID = "" }()
		}

		roomTopic := Topic("room:" + roomID)
		se.pubsub.Publish(ctx, roomTopic, Message{
			SessionID: se.session.ID(),
			Data:      payload, // TransportHeaderを除いたペイロードをRoomに渡す
		})

	default:
		slog.WarnContext(ctx, "unknown message type", "msgType", msgType)
	}
}

// handleControlEvent は制御チャネルからのイベントを処理し論理セッションの状態を更新する唯一の関数です。
func (se *SessionEndpoint) handleControlEvent(ctx context.Context, ev endpointEvent) {
	switch ev.kind {
	case evClose:
		se.close()
	case evPong:
		se.session.TouchPong()
	default:
		slog.WarnContext(ctx, "unknown endpoint event kind", "kind", ev.kind)
	}
}

func (se *SessionEndpoint) sendCtrlEvent(ctx context.Context, ev endpointEvent) {
	select {
	case se.ctrlCh <- ev:
	case <-ctx.Done():
	}
}
