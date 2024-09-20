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
    errorIllImport = errors.New("illegal import spec")
    errorIllJson   = errors.New("illegal json format")
    errorIllName   = errors.New("illegal name")
    errorIllXml    = errors.New("illegal xml format")
    errorNilExec   = errors.New("execute nil program")
    errorNoEntry   = errors.New("no matched rule")
    errorUpdated   = errors.New("target updated")
)

type (
    failureAssert      string
    failureUnreachable string
    failureTargetNotFound struct { project *project; target string }
    failurePathNotFound   struct { project *project; path *path }
    failureFileNotFound   struct { project *project; file *file }
    failure     struct { Context; reason string }
    termination struct { position Position }
)

func _failure(ctx Context, a ...any) failure {
    var s string
    if y := false; 0 < len(a) {
        if s, y = a[0].(string); y {
            if 1 < len(a) {
                s = fmt.Sprintf(s, a[1:]...)
            }
        }
    }
    return failure{ctx, s}
}

func (f *failure) Error() (s string) {
    s = "failed"
    if f.Context != nil { s += " : "+ts(f.Context) }
    if f.reason != "" { s += " : "+f.reason }
    return
}

func (s failureAssert) Error() string { return string(s) }
func (s failureUnreachable) Error() string { return string(s) }

func (e failureTargetNotFound) Error() string {
    return fmt.Sprintf("%s: %v: target not found", e.project.name, e.target)
}

func (e failurePathNotFound) Error() string {
    return fmt.Sprintf("%s: %v: path not found", e.project.name, e.path)
}

func (e failureFileNotFound) Error() string {
    if s, t := e.file.fullname(), e.file.filestub.name; t == s { // e.project.name
        return fmt.Sprintf(`"%v" not found`, t)
    } else {
        return fmt.Sprintf(`"%v" not found (at %s)`, t, s) //trimPromptString(s)
    }
}

func assert(cond bool, s string, a ...any) {
    if !cond { panic(failureAssert(fmt.Sprintf(s, a...))) }
}

func unreachable(a ...any) {
    panic(failureUnreachable(fmt.Sprint(a...)))
}
