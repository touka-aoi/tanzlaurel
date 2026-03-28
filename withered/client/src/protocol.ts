// ECSブランチ バイナリプロトコル定義
// Transport: [MsgType(varint)][TotalLen(varint)][Payload]
// RoomMessage Payload: [RoomIDLen(varint)][RoomID(string)][RoomMsgType(varint)][AppPayload]

// --- QUIC Varint (RFC 9000 Section 16) ---

export function readVarint(buf: Uint8Array, offset: number): { value: number; n: number } {
  const prefix = buf[offset] >> 6;
  const length = 1 << prefix;
  let value = 0;
  switch (length) {
    case 1:
      value = buf[offset] & 0x3f;
      break;
    case 2:
      value = ((buf[offset] & 0x3f) << 8) | buf[offset + 1];
      break;
    case 4:
      value =
        ((buf[offset] & 0x3f) << 24) |
        (buf[offset + 1] << 16) |
        (buf[offset + 2] << 8) |
        buf[offset + 3];
      break;
    case 8:
      // JSは53ビット精度なので上位を無視
      value =
        ((buf[offset + 2] & 0x3f) << 40) |
        (buf[offset + 3] << 32) |
        (buf[offset + 4] << 24) |
        (buf[offset + 5] << 16) |
        (buf[offset + 6] << 8) |
        buf[offset + 7];
      break;
  }
  return { value, n: length };
}

export function appendVarint(buf: number[], value: number): void {
  if (value <= 63) {
    buf.push(value);
  } else if (value <= 16383) {
    buf.push(0x40 | (value >> 8), value & 0xff);
  } else if (value <= 1073741823) {
    buf.push(
      0x80 | (value >> 24),
      (value >> 16) & 0xff,
      (value >> 8) & 0xff,
      value & 0xff
    );
  }
}

// --- MsgType ---
export const MSG_TYPE_ROOM_MESSAGE = 0x00;
export const MSG_TYPE_PING = 0x01;
export const MSG_TYPE_PONG = 0x02;
export const MSG_TYPE_ASSIGN = 0x03;

// --- RoomMsgType ---
export const ROOM_MSG_TYPE_JOIN = 0x00;
export const ROOM_MSG_TYPE_LEAVE = 0x01;
export const ROOM_MSG_TYPE_APP_DATA = 0x02;

// --- App DataType ---
export const DATA_TYPE_INPUT = 0;
export const DATA_TYPE_SNAPSHOT = 5;
export const PAYLOAD_HEADER_SIZE = 2;
export const INPUT_PAYLOAD_SIZE = 4;
export const SESSION_ID_SIZE = 16;

// --- ECS Tag ---
export const TAG_PLAYER = 1;
export const TAG_BOT = 2;
export const TAG_BULLET = 3;

// --- Component Mask ---
const M_POS = 1 << 0;
const M_VEL = 1 << 1;
const M_HP = 1 << 2;
const M_LIFE = 1 << 3;
const M_TTL = 1 << 4;
const M_OWN = 1 << 5;
const M_TAG = 1 << 6;

// ActorState
export const STATE_ALIVE = 0x01;
export const STATE_RESPAWNING = 0x02;
export const KIND_BOT = 0x10;

export function isAlive(state: number): boolean {
  return (state & STATE_ALIVE) !== 0;
}

export function isBot(tag: number): boolean {
  return tag === TAG_BOT;
}

// KeyMask
export const KEY_W = 0x01;
export const KEY_A = 0x02;
export const KEY_S = 0x04;
export const KEY_D = 0x08;

// --- Data Types ---

export interface Actor {
  entityId: number;
  x: number;
  y: number;
  hp: number;
  state: number;
  tag: number;
}

export interface Bullet {
  entityId: number;
  ownerId: number;
  x: number;
  y: number;
  vx: number;
  vy: number;
}

export interface GameState {
  actors: Actor[];
  bullets: Bullet[];
}

// --- Transport Header ---

export interface TransportMessage {
  msgType: number;
  payload: Uint8Array;
}

export function parseTransportHeader(data: Uint8Array): TransportMessage {
  const { value: msgType, n: n1 } = readVarint(data, 0);
  const { value: totalLen, n: n2 } = readVarint(data, n1);
  const payload = data.subarray(n1 + n2, n1 + n2 + totalLen);
  return { msgType, payload };
}

// --- Assign ---

export function decodeAssign(payload: Uint8Array): Uint8Array {
  // payload = [SessionID: 16bytes]
  return new Uint8Array(payload.buffer, payload.byteOffset, SESSION_ID_SIZE);
}

// --- Encode helpers ---

export function encodePong(): ArrayBuffer {
  const buf: number[] = [];
  appendVarint(buf, MSG_TYPE_PONG);
  appendVarint(buf, 0);
  return new Uint8Array(buf).buffer;
}

export function encodeRoomMessage(
  roomId: string,
  roomMsgType: number,
  appPayload?: Uint8Array
): ArrayBuffer {
  // Build inner payload: [RoomIDLen(varint)][RoomID][RoomMsgType(varint)][AppPayload]
  const encoder = new TextEncoder();
  const roomIdBytes = encoder.encode(roomId);

  const inner: number[] = [];
  appendVarint(inner, roomIdBytes.length);
  for (const b of roomIdBytes) inner.push(b);
  appendVarint(inner, roomMsgType);
  if (appPayload) {
    for (const b of appPayload) inner.push(b);
  }

  // Wrap: [MsgType(varint)][TotalLen(varint)][inner]
  const outer: number[] = [];
  appendVarint(outer, MSG_TYPE_ROOM_MESSAGE);
  appendVarint(outer, inner.length);
  for (const b of inner) outer.push(b);

  return new Uint8Array(outer).buffer;
}

export function encodeInputAppPayload(keyMask: number): Uint8Array {
  // [PayloadHeader(2bytes)][InputPayload(4bytes)]
  const buf = new ArrayBuffer(PAYLOAD_HEADER_SIZE + INPUT_PAYLOAD_SIZE);
  const view = new DataView(buf);
  view.setUint8(0, DATA_TYPE_INPUT);
  view.setUint8(1, 0);
  view.setUint32(2, keyMask, true);
  return new Uint8Array(buf);
}

// --- Snapshot Decoder ---

export function decodeSnapshot(payload: Uint8Array): GameState {
  // payload = [PayloadHeader(2)][Body]
  // Body = [EntityCount(u16)][Entity...]
  // Entity = [EntityID(u16)][Mask(u8)][Components...]
  const view = new DataView(
    payload.buffer,
    payload.byteOffset,
    payload.byteLength
  );
  let pos = PAYLOAD_HEADER_SIZE; // skip PayloadHeader

  const entityCount = view.getUint16(pos, true);
  pos += 2;

  const actors: Actor[] = [];
  const bullets: Bullet[] = [];

  for (let i = 0; i < entityCount; i++) {
    const entityId = view.getUint16(pos, true);
    pos += 2;
    const mask = view.getUint8(pos);
    pos += 1;

    let x = 0,
      y = 0,
      vx = 0,
      vy = 0;
    let hp = 0,
      state = 0,
      ttl = 0,
      ownerId = 0,
      tag = 0;

    if (mask & M_POS) {
      x = view.getFloat32(pos, true);
      pos += 4;
      y = view.getFloat32(pos, true);
      pos += 4;
    }
    if (mask & M_VEL) {
      vx = view.getFloat32(pos, true);
      pos += 4;
      vy = view.getFloat32(pos, true);
      pos += 4;
    }
    if (mask & M_HP) {
      hp = view.getUint8(pos);
      pos += 1;
    }
    if (mask & M_LIFE) {
      state = view.getUint8(pos);
      pos += 1;
    }
    if (mask & M_TTL) {
      ttl = view.getUint16(pos, true);
      pos += 2;
    }
    if (mask & M_OWN) {
      ownerId = view.getUint16(pos, true);
      pos += 2;
    }
    if (mask & M_TAG) {
      tag = view.getUint8(pos);
      pos += 1;
    }

    if (tag === TAG_PLAYER || tag === TAG_BOT) {
      actors.push({ entityId, x, y, hp, state, tag });
    } else if (tag === TAG_BULLET) {
      bullets.push({ entityId, ownerId, x, y, vx, vy });
    }
  }

  return { actors, bullets };
}

// --- Helpers ---

export function describeKeyMask(mask: number): string {
  const keys: string[] = [];
  if (mask & KEY_W) keys.push("W");
  if (mask & KEY_A) keys.push("A");
  if (mask & KEY_S) keys.push("S");
  if (mask & KEY_D) keys.push("D");
  return keys.join("+") || "none";
}

export function sessionIdToString(sessionId: Uint8Array): string {
  return Array.from(sessionId)
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}
