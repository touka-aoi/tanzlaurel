# フロントエンド実装計画

## 目的

Vite + HTML + TypeScript + Canvas でマルチプレイヤー2Dマップクライアントを実装する。

## 技術スタック

- **ビルドツール**: Vite
- **言語**: TypeScript
- **描画**: HTML5 Canvas
- **通信**: WebSocket (バイナリ)

## 完成の定義

- [ ] 2クライアント以上が同時接続できる
- [ ] WASDで移動でき、他クライアントにリアルタイム反映される
- [ ] 入退室が他クライアントに反映される
- [ ] マップ境界が視覚的にわかる
- [ ] 自分のアクターと他者のアクターが区別できる

## ディレクトリ構成

```
client/
├── index.html
├── package.json
├── tsconfig.json
├── vite.config.ts
└── src/
    ├── main.ts           # エントリーポイント
    ├── protocol.ts       # バイナリプロトコル (Header, Payload等)
    ├── websocket.ts      # WebSocket接続管理
    ├── input.ts          # キー入力管理
    ├── renderer.ts       # Canvas描画
    └── game.ts           # ゲームループ・状態管理
```

## プロトコル仕様（サーバーとの通信）

### 送信: Input メッセージ

```
[Header (13 bytes)] + [PayloadHeader (2 bytes)] + [KeyMask (4 bytes)]

Header:
  - Version: u8 (1)
  - SessionID: u32 (0 or 取得後の値)
  - Seq: u16 (連番)
  - Length: u16 (6 = PayloadHeader + KeyMask)
  - Timestamp: u32

PayloadHeader:
  - DataType: u8 (1 = Input)
  - SubType: u8 (0)

KeyMask: u32
  - 0x01: W (上)
  - 0x02: A (左)
  - 0x04: S (下)
  - 0x08: D (右)
```

### 受信: Actor Broadcast

```
[Header (13 bytes)] + [PayloadHeader (2 bytes)] + [ActorData]

ActorData:
  [ActorCount: u16] + [Actor1] + [Actor2] + ...

Actor (16 bytes):
  - SessionID: u64
  - X: f32
  - Y: f32
```

## 実装順序

1. **Vite プロジェクトセットアップ**
   - `npm create vite@latest client -- --template vanilla-ts`
   - 依存関係インストール

2. **protocol.ts - バイナリプロトコル**
   - Header encode/decode
   - PayloadHeader encode/decode
   - InputPayload encode
   - ActorBroadcast decode

3. **websocket.ts - WebSocket管理**
   - 接続/切断
   - バイナリメッセージ送受信
   - 再接続ロジック

4. **input.ts - キー入力**
   - keydown/keyup イベント
   - キーマスク計算
   - 入力メッセージ送信

5. **renderer.ts - Canvas描画**
   - マップ背景
   - アクター描画（自分 vs 他者で色分け）
   - 座標変換（ワールド → スクリーン）

6. **game.ts - ゲームループ**
   - requestAnimationFrame
   - 状態管理（アクター一覧）
   - 自分のSessionID特定

7. **main.ts - 統合**
   - 初期化
   - イベント接続

## マップ仕様

- サイズ: 100 x 100 ワールドユニット
- 初期位置: (50, 50)
- 境界: (0, 0) ～ (100, 100)

## UI仕様

- Canvas サイズ: 800 x 600 px
- スケール: 1ワールドユニット = 8px
- 自分: 緑色の円
- 他者: 青色の円
- マップ境界: グレーの枠線

## 検証

```bash
cd client
npm install
npm run dev
# ブラウザで http://localhost:5173 を複数タブで開く
```

## 修正対象ファイル

| ファイル | 内容 |
|---------|------|
| `client/index.html` | 新規: HTML |
| `client/package.json` | 新規: 依存関係 |
| `client/tsconfig.json` | 新規: TypeScript設定 |
| `client/vite.config.ts` | 新規: Vite設定 |
| `client/src/main.ts` | 新規: エントリーポイント |
| `client/src/protocol.ts` | 新規: プロトコル |
| `client/src/websocket.ts` | 新規: WebSocket |
| `client/src/input.ts` | 新規: 入力 |
| `client/src/renderer.ts` | 新規: 描画 |
| `client/src/game.ts` | 新規: ゲームループ |