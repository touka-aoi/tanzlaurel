package domain

import (
	"context"
	"log/slog"
	"time"
)

// HeartbeatService は定期的にpingメッセージを送信する死活監視サービスです。
type HeartbeatService struct {
	pingInterval time.Duration
	session      *Session
	writeCh      chan<- []byte
}

// NewHeartbeatService は新しいHeartbeatServiceを生成します。
func NewHeartbeatService(pingInterval time.Duration, session *Session, writeCh chan<- []byte) *HeartbeatService {
	return &HeartbeatService{
		pingInterval: pingInterval,
		session:      session,
		writeCh:      writeCh,
	}
}

// Run はpingInterval間隔でpingメッセージをwriteChに送信します。
// ctxがキャンセルされると終了します。
func (h *HeartbeatService) Run(ctx context.Context) {
	ticker := time.NewTicker(h.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pingMsg := EncodePingMessage(h.session.ID())
			select {
			case h.writeCh <- pingMsg:
				slog.DebugContext(ctx, "heartbeat: ping sent", "sessionID", h.session.ID())
			default:
				slog.WarnContext(ctx, "heartbeat: writeCh full, ping dropped", "sessionID", h.session.ID())
			}
		}
	}
}
