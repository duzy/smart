//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	"fmt"
	"io"
	"time"
)

var (
	l_t        = time.Now()
	l_config   = &ltracing{ tm: l_t }
	l_exec     = &ltracing{ tm: l_t }
	l_launch   = &ltracing{ tm: l_t }
	l_load     = &ltracing{ tm: l_t }
	l_parse    = &ltracing{ tm: l_t } // UNUSED
	l_traverse = &ltracing{ tm: l_t }
)

type l_tracer interface {
	elapsed() time.Duration
	tracef(string,...interface{})
	trace(...interface{})
	level(int)
}

func l_trace(t l_tracer, s string) l_tracer {
	t.trace(s, "(")
	t.level(+1)
	t.tracef("%v", t.elapsed())
	return t
}

func l_tracef(t l_tracer, f string, a ...interface{}) l_tracer {
	t.trace(fmt.Sprintf(f, a...), "(")
	t.level(+1)
	t.tracef("%v", t.elapsed())
	return t
}

// Usage:
//   defer un(trace(p, "..."))
//   defer un(tracef(p, "..."))
//   defer un(tr(p, "..."))
//   defer un(tt(p, t, "..."))
func un(t l_tracer) {
	t.tracef("%v", t.elapsed())
	t.level(-1)
	t.trace(")")
}

func tr(t l_tracer, i Value) l_tracer {
	t.tracef("%s (", ts(i))
	t.level(+1)
	t.tracef("%v", t.elapsed())
    return t
}

func tt(t l_tracer, ctx Context, i Value) l_tracer {
    // Note that t.args and t.arguments are different, they're
    // target execution args and argumented-prerequisite args.
    var a string = ts(_entry(ctx).destiny())
    if false { a += " " + ts(ctx) }
    t.trace(a, ":", ts(i), "(")
    t.level(+1)
	t.tracef("%v", t.elapsed())
    return t
}

type ltracing struct {
	// Tracing/debugging
	all bool
	enabled bool // (mode&Trace != 0)
	indent int  // indentation used for tracing output
	tm time.Time
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

func (p *ltracing) traceAt(pos Position, a ...interface{}) {
	fmt.Fprintf(stderr, "%7d:%3d: ", pos.Line, pos.Column)
	printIndentDots(p.indent, a...)
}

func (p *ltracing) trace(a ...interface{}) {
	printIndentDots(p.indent, a...)
}

func (p *ltracing) tracef(s string, a ...interface{}) {
	printIndentDots(p.indent, fmt.Sprintf(s, a...))
}

func (p *ltracing) level(n int) {
	p.indent += n
}

func (p *ltracing) elapsed() time.Duration {
	if false && p.tm.IsZero() { p.tm = time.Now() }
	return time.Now().Sub(p.tm)
}
