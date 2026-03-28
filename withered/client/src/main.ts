import "./style.css";
import { Game } from "./game";
import { LogPanel } from "./log-panel";

const canvas = document.getElementById("game") as HTMLCanvasElement;
if (!canvas) {
  throw new Error("Canvas element not found");
}

const logContainer = document.getElementById("event-log") as HTMLElement;
const logPanel = new LogPanel(logContainer);
logPanel.init();

const game = new Game(canvas);
game.start();