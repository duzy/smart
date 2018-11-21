//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "extbit.io/smart/token"
        "fmt"
)

type using struct {
        project *Project
        params []Value
}

func (p *using) refs(v Value) bool {
        for _, a := range p.params {
                if a.refs(v) { return true }
        }
        return false
}
func (p *using) closured() bool {
        for _, a := range p.params {
                if a.closured() { return true }
        }
        return false
}
func (p *using) expend(w expendwhat) (Value, error) {
        if params, num, err := expendall(w, p.params...); err != nil {
                return nil, err
        } else if num > 0 {
                return &using{ p.project, params }, nil
        }
        return p, nil
}
func (p *using) prepare(pc *preparer) (err error) {
        if trace_prepare {
                fmt.Printf("prepare:using\n")
        }
        if entry := p.project.DefaultEntry(); entry != nil {
                err = entry.prepare(pc)
        }
        return
}
func (p *using) Type() Type { return UsingType }
func (p *using) Integer() (i int64, err error) { return 0, nil }
func (p *using) Float() (f float64, err error) { return 0, nil }
func (p *using) String() string {
        if len(p.params) > 0 {
                return fmt.Sprintf("use %s %v", p.project.name, p.params)
        } else {
                return fmt.Sprintf("use %s", p.project.name)
        }
}
func (p *using) Strval() (s string, err error) {
        s = fmt.Sprintf("use %s %v", p.project.name, p.params)
        return
}

type usinglist struct {
        knownobject
        list []*using
}

func (p *usinglist) refs(v Value) bool {
        for _, a := range p.list {
                if a.refs(v) { return true }
        }
        return false
}
func (p *usinglist) closured() bool {
        for _, a := range p.list {
                if a.closured() { return true }
        }
        return false
}
func (p *usinglist) expend(w expendwhat) (Value, error) {
        var (list []*using; num int)
        for _, elem := range p.list {
                if v, err := elem.expend(w); err != nil {
                        return nil, err
                } else {
                        if v != elem { num += 1 }
                        list = append(list, v.(*using))
                }
        }
        if num > 0 {
                return &usinglist{ p.knownobject, list }, nil
        }
        return p, nil
}
func (p *usinglist) prepare(pc *preparer) error {
        if trace_prepare {
                fmt.Printf("prepare:usinglist\n")
        }
        for _, elem := range p.list {
                if err := elem.prepare(pc); err != nil {
                        return err
                }
        }
        return nil
}
func (p *usinglist) Type() Type { return UsingListType }
func (p *usinglist) String() string {
        var s string
        for i, elem := range p.list {
                if i > 0 { s += ", " }
                s += elem.project.name
        }
        return fmt.Sprintf("{%s uses %v}", p.name, s)
}

func (p *usinglist) append(proj *Project, params []Value) {
        p.list = append(p.list, &using{ proj, params })
}

func (p *usinglist) Call(pos token.Position, a... Value) (res Value, err error) {
        if false {
                list := new(List)
                for _, elem := range p.list {
                        list.Elems = append(list.Elems, elem)
                }
                res = list
        } else {
                res = p
        }
        return
}
