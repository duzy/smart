//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

type Interpreter interface {
        Evaluate(prog *Program, args []Value, recipes []Value) (Value, error)
}

type dialect struct {
        Interpreter
        s string
}

func (d dialect) name() string { return d.s }

var (
        dialects = make(map[string]*dialect)
)

func RegisterDialect(name string, int Interpreter) {
        dialects[name] = &dialect{ int, name }
}

func IsDialect(s string) (ok bool) {
        _, ok = dialects[s]
        return
}
