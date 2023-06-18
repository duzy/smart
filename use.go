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
var usePrepared = make(map[*Project]int)

type use struct {
        valbase
        project *Project
        params []Value
        opts useOpts
}

func (_ *use) Kind() Kind { return KindUse }
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
func (p *use) expandable(ctx Context, w facet) (res bool) {
        for _, a := range p.params {
                if res = a.expandable(ctx, w); res { return }
        }
        return
}
func (p *use) expand(ctx Context, w facet) (res Value) {
        var params, une, num = w.expand(ctx, p.params...)
        if num > 0 { res = &use{p.valbase,p.project,params,p.opts} } else { res = p }
        if une > 0 { res = unexpanded{res} }
        return
}
func (p *use) stat(ctx Context) (si *statinfo) {
        if entry := p.project.defaultEntry; entry != nil {
                // FIXME: entry maybe not pointing to the real target
                si = entry.stat(ctx)
        }
        return
}
func (p *use) traverse(ctx Context) {
        erro(at(ctx,p.position), "cant traverse 'use' %v", p.project).debug(1)
        return
}
/*func (p *use) _traverse(pc *traversal) {
        if options.traceTraversal { defer un(tt(t_traverse, pc, p)) }
        if _, done := usePrepared[p.project]; done {
                usePrepared[p.project] += 1
                // FIXME: allow re-use the project
                return
        }
        if entry := p.project.defaultEntry; entry != nil {
                if p.project.opts.traveUseLoop {
                        // FIXME: break use loop
                } else if entry.traverse(pc); pc.hasTravestates() {
                        // ...
                } else {
                        usePrepared[p.project] += 1
                }
        }
        return
}*/
func (p *use) stamp(ctx Context) (files []*File, err error) {
        if entry := p.project.defaultEntry; entry != nil {
                files, err = entry.stamp(ctx)
        }
        return
}
func (p *use) delete(ctx Context) (files []*File, err error) {
        if entry := p.project.defaultEntry; entry != nil {
                files, err = entry.delete(ctx)
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
func (p *use) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) { return }
func (p *use) stencil(ctx Context, stems []string) (val Value, rest []string) { return }
func (p *use) True(ctx Context) bool { return p.project != nil }
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
func (p *use) Strval(ctx Context) (s string) {
        s = fmt.Sprintf("use %s %v", p.project.name, p.params)
        return
}
func (_ *use) hit(ctx Context, cache hitch, bits int) (res *filemapCache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *use) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *use) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "collect unsupported: %v", cache).debug(32)
    return
}

type uselist struct {
        name string
        scope *Scope
        owner *Project
        list []*use
}
func (_ *uselist) Kind() Kind { return KindUseList }
func (p *uselist) Name(_ Context) string { return p.name }
func (p *uselist) DeclScope() *Scope { return p.scope }
func (p *uselist) OwnerProject() *Project { return p.owner }
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
func (p *uselist) Strval(ctx Context) (s string) {
        for i, elem := range p.list {
                if i > 0 { s += " " }
                s += elem.project.name
        }
        s = fmt.Sprintf("[%v]", s)
        return
}
func (p *uselist) True(ctx Context) bool { return len(p.list) > 0 }
func (p *uselist) Integer(ctx Context) (i int64, _ error) { return int64(len(p.list)), nil }
func (p *uselist) Float(ctx Context) (f float64, _ error) { return 0, nil }
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
func (p *uselist) stamp(ctx Context) (files []*File, err error) {
        for _, elem := range p.list {
                var a []*File
                if a, err = elem.stamp(ctx); err != nil { break }
                files = append(files, a...)
        }
        return
}
func (p *uselist) delete(ctx Context) (files []*File, err error) {
        for _, elem := range p.list {
                var a []*File
                if a, err = elem.delete(ctx); err != nil { break }
                files = append(files, a...)
        }
        return
}
func (p *uselist) cmp(ctx Context, v Value) (res cmpres) {
        if a, ok := v.(*uselist); ok {
                assert(ok, "value is not uselist")
                if p.name == a.name && p.owner == a.owner {
                        res = cmpEqual
                }
        }
        return
}
func (p *uselist) patterned(ctx Context) bool { return false }
func (p *uselist) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) { return }
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
func (p *uselist) expandable(ctx Context, w facet) (res bool) {
        for _, a := range p.list {
                if res = a.expandable(ctx, w); res { break }
        }
        return
}
func (p *uselist) expand(ctx Context, w facet) (res Value) {
        var ( list []*use; num int )
        for _, elem := range p.list {
                var v = elem.expand(ctx, w)
                if v != elem { num += 1 }
                list = append(list, v.(*use))
        }
        if num > 0 {
                res = &uselist{ p.name, p.scope, p.owner, list }
        } else {
                res = p
        }
        return
}
func (p *uselist) traverse(ctx Context) {
        erro(at(ctx,p.list[0].position), "cant traverse 'uselist'").debug(1)
        return
}
/*func (p *uselist) _traverse(pc *traversal) {
        if options.traceTraversal { defer un(tt(t_traverse, pc, p)) }
        for _, elem := range p.list {
                if elem.traverse(pc); pc.hasTravestates() { break }
        }
        return
}*/
func (p *uselist) rescope(ctx Context, scope *Scope) { panic("rescoping use list") }
func (p *uselist) append(ctx Context, proj *Project, params []Value, opts useOpts) {
        for _, elem := range p.list {
                if elem.project == proj {
                        return
                }
        }
        p.list = append(p.list, &use{valbase{ctx.Position()},proj,params,opts})
}

func (p *uselist) Get(ctx Context, name string) (result Value, err error) {
        var list []Value
        for _, usee := range p.list {
                if usee.opts.noVars { continue }
                //if _, obj := usee.project.scope.Find("use."+name); !isNil(obj) {
                if obj := usee.project.scope.Lookup("use."+name); !isNil(obj) {
                        list = append(list, obj)
                }
        }
        if err != nil {
                // failed
        } else if len(list) > 0 {
                result = MakeListOrScalar(p.Position(), list)
        } else {
                result = MakeNone(p.Position())
        }
        return
}

func (p *uselist) Call(ctx Context, _ []Value, a... Value) (result Value) {
        if p.list == nil { return }
        var targets []Value
        for _, usee := range p.list {
                if entry := usee.project.defaultEntry; entry != nil {
                        if usee.project.opts.traveUseLoop {
                                // FIXME: break use loop
                        } else if false {
                                usePrepared[usee.project] += 1
                        }
                        targets = append(targets, entry)
                }
        }
        if len(targets) > 0 {
                result = MakeListOrScalar(p.Position(), targets)
        }
        return
}

func (_ *uselist) hit(ctx Context, cache hitch, bits int) (res *filemapCache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *uselist) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *uselist) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "collect unsupported: %v", cache).debug(32)
    return
}
