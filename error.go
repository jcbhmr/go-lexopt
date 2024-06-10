package lexopt

import "fmt"

type Error [1]interface {
	isError()
}
type ErrorMissingValue struct {
	Option *string
}
type ErrorUnexpectedOption string
type ErrorUnexpectedArgument string
type ErrorUnexpectedValue struct {
	Option string
	Value  string
}
type ErrorParsingFailed struct {
	Value string
	Error error
}
type ErrorNonUnicodeValue string
type ErrorCustom [1]error

func (ErrorMissingValue) isError()       {}
func (ErrorUnexpectedOption) isError()   {}
func (ErrorUnexpectedArgument) isError() {}
func (ErrorUnexpectedValue) isError()    {}
func (ErrorParsingFailed) isError()      {}
func (ErrorNonUnicodeValue) isError()    {}
func (ErrorCustom) isError()             {}

func (e Error) Error() string {
	return e.String()
}

func (e Error) String() string {
	switch v := e[0].(type) {
	case ErrorMissingValue:
		if v.Option == nil {
			return "missing argument"
		} else {
			return fmt.Sprintf("missing value for option '%v'", *v.Option)
		}
	case ErrorUnexpectedOption:
		return fmt.Sprintf("invalid option '%v'", v)
	case ErrorUnexpectedArgument:
		return fmt.Sprintf("unexpected argument %#+v", v)
	case ErrorUnexpectedValue:
		return fmt.Sprintf("unexpected argument for option '%v': %#+v", v.Option, v.Value)
	case ErrorNonUnicodeValue:
		return fmt.Sprintf("argument is invalid unicode: %#+v", v)
	case ErrorParsingFailed:
		return fmt.Sprintf("cannot parse argument %#+v: %v", v.Value, v.Error)
	case ErrorCustom:
		return v[0].Error()
	default:
		panic("unreachable")
	}
}

func (e Error) GoString() string {
	return e.String()
}
