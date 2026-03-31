package domain

import "context"

type Broadcaster interface {
	Broadcast(ctx context.Context, data []byte)
	SendTo(ctx context.Context, sessionID SessionID, data []byte)
}

type PubSubBroadcaster struct {
	registry SessionRegistry
	pubsub   PubSub
}

func NewPubSubBroadcaster(registry SessionRegistry, pubsub PubSub) *PubSubBroadcaster {
	return &PubSubBroadcaster{
		registry: registry,
		pubsub:   pubsub,
	}
}

func (b *PubSubBroadcaster) Broadcast(ctx context.Context, data []byte) {
	for _, sessionID := range b.registry.List() {
		topic := Topic("session:" + sessionID.String())
		b.pubsub.Publish(ctx, topic, Message{Data: data})
	}
}

func (b *PubSubBroadcaster) SendTo(ctx context.Context, sessionID SessionID, data []byte) {
	topic := Topic("session:" + sessionID.String())
	b.pubsub.Publish(ctx, topic, Message{Data: data})
}
