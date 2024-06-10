package lexopt

import (
	"errors"
	"unicode/utf8"
)

type iterSeq[V any] func(yield func(V) bool)
type iterSeq2[K, V any] func(yield func(K, V) bool)

func firstCodepoint(bytes []byte) (rune, bool, error) {
	if len(bytes) > 4 {
		bytes = bytes[:4]
	}
	r, i := utf8.DecodeRune(bytes)
	if r == utf8.RuneError && i == 0 {
		return 0, false, errors.New(string(bytes[0]))
	}
	return r, true, nil
}
