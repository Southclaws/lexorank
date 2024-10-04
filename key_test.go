package lexorank

import (
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
