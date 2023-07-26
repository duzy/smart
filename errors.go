//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    "errors"
    "fmt"
)

var (
    ErrorIllImport = errors.New("illegal import spec")
    ErrorIllJson   = errors.New("illegal json format")
    ErrorIllName   = errors.New("illegal name")
    ErrorIllXml    = errors.New("illegal xml format")
    ErrorNilExec   = errors.New("execute nil program")
    ErrorNoEntry   = errors.New("no matched rule")
    ErrorUpdated   = errors.New("target updated")
)

type (
    failure struct {
        fmt string
        a []interface{}
    }

    termination struct {
        position Position
    }

    FailedAssertion string
    Unreachable string

    targetNotFoundError struct { project *Project; target string }
    pathNotFoundError   struct { project *Project; path *Path }
    fileNotFoundError   struct { project *Project; file *File }
)

func assert(cond bool, s string, a ...interface{}) {
    if !cond { panic(FailedAssertion(fmt.Sprintf(s, a...))) }
}

func unreachable(a ...interface{}) {
    panic(Unreachable(fmt.Sprint(a...)))
}

func ia(a ...interface{}) []interface{} { return a }

func (f *failure) Error() string { var a []interface{}
    for _, v := range f.a { if _, y := v.(Position); !y { a = append(a, v) }}
    return fmt.Sprintf(f.fmt, a...)
}
func (f *failure) at(ctx Context) Context {
    for _, a := range f.a { switch t := a.(type) {
    case []Value: if len(t) > 0 { return at(ctx, t[0].Position()) }
    case Value: return at(ctx, t.Position())
    case Position: return at(ctx, t)
    }}
    return ctx
}
func (f *failure) ia() (res []interface{}) {
    for i, a := range f.a { if _, y := a.(Position); i > 0 || !y { res = append(res, a) }}
    return
}

func (s FailedAssertion) Error() string { return string(s) }
func (s Unreachable) Error() string { return string(s) }

func (e targetNotFoundError) Error() string {
    return fmt.Sprintf("%s: %v: target not found", e.project.name, e.target)
}

func (e pathNotFoundError) Error() string {
    return fmt.Sprintf("%s: %v: path not found", e.project.name, e.path)
}

func (e fileNotFoundError) Error() string {
    if s, t := e.file.fullname(), e.file.filestub.name; t == s { // e.project.name
        return fmt.Sprintf(`"%v" not found`, t)
    } else {
        return fmt.Sprintf(`"%v" not found (at %s)`, t, s) //trimPromptString(s)
    }
}
