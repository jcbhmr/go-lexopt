/*
A very partial unfaithful implementation of cargo's command line.

This showcases some harier patterns, like subcommands and custom value parsing.
*/
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

const help = "cargo [+toolchain] [OPTIONS] [SUBCOMMAND]"

func Example_cargo() {
	os.Args = []string{"cargo", "+nightly", "--verbose", "--color=never", "install", "hello", "--root", "/home/octocat/project1", "--jobs", "8"}

	log.SetFlags(0)

	settings := globalSettings{
		toolchain: "stable",
		color:     colorAuto,
		offline:   false,
		quiet:     false,
		verbose:   false,
	}

	parser := lexopt.ParserFromEnv()
	for {
		arg, ok, err := parser.Next()
		fmt.Printf("parser=%#+v, arg=%#+v, ok=%#+v, err=%#+v\n", parser, arg, ok, err)
		if err != nil {
			log.Fatal(err)
		}
		if !ok {
			break
		}
		if argV, ok := arg.(Long); ok && argV.A == "color" {
			colorText, err := parser.Value()
			if err != nil {
				log.Fatal(err)
			}
			color, err2 := colorFromStr(colorText)
			if err2 != "" {
				err = &lexopt.ErrorCustom{errors.New(err2)}
				log.Fatal(err)
			}
			settings.color = color
		} else if argV, ok := arg.(Long); ok && argV.A == "offline" {
			settings.offline = true
		} else if argV, ok := arg.(Long); ok && argV.A == "quiet" {
			settings.quiet = true
			settings.verbose = false
		} else if argV, ok := arg.(Long); ok && argV.A == "verbose" {
			settings.quiet = false
			settings.verbose = true
		} else if argV, ok := arg.(Long); ok && argV.A == "help" {
			fmt.Println(help)
			os.Exit(0)
		} else if value, ok := arg.(Value); ok {
			if strings.HasPrefix(value.A, "+") {
				settings.toolchain = value.A[1:]
			} else if value.A == "install" {
				err := install(settings, parser)
				if err != nil {
					log.Fatal(err)
				}
				return
			} else {
				log.Fatalf("Unknown subcommand '%v'", value.A)
			}
		} else {
			log.Fatal(arg.Unexpected())
		}
	}

	fmt.Println(help)

	// Output:
	// Settings: lexopt_test.globalSettings{toolchain:"nightly", color:0, offline:false, quiet:false, verbose:true}
	// Installing hello into /home/octocat/project1 with 8 jobs
}

type globalSettings struct {
	toolchain string
	color     color
	offline   bool
	quiet     bool
	verbose   bool
}

func install(settings globalSettings, parser *lexopt.Parser) lexopt.Error {
	var package_ *string = nil
	var root *string = nil
	var jobs uint16 = getNoOfCPUs()

	for {
		arg, ok, err := parser.Next()
		if err != nil {
			return err
		}
		if !ok {
			break
		}
		argShort, argShortOk := arg.(Short)
		argLong, argLongOk := arg.(Long)
		if value, ok := arg.(Value); package_ == nil && ok {
			package_ = &value.A
		} else if (arg == Long{"root"}) {
			rootText, err := parser.Value()
			if err != nil {
				return err
			}
			root = &rootText
		} else if (argShortOk && argShort.A == 'j') || (argLongOk && argLong.A == "jobs") {
			jobsText, err := parser.Value()
			if err != nil {
				return err
			}
			jobs64, err2 := strconv.ParseUint(jobsText, 10, 16)
			if err2 != nil {
				return &lexopt.ErrorCustom{err2}
			}
			jobs = uint16(jobs64)
		} else if argV, ok := arg.(Long); ok && argV.A == "help" {
			fmt.Println("cargo install [OPTIONS] CRATE")
			os.Exit(0)
		} else {
			return arg.Unexpected()
		}
	}

	fmt.Printf("Settings: %#+v\n", settings)
	if package_ == nil {
		return &lexopt.ErrorCustom{errors.New("missing CRATE argument")}
	}
	fmt.Printf("Installing %v into %#v with %v jobs\n", *package_, root, jobs)

	return nil
}

type color uint8

const (
	colorAuto color = iota
	colorAlways
	colorNever
)

func colorFromStr(s string) (color, string) {
	switch strings.ToLower(s) {
	case "auto":
		return colorAuto, ""
	case "always":
		return colorAlways, ""
	case "never":
		return colorNever, ""
	default:
		return 0, fmt.Sprintf("Invalid style '%v' [pick from: auto, always, never]", s)
	}
}

func getNoOfCPUs() uint16 {
	return 4
}
