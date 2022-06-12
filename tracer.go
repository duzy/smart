//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	//"extbit.io/smart/token"
	"time"
	"fmt"
	"io"
)

var (
	t_launch   = &tracing{ tm: time.Now() }
	t_load     = &tracing{ tm: time.Now() }
	t_parse    = &tracing{ tm: time.Now() } // UNUSED
	t_traverse = &tracing{ tm: time.Now() }
	t_config   = &tracing{ tm: time.Now() }
	t_exec     = &tracing{ tm: time.Now() }
)

type tracer interface {
	elapsed() time.Duration
	tracef(s string, a ...interface{})
	trace(a ...interface{})
	level(n int)
}

func trace(t tracer, s string) tracer {
	t.trace(s, "(")
	t.level(+1)
	if true { t.tracef("%v", t.elapsed()) }
	return t
}

func tracef(t tracer, f string, a ...interface{}) tracer {
	t.trace(fmt.Sprintf(f, a...), "(")
	t.level(+1)
	if true { t.tracef("%v", t.elapsed()) }
	return t
}

// Usage:
//   defer un(trace(p, "..."))
//   defer un(tracef(p, "..."))
//   defer un(tr(p, "..."))
//   defer un(tt(p, t, "..."))
func un(t tracer) {
	if true { t.tracef("%v", t.elapsed()) }
	t.level(-1)
	t.trace(")")
}

func tr(t tracer, i Value) tracer {
	t.tracef("%s{%v} (", typeof(i), i)
	t.level(+1)
	if true { t.tracef("%v", t.elapsed()) }
    return t
}

func tt(t tracer, ctx Context, i Value) tracer {
    // Note that t.args and t.arguments are different, they're
    // target execution args and argumented-prerequisite args.
    var a string
	var pc = ctx.programCtx()
    if tar := ctx.entry().Target(); len(pc.params) > 0 {
        a = fmt.Sprintf("%s{%s}%s", typeof(tar), tar, pc.params)
    } else {
        a = fmt.Sprintf("%s{%v}", typeof(tar), tar)
    }
    var b = fmt.Sprintf("%s{%v}", typeof(i), i)
    t.trace(a, ":", b, "(")
    t.level(+1)
	if true { t.tracef("%v", t.elapsed()) }
    return t
}

type tracing struct {
	// Tracing/debugging
	all bool
	enabled bool // (mode&Trace != 0)
	indent int  // indentation used for tracing output
	tm time.Time
}

/*func (p *tracing) errorAt(pos token.Position, err interface{}, a ...interface{}) {
	// If AllErrors is not set, discard errors reported on the same line
	// as the last recorded error and stop parsing if there are more than
	// 10 errors.
	if p.all {
		n := diag.numErrors()
		if n > 10 { panic(bailout{}) }
	}

	var s string
	switch t := err.(type) {
	case error: diag.errorAt(Position(pos), "%T %v", t, t); return
	case string: s = t
	default: s = fmt.Sprintf("%v", err)
	}
	diag.errorAt(Position(pos), fmt.Sprintf(s, a...))
}*/

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

func (p *tracing) traceAt(pos Position, a ...interface{}) {
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

func (p *tracing) elapsed() time.Duration {
	if false && p.tm.IsZero() { p.tm = time.Now() }
	return time.Now().Sub(p.tm)
}
