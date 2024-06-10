package lexopt

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func parse(args string) Parser {
	return ParserFromArgs(strings.Fields(args))
}

func TestBasic(t *testing.T) {
	p := parse("-n 1- foo - -- baz -qux")

	next, ok, err := p.Next()
	require.Nil(t, err)
	require.True(t, ok)
	require.Equal(t, Arg{ArgShort('n')}, next)

	value, err := p.Value()
	require.Nil(t, err)
	i64, err := strconv.ParseInt(value, 10, 32)
	require.Nil(t, err)
	i := int(i64)
	require.Equal(t, 10, i)

	next, ok, err = p.Next()
	require.Nil(t, err)
	require.True(t, ok)
	require.Equal(t, Arg{ArgValue("foo")}, next)

	next, ok, err = p.Next()
	require.Nil(t, err)
	require.True(t, ok)
	require.Equal(t, Arg{ArgValue("-")}, next)

	next, ok, err = p.Next()
	require.Nil(t, err)
	require.True(t, ok)
	require.Equal(t, Arg{ArgValue("baz")}, next)

	next, ok, err = p.Next()
	require.Nil(t, err)
	require.True(t, ok)
	require.Equal(t, Arg{ArgValue("-qux")}, next)

	next, ok, err = p.Next()
	require.Nil(t, err)
	require.False(t, ok)

	next, ok, err = p.Next()
	require.Nil(t, err)
	require.False(t, ok)

	next, ok, err = p.Next()
	require.Nil(t, err)
	require.False(t, ok)
}
