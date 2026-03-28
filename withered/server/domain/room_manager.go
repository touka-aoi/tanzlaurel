package domain

import "context"

//go:generate go tool mockgen -destination=./mocks/room_manager_mock.go -package=mocks . RoomManager

// RoomManager はセッションに対するルーム割り当てを管理します。
// 将来的にマッチングサービスへの問い合わせ等に差し替え可能です。
type RoomManager interface {
	// GetRoom はセッションに割り当てるルームIDを返します。
	GetRoom(ctx context.Context, sessionID SessionID) (RoomID, error)
	JoinRoom(ctx context.Context, roomID RoomID, sessionID SessionID) error
	LeaveRoom(ctx context.Context, roomID RoomID, sessionID SessionID) error
}
