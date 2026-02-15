import { test as base } from "@playwright/test";
import { GamePage } from "../pages/game.page";

export const test = base.extend<{ gamePage: GamePage }>({
  gamePage: async ({ page }, use) => {
    const gamePage = new GamePage(page);
    await use(gamePage);
  },
});

export { expect } from "@playwright/test";
