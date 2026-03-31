package domain

type roomSendKind uint8

const (
	roomSendUnknown roomSendKind = iota
	roomSendBroadcast
	roomSendTo
)

type roomSend struct {
	kind      roomSendKind
	sessionID SessionID
	data      []byte
}
