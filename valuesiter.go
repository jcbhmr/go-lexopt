package lexopt

type ValuesIter iterSeq[string]

func newValuesIter(tookFirst bool, parser *Parser) ValuesIter {
	return func(yield func(string) bool) {
		for {
			if parser == nil {
				return
			}
			if tookFirst {
				v, ok := parser.nextIfNormal()
				if !ok {
					return
				}
				if !yield(v) {
					return
				}
			} else if value, hadEqSign, ok := parser.rawOptionalValue(); ok {
				if hadEqSign {
					parser = nil
				}
				tookFirst = true
				if !yield(value) {
					return
				}
			} else {
				value, ok := parser.nextIfNormal()
				if !ok {
					panic("ValuesIter must yield at least one value")
				}
				tookFirst = true
				if !yield(value) {
					return
				}
			}
		}
	}
}