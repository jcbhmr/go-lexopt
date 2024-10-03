package lexopt

import "iter"

type ValuesIter struct {
	tookFirst bool
	parser    *Parser
}

var _ iter.Seq[string] = (*ValuesIter)(nil).All

func (v *ValuesIter) Next() (string, bool) {
	parser := v.parser
	if v.tookFirst {
		return parser.nextIfNormal()
	} else if value, hadEqSign, ok := parser.rawOptionalValue(); ok {
		if hadEqSign {
			v.parser = nil
		}
		v.tookFirst = true
		return value, true
	} else {
		value, ok = parser.nextIfNormal()
		if !ok {
			panic("ValuesIter must yield at least one value")
		}
		v.tookFirst = true
		return value, true
	}
}

func (v *ValuesIter) All(yield func(string) bool) {
	for {
		value, ok := v.Next()
		if !ok {
			break
		}
		if !yield(value) {
			break
		}
	}
}
