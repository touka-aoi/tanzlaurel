// バイナリプロトコル定義

export const HEADER_SIZE = 25; // 1 + 16 + 2 + 2 + 4
export const PAYLOAD_HEADER_SIZE = 2;
export const INPUT_PAYLOAD_SIZE = 4;
export const SESSION_ID_SIZE = 16;

// DataType
export const DATA_TYPE_INPUT = 1;
export const DATA_TYPE_ACTOR = 2;
export const DATA_TYPE_VOICE = 3;
export const DATA_TYPE_CONTROL = 4;

// Control SubType
export const CONTROL_SUBTYPE_JOIN = 1;
export const CONTROL_SUBTYPE_LEAVE = 2;
export const CONTROL_SUBTYPE_ASSIGN = 7;

// KeyMask
export const KEY_W = 0x01;
export const KEY_A = 0x02;
export const KEY_S = 0x04;
export const KEY_D = 0x08;

export interface Header {
  version: number;
  sessionId: Uint8Array; // 16バイト
  seq: number;
  length: number;
  timestamp: number;
}

export interface Actor {
  sessionId: Uint8Array; // 16バイト
  x: number;
  y: number;
}

// Header を DataView に書き込む (25 bytes)
export function encodeHeader(view: DataView, offset: number, header: Header): void {
  view.setUint8(offset, header.version);
  // sessionId: 16バイト
  for (let i = 0; i < SESSION_ID_SIZE; i++) {
    view.setUint8(offset + 1 + i, header.sessionId[i] || 0);
  }
  view.setUint16(offset + 17, header.seq, true);
  view.setUint16(offset + 19, header.length, true);
  view.setUint32(offset + 21, header.timestamp, true);
}

// Input メッセージをエンコード
export function encodeInputMessage(sessionId: Uint8Array, seq: number, keyMask: number): ArrayBuffer {
  const payloadLength = PAYLOAD_HEADER_SIZE + INPUT_PAYLOAD_SIZE;
  const totalLength = HEADER_SIZE + payloadLength;

  const buf = new ArrayBuffer(totalLength);
  const view = new DataView(buf);

  // Header
  const header: Header = {
    version: 1,
    sessionId,
    seq,
    length: payloadLength,
    timestamp: Date.now() & 0xFFFFFFFF,
  };
  encodeHeader(view, 0, header);

  // PayloadHeader
  view.setUint8(HEADER_SIZE, DATA_TYPE_INPUT);
  view.setUint8(HEADER_SIZE + 1, 0); // subType

  // InputPayload
  view.setUint32(HEADER_SIZE + PAYLOAD_HEADER_SIZE, keyMask, true);

  return buf;
}

// Actor Broadcast をデコード
// 各Actor: SessionID([16]byte) + X(f32) + Y(f32) = 24 bytes
export function decodeActorBroadcast(data: ArrayBuffer): Actor[] {
  const view = new DataView(data);
  const ACTOR_SIZE = 24; // 16 + 4 + 4

  // Header + PayloadHeader をスキップ
  const offset = HEADER_SIZE + PAYLOAD_HEADER_SIZE;

  const actorCount = view.getUint16(offset, true);
  const actors: Actor[] = [];

  let pos = offset + 2;
  for (let i = 0; i < actorCount; i++) {
    const sessionId = new Uint8Array(data, pos, SESSION_ID_SIZE);
    const x = view.getFloat32(pos + 16, true);
    const y = view.getFloat32(pos + 20, true);
    actors.push({ sessionId: new Uint8Array(sessionId), x, y });
    pos += ACTOR_SIZE;
  }

  return actors;
}

// Header をデコード
export function decodeHeader(data: ArrayBuffer): Header {
  const view = new DataView(data);
  const sessionId = new Uint8Array(data, 1, SESSION_ID_SIZE);

  return {
    version: view.getUint8(0),
    sessionId: new Uint8Array(sessionId),
    seq: view.getUint16(17, true),
    length: view.getUint16(19, true),
    timestamp: view.getUint32(21, true),
  };
}

// DataType を取得
export function getDataType(data: ArrayBuffer): number {
  const view = new DataView(data);
  return view.getUint8(HEADER_SIZE);
}

// Control SubType を取得
export function getControlSubType(data: ArrayBuffer): number {
  const view = new DataView(data);
  return view.getUint8(HEADER_SIZE + 1);
}

// Assign メッセージからセッションIDをデコード
export function decodeAssignMessage(data: ArrayBuffer): Uint8Array {
  const header = decodeHeader(data);
  return header.sessionId;
}

// Control メッセージをエンコード
export function encodeControlMessage(sessionId: Uint8Array, seq: number, subType: number): ArrayBuffer {
  const payloadLength = PAYLOAD_HEADER_SIZE;
  const totalLength = HEADER_SIZE + payloadLength;

  const buf = new ArrayBuffer(totalLength);
  const view = new DataView(buf);

  // Header
  const header: Header = {
    version: 1,
    sessionId,
    seq,
    length: payloadLength,
    timestamp: Date.now() & 0xFFFFFFFF,
  };
  encodeHeader(view, 0, header);

  // PayloadHeader
  view.setUint8(HEADER_SIZE, DATA_TYPE_CONTROL);
  view.setUint8(HEADER_SIZE + 1, subType);

  return buf;
}

// RoomIDサイズ
export const ROOM_ID_SIZE = 16;

// Join メッセージをエンコード（RoomID付き）
// roomIdが省略またはnullの場合、ゼロ埋め16バイト（サーバーが自動割当）
export function encodeJoinMessage(sessionId: Uint8Array, seq: number, roomId?: Uint8Array | null): ArrayBuffer {
  const payloadLength = PAYLOAD_HEADER_SIZE + ROOM_ID_SIZE;
  const totalLength = HEADER_SIZE + payloadLength;

  const buf = new ArrayBuffer(totalLength);
  const view = new DataView(buf);

  // Header
  const header: Header = {
    version: 1,
    sessionId,
    seq,
    length: payloadLength,
    timestamp: Date.now() & 0xFFFFFFFF,
  };
  encodeHeader(view, 0, header);

  // PayloadHeader
  view.setUint8(HEADER_SIZE, DATA_TYPE_CONTROL);
  view.setUint8(HEADER_SIZE + 1, CONTROL_SUBTYPE_JOIN);

  // JoinPayload: RoomID (16バイト)
  const roomIdOffset = HEADER_SIZE + PAYLOAD_HEADER_SIZE;
  if (roomId && roomId.length >= ROOM_ID_SIZE) {
    for (let i = 0; i < ROOM_ID_SIZE; i++) {
      view.setUint8(roomIdOffset + i, roomId[i]);
    }
  }
  // roomIdがnull/undefined、または長さが足りない場合は0埋め（ArrayBufferはデフォルトで0）

  return buf;
}

// SessionIDを比較
export function sessionIdEquals(a: Uint8Array | null, b: Uint8Array | null): boolean {
  if (a === null || b === null) return a === b;
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++) {
    if (a[i] !== b[i]) return false;
  }
  return true;
}

// KeyMaskを人間が読める文字列に変換
export function describeKeyMask(mask: number): string {
  const keys: string[] = [];
  if (mask & KEY_W) keys.push("W");
  if (mask & KEY_A) keys.push("A");
  if (mask & KEY_S) keys.push("S");
  if (mask & KEY_D) keys.push("D");
  return keys.join("+") || "none";
}

// SessionIDを文字列に変換（デバッグ用）
export function sessionIdToString(sessionId: Uint8Array): string {
  return Array.from(sessionId)
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}