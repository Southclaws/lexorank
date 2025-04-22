package lexorank

import (
	"fmt"
)

var ErrOutOfBounds = fmt.Errorf("out of bounds")

type Orderable interface {
	GetKey() Key
}

type Mutable interface {
	SetKey(Key)
}

type Reorderable interface {
	Orderable
	Mutable
}

// ReorderableList represents a collection of orderable items, usually from a
// database. It's designed so that you read a range of items from your storage
// that you wish to apply one or more re-order operations to before saving them
// back in bulk. It supports automatic re-balancing of the keys if a between key
// goes beyond the advised length limit for the default lexorank length of 6.
//
// The Reorderable interface describes a type that supports mutating its own key
// in order to facilitate moving items or re-balancing the list.
//
// Rebalancing does not necessarily mean a write to every item, as the inline
// rebalance algorithm can operate on a small amount of neighbour items before
// falling back to normalising the entire list if it deems necessary.
//
// Reorderable list is assumed to be already ordered upon instantiation.
type ReorderableList []Reorderable

// Purely for testing purposes.
func (a ReorderableList) Len() int           { return len(a) }
func (a ReorderableList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ReorderableList) Less(i, j int) bool { return a[i].GetKey().String() < a[j].GetKey().String() }

func (l ReorderableList) Insert(position uint) (*Key, error) {
	if position > uint(len(l)) {
		return nil, ErrOutOfBounds
	}

	if position == 0 {
		k := l.Prepend()
		return &k, nil
	}

	if position == uint(len(l)) {
		k := l.Append()
		return &k, nil
	}

	prev := l[position-1].GetKey()
	next := l[position].GetKey()

	for {
		k, ok := prev.Between(next)
		if ok {
			return k, nil
		}

		l.rebalanceFrom(position, 1)

		// refresh prev/next keys
		prev = l[position-1].GetKey()
		next = l[position].GetKey()
	}
}

// Append does not change the size of the underlying list, but it may rebalance
// if necessary. It returns a new key which is ordered after the last item.
//
// In a worst case scenario, if the list already has a key at the maximum index,
// the list is rebalanced to make space at the end for the new generated key.
func (l ReorderableList) Append() Key {
	if len(l) == 0 {
		return Bottom
	}

	last := l[len(l)-1]
	for {
		k, ok := last.GetKey().Between(Top)
		if ok {
			return *k
		}

		l.rebalanceFrom(uint(len(l)-1), -1)
	}
}

// Prepend does not change the size of the underlying list, but it may rebalance
// if necessary. It returns a new key which is ordered before the first item.
//
// Same worst case scenario as Append.
func (l ReorderableList) Prepend() Key {
	if len(l) == 0 {
		return Top
	}

	for {
		k, ok := Bottom.Between(l[0].GetKey())
		if ok {
			return *k
		}

		l.rebalanceFrom(0, 1)
	}
}

func (l ReorderableList) rebalanceFrom(position uint, direction int) {
	ok := l.tryRebalanceFrom(position, direction)
	if ok {
		return
	}

	// If we're here, the worst case scenario was reached: every key is adjacent
	// to the next one. We need to normalise the entire list.

	l.Normalise()
}

func (l ReorderableList) tryRebalanceFrom(position uint, direction int) bool {
	if direction > 0 {
		for i := int(position); i < len(l)-1; i++ {
			curr := l[i].GetKey()
			next := l[i+1].GetKey()

			nextKey, ok := curr.Between(next)
			if ok {
				l[i+1].SetKey(*nextKey)
				if i == int(position) {
					// first pass worked, can exit early.
					return true
				}
			}

			// If not OK, continue to rebalance forwards by shifting every key
		}
	} else {
		for i := int(position); i > 0; i-- {
			curr := l[i].GetKey()
			next := l[i-1].GetKey()

			nextKey, ok := next.Between(curr)
			if ok {
				l[i-1].SetKey(*nextKey)
				if i == int(position) {
					// first pass worked, can exit early.
					return true
				}
			}

			// If not OK, continue to rebalance forwards by shifting every key
		}
	}

	return false
}

// Normalise will distribute the keys evenly across the key space.
func (l ReorderableList) Normalise() {
	for i := 0; i < len(l); i++ {
		f := float64(i) / float64(len(l))
		b := l[i].GetKey().bucket
		nextKey := KeyAt(b, f)
		l[i].SetKey(nextKey)
	}
}

func (l ReorderableList) IsSorted() bool {
	for i := 1; i < len(l); i++ {
		if l[i-1].GetKey().Compare(l[i].GetKey()) >= 0 {
			return false
		}
	}
	return true
}
