//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    //"time"
    "fmt"
)

// FIXME: locking for MT processing
var usePrepared = make(map[*project]int)

type use struct {
    valbase
    project *project
    params []Value
    opts useopts
}

func (_ *use) kind() Kind { return KindUse }
func (p *use) hash(ctx Context) uint64 { return fnv1(ctx, p, p.project.name) }
func (p *use) refs(ctx Context, v Value) bool {
    for _, a := range p.params {
        if a.refs(ctx, v) { return true }
    }
    return false
}
func (p *use) defs(ctx Context, s ...string) (res []*def) {
    for _, a := range p.params {
    res = append(res, a.defs(ctx, s...)...)
    }
    return
}
func (p *use) expandable(ctx Context) (res bool) {
    for _, a := range p.params {
        if res = a.expandable(ctx); res { return }
    }
    return
}
func (p *use) expand(ctx Context) (res Value) {
    if params := expand(ctx, p.params...); diff(ctx, params, p.params) {
        res = &use{p.valbase,p.project,params,p.opts}
    } else {
        res = p
    }
    return
}
func (p *use) stat(ctx Context) (_ *statinfo) {
    if entry := p.project.defaultEntry; entry != nil {
        // FIXME: entry maybe not pointing to the real target
        return entry.stat(ctx)
    }
    return
}
func (p *use) traverse(ctx Context) {
    erro(ctx, "cant traverse 'use' %v", p.project).trace()
    return
}
func (p *use) stamp(ctx Context) (_ []*file) {
    if entry := p.project.defaultEntry; entry != nil {
        return entry.stamp(ctx)
    }
    return
}
func (p *use) delete(ctx Context) (_ []*file) {
    if entry := p.project.defaultEntry; entry != nil {
        return entry.delete(ctx)
    }
    return
}
func (p *use) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*use); ok {
        assert(ok, "value is not use")
        if p.project == a.project {
            res = cmpEqual
        }
    }
    return
}
func (p *use) patterned(ctx Context) bool { return false }
func (p *use) match(ctx Context, i any) (full bool, s any, stems []string) { return }
func (p *use) stencil(ctx Context, stems []string) (val Value, rest []string) { return }
func (p *use) true(ctx Context) bool { return p.project != nil }
func (p *use) updated(ctx Context) (res bool) {
    if entry := p.project.defaultEntry; entry != nil {
        res = entry.updated(ctx)
    }
    return
}
func (p *use) updatedDeps(ctx Context, v ...Value) (res []Value) {
    if entry := p.project.defaultEntry; entry != nil {
        res = entry.updatedDeps(ctx, v...)
    }
    return
}
func (p *use) String() string {
    if len(p.params) > 0 {
        return fmt.Sprintf("%s(%v)", p.project.name, p.params)
    } else {
        return fmt.Sprintf("%s", p.project.name)
    }
}
func (p *use) string(ctx Context) (s string) {
    s = fmt.Sprintf("use %s %v", p.project.name, p.params)
    return
}

type uselist struct {
    owner_ *project
    name string
    scope *scope
    list []*use
}
func (_ *uselist) cond() bool { return false }
func (_ *uselist) kind() Kind { return KindUse|KindArray }
func (p *uselist) ident(_ Context) string { return p.name }
func (p *uselist) owner() *project { return p.owner_ }
func (p *uselist) Position() (pos Position) {
    if len(p.list) > 0 {
        pos = p.list[0].Position()
    }
    return
}
func (p *uselist) String() string {
    var s string
    for i, elem := range p.list {
        if i > 0 { s += "," }
        s += elem.project.name
    }
    return fmt.Sprintf("%s", s)
}
func (p *uselist) string(ctx Context) (s string) {
    for i, elem := range p.list {
        if i > 0 { s += " " }
        s += elem.project.name
    }
    s = fmt.Sprintf("[%v]", s)
    return
}
func (p *uselist) hash(ctx Context) uint64 {
    var a []any
    for _, v := range p.list { a = append(a, v) }
    return fnv1(ctx, p, a...)
}
func (p *uselist) true(ctx Context) bool { return len(p.list) > 0 }
func (p *uselist) int(ctx Context) (_ int64) { return int64(len(p.list)) }
func (p *uselist) float(ctx Context) (_ float64) { return }
func (p *uselist) updated(ctx Context) (res bool) {
    for _, elem := range p.list {
        res = res || elem.updated(ctx)
    }
    return
}
func (p *uselist) updatedDeps(ctx Context, v ...Value) (res []Value) {
    for _, elem := range p.list {
        res = append(res, elem.updatedDeps(ctx, v...)...)
    }
    return
}
func (p *uselist) stat(ctx Context) (si *statinfo) {
    if len(p.list) > 0 {
        for _, elem := range p.list {
            if ei := elem.stat(ctx); ei == nil {
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
func (p *uselist) stamp(ctx Context) (files []*file) {
    for _, elem := range p.list {
        files = append(files, elem.stamp(ctx)...)
    }
    return
}
func (p *uselist) delete(ctx Context) (files []*file) {
    for _, elem := range p.list {
        files = append(files, elem.delete(ctx)...)
    }
    return
}
func (p *uselist) cmp(ctx Context, v Value) (res cmpres) {
    if a, y := v.(*uselist); y { assert(y, "value is not uselist")
        if p.name == a.name && p.owner_ == a.owner_ { res = cmpEqual }
    }
    return
}
func (p *uselist) patterned(ctx Context) bool { return false }
func (p *uselist) match(ctx Context, i any) (full bool, s any, stems []string) { return }
func (p *uselist) stencil(ctx Context, stems []string) (val Value, rest []string) { return }
func (p *uselist) refs(ctx Context, v Value) bool {
    for _, a := range p.list {
        if a.refs(ctx, v) { return true }
    }
    return false
}
func (p *uselist) defs(ctx Context, s ...string) (res []*def) {
    for _, a := range p.list {
        res = append(res, a.defs(ctx, s...)...)
    }
    return
}
func (p *uselist) expandable(ctx Context) (res bool) {
    for _, a := range p.list {
        if res = a.expandable(ctx); res { break }
    }
    return
}
func (p *uselist) expand(ctx Context) (res Value) {
    var ( list []*use; num int )
    for _, elem := range p.list {
        var v = elem.expand(ctx)
        if v != elem { num += 1 }
        list = append(list, v.(*use))
    }
    if num > 0 {
        res = &uselist{ p.owner_, p.name, p.scope, list }
    } else {
        res = p
    }
    return
}
func (p *uselist) traverse(ctx Context) {
    erro(ctx, "cant traverse 'uselist'").trace()
    return
}
func (p *uselist) append(ctx Context, proj *project, params []Value, opts useopts) {
    for _, elem := range p.list {
        if elem.project == proj {
            return
        }
    }
    p.list = append(p.list, &use{valbase{_position(ctx)},proj,params,opts})
}

func (p *uselist) sel(ctx Context, name string) (result any) {
    var prefix string
    if m := name_prefix.FindStringSubmatch(name); m != nil {
        prefix, name = m[1], m[3]
    }

    var vals []Value
    var n = prefix+"use."+name
    for _, u := range p.list {
        if u.opts.noVars { continue }
        if o := u.project.Lookup(n); o != nil {
            vals = append(vals, o)
        }
    }
    return vals
}

func (p *uselist) _invoke(ctx Context, o, a []Value) (result Value) {
    var targets []Value
    if p.list != nil {
        for _, usee := range p.list {
            if entry := usee.project.defaultEntry; entry != nil {
                if usee.project.opt.traveUseLoop {
                    // FIXME: break use loop
                } else if false {
                    usePrepared[usee.project] += 1
                }
                targets = append(targets, entry)
            }
        }
        result = ease(ctx, targets)
    }
    return
}
