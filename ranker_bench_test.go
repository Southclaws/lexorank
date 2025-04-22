package lexorank_test

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/Southclaws/lexorank"
)

type reorderableNode struct {
	id   int
	sort lexorank.Key
}

func (n *reorderableNode) GetKey() lexorank.Key {
	return n.sort
}

func (n *reorderableNode) SetKey(k lexorank.Key) {
	n.sort = k
}

func BenchmarkReorderableList_FullSpace(b *testing.B) {
	const base = 75
	const precision = 3 // 4 and above = 10+ minute benchmark lol
	maxItems := int(math.Pow(base, precision))

	b.Log("Max items:", maxItems)

	list := make(lexorank.ReorderableList, maxItems)
	for i := 0; i < maxItems; i++ {
		list[i] = &reorderableNode{id: i, sort: lexorank.Key{}}
	}

	b.Log("Done initialising list:", len(list))

	b.ResetTimer()
	list.Normalise()
	b.StopTimer()

	b.Log("Done normalising list:")

	for i := 1; i < len(list); i++ {
		left := list[i-1].GetKey()
		right := list[i].GetKey()
		cmp := left.Compare(right)

		if cmp >= 0 {
			b.Fatalf("Keys not increasing at %d: left: %s, right: %s", i, left.String(), right.String())
		}
	}
}

func BenchmarkReorderableList_RandomInsert(b *testing.B) {
	r := rand.New(rand.NewSource(42))

	const base = 75
	const precision = 2
	maxItems := int(math.Pow(base, precision))

	b.Log("Max items:", maxItems)

	list := make(lexorank.ReorderableList, 0)

	b.ResetTimer()
	list.Normalise()
	b.StopTimer()

	b.Log("Done normalising list:")

	for i := 0; i < maxItems; i++ {
		pos := r.Intn(len(list) + 1)

		fmt.Println("insert", i, pos)

		key, err := list.Insert(uint(pos))
		if err != nil {
			list.Normalise()
			key, err = list.Insert(uint(pos))
			if err != nil {
				b.Fatalf("Insert failed after normalisation at iteration %d: %v", i, err)
			}
		}

		list = append(list, &reorderableNode{id: i, sort: *key})

	}
}
