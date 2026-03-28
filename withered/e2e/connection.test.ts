import { test, expect } from "./fixtures";

test.describe("WebSocket Connection", () => {
  test("should connect, receive session ID, join room, and display player", async ({
    gamePage,
  }) => {
    await gamePage.goto();

    // 接続確認
    await expect(gamePage.canvas).toHaveAttribute("data-connected", "true", {
      timeout: 10000,
    });

    // セッションID受信確認（32文字の16進数文字列）
    await expect(gamePage.canvas).toHaveAttribute(
      "data-session-id",
      /^[0-9a-f]{32}$/,
      { timeout: 10000 }
    );

    // Room参加確認
    await expect(gamePage.canvas).toHaveAttribute("data-room-joined", "true", {
      timeout: 10000,
    });

    // プレイヤー数が1以上であることを確認
    await expect(gamePage.canvas).toHaveAttribute(
      "data-player-count",
      /^[1-9]\d*$/,
      { timeout: 10000 }
    );

    // Canvasに緑色のプレイヤーが描画されていることを確認
    await expect
      .poll(() => gamePage.hasGreenPlayer(), { timeout: 10000 })
      .toBe(true);
  });
});
