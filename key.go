package lexorank

import (
	"bytes"
	"fmt"
	"math/rand"
	"strconv"
)

var ErrRebalance = fmt.Errorf("rebalance required")

const (
	Minimum  = '0'
	Midpoint = 'U'
	Maximum  = 'z'
)

var (
	Bottom = Key{
		raw:    []byte{'0', '|', Minimum},
		rank:   []byte{Minimum},
		bucket: 0,
	}
	Top = Key{
		raw:    []byte{'0', '|', Maximum, Maximum, Maximum, Maximum, Maximum, Maximum},
		rank:   []byte{Maximum, Maximum, Maximum, Maximum, Maximum, Maximum},
		bucket: 0,
	}
	Middle = Key{
		raw:    []byte{'0', '|', Midpoint, Midpoint, Midpoint, Midpoint, Midpoint, Midpoint},
		rank:   []byte{Midpoint, Midpoint, Midpoint, Midpoint, Midpoint, Midpoint},
		bucket: 0,
	}

	// When normalising a range, these defaults are the keys to use.
	NormaliseTop, _    = Middle.Between(Top)
	NormaliseBottom, _ = Middle.Between(Bottom)
)

const (
	keyLength  = 8 // the full key length "0|aaaaaa"
	rankLength = 6 // the part after the |: "aaaaaa"
)

type Key struct {
	raw    []byte // "0|aaaaaa"
	rank   Rank   // "aaaaaa"
	bucket uint8  // 0
}

func (k Key) String() string {
	return string(k.raw)
}

func (k Key) Compare(b Key) int {
	return bytes.Compare(k.raw, b.raw)
}

func (k *Key) SetBucket(b uint8) {
	if b > 2 {
		b = 0
	}
	k.bucket = b
}

type Keys []Key

type Rank []byte

func (r Rank) atMin(i int) byte {
	if i >= len(r) {
		return Minimum
	}

	return r[i]
}

func (r Rank) atMax(i int) byte {
	if i >= len(r) {
		return Maximum
	}

	return r[i]
}

func ParseKey(s string) (*Key, error) {
	if len(s) > keyLength {
		return nil, fmt.Errorf("invalid key length: %d", len(s))
	}

	bucket, err := strconv.Atoi(string(s[0]))
	if err != nil {
		return nil, err
	}

	rank := []byte(s[2:])
	for _, b := range rank {
		if b < Minimum || b > Maximum {
			return nil, fmt.Errorf("invalid byte value: %c", b)
		}
	}

	return &Key{
		raw:    []byte(s),
		rank:   rank,
		bucket: uint8(bucket),
	}, nil
}

// Between returns a new key that is between the current key and the second key.
// If the boolean return value is false, it indicates keys are getting too long
// and thus a rebalance is required. "Too long" is very subjective. The limit
// set in this library is 6 which gives you around 400k worst case re-orders.
func (k Key) Between(to Key) (*Key, bool) {
	mk := &Key{
		raw:    []byte{},
		rank:   Rank{},
		bucket: k.bucket,
	}

	for i := 0; ; i++ {
		prevChar := k.rank.atMin(i)
		nextChar := to.rank.atMax(i)

		if prevChar == nextChar {
			mk.rank = append(mk.rank, prevChar)
			continue
		}

		m, ok := mid(prevChar, nextChar)
		if !ok {
			mk.rank = append(mk.rank, prevChar)
			continue
		}

		mk.rank = append(mk.rank, m)
		break
	}

	if string(mk.rank) >= string(to.rank) {
		return &k, false
	}

	// ASCII representation of bucket value, ranges from 0-2 so 1 basic addition works fine
	bucketChar := k.bucket + 48

	mk.raw = append(mk.raw, []byte{bucketChar, '|'}...)
	mk.raw = append(mk.raw, mk.rank...)

	return mk, len(mk.rank) < rankLength
}

func mid(a, b byte) (byte, bool) {
	if a == b {
		return a, false
	}

	m := (a + b) / 2

	if m == a || m == b {
		return a, false
	}

	return m, true
}

func Random() Key {
	rank := []byte{random(), random(), random(), random(), random(), random()}
	raw := append([]byte{'0', '|'}, rank...)
	return Key{
		raw:    raw,
		rank:   rank,
		bucket: 0,
	}
}

func random() byte {
	return byte(Minimum + rand.Intn(Maximum-Minimum))
}
