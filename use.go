//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
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
func (p *use) _cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*use); ok {
        assert(ok, "value is not use")
        if p.project == a.project {
            res = cmpEqual
        }
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

type uselist struct {
    owner_ *project
    name string
    scope *scope
    list []*use
}
func (_ *uselist) kind() Kind { return KindUse }
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
func (p *uselist) _cmp(ctx Context, v Value) (res cmpres) {
    if a, y := v.(*uselist); y { assert(y, "value is not uselist")
        if p.name == a.name && p.owner_ == a.owner_ { res = cmpEqual }
    }
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

func (p *uselist) _invoke(ctx Context, o, a []Value) (result Value) {
    var targets []Value
    if p.list != nil {
        for _, usee := range p.list {
            if entry := usee.project.main; entry != nil {
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
