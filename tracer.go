//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "extbit.io/smart/token"
        "extbit.io/smart/scanner"
        "errors"
        "fmt"
)

type tracer interface {
        level(n int)
        traceAt(pos token.Position, a ...interface{})
        errorAt(pos token.Position, err interface{}, a ...interface{})
}

type tracing struct {
        errors scanner.Errors

	// Tracing/debugging
	mode   Mode // parsing mode
	enabled bool // (mode&Trace != 0)
	indent int  // indentation used for tracing output
}

func (p *tracing) errorAt(pos token.Position, err interface{}, a ...interface{}) {
	// If AllErrors is not set, discard errors reported on the same line
	// as the last recorded error and stop parsing if there are more than
	// 10 errors.
	if p.mode&AllErrors == 0 {
		n := len(p.errors)
		if n > 0 && p.errors[n-1].Pos.Line == pos.Line {
			return // discard - likely a spurious error
		}
		if n > 10 {
			panic(bailout{})
		}
	}

        var s string
        switch t := err.(type) {
        case error:  p.errors.Add(pos, t); return
        case string: s = t
        default: s = fmt.Sprintf("%v", err)
        }
        if len(a) > 0 {
                s = fmt.Sprintf(s, a...)
        }
        p.errors.Add(pos, errors.New(s))
}

func printIndentDots(indent int, a ...interface{}) {
	const dots = ". . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . "
	const n = len(dots)
	i := 2 * indent
	for i > n {
		fmt.Print(dots)
		i -= n
	}
	// i <= n
	fmt.Print(dots[0:i])
	if len(a) > 0 { fmt.Println(a...) }
}

func (p *tracing) traceAt(pos token.Position, a ...interface{}) {
	fmt.Fprintf(stderr, "%7d:%3d: ", pos.Line, pos.Column)
        printIndentDots(p.indent, a...)
}

func (p *tracing) level(n int) {
        p.indent += n
}
