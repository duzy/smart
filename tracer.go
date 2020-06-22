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
        "io"
)

var (
        t_launch = new(tracing)
        t_parse = new(tracing) // UNUSED
        t_traverse = new(tracing) // UNUSED
        t_exec = new(tracing)
        t_config = new(tracing)
)

type tracer interface {
        trace(a ...interface{})
        level(n int)
}

func trace(p tracer, msg string) tracer {
	p.trace(msg, "(")
	p.level(+1)
	return p
}

// Usage pattern: defer un(trace(p, "..."))
func un(p tracer) {
        p.level(-1)
        p.trace(")")
}

type tracing struct {
        errors scanner.Errors

	// Tracing/debugging
	tracemode Mode // parsing mode
	enabled bool // (mode&Trace != 0)
	indent int  // indentation used for tracing output
}

func (p *tracing) errorAt(pos token.Position, err interface{}, a ...interface{}) {
	// If AllErrors is not set, discard errors reported on the same line
	// as the last recorded error and stop parsing if there are more than
	// 10 errors.
	if p.tracemode&AllErrors == 0 {
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
                /*for _, v := range a {
                        if e, ok := v.(error); ok {
                                panic(fmt.Sprintf("embedded error: %s", e))
                        }
                }*/
                s = fmt.Sprintf(s, a...)
        }
        p.errors.Add(pos, errors.New(s))
}

// Printing fields (splitted by \t).
//var lenPrintField = lenPrintTab * 1

const (
        // Tab size helps formatting fields.
        lenPrintTab = 8

        dots = ". . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . "
        ndots = len(dots)
)

func fprintIndentDots(w io.Writer, indent int, a ...interface{}) {
	i := 2 * indent
	for i > ndots {
		fmt.Fprint(w, dots)
		i -= ndots
	}
	// i <= n
	fmt.Fprint(w, dots[0:i])
	if false && len(a) > 0 {
                fmt.Fprintln(w, a...)
        } else {
                var fieldLen = 0
                for i, v := range a {
                        if r, ok := v.(rune); ok && r == '\t' {
                                const sps = "                         "
                                if m := fieldLen % lenPrintTab; m > 0 {
                                        if m > len(sps) { m = len(sps)-1 }
                                        fmt.Fprint(w, sps[:m])
                                }
                                fieldLen = 0
                        } else if s := fmt.Sprint(v); s != "" {
                                if i > 0 {
                                        fmt.Fprint(w, " ", s)
                                        fieldLen += len(s) + 1
                                } else {
                                        fmt.Fprint(w, s)
                                        fieldLen += len(s)
                                }
                        }
                }
                fmt.Fprintln(w)
        }
}

func printIndentDots(indent int, a ...interface{}) {
        fprintIndentDots(stderr, indent, a...)
}

func (p *tracing) traceAt(pos token.Position, a ...interface{}) {
	fmt.Fprintf(stderr, "%7d:%3d: ", pos.Line, pos.Column)
        printIndentDots(p.indent, a...)
}

func (p *tracing) trace(a ...interface{}) {
        printIndentDots(p.indent, a...)
}

func (p *tracing) tracef(s string, a ...interface{}) {
        printIndentDots(p.indent, fmt.Sprintf(s, a...))
}

func (p *tracing) level(n int) {
        p.indent += n
}
