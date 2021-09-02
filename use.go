//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        //"time"
        "fmt"
)

// FIXME: locking for MT processing
var usingPrepared = make(map[*Project]int)

type using struct {
        valbase
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
func (p *using) defs(s string) (res []*Def) {
    for _, a := range p.params {
        res = append(res, a.defs(s)...)
    }
    return
}
func (p *using) expandible(w expandwhat) (res bool) {
        for _, a := range p.params {
                if res = a.expandible(w); res { return }
        }
        return
}
func (p *using) expand(w expandwhat) (res Value, err error) {
        var ( params []Value; num int )
        if params, num, err = expandall2(w, p.params...); err != nil {
                diag.errorAt(p.position, "%v", err).debug(1)
                return
        } else if num > 0 {
                return &using{p.valbase,p.project,params,p.opts}, nil
        }
        return
}
func (p *using) stat(t *traversal) (si *statinfo) {
        if entry := p.project.DefaultEntry(); entry != nil {
                // FIXME: entry maybe not pointing to the real target
                si = entry.stat(t)
        }
        return
}
func (p *using) traverse(pc *traversal) {
        diag.errorAt(p.position, "cant traverse 'using' %v", p.project).debug(1)
}
func (p *using) _traverse(pc *traversal) {
        if optionTraceTraversal { defer un(tt(t_traverse, pc, p)) }
        if _, done := usingPrepared[p.project]; done {
                usingPrepared[p.project] += 1
                // FIXME: allow re-using the project
                return
        }
        if entry := p.project.DefaultEntry(); entry != nil {
                if p.project.opts.breakUseLoop {
                        // FIXME: break use loop
                } else if entry.traverse(pc); pc.hasBreakers() {
                        // ...
                } else {
                        usingPrepared[p.project] += 1
                }
        }
        return
}
func (p *using) stamp(pc *traversal) (files []*File, err error) {
        if entry := p.project.DefaultEntry(); entry != nil {
                files, err = entry.stamp(pc)
        }
        return
}
func (p *using) cmp(v Value) (res cmpres) {
        if a, ok := v.(*using); ok {
                assert(ok, "value is not using")
                if p.project == a.project {
                        res = cmpEqual
                }
        }
        return
}
func (p *using) patterned() bool { return false }
func (p *using) match(i interface{}) (full bool, s string, stems []string) { return }
func (p *using) stencil(stems []string) (s string, rest []string) { return }
func (p *using) True() (bool, error) { return p.project != nil, nil }
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
func (p *usinglist) defs(s string) (res []*Def) {
    for _, a := range p.list {
        res = append(res, a.defs(s)...)
    }
    return
}
func (p *usinglist) expandible(w expandwhat) (res bool) {
        for _, a := range p.list {
                if res = a.expandible(w); res { break }
        }
        return
}
func (p *usinglist) expand(w expandwhat) (res Value, err error) {
        var ( list []*using; num int )
        for _, elem := range p.list {
                var v Value
                if v, err = elem.expand(w); err != nil {
                        diag.errorAt(elem.position, "%v", err).debug(1)
                        return
                } else {
                        if !isNil(v) { v = elem } else if v != elem { num += 1 }
                        list = append(list, v.(*using))
                }
        }
        if num > 0 {
                return &usinglist{ p.name, p.scope, p.owner, list }, nil
        }
        return
}
func (p *usinglist) traverse(pc *traversal) {
        diag.errorAt(p.list[0].position, "cant traverse 'usinglist'").debug(1)
}
func (p *usinglist) _traverse(pc *traversal) {
        if optionTraceTraversal { defer un(tt(t_traverse, pc, p)) }
        for _, elem := range p.list {
                if elem.traverse(pc); pc.hasBreakers() { break }
        }
        return
}
func (p *usinglist) redecl(scope *Scope) { panic("redeclaring using list") }
func (p *usinglist) DeclScope() *Scope { return p.scope }
func (p *usinglist) OwnerProject() *Project { return p.owner }
func (p *usinglist) Position() (pos Position) {
        if len(p.list) > 0 {
                pos = p.list[0].Position()
        }
        return
}
func (p *usinglist) True() (bool, error) { return len(p.list) > 0, nil }
func (p *usinglist) Name() string { return p.name }
func (p *usinglist) Integer() (int64, error) { return 0, nil }
func (p *usinglist) Float() (float64, error) { return 0, nil }
func (p *usinglist) stat(t *traversal) (si *statinfo) {
        if len(p.list) > 0 {
                for _, elem := range p.list {
                        if ei := elem.stat(t); ei == nil {
                                // FIXME: insert new statinfo or just discard it ??
                        } else if si == nil {
                                si = ei
                        } else {
                                si.next = ei
                        }
                }
        }
        return
}
func (p *usinglist) stamp(pc *traversal) (files []*File, err error) {
        for _, elem := range p.list {
                var a []*File
                if a, err = elem.stamp(pc); err != nil { break }
                files = append(files, a...)
        }
        return
}
func (p *usinglist) tryTraverse(t *traversal) (bool, []*breaker) { return false, nil }
func (p *usinglist) cmp(v Value) (res cmpres) {
        if a, ok := v.(*usinglist); ok {
                assert(ok, "value is not usinglist")
                if p.name == a.name && p.owner == a.owner {
                        res = cmpEqual
                }
        }
        return
}
func (p *usinglist) patterned() bool { return false }
func (p *usinglist) match(i interface{}) (full bool, s string, stems []string) { return }
func (p *usinglist) stencil(stems []string) (s string, rest []string) { return }
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

func (p *usinglist) append(pos Position, proj *Project, params []Value, opts useoptions) {
        for _, elem := range p.list {
                if elem.project == proj {
                        return
                }
        }
        p.list = append(p.list, &using{valbase{pos},proj,params,opts})
}

func (p *usinglist) Get(name string) (result Value, err error) {
        var list []Value
        for _, usee := range p.list {
                // Lookup only the project specific exported names. Don't use
                // scope.Find(...) invocation!
                obj := usee.project.scope.Lookup("using."+name)
                if !isNil(obj) { list = append(list, obj) }
        }
        /*if list == nil && err == nil {
                err = fmt.Errorf("no such property `%s` (%v)", name, p)
        }*/
        if err != nil {
                // failed
        } else if len(list) > 0 {
                result = MakeListOrScalar(p.Position(), list)
        } else {
                result = MakeNone(p.Position())
        }
        return
}

func (p *usinglist) Call(pos Position, a... Value) (result Value) {
        var targets []Value
        for _, usee := range p.list {
                if entry := usee.project.DefaultEntry(); entry != nil {
                        if usee.project.opts.breakUseLoop {
                                // FIXME: break use loop
                        } else if false {
                                usingPrepared[usee.project] += 1
                        }
                        targets = append(targets, entry)
                }
        }
        if len(targets) > 0 {
                result = MakeListOrScalar(p.Position(), targets)
        }
        return
}
