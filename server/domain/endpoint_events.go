package domain

type endpointEventKind uint8

const (
	// unknown
	unknown endpointEventKind = iota

	// I/O
	evPong // pong を受信した

	// ctrl
	evClose // セッション終了
)

type endpointEvent struct {
	kind endpointEventKind
	err  error
}
