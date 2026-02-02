// Canvas 描画

import type { Actor } from "./protocol";

const CANVAS_WIDTH = 800;
const CANVAS_HEIGHT = 600;
const SCALE = 8; // 1ワールドユニット = 8px
const ACTOR_RADIUS = 8;

export class Renderer {
  private canvas: HTMLCanvasElement;
  private ctx: CanvasRenderingContext2D;

  constructor(canvas: HTMLCanvasElement) {
    this.canvas = canvas;
    this.canvas.width = CANVAS_WIDTH;
    this.canvas.height = CANVAS_HEIGHT;

    const ctx = canvas.getContext("2d");
    if (!ctx) {
      throw new Error("Failed to get 2d context");
    }
    this.ctx = ctx;
  }

  clear(): void {
    this.ctx.fillStyle = "#1a1a1a";
    this.ctx.fillRect(0, 0, CANVAS_WIDTH, CANVAS_HEIGHT);
  }

  drawMap(): void {
    // マップ境界 (100x100 ワールドユニット)
    this.ctx.strokeStyle = "#444";
    this.ctx.lineWidth = 2;
    this.ctx.strokeRect(0, 0, 100 * SCALE, 100 * SCALE);

    // グリッド線 (10ユニット間隔)
    this.ctx.strokeStyle = "#333";
    this.ctx.lineWidth = 1;
    for (let i = 10; i < 100; i += 10) {
      // 縦線
      this.ctx.beginPath();
      this.ctx.moveTo(i * SCALE, 0);
      this.ctx.lineTo(i * SCALE, 100 * SCALE);
      this.ctx.stroke();
      // 横線
      this.ctx.beginPath();
      this.ctx.moveTo(0, i * SCALE);
      this.ctx.lineTo(100 * SCALE, i * SCALE);
      this.ctx.stroke();
    }
  }

  drawActors(actors: Actor[], mySessionId: bigint | null): void {
    for (const actor of actors) {
      const isMe = mySessionId !== null && actor.sessionId === mySessionId;
      this.drawActor(actor.x, actor.y, isMe);
    }
  }

  private drawActor(x: number, y: number, isMe: boolean): void {
    const screenX = x * SCALE;
    const screenY = y * SCALE;

    this.ctx.beginPath();
    this.ctx.arc(screenX, screenY, ACTOR_RADIUS, 0, Math.PI * 2);
    this.ctx.fillStyle = isMe ? "#4ade80" : "#60a5fa"; // 緑 vs 青
    this.ctx.fill();

    // 枠線
    this.ctx.strokeStyle = isMe ? "#22c55e" : "#3b82f6";
    this.ctx.lineWidth = 2;
    this.ctx.stroke();
  }

  render(actors: Actor[], mySessionId: bigint | null): void {
    this.clear();
    this.drawMap();
    this.drawActors(actors, mySessionId);
  }
}
