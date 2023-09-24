//
//  Copyright (C) 2012-2023, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	"strings"
	"testing"
)

func testTemplate(t *testing.T) {
	var ctx = load_testcase(t, "testdata/template", "testtemplate")
	if ctx.Context == nil {
		t.Errorf("fail")
		return
	}

	if v := ctx.get(".test.1"); v == nil {
		t.Errorf(".test.1")
	} else if v.String() != "xxx yyy zzz xxx yyy zzz" {
		ctx.err("%T %v", v, v)
	}

	if v := ctx.get(".test.2"); v == nil {
		t.Errorf(".test.2")
	} else if s := v.String(); s == "" {
		ctx.err("%T %v", v, v)
	} else if strings.Count(s, "xxx") != 2 {
		ctx.err("%T %v", v, v)
	} else if strings.Count(s, "yyy") != 2 {
		ctx.err("%T %v", v, v)
	} else if strings.Count(s, "zzz") != 2 {
		ctx.err("%T %v", v, v)
	}

	if v := ctx.get(".test.3"); v == nil {
		t.Errorf(".test.3")
	} else if s := v.String(); s == "" {
		ctx.err("%T %v", v, v)
	} else if strings.Count(s, "xxx") != 2 {
		ctx.err("%T %v", v, v)
	} else if strings.Count(s, "yyy") != 2 {
		ctx.err("%T %v", v, v)
	} else if strings.Count(s, "zzz") != 2 {
		ctx.err("%T %v", v, v)
	}

	ctx.flush()
}

func testTemplateForeach(t *testing.T) {
	var ctx = load_testcase(t, "testdata/template/foreach", "testtemplate")
	if ctx.Context == nil {
		t.Errorf("fail")
		return
	}

	if r := ctx.rule(".test.a"); r == nil {
		t.Errorf(".test.a")
	} else if r.String() != ".test.a(1)" {
		ctx.err("%T %v", r, r)
	} else if t, y := r.Entry.(*rule); !y {
		ctx.err("%T %v", r.Entry, r.Entry)
	} else if _, y := t.target.(*barecomp); !y {
		ctx.err("%T %v", t.target, t.target)
	} else if t.target.String() != ".test.a" {
		ctx.err("%T %v", t.target, t.target)
	} else if len(t.program_) != 1 {
		ctx.err("%v: %v", t.target, t.program_)
	} else if len(t.program_[0].depends) != 2 {
		ctx.err("%v: %v", t.target, t.program_[0].depends)
	} else if t.program_[0].depends[0].String() != "a" {
		ctx.err("%v: %v", t.target, t.program_[0].depends[0])
	} else if t.program_[0].depends[1].String() != "$(foreach a d e f,foo=$_)" {
		ctx.err("%v: %v", t.target, t.program_[0].depends[1])
	}

	if r := ctx.rule(".test.b"); r == nil {
		t.Errorf(".test.b")
	} else if r.String() != ".test.b(1)" {
		ctx.err("%T %v", r, r)
	} else if t, y := r.Entry.(*rule); !y {
		ctx.err("%T %v", r.Entry, r.Entry)
	} else if _, y := t.target.(*barecomp); !y {
		ctx.err("%T %v", t.target, t.target)
	} else if t.target.String() != ".test.b" {
		ctx.err("%T %v", t.target, t.target)
	} else if len(t.program_) != 1 {
		ctx.err("%v: %v", t.target, t.program_)
	} else if len(t.program_[0].depends) != 2 {
		ctx.err("%v: %v", t.target, t.program_[0].depends)
	} else if t.program_[0].depends[0].String() != "b" {
		ctx.err("%v: %v", t.target, t.program_[0].depends[0])
	} else if t.program_[0].depends[1].String() != "$(foreach b d e f,foo=$_)" {
		ctx.err("%v: %v", t.target, t.program_[0].depends[1])
	}

	if r := ctx.rule(".test.c"); r == nil {
		t.Errorf(".test.b")
	} else if r.String() != ".test.c(1)" {
		ctx.err("%T %v", r, r)
	} else if t, y := r.Entry.(*rule); !y {
		ctx.err("%T %v", r.Entry, r.Entry)
	} else if _, y := t.target.(*barecomp); !y {
		ctx.err("%T %v", t.target, t.target)
	} else if t.target.String() != ".test.c" {
		ctx.err("%T %v", t.target, t.target)
	} else if len(t.program_) != 1 {
		ctx.err("%v: %v", t.target, t.program_)
	} else if len(t.program_[0].depends) != 2 {
		ctx.err("%v: %v", t.target, t.program_[0].depends)
	} else if t.program_[0].depends[0].String() != "c" {
		ctx.err("%v: %v", t.target, t.program_[0].depends[0])
	} else if t.program_[0].depends[1].String() != "$(foreach c d e f,foo=$_)" {
		ctx.err("%v: %v", t.target, t.program_[0].depends[1])
	}

	ctx.flush()
}
