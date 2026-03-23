# ADR-001: パケットルーティングのプロトコル設計

## Status

Accepted

## Decision

Server → Room → Application の3層構造において、以下の設計を採用する。

### MsgType 一覧

各メッセージタイプが独立した MsgType を持つ:

| MsgType | 名前 | 説明 |
|---------|------|------|
| 0x00 | RoomMessage | Room層に委譲 |
| 0x01 | Ping | 死活確認（Server→Client） |
| 0x02 | Pong | 死活確認応答（Client→Server） |
| 0x03 | Assign | SessionID通知（Server→Client） |

### 3層ヘッダー構造

メッセージタイプごとにワイヤーフォーマットが異なる（MOQT と同じ方式）。

#### Ping / Pong

```
┌─────────────────────────────────────┐
│  Transport Header (Server が解釈)    │
│  - Message Type: varint (0x01/0x02) │
│  - Total Length: varint (= 0)       │
└─────────────────────────────────────┘
```

```
[MsgType: varint][TotalLen: varint]
```

payload なし。

#### Assign

```
┌─────────────────────────────────────┐
│  Transport Header (Server が解釈)    │
│  - Message Type: varint (0x03)      │
│  - Total Length: varint (= 16)      │
│  - SessionID: 16 bytes             │
└─────────────────────────────────────┘
```

```
[MsgType: varint][TotalLen: varint][SessionID: 16bytes]
```

#### RoomMessage

```
┌─────────────────────────────────────┐
│  Transport Header (Server が解釈)    │
│  - Message Type: varint (0x00)      │
│  - Total Length: varint             │
│  - Room ID: Length-Prefixed String  │
│    [len: varint][roomID: bytes]     │
├─────────────────────────────────────┤
│  Room Header (Room が解釈)           │
│  - Room Message Type: varint        │
│    Join / Leave / AppData           │
├─────────────────────────────────────┤
│  Application Payload (AppData 時)    │
│  - Application が自由に解釈          │
└─────────────────────────────────────┘
```

```
[MsgType: varint][TotalLen: varint][RoomIDLen: varint][RoomID: bytes][RoomMsgType: varint][AppPayload: bytes]
```

### Room層メッセージタイプ

| RoomMsgType | 名前 | 説明 |
|-------------|------|------|
| 0x00 | Join | ルーム参加 |
| 0x01 | Leave | ルーム離脱 |
| 0x02 | AppData | Application層に委譲 |

### ルーティングキー

- Room ID は**可変長の Length-Prefixed String** とする
- 長さは QUIC varint (RFC 9000) でエンコードする

### 各層の責務

- **Server**: TCP コネクション管理、Transport Header のデコード、Room ID によるディスパッチ、存在しない Room へのエラー返却
- **Room**: Room Header のデコード、Join/Leave によるメンバー管理、AppData 時は Application Payload をアプリケーション層にそのまま渡す、同一 Room 内のブロードキャスト判断
- **Application**: Payload を自由にデコード・処理

### レイヤー間のデータ受け渡し

各層は自分の担当ヘッダーを解釈した後、**パース済みオフセットとバッファスライスを次の層に渡す**（オフセット方式）。上位層は下位層のヘッダーを再パースしない。

```go
// Server (session_endpoint)
msgType, totalLen, hdrLen, _ := protocol.ParseTransportHeader(buf)
payload := buf[hdrLen : hdrLen+int(totalLen)]
switch msgType {
case protocol.MsgTypePong:
    handlePong()
case protocol.MsgTypeRoomMessage:
    roomID, n, _ := protocol.ParseRoomID(payload)
    room.Handle(session, payload[n:])
}

// Room
roomMsgType, n, _ := protocol.ParseRoomHeader(buf)
switch roomMsgType {
case protocol.RoomMsgTypeJoin:  // メンバー管理
case protocol.RoomMsgTypeLeave: // メンバー管理
case protocol.RoomMsgTypeAppData:
    app.Handle(buf[n:])
}
```

### 拡張性

基本エンコーディングとして **TLV (Type-Length-Value)** パターンを採用する。未知の Type は Length を読んでスキップできるため、後方互換性を維持しつつフィールド追加が可能。

## References

- [RFC 9000 - QUIC: A UDP-Based Multiplexed and Secure Transport](https://www.rfc-editor.org/rfc/rfc9000) - varint エンコーディング
- [Media over QUIC Transport (MOQT)](https://www.ietf.org/archive/id/draft-ietf-moq-transport-07.html) - Track Namespace, TLV パラメータ交換
