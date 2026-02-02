// ゲームループ・状態管理

import type { Actor } from "./protocol";
import {
  DATA_TYPE_ACTOR,
  decodeActorBroadcast,
  encodeInputMessage,
  getDataType,
} from "./protocol";
import { WebSocketClient } from "./websocket";
import { InputManager } from "./input";
import { Renderer } from "./renderer";

const SERVER_URL = "ws://localhost:9090/ws";

export class Game {
  private ws: WebSocketClient;
  private input: InputManager;
  private renderer: Renderer;

  private actors: Actor[] = [];
  private mySessionId: bigint | null = null;
  private seq: number = 0;
  private connected: boolean = false;

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
    console.log("Connected to server");
  }

  private onDisconnect(): void {
    this.connected = false;
    this.actors = [];
    this.mySessionId = null;
    console.log("Disconnected from server");
  }

  private onMessage(data: ArrayBuffer): void {
    const dataType = getDataType(data);

    if (dataType === DATA_TYPE_ACTOR) {
      this.actors = decodeActorBroadcast(data);

      // 最初のブロードキャストで自分のSessionIDを特定
      // (最初に追加されたアクターが自分である可能性が高い)
      if (this.mySessionId === null && this.actors.length > 0) {
        this.mySessionId = this.actors[0].sessionId;
        console.log("My SessionID:", this.mySessionId);
      }
    }
  }

  private gameLoop(): void {
    // 入力送信
    if (this.connected) {
      const keyMask = this.input.getKeyMask();
      if (keyMask !== 0) {
        const sessionId = this.mySessionId !== null ? Number(this.mySessionId) : 0;
        const msg = encodeInputMessage(sessionId, this.seq++, keyMask);
        this.ws.send(msg);
      }
    }

    // 描画
    this.renderer.render(this.actors, this.mySessionId);

    requestAnimationFrame(this.gameLoop.bind(this));
  }

  destroy(): void {
    this.ws.disconnect();
    this.input.destroy();
  }
}
