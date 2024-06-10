package lexopt

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/samber/mo"
)

type Parser struct {
	source      []string
	sourceIndex int
	state       state
	lastOption  lastOption
	binName     mo.Option[string]
}

type state [1]interface {
	isState()
}
type stateNone struct{}
type statePendingValue string
type stateShorts struct {
	A []byte
	B int
}
type stateFinishedOpts struct{}

func (stateNone) isState()         {}
func (statePendingValue) isState() {}
func (stateShorts) isState()       {}
func (stateFinishedOpts) isState() {}

type lastOption [1]interface {
	isLastOption()
}
type lastOptionNone struct{}
type lastOptionShort rune
type lastOptionLong string

func (lastOptionNone) isLastOption()  {}
func (lastOptionShort) isLastOption() {}
func (lastOptionLong) isLastOption()  {}

func (p *Parser) Next() (Arg, bool, error) {
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
