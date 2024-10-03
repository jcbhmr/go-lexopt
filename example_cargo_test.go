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

const Help = "cargo [+toolchain] [OPTIONS] [SUBCOMMAND]"

func init() {
	os.Args = []string{"cargo", "+nightly", "--verbose", "--color=never", "install", "hello", "--root", "/home/octocat/project1", "--jobs", "8"}
}

func Example_cargo() {
	settings := globalSettings{
		toolchain: "stable",
		color: ColorAuto,
		offline: false,
		quiet: false,
		verbose: false,
	}

	parser := lexopt.ParserFromEnv()
	for {
		arg, ok, err := parser.Next()
		if err != nil {
			log.Fatal(err)
		}
		if !ok {
			break
		}
		if (arg == Long{"color"}) {
			colorText, err := parser.Value()
			if err != nil {
				log.Fatal(err)
			}
			color, err2 := ColorFromStr(colorText)
			if err2 != "" {
				log.Fatal(&lexopt.ErrorCustom{errors.New(err2)})
			}
			settings.color = color
		} else if (arg == Long{"offline"}) {
			settings.offline = true
		} else if (arg == Long{"quiet"}) {
			settings.quiet = true
			settings.verbose = false
		} else if (arg == Long{"verbose"}) {
			settings.quiet = false
			settings.verbose = true
		} else if (arg == Long{"help"}) {
			fmt.Println(Help)
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

	fmt.Println(Help)

	// Output:
	// Settings: lexopt_test.globalSettings{toolchain:"nightly", color:0, offline:false, quiet:false, verbose:true}
	// Installing hello into /home/octocat/project1 with 8 jobs
}

type globalSettings struct {
	toolchain string
	color Color
	offline bool
	quiet bool
	verbose bool
}

func install(settings globalSettings, parser lexopt.Parser) lexopt.Error {
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
		if value, ok := arg.(Value); package_ == nil && ok {
			package_ = &value.A
		} else if (arg == Long{"root"}) {
			rootText, err := parser.Value()
			if err != nil {
				return err
			}
			root = &rootText
		} else if (arg == Short{'j'}) || (arg == Long{"jobs"}) {
			jobsText, err := parser.Value()
			if err != nil {
				return err
			}
			jobs64, err := strconv.ParseUint(jobsText, 10, 16)
			if err != nil {
				return &lexopt.ErrorCustom{err}
			}
			jobs = uint16(jobs64)
		} else if (arg == Long{"help"}) {
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

type Color uint8
const (
	ColorAuto Color = iota
	ColorAlways
	ColorNever
)

func ColorFromStr(s string) (Color, string) {
	switch strings.ToLower(s) {
	case "auto":
		return ColorAuto, ""
	case "always":
		return ColorAlways, ""
	case "never":
		return ColorNever, ""
	default:
		return 0, fmt.Sprintf("Invalid style '%v' [pick from: auto, always, never]", s)
	}
}

func getNoOfCPUs() uint16 {
	return 4
}
