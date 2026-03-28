import { test, expect } from "./fixtures";
import { writeFileSync, mkdirSync } from "fs";
import { join } from "path";

test.describe("ECS Game Screenshot", () => {
  test("should capture game with entities, bullets, and HP bars", async ({
    gamePage,
  }, testInfo) => {
    await gamePage.goto();

    // 接続・セッション割当・Room参加を待機
    await expect(gamePage.canvas).toHaveAttribute("data-connected", "true", {
      timeout: 10000,
    });
    await expect(gamePage.canvas).toHaveAttribute(
      "data-session-id",
      /^[0-9a-f]{32}$/,
      { timeout: 10000 }
    );
    await expect(gamePage.canvas).toHaveAttribute(
      "data-room-joined",
      "true",
      { timeout: 10000 }
    );

    // プレイヤーがスポーンされるまで待機
    await expect(gamePage.canvas).toHaveAttribute(
      "data-player-count",
      /^[1-9]\d*$/,
      { timeout: 10000 }
    );

    // ゲーム状態が安定するまで待機
    await gamePage.canvas.page().waitForTimeout(2000);

    // 緑色のプレイヤーが描画されるまで待機
    await expect
      .poll(() => gamePage.hasGreenPlayer(), { timeout: 15000 })
      .toBe(true);

    // スクリーンショット撮影
    const screenshotsDir = join(__dirname, "..", "screenshots");
    mkdirSync(screenshotsDir, { recursive: true });

    const pageScreenshot = await gamePage.canvas.page().screenshot({
      fullPage: true,
    });
    writeFileSync(join(screenshotsDir, "ecs-game-fullpage.png"), pageScreenshot);

    const canvasScreenshot = await gamePage.canvas.screenshot();
    writeFileSync(join(screenshotsDir, "ecs-game-canvas.png"), canvasScreenshot);
    await testInfo.attach("ecs-game-screenshot", {
      body: canvasScreenshot,
      contentType: "image/png",
    });
  });
});
