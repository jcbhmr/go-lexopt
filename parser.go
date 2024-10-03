package lexopt

import (
	"bytes"
	"errors"
	"fmt"
	"iter"
	"os"
	"strings"
	"unicode/utf8"
)

// A parser for command line arguments.
type Parser struct {
	source      iter.Seq[string]
	state       state
	// The last option we emitted.
	lastOption  lastOption
	// The name of the command (argv[0]).
	binName     *string
}

type state interface {
	isState()
}
// Nothing interesting is going on.
type stateNone struct{}
// We have a value left over from --option=value.
type statePendingValue struct {
	A string
}
// We're in the middle of -abc.
type stateShorts struct {
	A []byte
	B uint
}
// We saw -- and know no more options are coming.
type stateFinishedOpts struct{}

var _ state = (*stateNone)(nil)
var _ state = (*statePendingValue)(nil)
var _ state = (*stateShorts)(nil)
var _ state = (*stateFinishedOpts)(nil)

func (stateNone) isState()         {}
func (statePendingValue) isState() {}
func (stateShorts) isState()       {}
func (stateFinishedOpts) isState() {}

// We use this to keep track of the last emitted option, for error messages when
// an unexpected value is not found.
type lastOption interface {
	isLastOption()
}
type lastOptionNone struct{}
type lastOptionShort struct {
	A rune
}
type lastOptionLong struct {
	A string
}

var _ lastOption = (*lastOptionNone)(nil)
var _ lastOption = (*lastOptionShort)(nil)
var _ lastOption = (*lastOptionLong)(nil)

func (lastOptionNone) isLastOption()  {}
func (lastOptionShort) isLastOption() {}
func (lastOptionLong) isLastOption()  {}

// Get the next option or positional argument.
//
// A return value of (nil, false, nil) means there are no more arguments.
//
// # Errors
//
// ErrorUnexpectedValue is returned if the last option had a
// value that hasn't been consumed, as in --option=value or -o=value.
//
// It's possible to continue parsing after this error (but this is rarely useful).
func (p *Parser) Next() (Arg, bool, Error) {
	if v1, ok := p.state.(statePendingValue); ok {
		value := v1.A
		// Last time we got --long=value, and value hasn't been used.
		p.state = stateNone{}
		option, ok := p.formatLastOption()
		if !ok {
			panic("Should only have pending value after long option")
		}
		return nil, false, &ErrorUnexpectedValue{
			Option: option,
			Value: value,
		}
	} else if v2, ok := p.state.(stateShorts); ok {
		arg := v2.A
		pos := &v2.B
		// We're somewhere inside a -abc chain. Because we're in .next(),
		// not .value(), we can assume that the next character is another option.
		fcValue, fcOk, fcErr := firstCodepoint(arg[*pos:])
		if fcErr == nil && !fcOk {
			p.state = stateNone{}
		} else if *pos > 1 && fcErr == nil && fcOk && fcValue == '=' {
			// If we find "-=[...]" we interpret is as an option.
			// If we find "-o=..." then there's an unexpected value.
			// ('-=' as an option exists, see https://linux.die.net/man/1/a2ps.)
			// clap always interprets it as a short flag in this case, but
			// that feels sloppy.
			option, ok := p.formatLastOption()
			if !ok {
				panic("unreachable")
			}
			value, ok := p.OptionalValue()
			if !ok {
				panic("unreachable")
			}
			return nil, false, &ErrorUnexpectedValue{
				Option: option,
				Value: value,
			}
		} else if fcErr == nil && fcOk {
			*pos += uint(utf8.RuneLen(fcValue))
			p.lastOption = lastOptionShort{fcValue}
			return &ArgShort{fcValue}, true, nil
		} else if fcErr != nil {
			// Advancing may allow recovery.
			// This is a little iffy, there might be more bad unicode next.
			
		} else {
			panic("unreachable")
		}
	}

	switch v := p.state[0].(type) {
	case statePendingValue:
		value := string(v)
		p.state = state{stateNone{}}
		option, ok := p.formatLastOption()
		if !ok {
			panic("Should only have pending value after long option")
		}
		return Arg{}, false, Error{ErrorUnexpectedValue{Option: option, Value: value}}
	case stateShorts:
		arg := v.A
		pos := v.B
		fc, ok, err := firstCodepoint(arg[pos:])
		if err == nil {
			if !ok {
				p.state = state{stateNone{}}
			} else if fc == '=' && pos > 1 {
				option, ok := p.formatLastOption()
				if !ok {
					panic("unreachable")
				}
				value, ok := p.OptionalValue()
				if ok {
					panic("unreachable")
				}
				return Arg{}, false, Error{ErrorUnexpectedValue{Option: option, Value: value}}
			} else {
				v.B += utf8.RuneLen(fc)
				p.lastOption = lastOption{lastOptionShort(fc)}
				return Arg{ArgShort(fc)}, true, nil
			}
		} else {
			v.B += 1
			p.lastOption = lastOption{lastOptionShort('\uFFFD')}
			return Arg{ArgShort('\uFFFD')}, true, nil
		}
	case stateFinishedOpts:
		var x string
		var ok bool
		if p.sourceIndex < len(p.source) {
			x = p.source[p.sourceIndex]
			p.sourceIndex++
			ok = true
		} else {
			x = ""
			ok = false
		}
		if ok {
			return Arg{ArgValue(x)}, true, nil
		} else {
			return Arg{}, false, nil
		}
	case stateNone:
	default:
		panic("unreachable")
	}

	switch p.state[0].(type) {
	case stateNone:
	default:
		panic(fmt.Errorf("unepxected state %#+v", p.state))
	}

	var arg string
	var ok bool
	if p.sourceIndex < len(p.source) {
		arg = p.source[p.sourceIndex]
		p.sourceIndex++
		ok = true
	} else {
		arg = ""
		ok = false
	}
	if !ok {
		return Arg{}, false, nil
	}

	if arg == "--" {
		p.state = state{stateFinishedOpts{}}
		return p.Next()
	}

	arg2 := []byte(arg)
	if bytes.HasPrefix(arg2, []byte("--")) {
		ind := bytes.IndexByte(arg2, '=')
		if ind != -1 {
			p.state = state{statePendingValue(string(arg2[ind+1:]))}
			arg2 = arg2[:ind]
		}
		option := strings.ToValidUTF8(string(arg2), "\uFFFD")
		return p.setLong(option), true, nil
	} else if strings.HasPrefix(arg, "-") {
		p.state = state{stateShorts{[]byte(arg), 1}}
		return p.Next()
	} else {
		return Arg{ArgValue(arg)}, true, nil
	}
}

func (p *Parser) Value() (string, error) {
	optionalValue, ok := p.OptionalValue()
	if ok {
		return optionalValue, nil
	}

	var x string
	if p.sourceIndex < len(p.source) {
		x = p.source[p.sourceIndex]
		p.sourceIndex++
		ok = true
	} else {
		x = ""
		ok = false
	}
	if ok {
		return x, nil
	}

	option, ok := p.formatLastOption()
	var optionPtr *string
	if ok {
		optionPtr = &option
	} else {
		optionPtr = nil
	}
	return "", Error{ErrorMissingValue{Option: optionPtr}}
}

func (p *Parser) Values() (ValuesIter, error) {
	if p.hasPending() || p.nextIsNormal() {
		return newValuesIter(false, p), nil
	} else {
		option, ok := p.formatLastOption()
		var optionPtr *string
		if ok {
			optionPtr = &option
		} else {
			optionPtr = nil
		}
		return nil, Error{ErrorMissingValue{Option: optionPtr}}
	}
}

func (p *Parser) nextIfNormal() (string, bool) {
	if p.nextIsNormal() {
		if p.sourceIndex < len(p.source) {
			a := p.source[p.sourceIndex]
			p.sourceIndex++
			return a, true
		} else {
			return "", false
		}
	} else {
		return "", false
	}
}

func (p *Parser) nextIsNormal() bool {
	if p.hasPending() {
		panic("assertion failed")
	}
	var arg string
	if p.sourceIndex < len(p.source) {
		arg = p.source[p.sourceIndex]
	} else {
		return false
	}
	if arg == "--" {
		return false
	}
	leadDash := strings.HasPrefix(arg, "-")
	return !leadDash
}

func (p *Parser) RawArgs() (RawArgs, error) {
	value, ok := p.OptionalValue()
	if ok {
		option, ok := p.formatLastOption()
		if !ok {
			panic("unreachable")
		}
		return RawArgs{}, Error{ErrorUnexpectedValue{Option: option, Value: value}}
	}
	return RawArgs{slice: p.source[p.sourceIndex:]}, nil
}

func (p *Parser) TryRawArgs() (RawArgs, bool) {
	if p.hasPending() {
		return RawArgs{}, false
	} else {
		return RawArgs{slice: p.source[p.sourceIndex:]}, true
	}
}

func (p Parser) hasPending() bool {
	switch v := p.state[0].(type) {
	case stateNone:
		return false
	case stateFinishedOpts:
		return false
	case statePendingValue:
		return true
	case stateShorts:
		arg := v.A
		pos := v.B
		return pos < len(arg)
	default:
		panic("unreachable")
	}
}

func (p Parser) formatLastOption() (string, bool) {
	switch v := p.lastOption[0].(type) {
	case lastOptionNone:
		return "", false
	case lastOptionShort:
		return "-" + string(v), true
	case lastOptionLong:
		return string(v), true
	default:
		panic("unreachable")
	}
}

func (p Parser) BinName() (string, bool) {
	return p.binName.Get()
}

func (p Parser) OptionalValue() (string, bool) {
	raw, _, ok := p.rawOptionalValue()
	return raw, ok
}

func (p Parser) rawOptionalValue() (string, bool, bool) {
	oldState := p.state
	p.state = state{stateNone{}}
	switch v := oldState[0].(type) {
	case statePendingValue:
		return string(v), true, true
	case stateShorts:
		arg := v.A
		pos := v.B
		if pos >= len(arg) {
			return "", false, false
		}
		hadEqSign := false
		if arg[pos] == '=' {
			pos += 1
			hadEqSign = true
		}
		arg = arg[pos:]
		return string(arg), hadEqSign, true
	case stateFinishedOpts:
		p.state = state{stateFinishedOpts{}}
		return "", false, false
	case stateNone:
		return "", false, false
	default:
		panic("unreachable")
	}
}

func newParser(binName *string, source []string) Parser {
	return Parser{
		source:     source,
		state:      state{stateNone{}},
		lastOption: lastOption{lastOptionNone{}},
		binName:    mo.PointerToOption(binName),
	}
}

func ParserFromEnv() Parser {
	var binName *string
	var source []string
	if len(os.Args) > 0 {
		binName = &os.Args[0]
		source = os.Args[1:]
	} else {
		s := ""
		binName = &s
		source = os.Args
	}
	return newParser(binName, source)
}

func ParserFromIter(args []string) Parser {
	var binName *string
	var source []string
	if len(args) > 0 {
		binName = &args[0]
		source = args[1:]
	} else {
		s := ""
		binName = &s
		source = args
	}
	return newParser(binName, source)
}

func ParserFromArgs(args []string) Parser {
	return newParser(nil, args)
}

func (p *Parser) setLong(option string) Arg {
	p.lastOption = lastOption{lastOptionLong(option)}
	return Arg{ArgLong(option[2:])}
}


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