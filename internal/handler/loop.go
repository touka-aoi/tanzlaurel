package handler

import (
	"context"
	"errors"
	"log"
	"sync/atomic"
	"time"
)

// Handler processes requests submitted to the loop.
type Handler interface {
	Handle(req interface{}) error
}

// Config controls the behaviour of the single thread loop.
type Config struct {
	Handler   Handler
	QueueSize int
	Logger    *log.Logger
}

// Loop delivers incoming requests to the provided handler on a single goroutine.
type Loop struct {
	handler Handler
	queue   chan interface{}
	logger  *log.Logger

	started int32
	stopped int32

	done chan struct{}
}

// New creates a Loop with the supplied configuration.
func New(cfg Config) (*Loop, error) {
	if cfg.Handler == nil {
		return nil, errors.New("loop: handler is required")
	}
	queueSize := cfg.QueueSize
	if queueSize <= 0 {
		queueSize = 1024
	}
	logger := cfg.Logger
	if logger == nil {
		logger = log.Default()
	}
	return &Loop{
		handler: cfg.Handler,
		queue:   make(chan interface{}, queueSize),
		logger:  logger,
		done:    make(chan struct{}),
	}, nil
}

// Start launches the single-thread loop. It must be called once.
func (l *Loop) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&l.started, 0, 1) {
		return errors.New("loop: start called multiple times")
	}
	go l.run(ctx)
	return nil
}

func (l *Loop) run(ctx context.Context) {
	defer close(l.done)
	for {
		select {
		case <-ctx.Done():
			l.logger.Printf("loop: context cancelled, shutting down: %v", ctx.Err())
			return
		case req, ok := <-l.queue:
			if !ok {
				l.logger.Printf("loop: queue closed, exiting")
				return
			}
			if err := l.handler.Handle(req); err != nil {
				l.logger.Printf("loop: handler error: %v", err)
			}
		}
	}
}

// Submit enqueues a request to be processed by the loop.
func (l *Loop) Submit(ctx context.Context, req interface{}) error {
	if atomic.LoadInt32(&l.started) == 0 {
		return errors.New("loop: not started")
	}
	if atomic.LoadInt32(&l.stopped) == 1 {
		return errors.New("loop: stopped")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case l.queue <- req:
		return nil
	}
}

// Stop drains the loop and waits for graceful completion.
func (l *Loop) Stop(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&l.stopped, 0, 1) {
		return errors.New("loop: stop called multiple times")
	}
	close(l.queue)
	select {
	case <-l.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// DrainTimeout closes the queue and waits for completion with the given timeout.
func (l *Loop) DrainTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return l.Stop(ctx)
}
