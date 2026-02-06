# RoomID自動割当機能の実装プラン

## 概要
JoinメッセージでRoomIDが空（ゼロ埋め）の場合、サーバー側でRoomを自動割り当てする。

## 背景
- Assignメッセージ：SessionIDを通知する（identify）責任のみ ← **変更しない**
- JoinメッセージにはRoomID（16バイト）のペイロードが必要
- クライアントがRoomIDを知らない場合でもJoinできるようにしたい

## 設計方針
- **マッチングなし（今回）**: RoomIDが空ならサーバーがデフォルトRoomを割当
- **マッチングあり（将来）**: クライアントがRoomIDを指定してJoin

## 変更後のフロー
```
Client                          Server
   |                               |
   |-------- WS Connect ---------> |
   |                               | SessionEndpoint作成
   |<-- Assign(SessionID) -------- | SessionID通知（既存のまま）
   | mySessionId = ...             |
   |---- Join(SessionID, empty) -> | RoomID=ゼロ埋め16バイト
   |                               | RoomID空 → GetRoom()でデフォルト割当
   |                               | ルームに参加
```

## 変更ファイル

### 1. server/domain/session_endpoint.go
- `handleControlMessage`のJoin処理を修正
- RoomIDが空（全てゼロ）の場合、`RoomManager.GetRoom()`でデフォルトRoomIDを取得

### 2. server/domain/room.go
- `RoomID`の型を`string`から`[16]byte`に変更
- これにより`==`で比較可能（ゼロ値との比較で空判定）
- `IsEmpty()`メソッド追加: `return id == RoomID{}`

### 3. client/src/protocol.ts
- `encodeJoinMessage`関数を追加（RoomID付きJoinメッセージ）

### 4. client/src/game.ts
- `onMessage`: Assign受信後にJoinをRoomID空で送信

## 実装順序
1. room.go: `RoomID`を`[16]byte`に変更、`IsEmpty()`追加
2. protocol.go: `ParseJoinPayload`の修正（stringから[16]byteへ）
3. session_endpoint.go: Join処理でRoomID空チェック→自動割当
4. simple_room_manager.go: defaultRoomIDの型対応
5. cmd/main.go: RoomID初期化の修正
6. mocks/room_manager_mock.go: 再生成（make generate-mock）
7. protocol.ts: `encodeJoinMessage`追加
8. game.ts: RoomID空でJoin送信

## 検証方法
1. サーバー起動: `go run server/cmd/main.go`
2. クライアント起動: `cd client && npm run dev`
3. ブラウザでアクセスし、以下を確認:
   - `invalid join payload size`エラーが出ないこと
   - `session joined room`のログが出ること
   - ゲームが正常に動作すること

## 将来の拡張（今回はスコープ外）
- クライアントがRoomIDを指定してJoin
- ルーム一覧取得API