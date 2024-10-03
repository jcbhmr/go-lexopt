package lexopt_test

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/jcbhmr/go-lexopt"
	. "github.com/jcbhmr/go-lexopt/prelude"
)

type args struct {
	thing  string
	number uint32
	shout  bool
}

func parseArgs() (args, error) {
	var thing *string = nil
	var number uint32 = 1
	var shout bool = false
	parser := lexopt.ParserFromEnv()
	for {
		arg, ok, err := parser.Next()
		if err != nil {
			return args{}, err
		}
		if !ok {
			break
		}
		if (arg == Short{'n'}) || (arg == Long{"number"}) {
			numberText, err := parser.Value()
			if err != nil {
				return args{}, err
			}
			number64, err := strconv.ParseUint(numberText, 10, 32)
			if err != nil {
				return args{}, err
			}
			number = uint32(number64)
		} else if (arg == Long{"shout"}) {
			shout = true
		} else if val, ok := arg.(Value); thing == nil && ok {
			thing = &val.A
		} else if (arg == Long{"help"}) {
			fmt.Println("Usage: hello [-n|--number=NUM] [--shout] THING")
			os.Exit(0)
		} else {
			return args{}, arg.Unexpected()
		}
	}

	if thing == nil {
		return args{}, errors.New("missing argument THING")
	}
	return args{
		thing: *thing,
		number: number,
		shout: shout,
	}, nil
}

func init() {
	os.Args = []string{"hello", "--number=3", "--shout", "Alan Turing"}
}

func Example_hello() {
	args, err := parseArgs()
	if err != nil {
		log.Fatal(err)
	}
	message := fmt.Sprintf("Hello %s!", args.thing)
	if args.shout {
		message = strings.ToUpper(message)
	}
	for i := uint32(0); i < args.number; i++ {
		fmt.Println(message)
	}
	
	// Output:
	// HELLO ALAN TURING!
}
