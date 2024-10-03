package lexopt

// TODO: Split these tests into more manageable chunks.

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// import "github.com/jcbhmr/go-lexopt/prelude" import cycle
type Short = ArgShort
type Long = ArgLong
type Value = ArgValue

func parse(args string) *Parser {
	return ParserFromArgs(slices.Values(strings.Fields(args)))
}

func TestBasic(t *testing.T) {
	p := parse("-n 10 foo - -- baz -qux")
	t.Logf("p=%#+v", p)

	valuesIter, err := p.Values()
	if err != nil {
		t.Fatal(err)
	}
	values := slices.Collect(valuesIter.All)
	t.Log(values)
	return

	next, ok, err := p.Next()
	t.Logf("p=%#+v, next=%#+v, ok=%#+v, err=%#+v", p, next, ok, err)
	require.Nil(t, err)
	require.True(t, ok)
	require.Equal(t, (Arg)(&Short{'n'}), next)

	value, err := p.Value()
	t.Logf("p=%#+v, value=%#+v, err=%#+v", p, value, err)
	require.Nil(t, err)
	i64, err2 := strconv.ParseInt(value, 10, 32)
	require.Nil(t, err2)
	i := int32(i64)
	require.Equal(t, int32(10), i)

	next, ok, err = p.Next()
	t.Logf("p=%#+v, next=%#+v, ok=%#+v, err=%#+v", p, next, ok, err)
	require.Nil(t, err)
	require.True(t, ok)
	require.Equal(t, (Arg)(&Value{"foo"}), next)

	next, ok, err = p.Next()
	t.Logf("p=%#+v, next=%#+v, ok=%#+v, err=%#+v", p, next, ok, err)
	require.Nil(t, err)
	require.True(t, ok)
	require.Equal(t, (Arg)(&Value{"-"}), next)

	next, ok, err = p.Next()
	t.Logf("p=%#+v, next=%#+v, ok=%#+v, err=%#+v", p, next, ok, err)
	require.Nil(t, err)
	require.True(t, ok)
	require.Equal(t, (Arg)(&Value{"baz"}), next)

	next, ok, err = p.Next()
	t.Logf("p=%#+v, next=%#+v, ok=%#+v, err=%#+v", p, next, ok, err)
	require.Nil(t, err)
	require.True(t, ok)
	require.Equal(t, (Arg)(&Value{"-qux"}), next)

	next, ok, err = p.Next()
	t.Logf("p=%#+v, next=%#+v, ok=%#+v, err=%#+v", p, next, ok, err)
	require.Nil(t, err)
	require.False(t, ok)

	next, ok, err = p.Next()
	t.Logf("p=%#+v, next=%#+v, ok=%#+v, err=%#+v", p, next, ok, err)
	require.Nil(t, err)
	require.False(t, ok)

	next, ok, err = p.Next()
	t.Logf("p=%#+v, next=%#+v, ok=%#+v, err=%#+v", p, next, ok, err)
	require.Nil(t, err)
	require.False(t, ok)
}

func TestCombined(t *testing.T) {
	p := parse("-abc -fvalue -xfvalue")
	t.Logf("p=%#+v", p)

	valuesIter, err := p.Values()
	if err != nil {
		t.Fatal(err)
	}
	values := slices.Collect(valuesIter.All)
	t.Log(values)
	return

	next, ok, err := p.Next()
	t.Logf("p=%#+v, next=%#+v, ok=%#+v, err=%#+v", p, next, ok, err)
	require.Nil(t, err)
	require.True(t, ok)
	require.Equal(t, (Arg)(&Short{'a'}), next)

	next, ok, err = p.Next()
	t.Logf("p=%#+v, next=%#+v, ok=%#+v, err=%#+v", p, next, ok, err)
	require.Nil(t, err)
	require.True(t, ok)
	require.Equal(t, (Arg)(&Short{'b'}), next)

	next, ok, err = p.Next()
	t.Logf("p=%#+v, next=%#+v, ok=%#+v, err=%#+v", p, next, ok, err)
	require.Nil(t, err)
	require.True(t, ok)
	require.Equal(t, (Arg)(&Short{'c'}), next)

	next, ok, err = p.Next()
	t.Logf("p=%#+v, next=%#+v, ok=%#+v, err=%#+v", p, next, ok, err)
	require.Nil(t, err)
	require.True(t, ok)
	require.Equal(t, (Arg)(&Short{'f'}), next)

	value, err := p.Value()
	t.Logf("p=%#+v, value=%#+v, err=%#+v", p, value, err)
	require.Nil(t, err)
	require.Equal(t, "value", value)

	next, ok, err = p.Next()
	t.Logf("p=%#+v, next=%#+v, ok=%#+v, err=%#+v", p, next, ok, err)
	require.Nil(t, err)
	require.True(t, ok)
	require.Equal(t, (Arg)(&Short{'x'}), next)

	next, ok, err = p.Next()
	require.Nil(t, err)
	require.True(t, ok)
	require.Equal(t, (Arg)(&Short{'f'}), next)

	value, err = p.Value()
	t.Logf("p=%#+v, value=%#+v, err=%#+v", p, value, err)
	require.Nil(t, err)
	require.Equal(t, "value", value)

	next, ok, err = p.Next()
	require.Nil(t, err)
	require.False(t, ok)
}
