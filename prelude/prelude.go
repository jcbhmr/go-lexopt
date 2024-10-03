/*
A small prelude for processing arguments.

It allows you to write Short/Long/Value without an Arg prefix.
*/
package prelude

import "github.com/jcbhmr/go-lexopt"

type Short = lexopt.ArgShort
type Long = lexopt.ArgLong
type Value = lexopt.ArgValue
