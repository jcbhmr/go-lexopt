package lexopt

import (
	"bytes"
	"fmt"
	"iter"
	"os"
	"slices"
	"strings"
	"unicode/utf8"
)

// A parser for command line arguments.
type Parser struct {
	source struct {
		slice []string
		index int
	}
	state state
	// The last option we emitted.
	lastOption lastOption
	// The name of the command (argv[0]).
	binName *string
}

type state interface {
	isState()
}

// Nothing interesting is going on.
type stateNone struct{}

// We have a value left over from --option=value.
type statePendingValue struct {
	a string
}

// We're in the middle of -abc.
type stateShorts struct {
	a []byte
	b uint
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
		value := v1.a
		// Last time we got --long=value, and value hasn't been used.
		p.state = stateNone{}
		option, ok := p.formatLastOption()
		if !ok {
			panic("Should only have pending value after long option")
		}
		return nil, false, &ErrorUnexpectedValue{
			Option: option,
			Value:  value,
		}
	} else if v2, ok := p.state.(stateShorts); ok {
		arg := v2.a
		pos := v2.b
		// We're somewhere inside a -abc chain. Because we're in .next(),
		// not .value(), we can assume that the next character is another option.
		fcValue, fcOk, fcErr := firstCodepoint(arg[pos:])
		if fcErr == nil && !fcOk {
			p.state = stateNone{}
		} else if pos > 1 && fcErr == nil && fcOk && fcValue == '=' {
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
				Value:  value,
			}
		} else if fcErr == nil && fcOk {
			pos += uint(utf8.RuneLen(fcValue))
			p.state = stateShorts{arg, pos}
			p.lastOption = lastOptionShort{fcValue}
			return &ArgShort{fcValue}, true, nil
		} else if fcErr != nil {
			// Advancing may allow recovery.
			// This is a little iffy, there might be more bad unicode next.
			pos = uint(len(arg))
			p.state = stateShorts{arg, pos}
			p.lastOption = lastOptionShort{'\uFFFD'}
			return &ArgShort{'\uFFFD'}, true, nil
		} else {
			panic("unreachable")
		}
	} else if _, ok := p.state.(stateFinishedOpts); ok {
		if len(p.source.slice)-p.source.index > 0 {
			v := p.source.slice[p.source.index]
			p.source.index++
			return &ArgValue{v}, true, nil
		} else {
			return nil, false, nil
		}
	} else if _, ok := p.state.(stateNone); ok {
	} else {
		panic("unreachable")
	}

	if _, ok := p.state.(stateNone); ok {
	} else {
		panic(fmt.Errorf("unepxected state %#+v", p.state))
	}

	var arg2 string
	{
		if len(p.source.slice)-p.source.index > 0 {
			v := p.source.slice[p.source.index]
			p.source.index++
			arg2 = v
		} else {
			return nil, false, nil
		}
	}

	if arg2 == "--" {
		p.state = stateFinishedOpts{}
		return p.Next()
	}

	// Fast solution for platforms where strings are just UTF-8-ish bytes.
	arg3 := []byte(arg2)
	if bytes.HasPrefix(arg3, []byte("--")) {
		// Long options have two forms: --option and --option=value.
		if ind := bytes.IndexByte(arg3, '='); ind != -1 {
			// The value can be a non-UTF-8 string.
			p.state = statePendingValue{string(arg3[ind+1:])}
			arg3 = arg3[:ind]
		}
		// ...but the options has to be a string.
		option := strings.ToValidUTF8(string(arg3), "\uFFFD")
		return p.setLong(option), true, nil
	} else if len(arg3) > 1 && arg3[0] == '-' {
		p.state = stateShorts{arg3, 1}
		return p.Next()
	} else {
		return &ArgValue{string(arg3)}, true, nil
	}
}

// Get a value for an option.
//
// This function should normally be called right after seeing an option
// that expects a value, with positional arguments being collected
// using parser.Next().
//
// A value is collected even if it looks like an option
// (i.e., starts with -).
//
// # Errors
//
// An ErrorMissingValue is returned if the end of the command
// line is reached.
func (p *Parser) Value() (string, Error) {
	if value, ok := p.OptionalValue(); ok {
		return value, nil
	}

	if len(p.source.slice)-p.source.index > 0 {
		value := p.source.slice[p.source.index]
		p.source.index++
		return value, nil
	}

	option, ok := p.formatLastOption()
	var optionPtr *string
	if ok {
		optionPtr = &option
	}
	return "", &ErrorMissingValue{
		Option: optionPtr,
	}
}

// Gather multiple values for an option.
//
// This is used for options that take multiple arguments, such as a
// --command flag that's invoked as app --command echo 'Hello world'.
//
// It will gather arguments until another option is found, or -- is found, or
// the end of the command line is reached. This differs from .Value(), which
// takes a value even if it looks like an option.
//
// An equals sign (=) will limit this to a single value. That means -a=b c and
// --opt=b c will only yield "b" while -a b c and --opt b c will
// yield "b" and "c".
//
// # Errors
// If not at least one value is found then ErrorMissingValue is returned.
//
// # Example
//
//	parser := lexopt.ParserFromArgs([]string{"a", "b", "-x", "one", "two", "three", "four"})
//	argumentsIter, err := parser.Values()
//	if err != nil {
//	    return err
//	}
//	arguments := slices.Collect(argumentsIter)
//	require.Equal(t, []string{"a", "b"}, arguments)
//	parser.Next()
//	valuesIter, err := parser.Values()
//	if err != nil {
//	    return err
//	}
//	next, _ := iter.Pull(valuesIter)
//	atMostThreeFiles := []string{}
//	for i := 0; i < 3; i++ {
//	    value, ok := next()
//	    if !ok {
//	        break
//	    }
//	    atMostThreeFiles = append(atMostThreeFiles, value)
//	}
//	rawArgsIter, err := parser.RawArgs()
//	if err != nil {
//	    return err
//	}
//	rawArgs := slices.Collect(rawArgsIter)
//	require.Equal(t, rawArgs, []string{"four"})
//	valuesIter, err = parser.Values()
//	if err != nil {
//	    return err
//	}
//	for v := range valuesIter {
//	    // ...
//	}
func (p *Parser) Values() (*ValuesIter, Error) {
	// This code is designed so that just calling .Values() doesn't consume
	// any arguments as long as you don't use the iterator. It used to work
	// differently.
	// "--" is treated like an option and not consumed. This seems to me the
	// least unreasonable behavior, and it's the easiest to implement.
	if p.hasPending() || p.nextIsNormal() {
		return &ValuesIter{
			tookFirst: false,
			parser:    p,
		}, nil
	} else {
		option, ok := p.formatLastOption()
		var optionPtr *string
		if ok {
			optionPtr = &option
		}
		return nil, &ErrorMissingValue{
			Option: optionPtr,
		}
	}
}

// Inspect an argument and consume it if it's "normal" (not an option or --).
//
// Used by parser.Values().
//
// This method should not be called while partway through processing an
// argument.
func (p *Parser) nextIfNormal() (string, bool) {
	if p.nextIsNormal() {
		if len(p.source.slice)-p.source.index > 0 {
			value := p.source.slice[p.source.index]
			p.source.index++
			return value, true
		} else {
			return "", false
		}
	} else {
		return "", false
	}
}

// Execute the check for nextIfNormal().
func (p *Parser) nextIsNormal() bool {
	if p.hasPending() {
		panic("assertion failed")
	}
	var arg string
	{
		if len(p.source.slice)-p.source.index == 0 {
			// There has to be a next argument.
			return false
		} else {
			arg = p.source.slice[p.source.index]
		}
	}
	if _, ok := p.state.(stateFinishedOpts); ok {
		// If we already found a -- then we're really not supposed to be here,
		// but we shouldn't treat the next argument as an option.
		return true
	}
	if arg == "-" {
		// "-" is the one argument with a leading '-' that's allowed.
		return true
	}
	leadDash := len(arg) > 0 && arg[0] == '-'
	return !leadDash
}

// Take raw arguments from the original command line.
//
// This returns an iterator of strings. Any arguments that are not
// consumed are kept, so you can continue parsing after you're done with
// the iterator.
//
// TODO: To inspect an argument without consuming it, use rawArgs.Peek() or
// rawArgs.AsSlice().
//
// # Errors
//
// Returns an ErrorUnexpectedValue if the last option had a left-over
// argument, as in --option=value, -ovalue, or if it was midway through
// an option chan, as in -abc. The iterator only yields whole arguments.
// To avoid this, use TryRawArgs().
//
// After this error the method is guarenteed to succeed, as it consumes the
// rest of the argument.
//
// # Example
// As soon as a free-standing argument is found, consume the other arguments
// as-is, and build them into a command.
//
//	parser := lexopt.ParserFromArgs([]string{"-x", "echo", "-n", "'Hello world'"})
//	for {
//	    arg, ok, err := parser.Next()
//	    if err != nil {
//	        return err
//	    }
//	    if !ok {
//	        break
//	    }
//	    if prog, ok := arg.(Value); ok {
//	        rawArgsIter, err := parser.RawArgs()
//	        if err != nil {
//	            return err
//	        }
//	        args := slices.Collect(rawArgsIter)
//	        command := exec.Command(prog, args...)
//	    }
//	}
func (p *Parser) RawArgs() (*RawArgs, Error) {
	if value, ok := p.OptionalValue(); ok {
		option, ok := p.formatLastOption()
		if !ok {
			panic("unreachable")
		}
		return nil, &ErrorUnexpectedValue{
			Option: option,
			Value:  value,
		}
	}

	return &RawArgs{&p.source}, nil
}

// Take raw arguments from the original command line, *if* the current argument
// has finished processing.
//
// Unlike .RawArgs() this does not consume any value
// in case of a left-over argument. This makes it safe to call at any time.
//
// It returns (nil, false) exactly when .OptionalValue()
// would return (T, true).
//
// Note: If no arguments are left then it returns an empty iterator (not (nil, false)).
//
// # Example
// Process arguments of the form -123 as numbers. For a complete runnable version of
// this example, see example_nonstandard_test.go.
//
//	parser := lexopt.ParserFromArgs([]string{"-13"})
//	parseDashnum := func (parser *lexopt.Parser) (uint64, bool) {
//	    raw, ok := parser.TryRawArgs()
//	    if !ok {
//	        return 0, false
//	    }
//	    arg, ok := raw.Peek()
//	    if !ok {
//	        return 0, false
//	    }
//	    num, err := strconv.ParseUint(strings.TrimLeft(arg, "-"), 10, 64)
//	    if err != nil {
//	        return 0, false
//	    }
//	    raw.Next()
//	    return num, true
//	}
//
//	for {
//	    if num, ok := parseDashnum(parser); ok {
//	        fmt.Printf("Got number %v\n", num)
//	    } else {
//	        arg, ok, err := parser.Next()
//	        if err != nil {
//	            log.Fatal(err)
//	        }
//	        if ok {
//	            // ...
//	        } else {
//	            break
//	        }
//	    }
//	}
func (p *Parser) TryRawArgs() (*RawArgs, bool) {
	if p.hasPending() {
		return nil, false
	} else {
		return &RawArgs{&p.source}, true
	}
}

// Check whether we're halfway through an argument, or in other words,
// if parse.OptionalValue() would return (T, true).
func (p *Parser) hasPending() bool {
	_, noneOk := p.state.(stateNone)
	_, finishedOpsOk := p.state.(stateFinishedOpts)
	if noneOk || finishedOpsOk {
		return false
	} else if _, ok := p.state.(statePendingValue); ok {
		return true
	} else if shorts, ok := p.state.(stateShorts); ok {
		return shorts.b < uint(len(shorts.a))
	} else {
		panic("unreachable")
	}
}

func (p *Parser) formatLastOption() (string, bool) {
	if _, ok := p.lastOption.(lastOptionNone); ok {
		return "", false
	} else if short, ok := p.lastOption.(lastOptionShort); ok {
		return fmt.Sprintf("-%c", short.A), true
	} else if long, ok := p.lastOption.(lastOptionLong); ok {
		return long.A, true
	} else {
		panic("unreachable")
	}
}

// The name of the command, as in the zeroth argument of the process.
//
// This is intended for use in messages. If the name is not valid unicode
// it will be sanitized with replacement characters as by strings.ToValidUTF8.
//
// To get the current executable, use os.Executable.
//
// # Example
//
//	parser := lexopt.ParserFromEnv()
//	binName, ok := parser.BinName()
//	if !ok {
//	    binName = "myapp"
//	}
//	fmt.Printf("%v: Some message", binName)
func (p *Parser) BinName() (string, bool) {
	if p.binName == nil {
		return "", false
	}
	return strings.ToValidUTF8(*p.binName, "\uFFFD"), true
}

// Get a value only if it's concatenated to an option, as in -ovalue or
// --option=value or -o=value, but not -o value or --option value.
func (p *Parser) OptionalValue() (string, bool) {
	raw, _, ok := p.rawOptionalValue()
	if !ok {
		return "", false
	}
	return raw, true
}

// parser.OptionalValue(), but indicate whether the value was joined
// with an = sign. This matters for parser.Values().
func (p Parser) rawOptionalValue() (arg string, hadEqSign bool, ok bool) {
	prevState := p.state
	p.state = stateNone{}
	if pendingValue, ok := prevState.(statePendingValue); ok {
		return pendingValue.a, true, true
	} else if shorts, ok := prevState.(stateShorts); ok {
		arg := shorts.a
		pos := shorts.b
		if pos >= uint(len(arg)) {
			return "", false, false
		}
		hadEqSign := false
		if arg[pos] == '=' {
			// -o=value.
			// clap actually strips out all leading '='s, but that seems silly.
			// We allow -xo=value. Python's argparse doesn't strip the = in that case.
			pos += 1
			hadEqSign = true
		}
		arg = arg[pos:] // Reuse allocation
		return string(arg), hadEqSign, true
	} else if _, ok := prevState.(stateFinishedOpts); ok {
		// Not really supposed to be here, but it's benign and not our fault
		p.state = stateFinishedOpts{}
		return "", false, false
	} else if _, ok := prevState.(stateNone); ok {
		return "", false, false
	} else {
		panic("unreachable")
	}
}

func newParser(binName *string, source struct {
	slice []string
	index int
}) *Parser {
	var binName2 *string
	if binName != nil {
		v := strings.ToValidUTF8(*binName, "\uFFFD")
		binName2 = &v
	}
	return &Parser{
		source:     source,
		state:      stateNone{},
		lastOption: lastOptionNone{},
		binName:    binName2,
	}
}

// Create a parser from the environment using os.Args.
//
// This is the usual way to create a Parser.
func ParserFromEnv() *Parser {
	var binName *string
	var source []string
	if len(os.Args) > 0 {
		binName = &os.Args[0]
		source = os.Args[1:]
	} else {
		source = os.Args
	}
	return newParser(binName, struct {
		slice []string
		index int
	}{source, 0})
}

// Create a parser from an iterator. This is useful for testing among other things.
//
// The first item from the iterator **must** be the binary name, as from os.Args.
//
// The iterator is consumed immediately.
//
// # Example
//
//	args := []string{"myapp", "-n", "10", "./foo.bar"}
//	parser := lexopt.ParserFromIter(slices.Values(args))
func ParserFromIter(args iter.Seq[string]) *Parser {
	argsSlice := slices.Collect(args)
	var binName *string
	var source []string
	if len(argsSlice) > 0 {
		binName = &argsSlice[0]
		source = argsSlice[1:]
	} else {
		source = argsSlice
	}
	return newParser(binName, struct {
		slice []string
		index int
	}{source, 0})
}

// Create a parser from an iterator that does **not** include a binary name.
//
// The iterator is consumed immediately.
//
// .BinName() will return (T, false). Consider using
// ParserFromIter() instead.
func ParserFromArgs(args iter.Seq[string]) *Parser {
	argsSlice := slices.Collect(args)
	return newParser(nil, struct {
		slice []string
		index int
	}{argsSlice, 0})
}

// Store a long option so the caller can get it.
func (p *Parser) setLong(option string) Arg {
	p.lastOption = lastOptionLong{option}
	if lastOption, ok := p.lastOption.(lastOptionLong); ok {
		return &ArgLong{lastOption.A[2:]}
	} else {
		panic("unreachable")
	}
}

func firstCodepoint(bytes []byte) (char rune, ok bool, err error) {
	if len(bytes) == 0 {
		return 0, false, nil
	}
	r, _ := utf8.DecodeRune(bytes)
	if r == utf8.RuneError {
		return 0, false, fmt.Errorf("%v does not start with a valid UTF-8 codepoint", bytes)
	}
	return r, true, nil
}
