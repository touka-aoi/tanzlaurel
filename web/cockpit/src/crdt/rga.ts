export interface NodeID {
  siteId: string;
  timestamp: number;
}

export const OpType = {
  Insert: 1,
  Delete: 2,
} as const;

export type OpTypeValue = (typeof OpType)[keyof typeof OpType];

export interface Operation {
  requestId: string;
  opType: OpTypeValue;
  nodeId: NodeID;
  after: NodeID | null;
  value: string;
  authenticated?: boolean;
}

interface Node {
  id: NodeID;
  after: NodeID | null;
  value: string;
  deleted: boolean;
  authenticated: boolean;
}

function nodeIdEqual(a: NodeID, b: NodeID): boolean {
  return a.siteId === b.siteId && a.timestamp === b.timestamp;
}

function sameAfter(a: NodeID | null, b: NodeID | null): boolean {
  if (a === null && b === null) return true;
  if (a === null || b === null) return false;
  return nodeIdEqual(a, b);
}

// 並行挿入の優先順位: Timestamp大→左、同Timestampなら siteId辞書順小→左
function nodeIdPriority(a: NodeID, b: NodeID): boolean {
  if (a.timestamp !== b.timestamp) return a.timestamp > b.timestamp;
  return a.siteId < b.siteId;
}

let requestIdCounter = 0;
function genRequestId(): string {
  return `req-${++requestIdCounter}-${Date.now()}`;
}

export class RGA {
  private siteId: string;
  private counter: number = 0;
  private nodes: Node[] = [];
  private index: Map<string, number> = new Map();
  private seen: Set<string> = new Set();
  private pending: Operation[] = [];

  constructor(siteId: string) {
    this.siteId = siteId;
  }

  private nodeKey(id: NodeID): string {
    return `${id.siteId}:${id.timestamp}`;
  }

  private tick(): NodeID {
    this.counter++;
    return { siteId: this.siteId, timestamp: this.counter };
  }

  private updateClock(ts: number): void {
    if (ts > this.counter) this.counter = ts;
  }

  apply(op: Operation): boolean {
    if (this.seen.has(op.requestId)) return false;
    this.seen.add(op.requestId);
    this.updateClock(op.nodeId.timestamp);

    if (op.opType === OpType.Insert) {
      if (op.after !== null && !this.index.has(this.nodeKey(op.after))) {
        this.pending.push(op);
        return true;
      }
      this.applyInsert(op);
      this.flushPending();
    } else if (op.opType === OpType.Delete) {
      if (!this.index.has(this.nodeKey(op.nodeId))) {
        this.pending.push(op);
        return true;
      }
      this.applyDelete(op);
    }
    return true;
  }

  private applyInsert(op: Operation): void {
    const n: Node = {
      id: op.nodeId,
      after: op.after,
      value: op.value,
      deleted: false,
      authenticated: op.authenticated ?? true,
    };

    let insertIdx = 0;
    if (op.after !== null) {
      const afterIdx = this.index.get(this.nodeKey(op.after));
      if (afterIdx !== undefined) insertIdx = afterIdx + 1;
    }

    while (insertIdx < this.nodes.length) {
      const existing = this.nodes[insertIdx];
      if (!sameAfter(existing.after, op.after)) break;
      if (nodeIdPriority(op.nodeId, existing.id)) break;
      insertIdx = this.skipSubtree(insertIdx);
    }

    this.nodes.splice(insertIdx, 0, n);
    this.rebuildIndex(insertIdx);
  }

  private skipSubtree(idx: number): number {
    const parentId = this.nodes[idx].id;
    idx++;
    while (idx < this.nodes.length) {
      if (!this.isDescendantOf(idx, parentId)) break;
      idx++;
    }
    return idx;
  }

  private isDescendantOf(idx: number, parentId: NodeID): boolean {
    const n = this.nodes[idx];
    if (n.after === null) return false;
    if (nodeIdEqual(n.after, parentId)) return true;
    const ancestorIdx = this.index.get(this.nodeKey(n.after));
    if (ancestorIdx === undefined) return false;
    return this.isDescendantOf(ancestorIdx, parentId);
  }

  private applyDelete(op: Operation): void {
    const idx = this.index.get(this.nodeKey(op.nodeId));
    if (idx === undefined) return;
    this.nodes[idx].deleted = true;
  }

  private flushPending(): void {
    let applied = true;
    while (applied) {
      applied = false;
      const remaining: Operation[] = [];
      for (const op of this.pending) {
        if (op.opType === OpType.Insert) {
          if (op.after !== null && !this.index.has(this.nodeKey(op.after))) {
            remaining.push(op);
            continue;
          }
          this.applyInsert(op);
          applied = true;
        } else if (op.opType === OpType.Delete) {
          if (!this.index.has(this.nodeKey(op.nodeId))) {
            remaining.push(op);
            continue;
          }
          this.applyDelete(op);
          applied = true;
        }
      }
      this.pending = remaining;
    }
  }

  private rebuildIndex(from: number): void {
    for (let i = from; i < this.nodes.length; i++) {
      this.index.set(this.nodeKey(this.nodes[i].id), i);
    }
  }

  nodeAt(pos: number): NodeID | null {
    if (pos < 0) return null;
    let i = 0;
    for (const n of this.nodes) {
      if (n.deleted) continue;
      if (i === pos) return n.id;
      i++;
    }
    return null;
  }

  visibleNodes(): NodeID[] {
    return this.nodes.filter((n) => !n.deleted).map((n) => n.id);
  }

  isNodeAuthenticated(nodeId: NodeID): boolean {
    const idx = this.index.get(this.nodeKey(nodeId));
    if (idx === undefined) return true;
    return this.nodes[idx].authenticated;
  }

  text(): string {
    return this.nodes
      .filter((n) => !n.deleted)
      .map((n) => n.value)
      .join("");
  }

  insert(after: NodeID | null, value: string): Operation {
    const op: Operation = {
      requestId: genRequestId(),
      opType: OpType.Insert,
      nodeId: this.tick(),
      after,
      value,
    };
    this.apply(op);
    return op;
  }

  delete(nodeId: NodeID): Operation {
    const op: Operation = {
      requestId: genRequestId(),
      opType: OpType.Delete,
      nodeId,
      after: null,
      value: "",
    };
    this.apply(op);
    return op;
  }
}
