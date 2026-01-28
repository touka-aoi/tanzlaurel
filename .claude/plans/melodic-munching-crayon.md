# 2Dマップ実装計画

## 目的

WASD移動ができる2D長方形マップを実装する。

## 要件

- 2Dフィールド（X, Y座標）
- WASDで移動
- 長方形マップ（境界あり）

## 設計

### 1. Map構造体

```go
// server/domain/map.go
type Map struct {
    Width  float32  // マップ幅
    Height float32  // マップ高さ
    Actors map[SessionID]*Actor
}

type Actor struct {
    SessionID SessionID
    X, Y      float32  // 位置
}
```

### 2. 入力解釈（KeyMask）

```go
const (
    KeyW uint32 = 1 << 0  // 0x01: 上
    KeyA uint32 = 1 << 1  // 0x02: 左
    KeyS uint32 = 1 << 2  // 0x04: 下
    KeyD uint32 = 1 << 3  // 0x08: 右
)
```

### 3. WitheredApplication拡張

```go
type WitheredApplication struct {
    gameMap *Map
    speed   float32  // 移動速度
}

func (app *WitheredApplication) handleInput(...) {
    // KeyMaskから移動方向を計算
    // gameMap.Actors[sessionID]の位置を更新
}

func (app *WitheredApplication) Tick(ctx context.Context) interface{} {
    // 全アクターの位置をエンコードして返す
}
```

### 4. 位置更新ロジック

```go
func (m *Map) Move(sessionID SessionID, dx, dy float32) {
    actor := m.Actors[sessionID]
    actor.X = clamp(actor.X + dx, 0, m.Width)
    actor.Y = clamp(actor.Y + dy, 0, m.Height)
}
```

## 確認事項

1. **KeyMaskのビット割り当て**: W=0x01, A=0x02, S=0x04, D=0x08 でOK？
2. **マップサイズ**: デフォルト値は？（例: 100x100）
3. **移動速度**: 1 tick あたりの移動量は？
4. **スポーン位置**: 新規アクターの初期位置は？（中央？ランダム？）

## 実装順序

1. `server/domain/map.go` - Map, Actor構造体
2. `server/domain/input_keys.go` - KeyMask定数
3. `server/application/withered_application.go` - Map統合、移動処理
4. テスト作成

## 検証

```bash
go test ./server/domain/... -v
go test ./server/application/... -v
```
