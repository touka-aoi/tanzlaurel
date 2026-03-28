package domain

import "context"

//go:generate go tool mockgen -destination=./mocks/pubsub_mock.go -package=mocks . PubSub

// Topic はPubSubのトピックを表します。
type Topic string

// Message はPubSubで配送されるメッセージを表します。
type Message struct {
	SessionID SessionID
	Data      []byte
}

// PubSub はトピックベースのメッセージ配送を提供します。
type PubSub interface {
	// Subscribe はトピックを購読し、メッセージを受信するチャネルを返します。
	Subscribe(topic Topic) <-chan Message

	// Unsubscribe は購読を解除します。
	Unsubscribe(topic Topic, ch <-chan Message)

	// Publish はトピックにメッセージを配信します。
	// 配送はbest-effort（一部の購読者への配送失敗は無視して継続）。
	Publish(ctx context.Context, topic Topic, msg Message)
}
