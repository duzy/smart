//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
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
            if e.Pos.Same(&t.Pos) {
                s = t.Error()
            } else {
                s = fmt.Sprintf("%s\n%s: …from here", t, e.Pos)
            }
        default:
            s = fmt.Sprintf("%s: %T %s", e.Pos, t, t)
        }
        return
    }
    for _, err := range e.Errs {
        if s == "" {
            switch t := e.Errs[0].(type) {
            case *Error: s = t.Error()
            default: s = fmt.Sprintf("error:. %s", err)
            }
        } else {
            s = fmt.Sprintf("%s\n%s", s, err)
        }
    }
    if e.Pos.Filename != "" && e.Pos.IsValid() {
        if s == "" {
            s = fmt.Sprintf("%s: no errors", e.Pos)
        } else {
            s = fmt.Sprintf("%s\n%s: …from here", s, e.Pos)
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
    if pos.Same(&e.Pos) { return e }
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
        if v := reflect.ValueOf(err); err == nil || (v.Kind() == reflect.Ptr && v.IsNil()) {
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
