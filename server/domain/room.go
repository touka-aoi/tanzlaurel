package domain

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

type RoomID string

var ErrRoomBusy = errors.New("room control channel is full")

type Room struct {
	ID       RoomID
	sessions map[SessionID]struct{}

	pubsub      PubSub
	application Application // 外部からアプリケーションロジックを注入できる

	sendCh chan roomSend

	tickInterval time.Duration
}

func NewRoom(id RoomID, pubsub PubSub, application Application) *Room {
	return &Room{
		ID:           id,
		sessions:     make(map[SessionID]struct{}),
		pubsub:       pubsub,
		application:  application,
		sendCh:       make(chan roomSend, 1024),
		tickInterval: time.Second / 60,
	}
}

func (r *Room) Broadcast(ctx context.Context, data []byte) {
	for sessionID := range r.sessions {
		topic := Topic("session:" + sessionID.String())
		r.pubsub.Publish(ctx, topic, Message{Data: data})
	}
}

func (r *Room) SendTo(ctx context.Context, sessionID SessionID, data []byte) {
	topic := Topic("session:" + sessionID.String())
	r.pubsub.Publish(ctx, topic, Message{Data: data})
}

func (r *Room) EnqueueBroadcast(ctx context.Context, data []byte) error {
	return r.enqueueSend(ctx, roomSend{kind: roomSendBroadcast, data: data})
}

func (r *Room) EnqueueSendTo(ctx context.Context, sessionID SessionID, data []byte) error {
	return r.enqueueSend(ctx, roomSend{kind: roomSendTo, sessionID: sessionID, data: data})
}

func (r *Room) enqueueSend(ctx context.Context, msg roomSend) error {
	select {
	case <-ctx.Done():
		return nil
	case r.sendCh <- msg:
		return nil
	default:
		return ErrRoomBusy
	}
}

func (r *Room) Run(ctx context.Context) error {
	// room宛のメッセージを購読
	roomTopic := Topic("room:" + string(r.ID))
	msgCh := r.pubsub.Subscribe(roomTopic)
	defer r.pubsub.Unsubscribe(roomTopic, msgCh)

	// room制御用トピックを購読（join/leave）
	ctrlTopic := Topic("room:" + string(r.ID) + ":ctrl")
	ctrlCh := r.pubsub.Subscribe(ctrlTopic)
	defer r.pubsub.Unsubscribe(ctrlTopic, ctrlCh)

	ticker := time.NewTicker(r.tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// 制御メッセージを処理（join/leave）
		CTRL_LOOP:
			for {
				select {
				case ctrl := <-ctrlCh:
					r.handleControlMessage(ctrl)
				default:
					break CTRL_LOOP
				}
			}
			// 受信メッセージを処理
		RECEIVE_LOOP:
			for {
				select {
				case msg := <-msgCh:
					// アプリケーションロジックが担当する
					if err := r.application.HandleMessage(ctx, msg.SessionID, msg.Data); err != nil {
						slog.WarnContext(ctx, "room handle message failed", "err", err)
					}
				default:
					break RECEIVE_LOOP
				}
			}
			// 送信するデータがあれば送信する このデータは１フレーム前のデータになる
		SEND_LOOP:
			for {
				select {
				case msg := <-r.sendCh: // アプリケーションが１tick前に処理したデータがここに入っている
					r.handleSendMessage(ctx, msg)
				default:
					break SEND_LOOP
				}
			}
			// ApplicationのTick()を呼び出し、戻り値があればブロードキャスト
			if data := r.application.Tick(ctx); data != nil {
				if bytes, ok := data.([]byte); ok {
					r.Broadcast(ctx, bytes)
				}
			}
		}
	}
}

// handleControlMessage はjoin/leave制御メッセージを処理します。
// TODO: []byte("join"/"leave")をRoomMessage型に置き換え
func (r *Room) handleControlMessage(msg Message) {
	switch string(msg.Data) {
	case "join":
		r.sessions[msg.SessionID] = struct{}{}
	case "leave":
		delete(r.sessions, msg.SessionID)
	default:
	}
}

func (r *Room) handleSendMessage(ctx context.Context, msg roomSend) {
	switch msg.kind {
	case roomSendBroadcast:
		r.Broadcast(ctx, msg.data)
	case roomSendTo:
		r.SendTo(ctx, msg.sessionID, msg.data)
	default:
	}
}
