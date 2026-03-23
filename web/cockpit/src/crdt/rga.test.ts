import { describe, it, expect } from "vitest";
import { RGA, type Operation } from "./rga";

function applyAll(rga: RGA, ops: Operation[]) {
  for (const op of ops) {
    rga.apply(op);
  }
}

describe("RGA", () => {
  it("順次挿入でテキストが構築される", () => {
    const rga = new RGA("site-a");
    const op1 = rga.insert(null, "H");
    rga.insert(op1.nodeId, "i");
    expect(rga.text()).toBe("Hi");
  });

  it("先頭への挿入", () => {
    const rga = new RGA("site-a");
    rga.insert(null, "b");
    rga.insert(null, "a"); // 先頭に挿入（after=null, timestamp大→左）
    expect(rga.text()).toBe("ab");
  });

  it("削除でトゥームストーンになる", () => {
    const rga = new RGA("site-a");
    const op1 = rga.insert(null, "a");
    const insB = rga.insert(op1.nodeId, "b");
    rga.insert(insB.nodeId, "c");

    rga.delete(insB.nodeId);
    expect(rga.text()).toBe("ac");
  });

  it("冪等性: 同じrequest_idのopは2度適用されない", () => {
    const rga = new RGA("site-a");
    const op = rga.insert(null, "x");

    const applied = rga.apply(op);
    expect(applied).toBe(false);
    expect(rga.text()).toBe("x");
  });

  it("収束性: 同じopを異なる順序で適用しても同じ結果", () => {
    // サイトAで"abc"を構築
    const srcA = new RGA("site-a");
    const base: Operation[] = [];
    let after: { siteId: string; timestamp: number } | null = null;
    for (const ch of "abc") {
      const op = srcA.insert(after, ch);
      base.push(op);
      after = op.nodeId;
    }

    // サイトBで追加op
    const srcB = new RGA("site-b");
    applyAll(srcB, base);
    const opB = srcB.insert(base[0].nodeId, "X");

    // サイトCで追加op
    const srcC = new RGA("site-c");
    applyAll(srcC, base);
    const opC = srcC.insert(base[0].nodeId, "Y");

    // レプリカ1: base → B → C
    const r1 = new RGA("r1");
    applyAll(r1, base);
    r1.apply(opB);
    r1.apply(opC);

    // レプリカ2: base → C → B
    const r2 = new RGA("r2");
    applyAll(r2, base);
    r2.apply(opC);
    r2.apply(opB);

    expect(r1.text()).toBe(r2.text());
  });

  it("nodeAt: テキスト位置からNodeIDを取得できる", () => {
    const rga = new RGA("site-a");
    const op1 = rga.insert(null, "a");
    const op2 = rga.insert(op1.nodeId, "b");
    const op3 = rga.insert(op2.nodeId, "c");

    expect(rga.nodeAt(0)).toEqual(op1.nodeId);
    expect(rga.nodeAt(1)).toEqual(op2.nodeId);
    expect(rga.nodeAt(2)).toEqual(op3.nodeId);
    expect(rga.nodeAt(3)).toBeNull();
    expect(rga.nodeAt(-1)).toBeNull();
  });

  it("nodeAt: 削除済みノードをスキップする", () => {
    const rga = new RGA("site-a");
    const op1 = rga.insert(null, "a");
    const op2 = rga.insert(op1.nodeId, "b");
    const op3 = rga.insert(op2.nodeId, "c");
    rga.delete(op2.nodeId);

    // text() = "ac", pos0=a, pos1=c
    expect(rga.nodeAt(0)).toEqual(op1.nodeId);
    expect(rga.nodeAt(1)).toEqual(op3.nodeId);
  });

  it("visibleNodes: 可視ノードのIDリストを返す", () => {
    const rga = new RGA("site-a");
    const op1 = rga.insert(null, "a");
    const op2 = rga.insert(op1.nodeId, "b");
    const op3 = rga.insert(op2.nodeId, "c");
    rga.delete(op2.nodeId);

    const visible = rga.visibleNodes();
    expect(visible).toEqual([op1.nodeId, op3.nodeId]);
  });

  it("pendingバッファ: afterノードが未到着のopは後で適用される", () => {
    const src = new RGA("site-a");
    const op1 = src.insert(null, "a");
    const op2 = src.insert(op1.nodeId, "b");

    // op2を先に適用（op1が未到着）
    const dest = new RGA("site-b");
    dest.apply(op2);
    expect(dest.text()).toBe("");

    // op1を適用→op2もflushされる
    dest.apply(op1);
    expect(dest.text()).toBe("ab");
  });
});
