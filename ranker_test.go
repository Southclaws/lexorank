package lexorank

import (
	"sort"
	"testing"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Item is a simple struct that implements the Reorderable interface. In your
// application, this may be a Post or Issue data model for example.
type Item struct {
	ID   int
	Rank Key
}

// Implements the Orderable interface.
func (i Item) GetKey() Key { return i.Rank }

// Implements the Mutable interface
func (i *Item) SetKey(k Key) { i.Rank = k }

func TestReorderableList_Rebalance(t *testing.T) {
	a := assert.New(t)

	original := ReorderableList{
		item(0, "1|aaaaaa"),
		item(1, "1|aaaaab"),
		item(2, "1|aaaaac"),
		item(3, "1|aaaaad"),
		item(4, "1|aaaaae"),
		item(5, "1|aaaaaf"),
	}
	data := ReorderableList{
		item(0, "1|aaaaaa"),
		item(1, "1|aaaaab"),
		item(2, "1|aaaaac"),
		item(3, "1|aaaaad"),
		item(4, "1|aaaaae"),
		item(5, "1|aaaaaf"),
	}
	a.Equal(original, data)

	data.rebalanceFrom(0, 1)

	a.NotEqual(original, data)
	a.True(sort.IsSorted(data))

	a.NotEqual(pretty.Sprint(original), pretty.Sprint(data))

	for i := range data {
		before := original[i]
		after := data[i]

		a.Equal(before.GetKey().bucket, after.GetKey().bucket)
		t.Log(after.GetKey().String())
	}
}

func TestReorderableList_Insert(t *testing.T) {
	r := require.New(t)
	a := assert.New(t)

	list := ReorderableList{
		item(0, "1|aaa"),
		item(1, "1|aab"),
		item(2, "1|aac"),
		item(3, "1|aad"),
		item(4, "1|aae"),
		item(5, "1|aaf"),
	}
	before := list[2].GetKey()
	after := list[3].GetKey()

	newKey, err := list.Insert(3)
	r.NoError(err)

	t.Log("before", before)
	t.Log("newKey", newKey)
	t.Log("after:", after)

	a.Equal(newKey.Compare(before), 1, "placed after index 1")
	a.Equal(newKey.Compare(after), -1, "placed before index 2")
	a.Equal("1|aacU", newKey.String())
}

func TestInsert_TriggersRebalance(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// Saturated keys — no space between
	k1, _ := ParseKey("1|aaaaaa")
	k2 := Key{
		raw:    []byte("1|aaaaaa"),
		rank:   []byte{'a', 'a', 'a', 'a', 'a', 'a'},
		bucket: 1,
	}
	k2.rank[5]++ // very small diff
	k2.raw[7]++  // keep raw consistent

	list := ReorderableList{
		&Item{ID: 0, Rank: *k1},
		&Item{ID: 1, Rank: k2},
	}

	oldKey := list[1].GetKey().String()

	newKey, err := list.Insert(1) // insert between 0 and 1
	r.NoError(err)

	a.True(newKey.Compare(list[0].GetKey()) > 0)
	a.True(newKey.Compare(list[1].GetKey()) < 0)

	// The fallback rebalanced list[1], so its key must have changed
	a.NotEqual(oldKey, list[1].GetKey().String(), "rebalance should have changed the key")
}

func TestReorderableList_Append(t *testing.T) {
	a := assert.New(t)

	list := ReorderableList{
		item(0, "1|aaaaaa"),
		item(1, "1|aaaaab"),
		item(2, "1|aaaaac"),
		item(3, "1|aaaaad"),
		item(4, "1|aaaaae"),
		item(5, "1|aaaaaf"),
	}
	last := list[len(list)-1].GetKey()

	newKey := list.Append()

	a.Equal(newKey.Compare(last), 1, "newKey is sorted before the first item")
	a.Equal("1|m", newKey.String())

	for i := range list {
		t.Log("list", i, list[i].GetKey().String())
	}
	t.Log("newKey", newKey.String())
	t.Log("topKey", Top.String())
}

func TestReorderableList_AppendRebalance(t *testing.T) {
	a := assert.New(t)

	list := ReorderableList{
		item(0, "1|aaaaaa"),
		item(1, "1|aaaaab"),
		item(2, "1|aaaaac"),
		item(3, "1|aaaaad"),
		item(4, "1|aaaaae"),
		item(5, "1|zzzzzz"),
	}
	last := list[len(list)-1].GetKey()

	newKey := list.Append()

	a.Equal(newKey.Compare(last), -1, "newKey is sorted before the first item")
	a.NotEqual(list[len(list)-1].GetKey().String(), "1|zzzzzz", "last item has been rebalanced to the mid point between end and end-1")

	for i := range list {
		t.Log("list", i, list[i].GetKey().String())
	}
	t.Log("newKey", newKey.String())
	t.Log("topKey", Top.String())
}

func TestReorderableList_Prepend(t *testing.T) {
	a := assert.New(t)

	list := ReorderableList{
		item(0, "1|aaaaaa"),
		item(1, "1|aaaaab"),
		item(2, "1|aaaaac"),
		item(3, "1|aaaaad"),
		item(4, "1|aaaaae"),
		item(5, "1|aaaaaf"),
	}
	first := list[0].GetKey()

	newKey := list.Prepend()

	a.Equal(newKey.Compare(first), -1, "newKey is sorted before the first item")
}

func TestReorderableList_PrependRebalance(t *testing.T) {
	a := assert.New(t)

	list := ReorderableList{
		item(0, "1|aaaaaa"),
		item(1, "1|aaaaab"),
		item(2, "1|aaaaac"),
		item(3, "1|aaaaad"),
		item(4, "1|aaaaae"),
		item(5, "1|aaaaaf"),
	}
	first := list[0].GetKey()

	newKey := list.Prepend()

	a.Equal(newKey.Compare(first), -1, "newKey is sorted before the first item")
	a.NotEqual(list[0].GetKey().String(), "1|0", "first item has been rebalanced to the mid point between index 0 and index 1")
}

func TestInsert_OutOfBounds(t *testing.T) {
	list := ReorderableList{
		item(0, "1|aaa"),
		item(1, "1|aab"),
	}

	_, err := list.Insert(5)
	assert.Error(t, err)
	assert.Equal(t, ErrOutOfBounds, err)
}

func TestInsert_AtStart(t *testing.T) {
	list := ReorderableList{
		item(0, "1|aab"),
		item(1, "1|aac"),
	}

	key, err := list.Insert(0)
	assert.NoError(t, err)
	assert.True(t, key.Compare(list[0].GetKey()) < 0, "inserted key should sort before the first")
}

func TestInsert_AtEnd(t *testing.T) {
	list := ReorderableList{
		item(0, "1|aaa"),
		item(1, "1|aab"),
	}

	key, err := list.Insert(uint(len(list)))
	assert.NoError(t, err)
	assert.True(t, key.Compare(list[len(list)-1].GetKey()) > 0, "inserted key should sort after the last")
}

func TestReorderableList_Append_HitsBackwardsRebalance(t *testing.T) {
	a := assert.New(t)

	list := ReorderableList{
		item(0, "1|zzzzzz"), // Last key: max
	}

	newKey := list.Append() // Should trigger rebalanceFrom

	a.True(newKey.Compare(list[0].GetKey()) > 0, "newKey must sort after existing key")
	a.True(sort.IsSorted(list), "list must remain sorted")
}

func TestReorderableList_BackwardRebalanceLogic(t *testing.T) {
	a := assert.New(t)

	list := ReorderableList{
		item(0, "1|aaaaaa"),
		item(1, "1|aaaaab"),
		item(2, "1|aaaaac"),
		item(3, "1|aaaaad"),
		item(4, "1|aaaaae"),
		item(5, "1|aaaaaf"),
	}
	list.rebalanceFrom(5, -1)

	a.True(sort.IsSorted(list), "list should be sorted after backward rebalance")
}

func TestTryRebalanceFrom_BackwardFailsWithWrongBetweenOrder(t *testing.T) {
	a := assert.New(t)

	// Two adjacent keys, where Between(curr, prev) will fail
	start, _ := ParseKey("1|aaaaaa")
	end, _ := start.Between(Top) // something like 1|m

	list := ReorderableList{
		&Item{ID: 0, Rank: *start},
		&Item{ID: 1, Rank: *end},
	}

	// We intentionally call tryRebalanceFrom on index 1, going backward (-1)
	ok := list.tryRebalanceFrom(1, -1)
	a.True(ok, "should succeed if Between() arg order is correct")

	// If successful, keys should still be sorted
	a.True(sort.IsSorted(list), "list must be sorted after backward rebalance")
}

func TestTryRebalanceFrom_ForwardFirstPassSucceeds(t *testing.T) {
	a := assert.New(t)

	start, _ := ParseKey("1|aaaaaa")
	mid, _ := start.Between(Top) // enough space

	list := ReorderableList{
		&Item{ID: 0, Rank: *start},
		&Item{ID: 1, Rank: *mid},
	}

	ok := list.tryRebalanceFrom(0, 1)
	a.True(ok, "expected forward rebalance to succeed on first pass")
	a.True(sort.IsSorted(list), "list should still be sorted")
	a.NotEqual(mid.String(), list[1].GetKey().String(), "key should have changed during rebalance")
}

func item(id int, s string) Reorderable {
	o, err := ParseKey(s)
	if err != nil {
		panic(err)
	}
	return &Item{ID: id, Rank: *o}
}
