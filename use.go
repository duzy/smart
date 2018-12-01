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
func (p *using) expand(w expandwhat) (Value, error) {
        if params, num, err := expandall(w, p.params...); err != nil {
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
                if err = entry.prepare(pc); err != nil {
                        fmt.Printf("%v: using default entry `%s=>%s`: %v\n", entry.Position, p.project.name, entry, err)
                }
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
        name string
        scope *Scope
        owner *Project
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
func (p *usinglist) expand(w expandwhat) (Value, error) {
        var (list []*using; num int)
        for _, elem := range p.list {
                if v, err := elem.expand(w); err != nil {
                        return nil, err
                } else {
                        if v != elem { num += 1 }
                        list = append(list, v.(*using))
                }
        }
        if num > 0 {
                return &usinglist{ p.name, p.scope, p.owner, list }, nil
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
func (p *usinglist) redecl(scope *Scope) { panic("redeclaring using list") }
func (p *usinglist) DeclScope() *Scope { return p.scope }
func (p *usinglist) OwnerProject() *Project { return p.owner }
func (p *usinglist) Type() Type { return UsingListType }
func (p *usinglist) Name() string { return p.name }
func (p *usinglist) Integer() (int64, error) { return 0, nil }
func (p *usinglist) Float() (float64, error) { return 0, nil }
func (p *usinglist) Strval() (s string, err error) {
        for i, elem := range p.list {
                if i > 0 { s += " " }
                s += elem.project.name
        }
        s = fmt.Sprintf("use [%v]", s)
        return
}
func (p *usinglist) String() string {
        var s string
        for i, elem := range p.list {
                if i > 0 { s += ", " }
                s += elem.project.name
        }
        return fmt.Sprintf("{%s uses %v}", p.name, s)
}

func (p *usinglist) append(proj *Project, params []Value) {
        for _, elem := range p.list {
                if elem.project == proj {
                        return
                }
        }
        p.list = append(p.list, &using{ proj, params })
}

func (p *usinglist) Get(name string) (Value, error) {
        //switch name {
        //case "name":
        //}
        return nil, fmt.Errorf("no such property `%s`", name)
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
