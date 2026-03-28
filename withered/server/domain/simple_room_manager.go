package domain

import "context"

// SimpleRoomManager は常に固定のルームを返すシンプルな実装です。
// 将来的にマッチングサービスへの問い合わせ等に差し替えることを想定しています。
type SimpleRoomManager struct {
	defaultRoomID RoomID
}

// NewSimpleRoomManager は新しいSimpleRoomManagerを作成します。
func NewSimpleRoomManager(defaultRoomID RoomID) *SimpleRoomManager {
	return &SimpleRoomManager{defaultRoomID: defaultRoomID}
}

// GetRoom はセッションに割り当てるルームIDを返します。
// この実装では常に固定のデフォルトルームを返します。
func (m *SimpleRoomManager) GetRoom(ctx context.Context, sessionID SessionID) (RoomID, error) {
	return m.defaultRoomID, nil
}

func (m *SimpleRoomManager) JoinRoom(ctx context.Context, roomID RoomID, sessionID SessionID) error {
	return nil
}

func (m *SimpleRoomManager) LeaveRoom(ctx context.Context, roomID RoomID, sessionID SessionID) error {
	return nil
}
