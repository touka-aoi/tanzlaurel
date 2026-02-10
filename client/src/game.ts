// ゲームループ・状態管理

import type { Actor } from "./protocol";
import {
  CONTROL_SUBTYPE_ASSIGN,
  CONTROL_SUBTYPE_LEAVE,
  CONTROL_SUBTYPE_PING,
  CONTROL_SUBTYPE_PONG,
  DATA_TYPE_ACTOR,
  DATA_TYPE_CONTROL,
  HEADER_SIZE,
  PAYLOAD_HEADER_SIZE,
  decodeActorBroadcast,
  decodeAssignMessage,
  describeKeyMask,
  encodeControlMessage,
  encodeInputMessage,
  encodeJoinMessage,
  getControlSubType,
  getDataType,
  sessionIdToString,
} from "./protocol";
import { WebSocketClient } from "./websocket";
import { InputManager } from "./input";
import { Renderer } from "./renderer";
import { eventLogger } from "./event-logger";

const SERVER_URL = "ws://localhost:9090/ws";

export class Game {
  private ws: WebSocketClient;
  private input: InputManager;
  private renderer: Renderer;

  private actors: Actor[] = [];
  private mySessionId: Uint8Array | null = null;
  private seq: number = 0;
  private connected: boolean = false;
  private prevKeyMask: number = -1;

  constructor(canvas: HTMLCanvasElement) {
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
    eventLogger.log("connection", "info", "Connected to server");
  }

  private onDisconnect(): void {
    this.connected = false;
    this.actors = [];
    this.mySessionId = null;
    eventLogger.log("connection", "warn", "Disconnected from server");
  }

  private onMessage(data: ArrayBuffer): void {
    if (data.byteLength < HEADER_SIZE + PAYLOAD_HEADER_SIZE) {
      eventLogger.log("error", "error", `Message too short: ${data.byteLength} bytes`);
      return;
    }

    const dataType = getDataType(data);

    if (dataType === DATA_TYPE_CONTROL) {
      const subType = getControlSubType(data);
      if (subType === CONTROL_SUBTYPE_ASSIGN) {
        this.mySessionId = decodeAssignMessage(data);
        const sid = sessionIdToString(this.mySessionId);
        eventLogger.log("control", "info", `Session assigned: ${sid}`, { sessionId: sid });

        const joinMsg = encodeJoinMessage(this.mySessionId, this.seq++, null);
        this.ws.send(joinMsg);
        eventLogger.log("control", "info", "Sent JOIN (auto-assign room)");
      } else if (subType === CONTROL_SUBTYPE_PING && this.mySessionId !== null) {
        const pongMsg = encodeControlMessage(this.mySessionId, this.seq++, CONTROL_SUBTYPE_PONG);
        this.ws.send(pongMsg);
        eventLogger.log("control", "debug", "Received PING, sent PONG");
      }
    } else if (dataType === DATA_TYPE_ACTOR) {
      try {
        this.actors = decodeActorBroadcast(data);
        eventLogger.logActor(this.actors.length, {
          actors: this.actors.map((a) => ({
            sessionId: sessionIdToString(a.sessionId),
            x: a.x.toFixed(1),
            y: a.y.toFixed(1),
          })),
        });
      } catch (e) {
        eventLogger.log("error", "error", "Failed to decode actor broadcast", {
          error: String(e),
          byteLength: data.byteLength,
        });
      }
    } else {
      eventLogger.log("error", "warn", `Unknown dataType: ${dataType}`, {
        dataType,
        byteLength: data.byteLength,
      });
    }
  }

  private gameLoop(): void {
    // 入力送信
    if (this.connected && this.mySessionId !== null) {
      const keyMask = this.input.getKeyMask();
      if (keyMask !== 0) {
        const msg = encodeInputMessage(this.mySessionId, this.seq++, keyMask);
        this.ws.send(msg);
        if (keyMask !== this.prevKeyMask) {
          const desc = describeKeyMask(keyMask);
          eventLogger.log("input", "debug", `Input: ${desc} (mask=0x${keyMask.toString(16).padStart(2, "0")})`);
        }
      }
      this.prevKeyMask = keyMask;
    }

    // 描画
    this.renderer.render(this.actors, this.mySessionId);

    requestAnimationFrame(this.gameLoop.bind(this));
  }

  destroy(): void {
    // Control/Leave を送信（ベストエフォート）
    if (this.connected && this.mySessionId !== null) {
      const leaveMsg = encodeControlMessage(this.mySessionId, this.seq++, CONTROL_SUBTYPE_LEAVE);
      this.ws.send(leaveMsg);
    }
    this.ws.disconnect();
    this.input.destroy();
  }
}