// WebSocket 接続管理

import { eventLogger } from "./event-logger";

export type MessageHandler = (data: ArrayBuffer) => void;
export type ConnectionHandler = () => void;

export class WebSocketClient {
  private ws: WebSocket | null = null;
  private url: string;
  private onMessage: MessageHandler;
  private onConnect: ConnectionHandler;
  private onDisconnect: ConnectionHandler;

  constructor(
    url: string,
    onMessage: MessageHandler,
    onConnect: ConnectionHandler,
    onDisconnect: ConnectionHandler
  ) {
    this.url = url;
    this.onMessage = onMessage;
    this.onConnect = onConnect;
    this.onDisconnect = onDisconnect;
  }

  connect(): void {
    this.ws = new WebSocket(this.url);
    this.ws.binaryType = "arraybuffer";

    this.ws.onopen = () => {
      this.onConnect();
    };

    this.ws.onmessage = (event) => {
      if (event.data instanceof ArrayBuffer) {
        this.onMessage(event.data);
      }
    };

    this.ws.onclose = () => {
      this.onDisconnect();
    };

    this.ws.onerror = () => {
      eventLogger.log("error", "error", "WebSocket error");
    };
  }

  send(data: ArrayBuffer): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(data);
    }
  }

  disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }
}