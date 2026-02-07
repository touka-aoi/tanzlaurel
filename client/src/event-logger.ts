// イベントログ管理

export type LogEventType = "connection" | "control" | "input" | "actor" | "error";
export type LogSeverity = "info" | "warn" | "error" | "debug";

export interface LogEntry {
  id: number;
  timestamp: Date;
  type: LogEventType;
  severity: LogSeverity;
  summary: string;
  detail?: Record<string, unknown>;
}

export type LogListener = (entry: LogEntry) => void;

const MAX_ENTRIES = 1000;
const ACTOR_THROTTLE_MS = 1000;

export class EventLogger {
  private entries: LogEntry[] = [];
  private nextId = 1;
  private listeners: LogListener[] = [];

  // Actor スロットル用
  private pendingActorData: Record<string, unknown> | null = null;
  private actorTimer: ReturnType<typeof setTimeout> | null = null;

  log(type: LogEventType, severity: LogSeverity, summary: string, detail?: Record<string, unknown>): void {
    const entry: LogEntry = {
      id: this.nextId++,
      timestamp: new Date(),
      type,
      severity,
      summary,
      detail,
    };

    this.entries.push(entry);
    if (this.entries.length > MAX_ENTRIES) {
      this.entries.shift();
    }

    for (const listener of this.listeners) {
      listener(entry);
    }
  }

  logActor(actorCount: number, detail: Record<string, unknown>): void {
    this.pendingActorData = { actorCount, ...detail };

    if (this.actorTimer !== null) return;

    this.actorTimer = setTimeout(() => {
      this.actorTimer = null;
      if (this.pendingActorData) {
        const data = this.pendingActorData;
        this.pendingActorData = null;
        this.log("actor", "debug", `Actor update: ${data.actorCount} actor(s)`, data);
      }
    }, ACTOR_THROTTLE_MS);
  }

  subscribe(listener: LogListener): () => void {
    this.listeners.push(listener);
    return () => {
      this.listeners = this.listeners.filter((l) => l !== listener);
    };
  }

  getEntries(): readonly LogEntry[] {
    return this.entries;
  }

  clear(): void {
    this.entries = [];
  }
}

export const eventLogger = new EventLogger();
