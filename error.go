package lexopt

import (
	"fmt"
)

type Error interface {
	isError()
	fmt.Stringer
	fmt.GoStringer
	error
	Unwrap() error
}
type ErrorMissingValue struct {
	Option *string
}
type ErrorUnexpectedOption struct {
	A string
}
type ErrorUnexpectedArgument struct {
	A string
}
type ErrorUnexpectedValue struct {
	Option string
	Value  string
}
type ErrorParsingFailed struct {
	Value string
	Error2 error
}
type ErrorNonUnicodeValue struct {
	A string
}
type ErrorCustom struct {
	A error
}

var _ Error = (*ErrorMissingValue)(nil)
var _ Error = (*ErrorUnexpectedOption)(nil)
var _ Error = (*ErrorUnexpectedArgument)(nil)
var _ Error = (*ErrorUnexpectedValue)(nil)
var _ Error = (*ErrorParsingFailed)(nil)
var _ Error = (*ErrorNonUnicodeValue)(nil)
var _ Error = (*ErrorCustom)(nil)

func (ErrorMissingValue) isError()       {}
func (ErrorUnexpectedOption) isError()   {}
func (ErrorUnexpectedArgument) isError() {}
func (ErrorUnexpectedValue) isError()    {}
func (ErrorParsingFailed) isError()      {}
func (ErrorNonUnicodeValue) isError()    {}
func (ErrorCustom) isError()             {}

func (e *ErrorMissingValue) String() string {
	if e.Option == nil {
		return "missing argument"
	} else {
		return fmt.Sprintf("missing value for option '%v'", *e.Option)
	}
}
func (e *ErrorUnexpectedOption) String() string {
	return fmt.Sprintf("invalid option '%v'", e.A)
}
func (e *ErrorUnexpectedArgument) String() string {
	return fmt.Sprintf("unexpected argument %#+v", e.A)
}
func (e *ErrorUnexpectedValue) String() string {
	return fmt.Sprintf("unexpected argument for option '%v': %#+v", e.Option, e.Value)
}
func (e *ErrorNonUnicodeValue) String() string {
	return fmt.Sprintf("argument is invalid unicode: %#+v", e.A)
}
func (e *ErrorParsingFailed) String() string {
	return fmt.Sprintf("cannot parse argument %#+v: %v", e.Value, e.Error2)
}
func (e *ErrorCustom) String() string {
	return fmt.Sprint(e.A)
}

func (e *ErrorMissingValue) GoString() string {
	return e.String()
}
func (e *ErrorUnexpectedOption) GoString() string {
	return e.String()
}
func (e *ErrorUnexpectedArgument) GoString() string {
	return e.String()
}
func (e *ErrorUnexpectedValue) GoString() string {
	return e.String()
}
func (e *ErrorNonUnicodeValue) GoString() string {
	return e.String()
}
func (e *ErrorParsingFailed) GoString() string {
	return e.String()
}
func (e *ErrorCustom) GoString() string {
	return e.String()
}

func (e *ErrorMissingValue) Error() string {
	return e.String()
}
func (e *ErrorUnexpectedOption) Error() string {
	return e.String()
}
func (e *ErrorUnexpectedArgument) Error() string {
	return e.String()
}
func (e *ErrorUnexpectedValue) Error() string {
	return e.String()
}
func (e *ErrorNonUnicodeValue) Error() string {
	return e.String()
}
func (e *ErrorParsingFailed) Error() string {
	return e.String()
}
func (e *ErrorCustom) Error() string {
	return e.String()
}

func (e *ErrorMissingValue) Unwrap() error {
	return nil
}
func (e *ErrorUnexpectedOption) Unwrap() error {
	return nil
}
func (e *ErrorUnexpectedArgument) Unwrap() error {
	return nil
}
func (e *ErrorUnexpectedValue) Unwrap() error {
	return nil
}
func (e *ErrorNonUnicodeValue) Unwrap() error {
	return nil
}
func (e *ErrorParsingFailed) Unwrap() error {
	return e.Error2
}
func (e *ErrorCustom) Unwrap() error {
	return e.A
}