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

func item(id int, s string) Reorderable {
	o, err := ParseKey(s)
	if err != nil {
		panic(err)
	}
	return &Item{ID: id, Rank: *o}
}
