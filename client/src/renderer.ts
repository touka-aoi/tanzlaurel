// Canvas 描画

import type { Actor } from "./protocol";
import { sessionIdEquals, isAlive, isBot } from "./protocol";

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
    this.ctx.fillStyle = "#FAFAFA";
    this.ctx.fillRect(0, 0, CANVAS_WIDTH, CANVAS_HEIGHT);
  }

  drawMap(): void {
    // マップ境界 (100x100 ワールドユニット)
    this.ctx.strokeStyle = "#D0D0D0";
    this.ctx.lineWidth = 2;
    this.ctx.strokeRect(0, 0, 100 * SCALE, 100 * SCALE);

    // グリッド線 (10ユニット間隔)
    this.ctx.strokeStyle = "#E8E8E8";
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

  drawActors(actors: Actor[], mySessionId: Uint8Array | null): void {
    for (const actor of actors) {
      const isMe = sessionIdEquals(mySessionId, actor.sessionId);
      this.drawActor(actor, isMe);
    }
  }

  private drawActor(actor: Actor, isMe: boolean): void {
    const screenX = actor.x * SCALE;
    const screenY = actor.y * SCALE;
    const alive = isAlive(actor.state);
    const bot = isBot(actor.state);

    // 死亡時は半透明
    const prevAlpha = this.ctx.globalAlpha;
    if (!alive) {
      this.ctx.globalAlpha = 0.3;
    }

    // 色: 自分=緑, Bot=赤, 他人=青
    let fillColor: string;
    let strokeColor: string;
    if (isMe) {
      fillColor = "#4ade80";
      strokeColor = "#16a34a";
    } else if (bot) {
      fillColor = "#f87171";
      strokeColor = "#dc2626";
    } else {
      fillColor = "#60a5fa";
      strokeColor = "#2563eb";
    }

    // アクター円
    this.ctx.beginPath();
    this.ctx.arc(screenX, screenY, ACTOR_RADIUS, 0, Math.PI * 2);
    this.ctx.fillStyle = fillColor;
    this.ctx.fill();
    this.ctx.strokeStyle = strokeColor;
    this.ctx.lineWidth = 2;
    this.ctx.stroke();

    // HPバー
    if (alive) {
      const barWidth = ACTOR_RADIUS * 2.5;
      const barHeight = 3;
      const barX = screenX - barWidth / 2;
      const barY = screenY - ACTOR_RADIUS - 8;
      const hpRatio = actor.hp / 100;

      // 背景(灰)
      this.ctx.fillStyle = "#374151";
      this.ctx.fillRect(barX, barY, barWidth, barHeight);

      // HP(緑→赤)
      const hpColor = hpRatio > 0.5 ? "#4ade80" : hpRatio > 0.25 ? "#facc15" : "#f87171";
      this.ctx.fillStyle = hpColor;
      this.ctx.fillRect(barX, barY, barWidth * hpRatio, barHeight);
    }

    this.ctx.globalAlpha = prevAlpha;
  }

  render(actors: Actor[], mySessionId: Uint8Array | null): void {
    this.clear();
    this.drawMap();
    this.drawActors(actors, mySessionId);
  }
}