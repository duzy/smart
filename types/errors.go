//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package types

import (
        "errors"
        "fmt"
        "runtime"
)

var (
        ErrorUpdated   = errors.New("target updated")
        ErrorNilExec   = errors.New("execute nil program")
        ErrorNoEntry   = errors.New("no matched rule")
        ErrorIllXml    = errors.New("illegal xml format")
        ErrorIllJson   = errors.New("illegal json format")
)

type Returner struct {
        Value Value
}

func (p *Returner) Error() string {
        return "evaluation returned"
}

type Failure struct {
        msg string
}

func (f *Failure) Error() string { return f.msg }

func Fail(s string, a... interface{}) {
        if len(a) > 0 {
                s = fmt.Sprintf(s, a...)
        }
        panic(&Failure{s})
}

type RuntimeError struct {
        stack string
}

func (p *RuntimeError) Error() string {
        return p.stack
}

func GoFault(e interface{}) *RuntimeError {
        if _, ok := e.(runtime.Error); ok {
                s := fmt.Sprintf("%s\n\n%s", e, ParseStack(false))
                return &RuntimeError{ s }
        }
        return nil
}

func ParseStack(all bool) (s string) {
        // Alternative: debug.PrintStack
        // 1. runtime/stack.go
        // 2. runtime/debug/stack.go
        ba := make([]byte, 1<<16)
        ss := runtime.Stack(ba, all)
        s = string(ba[:ss])
        // TODO: trim off some sys frames
        return
}
