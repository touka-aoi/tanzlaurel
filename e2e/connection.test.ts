import { test, expect } from "@playwright/test";

test.describe("WebSocket Connection", () => {
  test("should connect, receive session ID, join room, and display player", async ({
    page,
  }) => {
    const logs: string[] = [];

    // コンソールログを監視
    page.on("console", (msg) => {
      if (msg.type() === "log") {
        logs.push(msg.text());
      }
    });

    // クライアントにアクセス
    await page.goto("/");

    // Canvasが表示されることを確認
    const canvas = page.locator("canvas#game");
    await expect(canvas).toBeVisible();

    // 接続ログを待機
    await expect
      .poll(() => logs.some((log) => log.includes("Connected to server")), {
        timeout: 10000,
        message: "Should connect to server",
      })
      .toBe(true);

    // セッションID受信を待機
    await expect
      .poll(() => logs.some((log) => log.includes("Received session ID:")), {
        timeout: 10000,
        message: "Should receive session ID",
      })
      .toBe(true);

    // Join送信を待機
    await expect
      .poll(() => logs.some((log) => log.includes("Sent Join message")), {
        timeout: 10000,
        message: "Should send Join message",
      })
      .toBe(true);

    // プレイヤー表示を待機（Actorブロードキャスト受信後）
    // 少し待ってから描画を確認
    await page.waitForTimeout(500);

    // Canvasから緑色のプレイヤーが描画されているか確認
    const hasGreenPlayer = await page.evaluate(() => {
      const canvas = document.querySelector(
        "canvas#game"
      ) as HTMLCanvasElement | null;
      if (!canvas) return false;

      const ctx = canvas.getContext("2d");
      if (!ctx) return false;

      const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
      const data = imageData.data;

      // 緑色 (#4ade80) のRGB値: R=74, G=222, B=128
      // 許容誤差を持たせて検索
      for (let i = 0; i < data.length; i += 4) {
        const r = data[i];
        const g = data[i + 1];
        const b = data[i + 2];

        // 緑色のプレイヤーを検出 (誤差 ±10)
        if (
          Math.abs(r - 74) < 10 &&
          Math.abs(g - 222) < 10 &&
          Math.abs(b - 128) < 10
        ) {
          return true;
        }
      }
      return false;
    });

    expect(hasGreenPlayer).toBe(true);
  });
});
