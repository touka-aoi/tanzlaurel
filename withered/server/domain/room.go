package domain

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"withered/server/domain/protocol"
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
	ID          RoomID
	registry    SessionRegistry
	broadcaster Broadcaster

	pubsub      PubSub      // Run()のSubscribe/Unsubscribeで使用
	application Application // 外部からアプリケーションロジックを注入できる

	sendCh chan roomSend

	tickInterval time.Duration
}

func NewRoom(id RoomID, pubsub PubSub, broadcaster Broadcaster, registry SessionRegistry, application Application) *Room {
	return &Room{
		ID:           id,
		registry:     registry,
		broadcaster:  broadcaster,
		pubsub:       pubsub,
		application:  application,
		sendCh:       make(chan roomSend, 1024),
		tickInterval: time.Second / 60,
	}
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
					r.HandleMessage(ctx, msg)
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
				r.broadcaster.Broadcast(ctx, data)
			}
		}
	}
}

// HandleMessage はPubSub経由で受信したメッセージを処理する。
// msg.Data はTransportHeaderを除いたペイロード（RoomID + RoomHeader + AppPayload）。
// Join/Leaveならsessions管理、AppDataならapplication.HandleMessageに委譲する。
func (r *Room) HandleMessage(ctx context.Context, msg Message) {
	// RoomIDをスキップ（session_endpointで既にパース済みだが、Roomにはペイロード全体が渡される）
	_, n1, err := protocol.ParseRoomID(msg.Data)
	if err != nil {
		slog.WarnContext(ctx, "room: failed to parse room ID", "err", err)
		return
	}

	roomMsgType, n2, err := protocol.ParseRoomHeader(msg.Data[n1:])
	if err != nil {
		slog.WarnContext(ctx, "room: failed to parse room header", "err", err)
		return
	}

	switch roomMsgType {
	case protocol.RoomMsgTypeJoin:
		r.registry.Add(msg.SessionID)
		slog.InfoContext(ctx, "room: session added", "roomID", r.ID, "sessionID", msg.SessionID)
	case protocol.RoomMsgTypeLeave:
		r.registry.Remove(msg.SessionID)
		slog.InfoContext(ctx, "room: session removed", "roomID", r.ID, "sessionID", msg.SessionID)
	case protocol.RoomMsgTypeAppData:
		appPayload := msg.Data[n1+n2:]
		if err := r.application.HandleMessage(ctx, msg.SessionID, appPayload); err != nil {
			slog.WarnContext(ctx, "room: application handle message failed", "err", err)
		}
	}
}

func (r *Room) handleSendMessage(ctx context.Context, msg roomSend) {
	switch msg.kind {
	case roomSendBroadcast:
		r.broadcaster.Broadcast(ctx, msg.data)
	case roomSendTo:
		r.broadcaster.SendTo(ctx, msg.sessionID, msg.data)
	default:
	}
}
