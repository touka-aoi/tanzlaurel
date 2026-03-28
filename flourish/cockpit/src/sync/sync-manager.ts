import { RGA, type Operation, type NodeID, OpType } from "../crdt/rga";
import { WSClient } from "./ws-client";

function genReqId(): string {
  return crypto.randomUUID();
}

export interface SyncState {
  text: string;
  connected: boolean;
  lastServerSeq: number;
  authenticated: boolean;
}

type SyncListener = (state: SyncState) => void;

export class SyncManager {
  private ws: WSClient;
  private rga: RGA;
  private entryId: string;
  private lastServerSeq = 0;
  private pendingAcks = new Map<string, Operation>();
  private confirmedReqIds = new Set<string>();
  private listeners: SyncListener[] = [];
  private removeWsHandler: (() => void) | null = null;
  private _authenticated = false;

  constructor(
    wsUrl: string,
    entryId: string,
    siteId: string,
  ) {
    this.ws = new WSClient(wsUrl);
    this.rga = new RGA(siteId);
    this.entryId = entryId;

  }

  connect(): void {
    this.removeWsHandler = this.ws.onMessage((data: any) => {
      this.handleMessage(data);
    });
    this.ws.connect();
  }

  disconnect(): void {
    this.removeWsHandler?.();
    this.ws.disconnect();
  }

  insert(after: NodeID | null, value: string): void {
    const op = this.rga.insert(after, value, this._authenticated);
    const reqId = genReqId();
    this.pendingAcks.set(reqId, op);

    this.ws.send({
      type: "op",
      request_id: reqId,
      entry_id: this.entryId,
      op_type: OpType.Insert,
      node_id: { site_id: op.nodeId.siteId, timestamp: op.nodeId.timestamp },
      after: op.after
        ? { site_id: op.after.siteId, timestamp: op.after.timestamp }
        : null,
      value: op.value,
    });

    this.notify();
  }

  delete(nodeId: NodeID): void {
    const op = this.rga.delete(nodeId);
    const reqId = genReqId();
    this.pendingAcks.set(reqId, op);

    this.ws.send({
      type: "op",
      request_id: reqId,
      entry_id: this.entryId,
      op_type: OpType.Delete,
      node_id: { site_id: nodeId.siteId, timestamp: nodeId.timestamp },
    });

    this.notify();
  }

  applyTextChange(newText: string): void {
    const oldText = this.rga.text();
    if (oldText === newText) return;

    // 共通接頭辞
    let prefixLen = 0;
    while (
      prefixLen < oldText.length &&
      prefixLen < newText.length &&
      oldText[prefixLen] === newText[prefixLen]
    ) {
      prefixLen++;
    }

    // 共通接尾辞（接頭辞と重ならないように）
    let suffixLen = 0;
    while (
      suffixLen < oldText.length - prefixLen &&
      suffixLen < newText.length - prefixLen &&
      oldText[oldText.length - 1 - suffixLen] ===
        newText[newText.length - 1 - suffixLen]
    ) {
      suffixLen++;
    }

    const deleteCount = oldText.length - prefixLen - suffixLen;
    const insertChars = newText.slice(prefixLen, newText.length - suffixLen);

    const visibleNodes = this.rga.visibleNodes();

    // 後ろから削除（インデックスがずれないように）
    for (let i = deleteCount - 1; i >= 0; i--) {
      const nodeId = visibleNodes[prefixLen + i];
      if (nodeId) {
        // 非認証時は認証ノードの削除をスキップ
        if (!this._authenticated && this.rga.isNodeAuthenticated(nodeId)) {
          continue;
        }
        this.delete(nodeId);
      }
    }

    // 挿入
    let after =
      prefixLen > 0 ? visibleNodes[prefixLen - 1] ?? null : null;
    for (const ch of insertChars) {
      const afterCopy = after;
      const op = this.rga.insert(afterCopy, ch, this._authenticated);
      const reqId = genReqId();
      this.pendingAcks.set(reqId, op);

      this.ws.send({
        type: "op",
        request_id: reqId,
        entry_id: this.entryId,
        op_type: OpType.Insert,
        node_id: {
          site_id: op.nodeId.siteId,
          timestamp: op.nodeId.timestamp,
        },
        after: op.after
          ? { site_id: op.after.siteId, timestamp: op.after.timestamp }
          : null,
        value: op.value,
      });

      after = op.nodeId;
    }

    this.notify();
  }

  getText(): string {
    return this.rga.text();
  }

  get authenticated(): boolean {
    return this._authenticated;
  }

  getState(): SyncState {
    return {
      text: this.rga.text(),
      connected: this.ws.connected,
      lastServerSeq: this.lastServerSeq,
      authenticated: this._authenticated,
    };
  }

  onChange(listener: SyncListener): () => void {
    this.listeners.push(listener);
    return () => {
      this.listeners = this.listeners.filter((l) => l !== listener);
    };
  }

  private notify(): void {
    const state = this.getState();
    this.listeners.forEach((l) => l(state));
  }

  private handleMessage(data: any): void {
    switch (data.type) {
      case "__connected":
        this.sendSyncRequest();
        this.notify();
        break;

      case "__disconnected":
        this.notify();
        break;

      case "ack":
        this.confirmedReqIds.add(data.request_id);
        this.pendingAcks.delete(data.request_id);
        if (data.server_seq > this.lastServerSeq) {
          this.lastServerSeq = data.server_seq;
        }
        break;

      case "sync":
        this.handleSync(data);
        break;

      case "auth_status":
        this._authenticated = !!data.authenticated;
        this.notify();
        break;

      case "error":
        console.error("WS error:", data);
        break;
    }
  }

  private handleSync(data: any): void {
    if (data.latest_server_seq > this.lastServerSeq) {
      this.lastServerSeq = data.latest_server_seq;
    }

    for (const syncOp of data.ops || []) {
      // 自分が送ったopはRGAに既に適用済みなのでスキップ
      if (this.pendingAcks.has(syncOp.request_id) || this.confirmedReqIds.delete(syncOp.request_id)) {
        continue;
      }

      const nodeId: NodeID = {
        siteId: syncOp.node_id?.site_id ?? "",
        timestamp: syncOp.node_id?.timestamp ?? 0,
      };

      const op: Operation = {
        requestId: syncOp.request_id,
        opType: syncOp.op_type,
        nodeId,
        after: syncOp.after
          ? { siteId: syncOp.after.site_id, timestamp: syncOp.after.timestamp }
          : null,
        value: syncOp.value ?? "",
        authenticated: syncOp.authenticated ?? true,
      };

      this.rga.apply(op);
    }

    this.notify();
  }

  private sendSyncRequest(): void {
    this.ws.send({
      type: "sync_request",
      request_id: genReqId(),
      entry_id: this.entryId,
      last_server_seq: this.lastServerSeq,
    });
  }
}
