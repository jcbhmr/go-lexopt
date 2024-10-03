package lexopt

import "iter"

type RawArgs struct {
	slice []string
	index int
}

func (a RawArgs) Iter() iter.Seq[string] {
	return func(yield func(string) bool) {
		for i := a.index; i < len(a.slice); i++ {
			if !yield(a.slice[i]) {
				return
			}
		}
	}
}

func (a RawArgs) Peek() (string, bool) {
	if a.index < len(a.slice) {
		return a.slice[a.index], true
	}
	return "", false
}

func (a RawArgs) NextIf(f func(string) bool) (string, bool) {
	v, ok := a.Peek()
	if ok && f(v) {
		a.index++
		return v, true
	}
	return "", false
}

func (a RawArgs) AsSlice() []string {
	return a.slice[a.index:]
}
