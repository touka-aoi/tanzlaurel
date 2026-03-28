package adapterwebsocker

import (
	"context"

	"withered/server/domain"

	"github.com/coder/websocket"
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
	return t.conn.Write(ctx, websocket.MessageBinary, data)
}

// エラーがあるならエラーを定義しておきたい
func (t *wsTransport) Close(code int32, reason string) error {
	// Close(err error) error
	// if err != nil そうじゃなかったら正常切断にする
	return t.conn.Close(websocket.StatusCode(code), reason)
}
