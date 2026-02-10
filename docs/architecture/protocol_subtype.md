# SubType設計

## 概要

`PayloadHeader.SubType`は`uint8`として定義。`DataType`によって解釈が変わる。

## SubTypeの解釈

| dataType | subTypeの解釈 |
|----------|---------------|
| actor2D (2) | ActorSubType (spawn=1, update=2, despawn=3) |
| actor3D (5) | ActorSubType (spawn=1, update=2, despawn=3) |
| control (4) | ControlSubType (join=1, leave=2, kick=3, ping=4, pong=5, error=6, assign=7) |

actor2Dとactor3Dは同じActorSubType定数を共有する。ペイロード構造のみが異なる:
- actor2D: Position2D (8バイト)
- actor3D: Position (28バイト) + BoneData (可変長)

## 実装

```go
type PayloadHeader struct {
    DataType DataType
    SubType  uint8  // DataTypeに応じてキャストする
}

// 使用例
switch ph.DataType {
case DataTypeActor2D:
    actorSubType := ActorSubType(ph.SubType)
case DataTypeActor3D:
    actorSubType := ActorSubType(ph.SubType)
case DataTypeControl:
    controlSubType := ControlSubType(ph.SubType)
}
```
