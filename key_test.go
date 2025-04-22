package lexorank

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKey_Defaults(t *testing.T) {
	fmt.Println(Bottom)
	fmt.Println(Top)
	fmt.Println(Middle)
	fmt.Println(NormaliseTop)
	fmt.Println(NormaliseBottom)
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
	r.Equal("1|aaaaaaU", got.String())
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
	r.Equal("1|0", got.String())
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
