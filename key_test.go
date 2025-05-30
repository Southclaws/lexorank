package lexorank

import (
	"encoding/json"
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKey_Defaults(t *testing.T) {
	fmt.Println(Bottom)
	fmt.Println(Top)
	fmt.Println(Middle)
}

func TestKey_Between_Insert(t *testing.T) {
	r := require.New(t)
	a := assert.New(t)

	current, err := ParseKey("1|a")
	r.NoError(err)

	next, err := ParseKey("1|b")
	r.NoError(err)

	got, ok := current.Between(*next)
	r.True(ok)
	a.Equal("1|aU", got.String())
}

func TestKey_Between_Rebalance(t *testing.T) {
	r := require.New(t)
	// a := assert.New(t)

	current, err := ParseKey("1|aaaaaa")
	r.NoError(err)

	next, err := ParseKey("1|aaaaab")
	r.NoError(err)

	got, ok := current.Between(*next)
	r.False(ok)
	r.Nil(got)
}

func TestKey_Between_AtStart(t *testing.T) {
	r := require.New(t)
	// a := assert.New(t)

	current, err := ParseKey("1|0")
	r.NoError(err)

	next, err := ParseKey("1|0")
	r.NoError(err)

	got, ok := current.Between(*next)
	r.False(ok)
	r.Nil(got)
}

func TestKey_Between_AtTopClose(t *testing.T) {
	r := require.New(t)

	current, err := ParseKey("0|yyyyy")
	r.NoError(err)

	next, err := ParseKey("0|zzzzzz")
	r.NoError(err)

	got, ok := current.Between(*next)
	r.True(ok)
	r.Equal("0|yyyyyU", got.String())
	r.True(sort.StringsAreSorted([]string{current.String(), got.String(), next.String()}))
}

func TestKey_Between_AtTopNoSpace(t *testing.T) {
	r := require.New(t)

	current, err := ParseKey("0|zzzzzz")
	r.NoError(err)

	got, ok := current.Between(Top)
	r.False(ok)
	r.Nil(got)
}

func TestKey_After(t *testing.T) {
	r := require.New(t)
	a := assert.New(t)

	start, err := ParseKey("0|0")
	r.NoError(err)

	after, ok := start.After(10)
	r.True(ok)
	a.Equal("0|:", after.String())

	a.True(start.Compare(*after) < 0, "expected start < after")

	startIndex := decodeBase75(start.rank)
	afterIndex := decodeBase75(after.rank)
	a.Equal(int64(10), afterIndex-startIndex)
}

func TestKey_AfterLong(t *testing.T) {
	r := require.New(t)
	a := assert.New(t)

	start, err := ParseKey("0|zh2:dA")
	r.NoError(err)

	after, ok := start.After(10000)
	r.True(ok)
	a.Equal("0|zh2<SZ", after.String())

	a.True(start.Compare(*after) < 0, "expected start < after")

	startIndex := decodeBase75(start.rank)
	afterIndex := decodeBase75(after.rank)
	a.Equal(int64(10000), afterIndex-startIndex)
}

func TestKey_Before(t *testing.T) {
	r := require.New(t)
	a := assert.New(t)

	start, err := ParseKey("0|zzzzzz")
	r.NoError(err)

	before, ok := start.Before(10)
	r.True(ok)
	a.Equal("0|zzzzzp", before.String())

	a.True(start.Compare(*before) > 0, "expected start > before")

	startIndex := decodeBase75(start.rank)
	afterIndex := decodeBase75(before.rank)
	a.Equal(int64(10), startIndex-afterIndex)
}

func TestKey_BeforeLong(t *testing.T) {
	r := require.New(t)
	a := assert.New(t)

	start, err := ParseKey("0|zh2:dA")
	r.NoError(err)

	before, ok := start.Before(10000)
	r.True(ok)
	a.Equal("0|zh28ts", before.String())

	a.True(start.Compare(*before) > 0, "expected start > before")

	startIndex := decodeBase75(start.rank)
	afterIndex := decodeBase75(before.rank)
	a.Equal(int64(10000), startIndex-afterIndex)
}

func TestBase75Encoding(t *testing.T) {
	r := require.New(t)
	a := assert.New(t)

	current, err := ParseKey("0|zzzzzz")
	r.NoError(err)

	rawrank := []byte(current.rank)

	decoded := decodeBase75(rawrank)
	r.Equal(int64(177978515624), decoded)

	encoded := encodeBase75(decoded)
	a.Equal(rawrank, encoded)
}

func TestKey_Random(t *testing.T) {
	r := require.New(t)
	// a := assert.New(t)

	k := Random()
	r.NotEmpty(k)
	fmt.Println(k)
}

func TestMarshalUnmarshalText(t *testing.T) {
	orig := Middle
	text, err := orig.MarshalText()
	if err != nil {
		t.Fatalf("marshal text failed: %v", err)
	}

	var out Key
	if err := out.UnmarshalText(text); err != nil {
		t.Fatalf("unmarshal text failed: %v", err)
	}

	if orig.Compare(out) != 0 {
		t.Errorf("expected %v, got %v", orig, out)
	}
}

func TestMarshalUnmarshalJSON(t *testing.T) {
	orig := Middle
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal json failed: %v", err)
	}

	var out Key
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal json failed: %v", err)
	}

	if orig.Compare(out) != 0 {
		t.Errorf("expected %v, got %v", orig, out)
	}
}

func TestSQLDriverValuer(t *testing.T) {
	orig := Middle
	val, err := orig.Value()
	if err != nil {
		t.Fatalf("value failed: %v", err)
	}
	strVal, ok := val.(string)
	if !ok {
		t.Fatalf("expected string, got %T", val)
	}
	if strVal != orig.String() {
		t.Errorf("expected %s, got %s", orig.String(), strVal)
	}
}

func TestSQLScanner(t *testing.T) {
	orig := Middle
	input := orig.String()

	var k Key
	if err := k.Scan(input); err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if orig.Compare(k) != 0 {
		t.Errorf("expected %v, got %v", orig, k)
	}

	// Test scanning from []byte
	if err := k.Scan([]byte(input)); err != nil {
		t.Fatalf("scan from bytes failed: %v", err)
	}
	if orig.Compare(k) != 0 {
		t.Errorf("expected %v, got %v", orig, k)
	}

	// Invalid type
	err := k.Scan(123)
	if err == nil {
		t.Fatal("expected error when scanning int, got nil")
	}
}

func TestBetween_OrderIndependent(t *testing.T) {
	a, _ := ParseKey("0|a")
	b, _ := ParseKey("0|z")

	forward, okA := a.Between(*b)
	if !okA {
		t.Fatal("Expected a.Between(b) to succeed")
	}

	backward, okB := b.Between(*a)
	if !okB {
		t.Fatal("Expected b.Between(a) to succeed")
	}

	if backward != nil && forward.String() != backward.String() {
		t.Errorf("Between should be symmetric, but got %s vs %s", forward.String(), backward.String())
	}
}
