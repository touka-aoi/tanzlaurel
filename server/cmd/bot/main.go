package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/coder/websocket"

	"withered/server/application"
	"withered/server/domain"
	"withered/utils"
)

var byteOrder = binary.LittleEndian

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	addr := utils.GetEnvDefault("ADDR", "localhost")
	port := utils.GetEnvDefault("PORT", "9090")
	botCountStr := utils.GetEnvDefault("BOT_COUNT", "3")
	botCount, err := strconv.Atoi(botCountStr)
	if err != nil {
		slog.Error("invalid BOT_COUNT", "value", botCountStr)
		os.Exit(1)
	}

	serverURL := fmt.Sprintf("ws://%s:%s/ws", addr, port)
	slog.Info("starting bots", "count", botCount, "server", serverURL)

	var wg sync.WaitGroup
	for i := range botCount {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			runBot(ctx, serverURL, id)
		}(i)
	}

	wg.Wait()
	slog.Info("all bots stopped")
}

func runBot(ctx context.Context, serverURL string, id int) {
	logger := slog.With("botID", id)

	for {
		if ctx.Err() != nil {
			return
		}
		err := botSession(ctx, serverURL, logger)
		if err != nil && ctx.Err() == nil {
			logger.Warn("bot session ended, reconnecting", "err", err)
			time.Sleep(2 * time.Second)
		}
	}
}

func botSession(ctx context.Context, serverURL string, logger *slog.Logger) error {
	conn, _, err := websocket.Dial(ctx, serverURL, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.CloseNow()

	logger.Info("connected")

	var sessionID domain.SessionID
	var seq uint16
	controller := application.NewRuleBotController()

	// ゲームステートを保持
	var actors []*application.Actor
	var bullets []*application.Bullet
	var mu sync.Mutex

	// 受信ループ
	go func() {
		for {
			if ctx.Err() != nil {
				return
			}
			_, data, err := conn.Read(ctx)
			if err != nil {
				if ctx.Err() == nil {
					logger.Warn("read error", "err", err)
				}
				return
			}

			if len(data) < domain.HeaderSize+domain.PayloadHeaderSize {
				continue
			}

			payloadHeader, err := domain.ParsePayloadHeader(data[domain.HeaderSize:])
			if err != nil {
				continue
			}

			switch payloadHeader.DataType {
			case domain.DataTypeControl:
				subType := domain.ControlSubType(payloadHeader.SubType)
				switch subType {
				case domain.ControlSubTypeAssign:
					header, err := domain.ParseHeader(data)
					if err != nil {
						continue
					}
					sessionID = domain.SessionIDFromBytes(header.SessionID)
					logger.Info("session assigned", "sessionID", sessionID)

					// Join送信
					joinMsg := encodeJoinMessage(sessionID, seq)
					seq++
					if err := conn.Write(ctx, websocket.MessageBinary, joinMsg); err != nil {
						logger.Warn("failed to send join", "err", err)
						return
					}
					logger.Info("joined room")

				case domain.ControlSubTypePing:
					pongMsg := encodeControlMessage(sessionID, seq, domain.ControlSubTypePong)
					seq++
					if err := conn.Write(ctx, websocket.MessageBinary, pongMsg); err != nil {
						logger.Warn("failed to send pong", "err", err)
						return
					}
				}

			case domain.DataTypeActor2D:
				a, b := decodeGameState(data)
				mu.Lock()
				actors = a
				bullets = b
				mu.Unlock()
			}
		}
	}()

	// 判断・送信ループ (60FPS相当)
	ticker := time.NewTicker(time.Second / 60)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			conn.Close(websocket.StatusNormalClosure, "shutdown")
			return nil
		case <-ticker.C:
			if sessionID == "" {
				continue
			}

			mu.Lock()
			self := findSelf(actors, sessionID)
			if self == nil || !self.IsAlive() {
				mu.Unlock()
				continue
			}
			action := controller.Decide(self, actors, bullets)
			mu.Unlock()

			keyMask := directionToKeyMask(action.MoveDirection)
			if keyMask == 0 {
				continue
			}

			inputMsg := encodeInputMessage(sessionID, seq, keyMask)
			seq++
			if err := conn.Write(ctx, websocket.MessageBinary, inputMsg); err != nil {
				return fmt.Errorf("write: %w", err)
			}
		}
	}
}

func findSelf(actors []*application.Actor, sessionID domain.SessionID) *application.Actor {
	for _, a := range actors {
		if a.SessionID == sessionID {
			return a
		}
	}
	return nil
}

func directionToKeyMask(dir domain.Position2D) uint32 {
	var mask uint32
	if dir.Y < -0.3 {
		mask |= 0x01 // W
	}
	if dir.X < -0.3 {
		mask |= 0x02 // A
	}
	if dir.Y > 0.3 {
		mask |= 0x04 // S
	}
	if dir.X > 0.3 {
		mask |= 0x08 // D
	}
	return mask
}

// --- プロトコルエンコード/デコード ---

func encodeHeader(buf []byte, sessionID domain.SessionID, seq uint16, length uint16) {
	buf[0] = 1 // version
	sid := sessionID.Bytes()
	copy(buf[1:17], sid[:])
	byteOrder.PutUint16(buf[17:19], seq)
	byteOrder.PutUint16(buf[19:21], length)
	byteOrder.PutUint32(buf[21:25], uint32(time.Now().UnixMilli()&0xFFFFFFFF))
}

func encodeInputMessage(sessionID domain.SessionID, seq uint16, keyMask uint32) []byte {
	payloadLen := uint16(domain.PayloadHeaderSize + 4)
	buf := make([]byte, domain.HeaderSize+int(payloadLen))
	encodeHeader(buf, sessionID, seq, payloadLen)
	buf[domain.HeaderSize] = uint8(domain.DataTypeInput)
	buf[domain.HeaderSize+1] = 0
	byteOrder.PutUint32(buf[domain.HeaderSize+domain.PayloadHeaderSize:], keyMask)
	return buf
}

func encodeControlMessage(sessionID domain.SessionID, seq uint16, subType domain.ControlSubType) []byte {
	payloadLen := uint16(domain.PayloadHeaderSize)
	buf := make([]byte, domain.HeaderSize+int(payloadLen))
	encodeHeader(buf, sessionID, seq, payloadLen)
	buf[domain.HeaderSize] = uint8(domain.DataTypeControl)
	buf[domain.HeaderSize+1] = uint8(subType)
	return buf
}

func encodeJoinMessage(sessionID domain.SessionID, seq uint16) []byte {
	payloadLen := uint16(domain.PayloadHeaderSize + domain.JoinPayloadSize)
	buf := make([]byte, domain.HeaderSize+int(payloadLen))
	encodeHeader(buf, sessionID, seq, payloadLen)
	buf[domain.HeaderSize] = uint8(domain.DataTypeControl)
	buf[domain.HeaderSize+1] = uint8(domain.ControlSubTypeJoin)
	// RoomID = all zeros (auto-assign)
	return buf
}

func decodeGameState(data []byte) ([]*application.Actor, []*application.Bullet) {
	const actorSize = 26
	const bulletSize = 34

	pos := domain.HeaderSize + domain.PayloadHeaderSize

	if len(data) < pos+2 {
		return nil, nil
	}

	// アクター部
	actorCount := int(byteOrder.Uint16(data[pos:]))
	pos += 2

	actors := make([]*application.Actor, 0, actorCount)
	for i := 0; i < actorCount; i++ {
		if pos+actorSize > len(data) {
			break
		}
		var sidBytes [16]byte
		copy(sidBytes[:], data[pos:pos+16])
		sid := domain.SessionIDFromBytes(sidBytes)
		x := math.Float32frombits(byteOrder.Uint32(data[pos+16:]))
		y := math.Float32frombits(byteOrder.Uint32(data[pos+20:]))
		hp := data[pos+24]
		state := application.ActorState(data[pos+25])
		actors = append(actors, &application.Actor{
			SessionID: sid,
			Position:  domain.Position2D{X: x, Y: y},
			HP:        hp,
			State:     state,
		})
		pos += actorSize
	}

	// 弾丸部
	if pos+2 > len(data) {
		return actors, nil
	}
	bulletCount := int(byteOrder.Uint16(data[pos:]))
	pos += 2

	bullets := make([]*application.Bullet, 0, bulletCount)
	for i := 0; i < bulletCount; i++ {
		if pos+bulletSize > len(data) {
			break
		}
		id := byteOrder.Uint16(data[pos:])
		var ownerBytes [16]byte
		copy(ownerBytes[:], data[pos+2:pos+18])
		ownerID := domain.SessionIDFromBytes(ownerBytes)
		x := math.Float32frombits(byteOrder.Uint32(data[pos+18:]))
		y := math.Float32frombits(byteOrder.Uint32(data[pos+22:]))
		vx := math.Float32frombits(byteOrder.Uint32(data[pos+26:]))
		vy := math.Float32frombits(byteOrder.Uint32(data[pos+30:]))
		bullets = append(bullets, &application.Bullet{
			ID:       id,
			OwnerID:  ownerID,
			Position: domain.Position2D{X: x, Y: y},
			Velocity: domain.Position2D{X: vx, Y: vy},
		})
		pos += bulletSize
	}

	return actors, bullets
}
