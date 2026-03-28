package domain

import (
	"context"
	"log/slog"
	"sync"
)

const (
	// DefaultChannelBuffer はSubscribeで作成されるチャネルのデフォルトバッファサイズです。
	DefaultChannelBuffer = 1024
)

// SimplePubSub はインメモリのPubSub実装です。
type SimplePubSub struct {
	mu          sync.RWMutex
	subscribers map[Topic][]chan Message
}

// NewSimplePubSub は新しいSimplePubSubを作成します。
func NewSimplePubSub() *SimplePubSub {
	return &SimplePubSub{
		subscribers: make(map[Topic][]chan Message),
	}
}

// Subscribe はトピックを購読し、メッセージを受信するチャネルを返します。
func (p *SimplePubSub) Subscribe(topic Topic) <-chan Message {
	p.mu.Lock()
	defer p.mu.Unlock()

	ch := make(chan Message, DefaultChannelBuffer)
	p.subscribers[topic] = append(p.subscribers[topic], ch)
	return ch
}

// Unsubscribe は購読を解除します。
func (p *SimplePubSub) Unsubscribe(topic Topic, ch <-chan Message) {
	p.mu.Lock()
	defer p.mu.Unlock()

	subs := p.subscribers[topic]
	for i, sub := range subs {
		if sub == ch {
			// チャネルを閉じる
			close(sub)
			// スライスから削除
			p.subscribers[topic] = append(subs[:i], subs[i+1:]...)
			break
		}
	}

	// 購読者がいなくなったらトピックを削除
	if len(p.subscribers[topic]) == 0 {
		delete(p.subscribers, topic)
	}
}

// Publish はトピックにメッセージを配信します。
// 配送はbest-effort: チャネルが満杯の購読者はスキップして継続します。
func (p *SimplePubSub) Publish(ctx context.Context, topic Topic, msg Message) {
	p.mu.RLock()
	subs := p.subscribers[topic]
	p.mu.RUnlock()

	for _, ch := range subs {
		select {
		case <-ctx.Done():
			return
		case ch <- msg:
			// 送信成功
		default:
			// チャネルが満杯の場合はスキップしてログ出力
			slog.Warn("pub/sub: channel full, message dropped", "topic", topic)
		}
	}
}
