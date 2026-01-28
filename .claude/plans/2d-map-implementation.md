# 2Dマップ実装計画

## 目的

リアルタイム通信システムの動作を視覚的に確認できる**デモアプリケーション**を実装する。

複数クライアントがWebSocket経由で同じマップに接続し、WASD入力でリアルタイムに移動する様子を互いに確認できる。

## 完成の定義

- [ ] 2クライアント以上が同時接続できる
- [ ] WASDで移動でき、他クライアントにリアルタイム反映される
- [ ] 入退室が他クライアントに反映される
- [ ] マップ境界が視覚的にわかる
- [ ] 自分のアクターと他者のアクターが区別できる

## 技術スタック

| レイヤー | 技術 |
|----------|------|
| サーバー | Go（既存アーキテクチャ） |
| クライアント | MoonBit → JS + Canvas |

## 非機能要件

- 同時接続数: 2-5人（デモ用途）

## 機能要件

- マルチプレイヤー必須（複数クライアントが同時接続し互いの位置を確認）
- ブラウザで動作するクライアントを含む
- 2Dフィールド（X, Y座標のみ、軽量な新規Position2D）
- WASDで移動
- 長方形マップ（境界あり）
- 設定はYAMLで定義（ルーム起動時に読み込み）
- スポーン位置はマップ中央
- アクター追加/削除: Control Join/Leave
- Join時に現在の全アクター一覧を返す
- 衝突判定なし（アクター同士は重なれる）
- Map:Room = 1:1

## UI仕様

- アクター表示: アイコン/スプライト
- 自他区別: 自分は特別な色/形で表示

## 設計

### 1. 設定ファイル (YAML)

```yaml
# config/map.yaml
map:
  width: 100.0
  height: 100.0

player:
  speed: 1.0
```

### 2. Config構造体

```go
// server/config/config.go
type Config struct {
    Map    MapConfig    `yaml:"map"`
    Player PlayerConfig
    Input  InputConfig
}

type MapConfig struct {
    Width  float32 `yaml:"width"`
    Height float32 `yaml:"height"`
}

type PlayerConfig struct {
    Speed float32
}

func LoadConfig(path string) (*Config, error)
```

### 3. Position2D（新規・軽量）

```go
// server/domain/position2d.go
const Position2DSize = 8  // 2 * float32

type Position2D struct {
    X, Y float32
}

func ParsePosition2D(data []byte) (*Position2D, error)
func (p *Position2D) Encode() []byte
```

### 4. Map構造体

```go
// server/domain/map.go
type Map struct {
    Width  float32
    Height float32
    Actors map[SessionID]*Actor
}

type Actor struct {
    SessionID SessionID
    Position  Position2D
}

func NewMap(width, height float32) *Map
func (m *Map) SpawnAtCenter(sessionID SessionID) *Actor
func (m *Map) Move(sessionID SessionID, dx, dy float32)
func (m *Map) Remove(sessionID SessionID)
func (m *Map) GetAllActors() []*Actor
```

### 5. WitheredApplication拡張

```go
type WitheredApplication struct {
    config  *Config
    gameMap *Map
}

func NewWitheredApplication(config *Config) *WitheredApplication

// Control Join: アクター追加 + 全アクター一覧を返す
func (app *WitheredApplication) handleControlJoin(sessionID) []byte

// Control Leave: アクター削除
func (app *WitheredApplication) handleControlLeave(sessionID)

// Input: 移動処理
func (app *WitheredApplication) handleInput(sessionID, keyMask) {
    dx, dy := app.keyMaskToDirection(keyMask)
    app.gameMap.Move(sessionID, dx * app.config.Player.Speed, dy * app.config.Player.Speed)
}

// Tick: 全アクターのPosition2Dをエンコードして返す
func (app *WitheredApplication) Tick(ctx context.Context) interface{} {
    actors := app.gameMap.GetAllActors()
    return encodeActorPositions(actors)
}
```

### 6. ブロードキャストデータ形式

```
Tick出力: [ActorCount(u16)] + [Actor1] + [Actor2] + ...
Actor: [SessionID(u32)] + [X(f32)] + [Y(f32)]  = 12 bytes/actor
```

## 実装順序

1. `server/config/config.go` - Config構造体、YAML読み込み
2. `config/map.yaml` - 設定ファイル
3. `server/domain/position2d.go` - Position2D構造体
4. `server/domain/map.go` - Map, Actor構造体
5. `server/application/withered_application.go` - Config/Map統合、ロジック実装
6. テスト作成

## 修正対象ファイル

| ファイル | 変更内容 |
|---------|---------|
| `server/config/config.go` | 新規: Config構造体、YAML読み込み |
| `config/map.yaml` | 新規: 設定ファイル |
| `server/domain/position2d.go` | 新規: Position2D構造体 |
| `server/domain/map.go` | 新規: Map, Actor構造体 |
| `server/application/withered_application.go` | 修正: Config/Map統合 |

## 検証

```bash
go test ./server/config/... -v
go test ./server/domain/... -v
go test ./server/application/... -v
```
