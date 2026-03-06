package crdt_test

import (
	"testing"

	"flourish/server/domain/crdt"

	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// --- ヘルパー ---

func applyAll(r *crdt.RGA, ops []crdt.Operation) {
	for _, op := range ops {
		r.Apply(op)
	}
}

// randomString はrapid用のランダム文字列ジェネレータ。
func randomString(chars []rune, minLen, maxLen int) *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		length := rapid.IntRange(minLen, maxLen).Draw(t, "len")
		result := make([]rune, length)
		for i := range result {
			result[i] = rapid.SampledFrom(chars).Draw(t, "ch")
		}
		return string(result)
	})
}

// interleave は2つのオペレーション列を因果順序を保ちながらランダムにインターリーブする。
func interleave(t *rapid.T, ops1, ops2 []crdt.Operation) []crdt.Operation {
	result := make([]crdt.Operation, 0, len(ops1)+len(ops2))
	i, j := 0, 0
	for i < len(ops1) || j < len(ops2) {
		if i < len(ops1) && (j >= len(ops2) || rapid.Bool().Draw(t, "pick")) {
			result = append(result, ops1[i])
			i++
		} else if j < len(ops2) {
			result = append(result, ops2[j])
			j++
		}
	}
	return result
}

// --- PBT ---

// 親子順序: after:X で挿入したノードは、Text()上でXの直後に出現する。
func TestPBT_ParentChildOrder(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		r := crdt.NewRGA(uuid.New())

		// ランダムなベーステキストを構築
		baseText := randomString([]rune{'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h'}, 2, 10).Draw(t, "base")
		var nodeIDs []crdt.NodeID
		var after *crdt.NodeID
		for _, ch := range baseText {
			op := r.Insert(after, ch)
			nodeIDs = append(nodeIDs, op.NodeID)
			after = &op.NodeID
		}

		// ランダムな位置の後ろに挿入
		pos := rapid.IntRange(0, len(nodeIDs)-1).Draw(t, "pos")
		parentID := nodeIDs[pos]
		r.Insert(&parentID, 'Z')

		runes := []rune(r.Text())
		if runes[pos+1] != 'Z' {
			t.Errorf("親子順序違反: pos=%d, text=%q", pos, string(runes))
		}
	})
}

// 削除: 削除したノードはText()に含まれない。削除前のテキストから対象文字を抜いた結果と一致する。
func TestPBT_DeleteRemovesFromText(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		r := crdt.NewRGA(uuid.New())

		// ランダムなベーステキストを構築
		baseText := randomString([]rune{'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h'}, 2, 10).Draw(t, "base")
		var nodeIDs []crdt.NodeID
		var after *crdt.NodeID
		for _, ch := range baseText {
			op := r.Insert(after, ch)
			nodeIDs = append(nodeIDs, op.NodeID)
			after = &op.NodeID
		}

		before := []rune(r.Text())
		target := rapid.IntRange(0, len(nodeIDs)-1).Draw(t, "target")
		r.Delete(nodeIDs[target])

		// 削除前のテキストからtarget位置の文字を除いたものと一致すべき
		expected := string(append(before[:target], before[target+1:]...))
		if got := r.Text(); got != expected {
			t.Errorf("削除結果が不正: got %q, want %q", got, expected)
		}
	})
}

// 収束: 同じオペレーション群を異なる順序で適用しても同じText()になる。
func TestPBT_Convergence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// ベースを構築
		src := crdt.NewRGA(uuid.New())
		baseText := randomString([]rune{'a', 'b', 'c', 'd', 'e'}, 2, 8).Draw(t, "base")
		var baseOps []crdt.Operation
		var after *crdt.NodeID
		for _, ch := range baseText {
			op := src.Insert(after, ch)
			after = &op.NodeID
			baseOps = append(baseOps, op)
		}

		// 2つのソースRGAでops生成
		src1 := crdt.NewRGA(uuid.New())
		src2 := crdt.NewRGA(uuid.New())
		applyAll(src1, baseOps)
		applyAll(src2, baseOps)

		var ops1, ops2 []crdt.Operation
		nOps := rapid.IntRange(1, 5).Draw(t, "nOps")
		for range nOps {
			pos := rapid.IntRange(0, len(baseOps)).Draw(t, "pos1")
			var a *crdt.NodeID
			if pos > 0 {
				nid := baseOps[pos-1].NodeID
				a = &nid
			}
			ops1 = append(ops1, src1.Insert(a, 'X'))
		}
		for range nOps {
			pos := rapid.IntRange(0, len(baseOps)).Draw(t, "pos2")
			var a *crdt.NodeID
			if pos > 0 {
				nid := baseOps[pos-1].NodeID
				a = &nid
			}
			ops2 = append(ops2, src2.Insert(a, 'Y'))
		}

		// 3つの検証用レプリカ
		a := crdt.NewRGA(uuid.New())
		applyAll(a, baseOps)
		applyAll(a, ops1)
		applyAll(a, ops2)

		b := crdt.NewRGA(uuid.New())
		applyAll(b, baseOps)
		applyAll(b, ops2)
		applyAll(b, ops1)

		c := crdt.NewRGA(uuid.New())
		applyAll(c, baseOps)
		applyAll(c, interleave(t, ops1, ops2))

		if a.Text() != b.Text() || a.Text() != c.Text() {
			t.Errorf("収束性違反: A=%q, B=%q, C=%q", a.Text(), b.Text(), c.Text())
		}
	})
}
