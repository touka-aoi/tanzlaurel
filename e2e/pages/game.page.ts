import type { Locator, Page } from "@playwright/test";

export class GamePage {
  readonly canvas: Locator;

  constructor(private readonly page: Page) {
    this.canvas = page.locator("canvas#game");
  }

  async goto(): Promise<void> {
    await this.page.goto("/");
    await this.canvas.waitFor({ state: "visible" });
  }

  async hasGreenPlayer(): Promise<boolean> {
    return this.page.evaluate(() => {
      const canvas = document.querySelector(
        "canvas#game"
      ) as HTMLCanvasElement | null;
      if (!canvas) return false;

      const ctx = canvas.getContext("2d");
      if (!ctx) return false;

      const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
      const data = imageData.data;

      // 緑色 (#4ade80) のRGB値: R=74, G=222, B=128
      for (let i = 0; i < data.length; i += 4) {
        const r = data[i];
        const g = data[i + 1];
        const b = data[i + 2];

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
  }
}
