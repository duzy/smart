//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//  Note that the Error and Errors defined in this file are the same as the
//  Error and ErrorList in go/scanner.
//  
package scanner

import (
        "extbit.io/smart/token"
        "errors"
	"sort"
	"fmt"
	"io"
)

// In an Errors, an error is represented by an *Error.
// The position Pos, if valid, points to the beginning of
// the offending token, and the error condition is described
// by Msg.
//
type Error struct {
	Pos token.Position
	Err error // Underlying error
}

// Error implements the error interface.
func (e *Error) Error() (s string) {
	if e.Pos.Filename != "" || e.Pos.IsValid() {
                switch t := e.Err.(type) {
                case *Error:
                        //s = fmt.Sprintf("%s: …\n%s", e.Pos, t)
                        s = fmt.Sprintf("%s\n%s: …", t, e.Pos)
                default:
                        s = fmt.Sprintf("%s: %s", e.Pos, e.Err)
                }
	} else {
                s = e.Err.Error()
        }
	return
}

// Errors is a list of *Errors.
// The zero value for an Errors is an empty Errors ready to use.
//
type Errors []*Error

// Add adds an Error with given position and error message to an Errors.
func (p *Errors) Add(pos token.Position, err error) {
        switch t := err.(type) {
        case *Error:
                *p = append(*p, t)
                *p = append(*p, &Error{pos, errors.New("from here")})
        case *Errors:
                for _, e := range *t { *p = append(*p, e) }
        case Errors:
                for _, e := range t { *p = append(*p, e) }
        default:
                *p = append(*p, &Error{pos, err})
        }
}

// Reset resets an Errors to no errors.
func (p *Errors) Reset() { *p = (*p)[0:0] }

// Errors implements the sort Interface.
func (p Errors) Len() int      { return len(p) }
func (p Errors) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func (p Errors) Less(i, j int) bool {
	e := &p[i].Pos
	f := &p[j].Pos
	// Note that it is not sufficient to simply compare file offsets because
	// the offsets do not reflect modified line information (through //line
	// comments).
	if e.Filename != f.Filename {
		return e.Filename < f.Filename
	}
	if e.Line != f.Line {
		return e.Line < f.Line
	}
	if e.Column != f.Column {
		return e.Column < f.Column
	}
	return p[i].Err.Error() < p[j].Err.Error()
}

// Sort sorts an Errors. *Error entries are sorted by position,
// other errors are sorted by error message, and before any *Error
// entry.
//
func (p Errors) Sort() { sort.Sort(p) }

// RemoveMultiples sorts an Errors and removes all but the first error per line.
func (p *Errors) RemoveMultiples() {
	sort.Sort(p)
	var last token.Position // initial last.Line is != any legal error line
	i := 0
	for _, e := range *p {
		if e.Pos.Filename != last.Filename || e.Pos.Line != last.Line {
			last = e.Pos
			(*p)[i] = e
			i++
		}
	}
	(*p) = (*p)[0:i]
}

// An Errors implements the error interface.
func (p Errors) Error() string {
	switch len(p) {
	case 0:
		return "no errors"
	case 1:
		return p[0].Error()
	}
	return fmt.Sprintf("%s (and %d more errors)", p[0], len(p)-1)
}

// Err returns an error equivalent to this error list.
// If the list is empty, Err returns nil.
func (p Errors) Err() error {
	if len(p) == 0 {
		return nil
	}
	return p
}

// PrintError is a utility function that prints a list of errors to w,
// one error per line, if the err parameter is an Errors. Otherwise
// it prints the err string.
//
func PrintError(w io.Writer, err error) {
        switch e := err.(type) {
        case Errors: for _, i := range e { PrintError(w, i) }
        /*case *Error:
                if _, ok := e.Err.(*Error); ok {
                        fmt.Fprintf(w, "%s: …\n", e.Pos)
                        PrintError(w, e.Err)
                } else {
                        fmt.Fprintf(w, "%s\n", err)
                }*/
        default:
		fmt.Fprintf(w, "%s\n", err)
	}
}

func Errorf(pos token.Position, s string, args... interface{}) (err error) {
        for _, a := range args {
                switch e := a.(type) {
                case *Error: panic(e)
                case Errors: panic(e)
                }
        }
        err = &Error{pos, fmt.Errorf(s, args...)}
        return
}

func WrapError(pos token.Position, args ...error) (err error) {
        var errs Errors
        for _, a := range args {
                switch e := a.(type) {
                case *Error:
                        errs = append(Errors{e}, errs...)
                case Errors:
                        if len(e) == 0 { continue }
                        var t = &Error{pos, fmt.Errorf("…from here")}
                        errs = append(append(e, t), errs...)
                default:
                        var t = &Error{pos, e}
                        errs = append(Errors{t}, errs...)
                }
        }
        if n := len(errs); n == 1 {
                err = errs[0]
        } else if n > 1 {
                err = errs
        }
        return
}
