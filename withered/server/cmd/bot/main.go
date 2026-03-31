package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"withered/server/domain/protocol"
	"withered/utils"

	"github.com/coder/websocket"
)

const defaultRoomID = "00000000000000000000000000000001"

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	botCount, _ := strconv.Atoi(utils.GetEnvDefault("BOT_COUNT", "3"))
	serverURL := utils.GetEnvDefault("SERVER_URL", "ws://localhost:9090/ws")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var connected atomic.Int64
	var errors atomic.Int64

	// 段階的に接続（10ms間隔）
	for i := range botCount {
		go func() {
			if err := runBot(ctx, serverURL, &connected); err != nil {
				errors.Add(1)
				slog.Warn("bot stopped", "id", i, "err", err)
			}
			connected.Add(-1)
		}()

		if (i+1)%1000 == 0 {
			slog.Info("bots launching", "launched", i+1, "total", botCount)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Millisecond):
		}
	}

	slog.Info("all bots launched", "count", botCount)

	// 定期的にステータス出力
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("shutting down", "connected", connected.Load(), "errors", errors.Load())
			return
		case <-ticker.C:
			slog.Info("status", "connected", connected.Load(), "errors", errors.Load())
		}
	}
}

func runBot(ctx context.Context, serverURL string, connected *atomic.Int64) error {
	conn, _, err := websocket.Dial(ctx, serverURL, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.CloseNow()
	connected.Add(1)

	// Assignメッセージを待つ
	if err := waitAssign(ctx, conn); err != nil {
		return fmt.Errorf("assign: %w", err)
	}

	// Room Join
	joinMsg := protocol.EncodeRoomMessage(defaultRoomID, protocol.RoomMsgTypeJoin, nil)
	if err := conn.Write(ctx, websocket.MessageBinary, joinMsg); err != nil {
		return fmt.Errorf("join: %w", err)
	}

	// readLoop と writeLoop を並行実行
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 2)

	go func() {
		errCh <- readLoop(ctx, conn)
	}()
	go func() {
		errCh <- writeLoop(ctx, conn)
	}()

	// どちらかがエラーになったら終了
	err = <-errCh
	cancel()

	// Leave送信（best effort）
	leaveCtx, leaveCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer leaveCancel()
	leaveMsg := protocol.EncodeRoomMessage(defaultRoomID, protocol.RoomMsgTypeLeave, nil)
	_ = conn.Write(leaveCtx, websocket.MessageBinary, leaveMsg)
	_ = conn.Close(websocket.StatusNormalClosure, "")

	return err
}

func waitAssign(ctx context.Context, conn *websocket.Conn) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, data, err := conn.Read(ctx)
	if err != nil {
		return err
	}

	msgType, _, _, err := protocol.ParseTransportHeader(data)
	if err != nil {
		return err
	}
	if msgType != protocol.MsgTypeAssign {
		return fmt.Errorf("expected Assign, got %d", msgType)
	}
	return nil
}

func readLoop(ctx context.Context, conn *websocket.Conn) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		_, data, err := conn.Read(ctx)
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}

		if len(data) < 1 {
			continue
		}

		// Pingに応答
		msgType, _, _, err := protocol.ParseTransportHeader(data)
		if err != nil {
			continue
		}
		if msgType == protocol.MsgTypePing {
			pong := protocol.EncodePong()
			if err := conn.Write(ctx, websocket.MessageBinary, pong); err != nil {
				return fmt.Errorf("pong: %w", err)
			}
		}
		// Snapshot等は読み捨て
	}
}

func writeLoop(ctx context.Context, conn *websocket.Conn) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			keyMask := randomKeyMask()
			appPayload := encodeInputPayload(keyMask)
			msg := protocol.EncodeRoomMessage(defaultRoomID, protocol.RoomMsgTypeAppData, appPayload)
			if err := conn.Write(ctx, websocket.MessageBinary, msg); err != nil {
				return fmt.Errorf("write: %w", err)
			}
		}
	}
}

func randomKeyMask() uint32 {
	// WASD の組み合わせをランダムに生成
	return rand.Uint32N(16) // 0x00 ~ 0x0F
}

func encodeInputPayload(keyMask uint32) []byte {
	// [DataType=0(Input)][SubType=0][KeyMask: u32 LE]
	buf := make([]byte, 6)
	buf[0] = 0 // DataTypeInput
	buf[1] = 0 // SubType
	binary.LittleEndian.PutUint32(buf[2:], keyMask)
	return buf
}
