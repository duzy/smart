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
        "reflect"
	"sort"
	"fmt"
)

const maxErrors = 120
var errTooManyErrors = errors.New("too many errors")

// In an Errors, an error is represented by an *Error.
// The position Pos, if valid, points to the beginning of
// the offending token, and the error condition is described
// by Msg.
//
type Error struct {
	Pos token.Position
	Errs []error // Underlying errors
}

// Error implements the error interface.
func (e *Error) Error() (s string) {
        if e == nil { return }
        if len(e.Errs) == 1 {
                switch t := e.Errs[0].(type) {
                case *Error:
                        if e.Pos.Equals(&t.Pos) {
                                s = t.Error()
                        } else {
                                s = fmt.Sprintf("%s\n%s: …from here", t, e.Pos)
                        }
                default:
                        s = fmt.Sprintf("%s: %s", e.Pos, t)
                }
                return
        }
        for _, err := range e.Errs {
                if s == "" {
                        switch t := e.Errs[0].(type) {
                        case *Error: s = t.Error()
                        default: s = fmt.Sprintf("error: %s", err)
                        }
                } else {
                        s = fmt.Sprintf("%s\n%s", s, err)
                }
        }
        if e.Pos.Filename != "" && e.Pos.IsValid() {
                if s == "" {
                        s = fmt.Sprintf("%s: no errors", e.Pos)
                } else {
                        s = fmt.Sprintf("%s\n%s: …from here!", s, e.Pos)
                }
        }
	return
}

func (e *Error) Brief() (s string) {
        if n := len(e.Errs); n == 0 {
                s = "no errors"
        } else {
                if t, ok := e.Errs[0].(*Error); ok {
                        s = t.Brief()
                } else {
                        s = t.Error()
                }
                if n > 1 {
                        s = fmt.Sprintf("%s, and %v more", s, n-1)
                }
        }
	return
}

func (e *Error) getErrorAt(pos token.Position) (res *Error) {
        if pos.Equals(&e.Pos) { return e }
        for _, err := range e.Errs {
                if t, ok := err.(*Error); ok {
                        if res = t.getErrorAt(pos); res != nil { return }
                }
        }
        return
}

func (e *Error) find(err error) int {
        if _, ok := err.(*Error); !ok {
                for i, e := range e.Errs {
                        if _, ok := e.(*Error); ok { continue }
                        if e == err || e.Error() == err.Error() {
                                return i
                        }
                }
        }
        return -1
}

func (result *Error) Merge(errs ...error) {
ForErrs:
        for _, err := range errs {
                if v := reflect.ValueOf(err); err == nil || v.IsNil() {
                        continue
                } else if len(result.Errs) > maxErrors {
                        result.Errs = result.Errs[maxErrors:]
                }

                if e, ok := err.(*Error); ok {
                        if t := result.getErrorAt(e.Pos); t != nil {
                                t.Merge(e.Errs...)
                        } else {
                                for i, f := range result.Errs {
                                        if j := e.find(f); j >= 0 {
                                                result.Errs = append(result.Errs[0:i], result.Errs[i+1:]...)
                                        }
                                }
                                result.Errs = append(result.Errs, err)
                        }
                        continue ForErrs
                }

                var s string
                for _, e := range result.Errs {
                        if e == err { continue ForErrs }
                        if e, ok := e.(*Error); !ok {
                                if s == "" { s = err.Error() }
                                if e.Error() == s { continue ForErrs }
                        }
                }

                result.Errs = append(result.Errs, err)
        }
}

// Errors is a list of *Errors.
// The zero value for an Errors is an empty Errors ready to use.
//
type Errors []*Error

// Add adds an Error with given position and error message to an Errors.
func (p *Errors) Add(pos token.Position, err error) {
        *p = append(*p, &Error{pos, []error{err}})
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
	return p[i].Errs[0].Error() < p[j].Errs[0].Error()
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

func Errorf(pos token.Position, s string, args... interface{}) (err error) {
        for _, a := range args {
                switch e := a.(type) {
                case *Error: panic(e) // use WrapErrors instead!
                }
        }
        err = &Error{pos, []error{fmt.Errorf(s, args...)}}
        return
}

func WrapErrors(pos token.Position, errs ...error) (err error) {
        if len(errs) == 0 { panic("no errors") }
        var result = &Error{Pos:pos}
        result.Merge(errs...)
        if len(result.Errs) == 0 { panic("no errors") }
        err = result
        return
}
