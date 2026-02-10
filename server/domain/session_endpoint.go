package domain

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"time"

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
	roomID      RoomID // 実行時にRoomManagerから取得

	ctrlCh  chan endpointEvent // 制御用チャネル
	writeCh chan []byte        // 書き込み用チャネル

	// lifecycle
	closed atomic.Bool
}

func NewSessionEndpoint(session *Session, connection *Connection, pubsub PubSub, roomManager RoomManager) (*SessionEndpoint, error) {
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
	ctx, cancel := context.WithCancel(context.Background())
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
	assignMsg := EncodeAssignMessage(se.session.ID())
	if err := se.Send(assignMsg); err != nil {
		return err
	}

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

func (se *SessionEndpoint) ForceClose() {
	se.close()
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
				se.sendCtrlEvent(ctx, endpointEvent{kind: evReadError, err: err})
				continue
			}
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
				se.sendCtrlEvent(ctx, endpointEvent{kind: evWriteError, err: err})
				continue
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
	se.cancel()
	se.session.Close()
	se.connection.Close()
}

func (se *SessionEndpoint) handleData(ctx context.Context, data []byte) {
	header, err := ParseHeader(data)
	if err != nil {
		slog.WarnContext(ctx, "failed to parse header", "err", err)
		return
	}
	expectedBytes := se.session.ID().Bytes()
	if header.SessionID != expectedBytes {
		slog.WarnContext(ctx, "session ID mismatch", "expected", se.session.ID(), "got", SessionIDFromBytes(header.SessionID))
		return
	}
	payloadHeader, err := ParsePayloadHeader(data[HeaderSize:])
	if err != nil {
		slog.WarnContext(ctx, "failed to parse payload header", "err", err)
		return
	}

	switch payloadHeader.DataType {
	case DataTypeControl:
		se.handleControlMessage(ctx, ControlSubType(payloadHeader.SubType), data)
		return
	case DataTypeInput, DataTypeActor2D:
		// データメッセージをroom topicに転送
		if se.roomID.IsEmpty() {
			slog.WarnContext(ctx, "received data message before joining a room", "sessionID", se.session.ID())
			return
		}
		roomTopic := Topic("room:" + se.roomID.String())
		se.pubsub.Publish(ctx, roomTopic, Message{
			SessionID: se.session.ID(),
			Data:      data,
		})
	default:
		slog.WarnContext(ctx, "unknown data type", "dataType", payloadHeader.DataType)
	}
}

func (se *SessionEndpoint) handleControlMessage(ctx context.Context, subType ControlSubType, data []byte) {
	switch subType {
	case ControlSubTypeJoin:
		payload, err := ParseJoinPayload(data[HeaderSize+PayloadHeaderSize:])
		if err != nil {
			slog.WarnContext(ctx, "failed to parse join message", "err", err)
			return
		}
		roomID := payload.RoomID
		// RoomIDが空の場合、RoomManagerからデフォルトルームを取得
		if roomID.IsEmpty() {
			defaultRoomID, err := se.roomManager.GetRoom(ctx, se.session.ID())
			if err != nil {
				slog.ErrorContext(ctx, "failed to get default room", "err", err)
				return
			}
			roomID = defaultRoomID
			slog.DebugContext(ctx, "auto-assigned room", "sessionID", se.session.ID(), "roomID", roomID)
		}
		se.roomID = roomID
		slog.InfoContext(ctx, "session joined room", "sessionID", se.session.ID(), "roomID", se.roomID)
		// room topicにJoinメッセージをpublish（Room.HandleMessageでsessions追加）
		roomTopic := Topic("room:" + se.roomID.String())
		se.pubsub.Publish(ctx, roomTopic, Message{SessionID: se.session.ID(), Data: data})
	case ControlSubTypeLeave:
		if se.roomID.IsEmpty() {
			slog.WarnContext(ctx, "session not in any room, cannot leave", "sessionID", se.session.ID())
			return
		}
		// room topicにLeaveメッセージをpublish（Room.HandleMessageでsessions削除）
		roomTopic := Topic("room:" + se.roomID.String())
		se.pubsub.Publish(ctx, roomTopic, Message{SessionID: se.session.ID(), Data: data})
		slog.InfoContext(ctx, "session left room", "sessionID", se.session.ID(), "roomID", se.roomID)
		se.roomID = RoomID{}
	}
}

// handleControlEvent は制御チャネルからのイベントを処理し論理セッションの状態を更新する唯一の関数です。
func (se *SessionEndpoint) handleControlEvent(ctx context.Context, ev endpointEvent) {
	switch ev.kind {
	case evClose:
		se.close()
	case evPong:
		se.session.TouchPong()
	case evReadError:
		return
	case evWriteError:
		return
	case evDispatchError:
		return

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
