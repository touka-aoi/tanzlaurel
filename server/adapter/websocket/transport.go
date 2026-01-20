package adapterwebsocker

import (
	"context"

	"github.com/coder/websocket"
	"github.com/touka-aoi/paralle-vs-single/server/domain"
)

type wsTransport struct {
	conn *websocket.Conn
}

func NewTransportFrom(conn *websocket.Conn) domain.Transport {
	return &wsTransport{conn: conn}
}

func (t *wsTransport) Read(ctx context.Context) ([]byte, error) {
	_, data, err := t.conn.Read(ctx)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (t *wsTransport) Write(ctx context.Context, data []byte) error {
	return t.conn.Write(ctx, websocket.MessageText, data)
}

func (t *wsTransport) Close(code int32, reason string) error {
	return t.conn.Close(websocket.StatusCode(code), reason)
}
