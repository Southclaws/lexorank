package lexorank

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding"
	"encoding/json"
	"fmt"
	"math"
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

	// Charset for encoding positions
	charset  = []byte("0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz")
	maxValue = int(math.Pow(float64(len(charset)), float64(6))) // full keyspace

)

func TopOf(b uint8) Key {
	return Key{
		raw:    []byte{byte(b + '0'), '|', Maximum, Maximum, Maximum, Maximum, Maximum, Maximum},
		rank:   []byte{Maximum, Maximum, Maximum, Maximum, Maximum, Maximum},
		bucket: b,
	}
}

func BottomOf(b uint8) Key {
	return Key{
		raw:    []byte{byte(b + '0'), '|', Minimum},
		rank:   []byte{Minimum},
		bucket: b,
	}
}

func MiddleOf(b uint8) Key {
	return Key{
		raw:    []byte{byte(b + '0'), '|', Midpoint, Midpoint, Midpoint, Midpoint, Midpoint, Midpoint},
		rank:   []byte{Midpoint, Midpoint, Midpoint, Midpoint, Midpoint, Midpoint},
		bucket: b,
	}
}

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

func (k Key) GoString() string {
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

	return parseRaw(uint8(bucket), rank)
}

func parseRaw(bucket uint8, rank []byte) (*Key, error) {
	for _, b := range rank {
		if b < Minimum || b > Maximum {
			return nil, fmt.Errorf("invalid byte value: %c", b)
		}
	}

	raw := append([]byte{byte(bucket + 48), '|'}, rank...)

	return &Key{
		raw:    raw,
		rank:   rank,
		bucket: bucket,
	}, nil
}

// KeyAt generates a key from a specific numeric position in the key space.
func KeyAt(bucket uint8, f float64) Key {
	bucketChar := byte(bucket + 48)

	base := float64(len(charset)) // 75
	key := make([]byte, 0, rankLength)

	for i := 0; i < rankLength; i++ {
		f *= base
		index := int(f)
		if index >= len(charset) {
			index = len(charset) - 1
		}
		key = append(key, charset[index])
		f -= float64(index)

		if f <= 0.0 {
			break
		}
	}

	k, err := ParseKey(string(append([]byte{bucketChar, '|'}, key...)))
	if err != nil {
		panic(err)
	}

	return *k
}

// Between returns a new key that is between the current key and the second key.
// If the boolean return value is false, it indicates keys are getting too long
// and thus a rebalance is required. "Too long" is very subjective. The limit
// set in this library is 6 which gives you around 400k worst case re-orders.
func (k Key) Between(to Key) (*Key, bool) {
	if k.Compare(to) > 0 {
		return to.Between(k)
	}

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

	valid := len(mk.rank) <= rankLength
	if !valid {
		return nil, false
	}

	if string(mk.rank) >= string(to.rank) {
		return nil, false
	}

	// ASCII representation of bucket value, ranges from 0-2 so 1 basic addition works fine
	bucketChar := k.bucket + 48

	mk.raw = append(mk.raw, []byte{bucketChar, '|'}...)
	mk.raw = append(mk.raw, mk.rank...)

	return mk, valid
}

func (k Key) After(distance int64) (*Key, bool) {
	index := decodeBase75(k.rank)
	next := encodeBase75(index + distance)

	n, err := parseRaw(k.bucket, next)
	if err != nil {
		return nil, false
	}

	return n, true
}

func (k Key) Before(distance int64) (*Key, bool) {
	index := decodeBase75(k.rank)
	if index-distance < 0 {
		return nil, false
	}

	next := encodeBase75(index - distance)

	n, err := parseRaw(k.bucket, next)
	if err != nil {
		return nil, false
	}

	return n, true
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

func decodeBase75(rank []byte) int64 {
	var index int64
	for _, c := range rank {
		pos := int64(bytes.IndexByte(charset, c))
		if pos == -1 {
			panic("invalid character in rank")
		}
		index = index*int64(len(charset)) + pos
	}
	return index
}

func encodeBase75(val int64) []byte {
	if val == 0 {
		return []byte{charset[0]}
	}
	var out []byte
	for val > 0 {
		rem := val % int64(len(charset))
		out = append([]byte{charset[rem]}, out...)
		val = val / int64(len(charset))
	}
	return out
}

func Random() Key {
	f := rand.Float64()
	return KeyAt(0, f)
}

var (
	_ encoding.TextMarshaler   = (*Key)(nil)
	_ encoding.TextUnmarshaler = (*Key)(nil)
	_ json.Marshaler           = (*Key)(nil)
	_ json.Unmarshaler         = (*Key)(nil)
	_ driver.Valuer            = (*Key)(nil)
	_ sql.Scanner              = (*Key)(nil)
)

// TextMarshaler
func (k Key) MarshalText() ([]byte, error) {
	return []byte(k.String()), nil
}

// TextUnmarshaler
func (k *Key) UnmarshalText(text []byte) error {
	parsed, err := ParseKey(string(text))
	if err != nil {
		return err
	}
	*k = *parsed
	return nil
}

// JSON Marshaler
func (k Key) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.String())
}

// JSON Unmarshaler
func (k *Key) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := ParseKey(s)
	if err != nil {
		return err
	}
	*k = *parsed
	return nil
}

// SQL Valuer
func (k Key) Value() (driver.Value, error) {
	return k.String(), nil
}

// SQL Scanner
func (k *Key) Scan(value any) error {
	switch v := value.(type) {
	case string:
		parsed, err := ParseKey(v)
		if err != nil {
			return err
		}
		*k = *parsed
		return nil
	case []byte:
		parsed, err := ParseKey(string(v))
		if err != nil {
			return err
		}
		*k = *parsed
		return nil
	default:
		return fmt.Errorf("cannot scan type %T into Key", value)
	}
}
