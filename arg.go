package lexopt

type Arg [1]interface {
	isArg()
}
type ArgShort rune
type ArgLong string
type ArgValue string

func (ArgShort) isArg() {}
func (ArgLong) isArg()  {}
func (ArgValue) isArg() {}

func (a Arg) Unexpected() error {
	switch a := a[0].(type) {
	case ArgShort:
		return Error{ErrorUnexpectedOption(a)}
	case ArgLong:
		return Error{ErrorUnexpectedOption(a)}
	case ArgValue:
		return Error{ErrorUnexpectedArgument(a)}
	default:
		panic("unreachable")
	}
}
