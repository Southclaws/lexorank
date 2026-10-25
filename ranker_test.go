package lexorank

import (
	"bytes"
	"slices"
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

func TestInsert_PartialRebalance(t *testing.T) {
	for _, tt := range []struct {
		name      string
		keys      []string
		position  uint
		unchanged []int
	}{
		{
			name:      "issue 2",
			keys:      []string{"0|AAAAAA", "0|AAAAAB", "0|B"},
			position:  1,
			unchanged: []int{0, 2},
		},
		{
			name:      "space beyond adjacent keys",
			keys:      []string{"0|AAAAAA", "0|AAAAAB", "0|AAAAAC", "0|B"},
			position:  1,
			unchanged: []int{0, 3},
		},
		{
			name:      "middle of list",
			keys:      []string{"0|9", "0|AAAAAA", "0|AAAAAB", "0|B", "0|C"},
			position:  2,
			unchanged: []int{0, 1, 3, 4},
		},
		{
			name:      "prepend with immediate space",
			keys:      []string{"0|000000", "0|B"},
			position:  0,
			unchanged: []int{1},
		},
		{
			name:      "prepend at bottom boundary",
			keys:      []string{"0|000000", "0|000001", "0|B"},
			position:  0,
			unchanged: []int{2},
		},
		{
			name:     "prepend needs full normalisation",
			keys:     []string{"0|000000", "0|000002"},
			position: 0,
		},
		{
			name:     "append needs full normalisation",
			keys:     []string{"0|zzzzzx", "0|zzzzzz"},
			position: 2,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			list := make(ReorderableList, len(tt.keys))
			for i, key := range tt.keys {
				list[i] = item(i, key)
			}

			key, err := list.Insert(tt.position)
			require.NoError(t, err)
			require.NotNil(t, key)
			if tt.position > 0 {
				assert.Less(t, list[tt.position-1].GetKey().String(), key.String())
			}
			if tt.position < uint(len(list)) {
				assert.Less(t, key.String(), list[tt.position].GetKey().String())
			}
			assert.True(t, list.IsSorted())
			for _, i := range tt.unchanged {
				assert.Equal(t, tt.keys[i], list[i].GetKey().String(), "unaffected item %d", i)
			}
		})
	}
}

func FuzzReorderableList_Insert(f *testing.F) {
	initial := [][]string{
		{"AAAAAA", "AAAAAB", "B"},
		{"AAAAAA", "AAAAAB", "AAAAAC", "B"},
		{"000000", "B"},
		{"000000", "000001", "B"},
		{"y", "zzzzzy", "zzzzzz"},
		{"000000", "000001", "000002"},
		{"zzzzzx", "zzzzzy", "zzzzzz"},
	}
	for i := range initial {
		for bucket := uint8(0); bucket < 3; bucket++ {
			f.Add(uint8(i), bucket, []byte{1, 0, 255, 2, 1})
			f.Add(uint8(i), bucket, bytes.Repeat([]byte{1}, 128))
		}
	}
	f.Fuzz(func(t *testing.T, start, bucket uint8, positions []byte) {
		// Bound each sequence while retaining repeated inserts into dense gaps.
		if len(positions) > 256 {
			positions = positions[:256]
		}
		prefix := string([]byte{'0' + bucket%3, '|'})
		var list ReorderableList
		for i, rank := range initial[int(start)%len(initial)] {
			list = append(list, item(i, prefix+rank))
		}

		for step, p := range positions {
			position := int(p) % (len(list) + 1)
			length := len(list)
			key, err := list.Insert(uint(position))
			require.NoError(t, err, "step %d at position %d", step, position)
			require.NotNil(t, key)
			require.Len(t, list, length, "Insert must only generate a key")

			list = slices.Insert(list, position, Reorderable(&Item{ID: length, Rank: *key}))
			for i, node := range list {
				value := node.GetKey().String()
				require.Equal(t, prefix, value[:2], "bucket must be preserved")
				require.LessOrEqual(t, len(value), 8, "rank must fit six characters")
				if i > 0 {
					require.Less(t, list[i-1].GetKey().String(), value,
						"keys must be unique and ordered after step %d", step)
				}
			}
		}
	})
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
	end, _ := start.Between(TopOf(1)) // something like 1|m

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
	mid, _ := start.Between(TopOf(1)) // enough space

	list := ReorderableList{
		&Item{ID: 0, Rank: *start},
		&Item{ID: 1, Rank: *mid},
	}

	ok := list.tryRebalanceFrom(0, 1)
	a.True(ok, "expected forward rebalance to succeed on first pass")
	a.True(sort.IsSorted(list), "list should still be sorted")
	a.NotEqual(start.String(), list[0].GetKey().String(), "requested item should move towards its successor")
	a.Equal(mid.String(), list[1].GetKey().String(), "successor should be unchanged")
}

func TestTryRebalanceFrom_TightAtTop(t *testing.T) {
	a := assert.New(t)

	list := ReorderableList{
		item(5, "0|UUUUUU"), item(6, "0|g"), item(7, "0|g"), item(8, "0|g"), item(9, "0|g"), item(10, "0|g"), item(11, "0|k"), item(12, "0|p"), item(13, "0|p"), item(14, "0|p"), item(15, "0|p"), item(16, "0|u"), item(17, "0|u"), item(18, "0|w"), item(19, "0|x"), item(20, "0|y"), item(21, "0|yU"), item(22, "0|yg"), item(23, "0|yp"), item(24, "0|yu"), item(25, "0|yw"), item(26, "0|yx"), item(27, "0|yy"), item(28, "0|yyU"), item(29, "0|yyg"), item(30, "0|yyp"), item(31, "0|yyu"), item(32, "0|yyw"), item(33, "0|yyx"), item(34, "0|yyx"), item(35, "0|yyy"), item(36, "0|yyyU"), item(37, "0|yyyp"), item(38, "0|yyyu"), item(39, "0|yyyw"), item(40, "0|yyyy"), item(41, "0|yyyyB"), item(42, "0|yyyyU"), item(43, "0|yyyyp"), item(44, "0|yyyyr"), item(45, "0|yyyyu"), item(46, "0|yyyyw"), item(47, "0|yyyyx"), item(48, "0|yyyyy"),
	}

	list.Normalise()

	first := list[0]
	last := list[len(list)-1]
	a.Equal("0|3>K9[6", first.GetKey().String())
	a.Equal("0|wl_qOt", last.GetKey().String())
}

func item(id int, s string) Reorderable {
	o, err := ParseKey(s)
	if err != nil {
		panic(err)
	}
	return &Item{ID: id, Rank: *o}
}
