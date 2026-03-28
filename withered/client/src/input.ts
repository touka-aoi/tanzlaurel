// キー入力管理

import { KEY_W, KEY_A, KEY_S, KEY_D } from "./protocol";

export class InputManager {
  private keys: Set<string> = new Set();

  constructor() {
    window.addEventListener("keydown", this.onKeyDown.bind(this));
    window.addEventListener("keyup", this.onKeyUp.bind(this));
  }

  private onKeyDown(event: KeyboardEvent): void {
    const key = event.key.toLowerCase();
    if (["w", "a", "s", "d"].includes(key)) {
      this.keys.add(key);
    }
  }

  private onKeyUp(event: KeyboardEvent): void {
    const key = event.key.toLowerCase();
    this.keys.delete(key);
  }

  getKeyMask(): number {
    let mask = 0;
    if (this.keys.has("w")) mask |= KEY_W;
    if (this.keys.has("a")) mask |= KEY_A;
    if (this.keys.has("s")) mask |= KEY_S;
    if (this.keys.has("d")) mask |= KEY_D;
    return mask;
  }

  hasInput(): boolean {
    return this.keys.size > 0;
  }

  destroy(): void {
    window.removeEventListener("keydown", this.onKeyDown.bind(this));
    window.removeEventListener("keyup", this.onKeyUp.bind(this));
  }
}
