//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        "github.com/duzy/smart/token"
        "errors"
        "fmt"
)

var (
        ErrorUpdated   = errors.New("target updated")
        ErrorNilExec   = errors.New("execute nil program")
        ErrorNoDialect = errors.New("unknown dialect")
        ErrorNoEntry   = errors.New("no matched rule")
        ErrorIllXml    = errors.New("illegal xml format")
        ErrorIllJson   = errors.New("illegal json format")
)

type Failure struct {
        pos token.Pos
        msg string
}

func (f *Failure) Pos() token.Pos { return f.pos }
func (f *Failure) Error() string { return f.msg }

func Fail(s string, a... interface{}) {
        if len(a) > 0 {
                s = fmt.Sprintf(s, a...)
        }
        panic(&Failure{token.NoPos, s})
}

func FailAt(pos token.Pos, s string, a... interface{}) {
        if len(a) > 0 {
                s = fmt.Sprintf(s, a...)
        }
        panic(&Failure{pos, s})
}
