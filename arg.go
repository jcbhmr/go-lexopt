package lexopt

type Arg interface {
	isArg()
	Unexpected() Error
}
type ArgShort struct {
	A rune
}
type ArgLong struct {
	A string
}
type ArgValue struct {
	A string
}

var _ Arg = (*ArgShort)(nil)
var _ Arg = (*ArgLong)(nil)
var _ Arg = (*ArgValue)(nil)

func (ArgShort) isArg() {}
func (ArgLong) isArg()  {}
func (ArgValue) isArg() {}

func (a ArgShort) Unexpected() Error {
	return &ErrorUnexpectedOption{string(a.A)}
}
func (a ArgLong) Unexpected() Error {
	return &ErrorUnexpectedOption{a.A}
}
func (a ArgValue) Unexpected() Error {
	return &ErrorUnexpectedArgument{a.A}
}
