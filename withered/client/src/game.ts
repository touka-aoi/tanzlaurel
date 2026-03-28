// ゲームループ・状態管理

import type { Actor, Bullet } from "./protocol";
import {
  MSG_TYPE_ASSIGN,
  MSG_TYPE_PING,
  MSG_TYPE_ROOM_MESSAGE,
  ROOM_MSG_TYPE_JOIN,
  ROOM_MSG_TYPE_LEAVE,
  ROOM_MSG_TYPE_APP_DATA,
  DATA_TYPE_SNAPSHOT,
  PAYLOAD_HEADER_SIZE,
  parseTransportHeader,
  decodeAssign,
  decodeSnapshot,
  describeKeyMask,
  encodeRoomMessage,
  encodeInputAppPayload,
  encodePong,
  sessionIdToString,
  readVarint,
} from "./protocol";
import { WebSocketClient } from "./websocket";
import { InputManager } from "./input";
import { Renderer } from "./renderer";
import { eventLogger } from "./event-logger";

const SERVER_URL = import.meta.env.VITE_SERVER_URL || "ws://localhost:9090/ws";

// デフォルトルームID（サーバー側の固定ID: 00000000000000000000000000000001）
const DEFAULT_ROOM_ID = "00000000000000000000000000000001";

export class Game {
  private ws: WebSocketClient;
  private input: InputManager;
  private renderer: Renderer;
  private canvas: HTMLCanvasElement;

  private actors: Actor[] = [];
  private bullets: Bullet[] = [];
  private lastBulletUpdate: number = 0;
  private mySessionId: Uint8Array | null = null;
  private myEntityId: number | null = null;
  private connected: boolean = false;
  private prevKeyMask: number = -1;

  constructor(canvas: HTMLCanvasElement) {
    this.canvas = canvas;
    this.canvas.dataset.connected = "false";
    this.canvas.dataset.sessionId = "";
    this.canvas.dataset.roomJoined = "false";
    this.canvas.dataset.playerCount = "0";

    this.input = new InputManager();
    this.renderer = new Renderer(canvas);
    this.ws = new WebSocketClient(
      SERVER_URL,
      this.onMessage.bind(this),
      this.onConnect.bind(this),
      this.onDisconnect.bind(this)
    );
  }

  start(): void {
    this.ws.connect();
    this.gameLoop();
  }

  private onConnect(): void {
    this.connected = true;
    this.canvas.dataset.connected = "true";
    eventLogger.log("connection", "info", "Connected to server");
  }

  private onDisconnect(): void {
    this.connected = false;
    this.actors = [];
    this.bullets = [];
    this.mySessionId = null;
    this.myEntityId = null;
    this.canvas.dataset.connected = "false";
    this.canvas.dataset.sessionId = "";
    this.canvas.dataset.roomJoined = "false";
    this.canvas.dataset.playerCount = "0";
    eventLogger.log("connection", "warn", "Disconnected from server");
  }

  private onMessage(data: ArrayBuffer): void {
    if (data.byteLength < 2) {
      eventLogger.log(
        "error",
        "error",
        `Message too short: ${data.byteLength} bytes`
      );
      return;
    }

    const bytes = new Uint8Array(data);
    const firstByte = bytes[0];

    // TransportHeaderのMsgType(0-3)か、rawブロードキャスト(PayloadHeader)かを判定
    // MsgType: 0=RoomMessage, 1=Ping, 2=Pong, 3=Assign
    // Broadcast: PayloadHeader[0] = DataType (5=Snapshot)
    if (firstByte <= MSG_TYPE_ASSIGN) {
      const msg = parseTransportHeader(bytes);
      switch (msg.msgType) {
        case MSG_TYPE_ASSIGN:
          this.handleAssign(msg.payload);
          break;
        case MSG_TYPE_PING:
          this.ws.send(encodePong());
          eventLogger.log("control", "debug", "Received PING, sent PONG");
          break;
        case MSG_TYPE_ROOM_MESSAGE:
          this.handleRoomMessage(msg.payload);
          break;
        default:
          eventLogger.log(
            "error",
            "warn",
            `Unknown msgType: ${msg.msgType}`
          );
      }
    } else {
      // Rawブロードキャスト（TransportHeaderなし）= AppPayload直接
      this.handleAppPayload(bytes);
    }
  }

  private handleAssign(payload: Uint8Array): void {
    this.mySessionId = decodeAssign(payload);
    const sid = sessionIdToString(this.mySessionId);
    this.canvas.dataset.sessionId = sid;
    eventLogger.log("control", "info", `Session assigned: ${sid}`, {
      sessionId: sid,
    });

    // 自動的にデフォルトルームにJoin
    const joinMsg = encodeRoomMessage(DEFAULT_ROOM_ID, ROOM_MSG_TYPE_JOIN);
    this.ws.send(joinMsg);
    this.canvas.dataset.roomJoined = "true";
    eventLogger.log("control", "info", "Sent JOIN (default room)");
  }

  private handleRoomMessage(payload: Uint8Array): void {
    // RoomMessage payload from server = broadcast data (AppPayload直接)
    // サーバーのRoom.Broadcastはpubsub→subscribeLoop→writeChで送信
    // これはTransportHeaderなしのrawデータ（codecの出力そのまま）
    // 実際にはサーバーからのブロードキャストはTransportHeaderでラップされていない
    // room.go の Broadcast は直接データをpublishするので、
    // subscribeLoop→writeLoop→connection.Writeで送られる
    // つまりサーバーからのブロードキャストにはTransportHeaderがない

    // ただし実際はこのpayloadはparseTransportHeaderで既にunwrapされたもの
    // RoomMessage payloadの中身を解析
    try {
      // [RoomIDLen(varint)][RoomID][RoomMsgType(varint)][AppPayload]
      const { value: roomIdLen, n: n1 } = readVarint(payload, 0);
      const n2 = n1 + roomIdLen;
      const { value: roomMsgType, n: n3 } = readVarint(payload, n2);

      if (roomMsgType === ROOM_MSG_TYPE_APP_DATA) {
        const appPayload = payload.subarray(n2 + n3);
        this.handleAppPayload(appPayload);
      }
    } catch (e) {
      eventLogger.log("error", "error", "Failed to handle room message", {
        error: String(e),
      });
    }
  }

  private handleAppPayload(appPayload: Uint8Array): void {
    if (appPayload.length < PAYLOAD_HEADER_SIZE) return;

    const dataType = appPayload[0];

    if (dataType === DATA_TYPE_SNAPSHOT) {
      try {
        const state = decodeSnapshot(appPayload);
        this.actors = state.actors;
        this.bullets = state.bullets;
        this.lastBulletUpdate = performance.now();
        this.canvas.dataset.playerCount = String(this.actors.length);

        // 最初のactorを自分のEntityとして推定（サーバーからEntityID通知がないため暫定）
        if (this.myEntityId === null && this.actors.length > 0) {
          this.myEntityId = this.actors[0].entityId;
        }

        eventLogger.logActor(this.actors.length, {
          actors: this.actors.map((a) => ({
            entityId: a.entityId,
            x: a.x.toFixed(1),
            y: a.y.toFixed(1),
          })),
        });
      } catch (e) {
        eventLogger.log("error", "error", "Failed to decode snapshot", {
          error: String(e),
          byteLength: appPayload.length,
        });
      }
    }
  }

  private gameLoop(): void {
    // 入力送信
    if (this.connected && this.mySessionId !== null) {
      const keyMask = this.input.getKeyMask();
      if (keyMask !== 0) {
        const appPayload = encodeInputAppPayload(keyMask);
        const msg = encodeRoomMessage(
          DEFAULT_ROOM_ID,
          ROOM_MSG_TYPE_APP_DATA,
          appPayload
        );
        this.ws.send(msg);
        if (keyMask !== this.prevKeyMask) {
          const desc = describeKeyMask(keyMask);
          eventLogger.log(
            "input",
            "debug",
            `Input: ${desc} (mask=0x${keyMask.toString(16).padStart(2, "0")})`
          );
        }
      }
      this.prevKeyMask = keyMask;
    }

    // 弾丸位置の補間（サーバーtick間を速度で補間）
    const now = performance.now();
    const dtSec = (now - this.lastBulletUpdate) / 1000;
    const interpolatedBullets = this.bullets.map((b) => ({
      ...b,
      x: b.x + b.vx * dtSec * 60,
      y: b.y + b.vy * dtSec * 60,
    }));

    // 描画
    this.renderer.render(this.actors, interpolatedBullets, this.myEntityId);

    requestAnimationFrame(this.gameLoop.bind(this));
  }

  destroy(): void {
    if (this.connected) {
      const leaveMsg = encodeRoomMessage(DEFAULT_ROOM_ID, ROOM_MSG_TYPE_LEAVE);
      this.ws.send(leaveMsg);
    }
    this.ws.disconnect();
    this.input.destroy();
  }
}
