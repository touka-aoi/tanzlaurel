package domain

import "context"

type ConnectionID string

// Connection は物理的な接続を表します。
type Connection struct {
	SessionID    SessionID
	ConnectionID ConnectionID
	transport    Transport
}

func NewConnection(sessionID SessionID, transport Transport) *Connection {
	return &Connection{
		SessionID:    sessionID,
		ConnectionID: ConnectionID(sessionID),
		transport:    transport,
	}
}

func (c *Connection) Write(ctx context.Context, data []byte) error {
	return c.transport.Write(ctx, data)
}

func (c *Connection) Read(ctx context.Context) ([]byte, error) {
	return c.transport.Read(ctx)
}

func (c *Connection) Close() {
	_ = c.transport.Close(1000, "")
}
