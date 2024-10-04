/*
Some programs accept options with an unusual syntax. For example, tail
accepts -13 as an alias for -n 13.

This program shows how to use parser.TryRawArgs() to handle them
manually.

(Note: actual tail implementations handle it slightly differently! This
is just an example.)
*/
package lexopt_test

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/jcbhmr/go-lexopt"
	. "github.com/jcbhmr/go-lexopt/prelude"
)

func parseDashnum(parser *lexopt.Parser) (uint64, bool) {
	raw, ok := parser.TryRawArgs()
	if !ok {
		return 0, false
	}
	arg, ok := raw.Peek()
	if !ok {
		return 0, false
	}
	num, err := strconv.ParseUint(strings.TrimPrefix(arg, "-"), 10, 64)
	if err != nil {
		return 0, false
	}
	raw.Next() // Consume the argument we just parsed
	return num, true
}

func Example_nonstandard() {
	os.Args = []string{"nonstandard", "-13", "--number=42", "--follow", "file.txt"}

	log.SetFlags(0)

	parser := lexopt.ParserFromEnv()
	for {
		if num, ok := parseDashnum(parser); ok {
			log.Printf("Got number %v", num)
		} else {
			arg, ok, err := parser.Next()
			if err != nil {
				log.Fatal(err)
			}
			if ok {
				argShort, argShortOk := arg.(Short)
				argLong, argLongOk := arg.(Long)
				if (argShortOk && argShort.A == 'f') || (argLongOk && argLong.A == "follow") {
					log.Println("Got --follow")
				} else if (argShortOk && argShort.A == 'n') || (argLongOk && argLong.A == "number") {
					numText, err := parser.Value()
					if err != nil {
						log.Fatal(err)
					}
					num, err2 := strconv.ParseUint(numText, 10, 64)
					if err2 != nil {
						log.Fatal(err2)
					}
					log.Printf("Got number %v", num)
				} else if argV, ok := arg.(Value); ok {
					log.Printf("Got file %v", argV.A)
				}
			} else {
				break
			}
		}
	}
}
