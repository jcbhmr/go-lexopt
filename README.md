# Go lexopt

🐿️ Rust's [`lexopt` crate](https://docs.rs/lexopt) ported to Go

<table align=center><td>


</table>

## Usage

```go
package main

import (
    "errors"
    "fmt"
    "log"
    "os"
    "strconv"
    "strings"

    . "github.com/jcbhmr/go-lexopt/prelude"
    "github.com/jcbhmr/go-lexopt"
)

type args struct {
    thing string
    number uint32
    shout bool
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
        argShort, argShortOk := arg.(Short)
        argLong, argLongOk := arg.(Long)
        if (argShortOk && argShort == 'n') || (argLongOk && argLong == "number") {
            value, err := parser.Value()
            if err != nil {
                return args{}, err
            }
            number, err = strconv.ParseUint(value, 10, 32)
            if err != nil {
                return args{}, err
            }
        } else if v, ok := arg.(Long); ok && v == "shout" {
            shout = true
        } else if val, ok := arg.(Value); thing == nil && ok {
            thing = &val
        } else if v, ok := arg.(Long); ok && v == "help" {
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

func main() {
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
}
```

Let's walk through this:

- We start parsing with `lexopt.ParserFromEnv()`
- We call `parser.Next()` in a loop to get all the arguments until they run out
- We use `if` statements to structurally match on the arguments. `Short` and `Long` indicate an option.
- To get the value that belongs to an option (like `10` in `-n 10`) we call `parser.Value()`.
    - This returns a standard `string`.
    - For convenience, `import . "github.com/jcbhmr/go-lexopt/prelude"` adds a `
