//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "extbit.io/smart/scanner"
        "extbit.io/smart/token"
        "fmt"
)

// FIXME: locking for MT processing
var usingPrepared = make(map[*Project]int)

type using struct {
        project *Project
        params []Value
        opts useoptions
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
                return &using{ p.project, params, p.opts }, nil
        }
        return p, nil
}
func (p *using) traverse(pc *traversal) (err error) {
        if optionTracePrepare { defer prepun(preptrace(pc, p)) }
        if _, done := usingPrepared[p.project]; done {
                usingPrepared[p.project] += 1
                // FIXME: allow re-using the project
                return
        }
        if entry := p.project.DefaultEntry(); entry != nil {
                if err = entry.traverse(pc); err != nil {
                        //fmt.Fprintf(os.Stderr, "%s: `%s` using error: %s (default entry '%s')\n", entry.Position, p.project.name, err, entry.target)
                        if br, ok := err.(*breaker); ok {
                                switch br.what {
                                case breakGood, breakUpdates:
                                        return nil
                                }
                        }
                        err = scanner.WrapErrors(token.Position(entry.Position), err)
                } else {
                        usingPrepared[p.project] += 1
                }
        }
        return
}
func (p *using) cmp(v Value) (res cmpres) {
        if v.Type() == UsingType {
                a, ok := v.(*using)
                assert(ok, "value is not using")
                if p.project == a.project {
                        res = cmpEqual
                }
        }
        return
}
func (p *using) Type() Type { return UsingType }
func (p *using) True() bool { return p.project != nil }
func (p *using) Integer() (i int64, err error) { return 0, nil }
func (p *using) Float() (f float64, err error) { return 0, nil }
func (p *using) String() string {
        if len(p.params) > 0 {
                return fmt.Sprintf("%s(%v)", p.project.name, p.params)
        } else {
                return fmt.Sprintf("%s", p.project.name)
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
func (p *usinglist) traverse(pc *traversal) error {
        if optionTracePrepare { defer prepun(preptrace(pc, p)) }
        for _, elem := range p.list {
                if err := elem.traverse(pc); err != nil {
                        return err
                }
        }
        return nil
}
func (p *usinglist) redecl(scope *Scope) { panic("redeclaring using list") }
func (p *usinglist) DeclScope() *Scope { return p.scope }
func (p *usinglist) OwnerProject() *Project { return p.owner }
func (p *usinglist) Type() Type { return UsingListType }
func (p *usinglist) True() bool { return len(p.list) > 0 }
func (p *usinglist) Name() string { return p.name }
func (p *usinglist) Integer() (int64, error) { return 0, nil }
func (p *usinglist) Float() (float64, error) { return 0, nil }
func (p *usinglist) cmp(v Value) (res cmpres) {
        if v.Type() == UsingListType {
                a, ok := v.(*usinglist)
                assert(ok, "value is not usinglist")
                if p.name == a.name && p.owner == a.owner {
                        res = cmpEqual
                }
        }
        return
}
func (p *usinglist) Strval() (s string, err error) {
        for i, elem := range p.list {
                if i > 0 { s += " " }
                s += elem.project.name
        }
        s = fmt.Sprintf("[%v]", s)
        return
}
func (p *usinglist) String() string {
        var s string
        for i, elem := range p.list {
                if i > 0 { s += "," }
                s += elem.project.name
        }
        return fmt.Sprintf("%s", s)
}

func (p *usinglist) append(proj *Project, params []Value, opts useoptions) {
        for _, elem := range p.list {
                if elem.project == proj {
                        return
                }
        }
        p.list = append(p.list, &using{ proj, params, opts })
}

func (p *usinglist) Get(name string) (result Value, err error) {
        var list []Value
        for _, usee := range p.list {
                // Lookup only the project specific exported names. Don't use
                // scope.Find(...) invocation!
                obj := usee.project.scope.Lookup("using."+name)
                if obj != nil { list = append(list, obj) }
        }
        /*if list == nil && err == nil {
                err = fmt.Errorf("no such property `%s` (%v)", name, p)
        }*/
        if err == nil {
                if list != nil {
                        result = MakeListOrScalar(list)
                } else {
                        result = universalnone
                }
        }
        return
}

func (p *usinglist) Call(pos Position, a... Value) (res Value, err error) {
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
