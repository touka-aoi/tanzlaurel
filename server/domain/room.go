package domain

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

type RoomID [16]byte

// IsEmpty はRoomIDが空（ゼロ値）かどうかを判定します
func (id RoomID) IsEmpty() bool {
	return id == RoomID{}
}

// String はRoomIDを16進数文字列で返します
func (id RoomID) String() string {
	return fmt.Sprintf("%x", id[:])
}

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
	roomTopic := Topic("room:" + r.ID.String())
	msgCh := r.pubsub.Subscribe(roomTopic)
	defer r.pubsub.Unsubscribe(roomTopic, msgCh)

	ticker := time.NewTicker(r.tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// 受信メッセージを処理
		RECEIVE_LOOP:
			for {
				select {
				case msg := <-msgCh:
					// Roomの責務に関する処理
					r.HandleMessage(ctx, msg)
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
			if data, err := r.application.Tick(ctx); err != nil {
				slog.WarnContext(ctx, "room tick failed", "err", err)
			} else if data != nil {
				r.Broadcast(ctx, data)
			}
		}
	}
}

// HandleMessage はPubSub経由で受信したメッセージを処理し、
// Control/JoinならsessionsにセッションIDを追加、Control/Leaveなら削除する。
func (r *Room) HandleMessage(ctx context.Context, msg Message) {
	if len(msg.Data) < HeaderSize+PayloadHeaderSize {
		return
	}
	payloadHeader, err := ParsePayloadHeader(msg.Data[HeaderSize:])
	if err != nil {
		return
	}
	if payloadHeader.DataType != DataTypeControl {
		return
	}
	switch ControlSubType(payloadHeader.SubType) {
	case ControlSubTypeJoin:
		r.sessions[msg.SessionID] = struct{}{}
		slog.InfoContext(ctx, "room: session added", "roomID", r.ID, "sessionID", msg.SessionID)
	case ControlSubTypeLeave:
		delete(r.sessions, msg.SessionID)
		slog.InfoContext(ctx, "room: session removed", "roomID", r.ID, "sessionID", msg.SessionID)
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
