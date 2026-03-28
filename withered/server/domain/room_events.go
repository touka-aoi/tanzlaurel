package domain

type roomCtrlKind uint8

const (
	roomCtrlUnknown roomCtrlKind = iota
	roomCtrlAdd
	roomCtrlRemove
)

type roomCtrl struct {
	kind      roomCtrlKind
	sessionID SessionID
}

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
