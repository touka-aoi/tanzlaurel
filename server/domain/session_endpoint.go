package domain

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
)

var (
	// ErrSessionAlreadyAttached はセッションに既に接続が紐付けられている場合に返されるエラーです。
	ErrSessionAlreadyAttached = errors.New("session already has an attached connection")
	// ErrSessionNotAttached はセッションに接続が紐付けられていない場合に返されるエラーです。
	ErrSessionNotAttached = errors.New("session has no attached connection")
	// ErrBackpressure は書き込みチャネルが満杯の場合に返されるエラーです。
	ErrBackpressure = errors.New("write channel is full, apply backpressure")
	// ErrInitializationFailed はセッションエンドポイントの初期化に失敗した場合に返されるエラーです。
	ErrInitializationFailed = errors.New("failed to initialize session endpoint")
)

type SessionEndpoint struct {
	ctx    context.Context
	cancel context.CancelFunc

	session    *Session
	connection *Connection
	dispatcher Dispatcher

	ctrlCh  chan endpointEvent // 制御用チャネル
	writeCh chan []byte        // 書き込み用チャネル

	// lifecycle
	closed atomic.Bool
}

func NewSessionEndpoint(session *Session, connection *Connection, dispatcher Dispatcher) (*SessionEndpoint, error) {
	if session == nil {
		return nil, ErrInitializationFailed
	}
	if connection == nil {
		return nil, ErrInitializationFailed
	}
	if dispatcher == nil {
		return nil, ErrInitializationFailed
	}
	ctx, cancel := context.WithCancel(context.Background())
	se := &SessionEndpoint{
		ctx:        ctx,
		cancel:     cancel,
		session:    session,
		connection: connection,
		dispatcher: dispatcher,
		ctrlCh:     make(chan endpointEvent, 16),
		writeCh:    make(chan []byte, 1024),
	}
	return se, nil
}

func (se *SessionEndpoint) Run() error {
	eg, ctx := errgroup.WithContext(se.ctx)
	eg.Go(func() error {
		se.ownerLoop(ctx)
		return nil
	})
	eg.Go(func() error {
		se.readLoop(ctx)
		return nil
	})
	eg.Go(func() error {
		se.writeLoop(ctx)
		return nil
	})
	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}

func (se *SessionEndpoint) Send(data []byte) error {
	select {
	case se.writeCh <- data:
		return nil
	default:
		return ErrBackpressure
	}
}

func (se *SessionEndpoint) Close(ctx context.Context) {
	se.sendCtrlEvent(ctx, endpointEvent{kind: evClose, err: nil})
}

func (se *SessionEndpoint) ForceClose() {
	se.close()
}

// ownerLoop は論理セッションの状態を監視し、必要に応じて接続の管理を行います。
func (se *SessionEndpoint) ownerLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-se.ctrlCh:
			se.handleControlEvent(ctx, ev)
		case <-ticker.C:
			ok, reason := se.session.IsIdle(30 * time.Second)
			if ok {
				se.handleControlEvent(ctx, endpointEvent{
					kind: evClose,
					err:  errors.New(reason.String()),
				})
			}
		}
	}
}

func (se *SessionEndpoint) readLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			data, err := se.connection.Read(ctx)
			if err != nil {
				se.sendCtrlEvent(ctx, endpointEvent{kind: evReadError, err: err})
				continue
			}
			se.session.TouchRead()
			err = se.dispatcher.Dispatch(ctx, data)
			if err != nil {
				se.sendCtrlEvent(ctx, endpointEvent{kind: evDispatchError, err: err})
			}
		}
	}
}

func (se *SessionEndpoint) writeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case data := <-se.writeCh:
			err := se.connection.Write(ctx, data)
			if err != nil {
				se.sendCtrlEvent(ctx, endpointEvent{kind: evWriteError, err: err})
				continue
			}
			se.session.TouchWrite()
		}
	}
}

func (se *SessionEndpoint) close() {
	if !se.closed.CompareAndSwap(false, true) {
		return
	}
	se.cancel()
	se.session.Close()
	se.connection.Close()
	_ = se.dispatcher.Dispatch(se.ctx, nil)
}

// handleControlEvent は制御チャネルからのイベントを処理し論理セッションの状態を更新する唯一の関数です。
func (se *SessionEndpoint) handleControlEvent(ctx context.Context, ev endpointEvent) {
	switch ev.kind {
	case evClose:
		se.close()
	case evPong:
		se.session.TouchPong()
	case evReadError:
		return
	case evWriteError:
		return
	case evDispatchError:
		return

	default:
		slog.WarnContext(ctx, "unknown endpoint event kind", "kind", ev.kind)
	}
}

func (se *SessionEndpoint) sendCtrlEvent(ctx context.Context, ev endpointEvent) {
	select {
	case se.ctrlCh <- ev:
	case <-ctx.Done():
	}
}
