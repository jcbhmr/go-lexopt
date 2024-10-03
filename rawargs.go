package lexopt

import "iter"

type RawArgs struct {
	a *struct {slice []string; index int}
}

var _ iter.Seq[string] = (*RawArgs)(nil).All

func (r *RawArgs) Next() (string, bool) {
	if len(r.a.slice) - r.a.index == 0 {
		return "", false
	}
	v := r.a.slice[r.a.index]
	r.a.index++
	return v, true
}

func (r *RawArgs) All(yield func(string) bool) {
	for {
		value, ok := r.Next()
		if !ok {
			break
		}
		if !yield(value) {
			break
		}
	}
}

func (r *RawArgs) Peek() (string, bool) {
	if len(r.a.slice) - r.a.index == 0 {
		return "", false
	}
	return r.a.slice[r.a.index], true
}

func (r *RawArgs) NextIf(func_ func (string) bool) (string, bool) {
	if arg, ok := r.Peek(); ok && func_(arg) {
		return r.Next()
	} else {
		return "", false
	}
}

func (r *RawArgs) AsSlice() []string {
	return append(r.a.slice[:0:0], r.a.slice[r.a.index:]...)
}