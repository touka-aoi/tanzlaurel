// バイナリプロトコル定義

export const HEADER_SIZE = 13;
export const PAYLOAD_HEADER_SIZE = 2;
export const INPUT_PAYLOAD_SIZE = 4;

// DataType
export const DATA_TYPE_INPUT = 1;
export const DATA_TYPE_ACTOR = 2;
export const DATA_TYPE_VOICE = 3;
export const DATA_TYPE_CONTROL = 4;

// KeyMask
export const KEY_W = 0x01;
export const KEY_A = 0x02;
export const KEY_S = 0x04;
export const KEY_D = 0x08;

export interface Header {
  version: number;
  sessionId: number;
  seq: number;
  length: number;
  timestamp: number;
}

export interface Actor {
  sessionId: bigint;
  x: number;
  y: number;
}

// Header を DataView に書き込む (13 bytes)
export function encodeHeader(view: DataView, offset: number, header: Header): void {
  view.setUint8(offset, header.version);
  view.setUint32(offset + 1, header.sessionId, true);
  view.setUint16(offset + 5, header.seq, true);
  view.setUint16(offset + 7, header.length, true);
  view.setUint32(offset + 9, header.timestamp, true);
}

// Input メッセージをエンコード
export function encodeInputMessage(sessionId: number, seq: number, keyMask: number): ArrayBuffer {
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
export function decodeActorBroadcast(data: ArrayBuffer): Actor[] {
  const view = new DataView(data);

  // Header + PayloadHeader をスキップ
  const offset = HEADER_SIZE + PAYLOAD_HEADER_SIZE;

  const actorCount = view.getUint16(offset, true);
  const actors: Actor[] = [];

  let pos = offset + 2;
  for (let i = 0; i < actorCount; i++) {
    const sessionId = view.getBigUint64(pos, true);
    const x = view.getFloat32(pos + 8, true);
    const y = view.getFloat32(pos + 12, true);
    actors.push({ sessionId, x, y });
    pos += 16;
  }

  return actors;
}

// Header をデコード
export function decodeHeader(data: ArrayBuffer): Header {
  const view = new DataView(data);

  return {
    version: view.getUint8(0),
    sessionId: view.getUint32(1, true),
    seq: view.getUint16(5, true),
    length: view.getUint16(7, true),
    timestamp: view.getUint32(9, true),
  };
}

// DataType を取得
export function getDataType(data: ArrayBuffer): number {
  const view = new DataView(data);
  return view.getUint8(HEADER_SIZE);
}