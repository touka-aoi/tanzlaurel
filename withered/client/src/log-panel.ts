// イベントログパネル描画

import { eventLogger, type LogEntry, type LogEventType } from "./event-logger";

export class LogPanel {
  private container: HTMLElement;
  private entriesEl: HTMLElement;
  private autoScroll = true;
  private autoScrollBtn: HTMLElement;
  private pendingEntries: LogEntry[] = [];
  private rafScheduled = false;
  private filters: Map<LogEventType, boolean>;

  constructor(container: HTMLElement) {
    this.container = container;
    this.entriesEl = container.querySelector("#log-entries") as HTMLElement;
    this.autoScrollBtn = container.querySelector("#log-autoscroll") as HTMLElement;
    this.filters = new Map<LogEventType, boolean>([
      ["connection", true],
      ["control", true],
      ["input", true],
      ["actor", false],
      ["error", true],
    ]);
  }

  init(): void {
    // フィルターチップ
    const chips = this.container.querySelectorAll<HTMLElement>(".log-chip");
    for (const chip of chips) {
      const type = chip.dataset.type as LogEventType;
      chip.addEventListener("click", () => this.onFilterClick(type, chip));
    }

    // Actorはデフォルト非表示なのでCSSクラスを設定
    this.entriesEl.classList.add("hide-actor");

    // Clear ボタン
    const clearBtn = this.container.querySelector("#log-clear") as HTMLElement;
    clearBtn.addEventListener("click", () => {
      eventLogger.clear();
      this.entriesEl.innerHTML = "";
    });

    // Auto-scroll ボタン
    this.autoScrollBtn.classList.add("active");
    this.autoScrollBtn.addEventListener("click", () => {
      this.autoScroll = !this.autoScroll;
      this.autoScrollBtn.classList.toggle("active", this.autoScroll);
      if (this.autoScroll) {
        this.scrollToBottom();
      }
    });

    // スクロール検知: ユーザーが上にスクロールしたらauto-scroll OFF、底に戻したらON
    this.entriesEl.addEventListener("scroll", () => {
      const el = this.entriesEl;
      const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 4;
      if (atBottom && !this.autoScroll) {
        this.autoScroll = true;
        this.autoScrollBtn.classList.add("active");
      } else if (!atBottom && this.autoScroll) {
        this.autoScroll = false;
        this.autoScrollBtn.classList.remove("active");
      }
    });

    // 既存エントリ描画
    for (const entry of eventLogger.getEntries()) {
      this.pendingEntries.push(entry);
    }
    this.scheduleRender();

    // 新規エントリ購読
    eventLogger.subscribe((entry) => {
      this.pendingEntries.push(entry);
      this.scheduleRender();
    });
  }

  private scheduleRender(): void {
    if (this.rafScheduled) return;
    this.rafScheduled = true;
    requestAnimationFrame(() => this.flushRender());
  }

  private flushRender(): void {
    this.rafScheduled = false;
    if (this.pendingEntries.length === 0) return;

    const fragment = document.createDocumentFragment();
    for (const entry of this.pendingEntries) {
      fragment.appendChild(this.createRowElement(entry));
    }
    this.pendingEntries = [];

    this.entriesEl.appendChild(fragment);

    // リングバッファと同期: DOM上の行数が多すぎたら古い行を削除
    while (this.entriesEl.childElementCount > 1000) {
      this.entriesEl.removeChild(this.entriesEl.firstElementChild!);
    }

    if (this.autoScroll) {
      this.scrollToBottom();
    }
  }

  private createRowElement(entry: LogEntry): HTMLElement {
    const row = document.createElement("div");
    row.className = "log-row";
    row.dataset.type = entry.type;

    // Compact行
    const compact = document.createElement("div");
    compact.className = "log-row-compact";

    // タイムスタンプ
    const time = document.createElement("span");
    time.className = "log-time";
    time.textContent = this.formatTime(entry.timestamp);

    // Severity ドット
    const severity = document.createElement("span");
    severity.className = `log-severity ${entry.severity}`;

    // タイプバッジ
    const badge = document.createElement("span");
    badge.className = `log-badge ${entry.type}`;
    badge.textContent = entry.type;

    // サマリー
    const summary = document.createElement("span");
    summary.className = "log-summary";
    summary.textContent = entry.summary;

    // 展開アロー（詳細がある場合のみ）
    const arrow = document.createElement("span");
    arrow.className = "log-arrow";
    arrow.textContent = entry.detail ? "\u203a" : "";

    compact.append(arrow, time, severity, badge, summary);
    row.appendChild(compact);

    // 詳細セクション
    if (entry.detail) {
      const detail = document.createElement("div");
      detail.className = "log-detail";

      for (const [key, value] of Object.entries(entry.detail)) {
        const line = document.createElement("div");
        line.className = "log-detail-line";

        const keyEl = document.createElement("span");
        keyEl.className = "log-detail-key";
        keyEl.textContent = key + ":";

        const valEl = document.createElement("span");
        valEl.className = "log-detail-value";
        valEl.textContent = typeof value === "string" ? value : JSON.stringify(value);

        line.append(keyEl, valEl);
        detail.appendChild(line);
      }

      row.appendChild(detail);

      compact.addEventListener("click", () => {
        row.classList.toggle("expanded");
      });
    }

    return row;
  }

  private onFilterClick(type: LogEventType, chip: HTMLElement): void {
    const isActive = this.filters.get(type)!;
    this.filters.set(type, !isActive);
    chip.classList.toggle("active", !isActive);
    this.entriesEl.classList.toggle(`hide-${type}`, isActive);
  }

  private scrollToBottom(): void {
    this.entriesEl.scrollTop = this.entriesEl.scrollHeight;
  }

  private formatTime(date: Date): string {
    const h = String(date.getHours()).padStart(2, "0");
    const m = String(date.getMinutes()).padStart(2, "0");
    const s = String(date.getSeconds()).padStart(2, "0");
    const ms = String(date.getMilliseconds()).padStart(3, "0");
    return `${h}:${m}:${s}.${ms}`;
  }
}
