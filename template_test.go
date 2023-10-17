//
//  Copyright (C) 2012-2023, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	"strings"
)

func testTemplate(ctx *testcase) {
	if s := ".test.1"; false {
	} else if v := ctx.get(s); v == nil {
		ctx.err(s)
	} else if v.String() != "xxx yyy zzz xxx yyy zzz" {
		ctx.err("%v", v)
	} else if v.string(ctx) != "xxx yyy zzz xxx yyy zzz" {
		ctx.err("%v", v)
	}

	if s := ".test.2"; false {
	} else if v := ctx.get(s); v == nil {
		ctx.err(s)
	} else if s := v.String(); s == "" {
		ctx.err("%v", v)
	} else if strings.Count(s, "xxx") != 2 {
		ctx.err("%v", v)
	} else if strings.Count(s, "yyy") != 2 {
		ctx.err("%v", v)
	} else if strings.Count(s, "zzz") != 2 {
		ctx.err("%v", v)
	} else if s := v.string(ctx); s == "" {
		ctx.err("%v", v)
	} else if strings.Count(s, "xxx") != 2 {
		ctx.err("%v", v)
	} else if strings.Count(s, "yyy") != 2 {
		ctx.err("%v", v)
	} else if strings.Count(s, "zzz") != 2 {
		ctx.err("%v", v)
	}

	if s := ".test.3"; false {
	} else if v := ctx.get(s); v == nil {
		ctx.err(s)
	} else if s := v.String(); s == "" {
		ctx.err("%v", v)
	} else if strings.Count(s, "xxx") != 2 {
		ctx.err("%v", v)
	} else if strings.Count(s, "yyy") != 2 {
		ctx.err("%v", v)
	} else if strings.Count(s, "zzz") != 2 {
		ctx.err("%v", v)
	}

	if s := ".test.4"; false {
	} else if v := ctx.get(s); v == nil {
		ctx.err(s)
	} else if s := v.String(); s == "" {
		ctx.err("%v", v)
	} else if len(strings.Fields(s)) != 9 {
		ctx.err("%v", v)
	} else if strings.Count(s, "a=x") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "a=y") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "a=z") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "b=x") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "b=y") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "b=z") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "c=x") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "c=y") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "c=z") != 1 {
		ctx.err("%v", v)
	} else if s != "a=x a=y a=z b=x b=y b=z c=x c=y c=z" {
		ctx.err("%v", v)
	}

	if s := ".test.5"; false {
	} else if v := ctx.get(s); v == nil {
		ctx.err(s)
	} else if s := v.String(); s == "" {
		ctx.err("%v", v)
	} else if len(strings.Fields(s)) != 6 {
		ctx.err("%v", v)
	} else if strings.Count(s, "a=x") != 2 {
		ctx.err("%v", v)
	} else if strings.Count(s, "a=y") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "b=y") != 2 {
		ctx.err("%v", v)
	} else if strings.Count(s, "b=x") != 1 {
		ctx.err("%v", v)
	} else if s != "a=x a=y a=x b=y b=x b=y" {
		ctx.err("%v", v)
	}

	if s := ".test.6"; false {
	} else if v := ctx.get(s); v == nil {
		ctx.err(s)
	} else if s := v.String(); s == "" {
		ctx.err("%v", v)
	} else if len(strings.Fields(s)) != 6 {
		ctx.err("%v", v)
	} else if strings.Count(s, "a=x") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "a=y") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "b=x") != 2 {
		ctx.err("%v", v)
	} else if strings.Count(s, "c=x") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "c=y") != 1 {
		ctx.err("%v", v)
	} else if s != "a=x a=y b=x b=x c=y c=x" {
		ctx.err("%v", v)
	}

	if s := ".test.7"; false {
	} else if v := ctx.get(s); v == nil {
		ctx.err(s)
	} else if s := v.String(); s == "" {
		ctx.err("%v", v)
	} else if len(strings.Fields(s)) != 9+9 {
		ctx.err("%v", v)
	} else if strings.Count(s, "a1=x a2=x") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "a1=y a2=y") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "a1=z a2=z") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "b1=x b2=x") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "b1=y b2=y") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "b1=z b2=z") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "c1=x c2=x") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "c1=y c2=y") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "c1=z c2=z") != 1 {
		ctx.err("%v", v)
	} else if s != "a1=x a2=x a1=y a2=y a1=z a2=z b1=x b2=x b1=y b2=y b1=z b2=z c1=x c2=x c1=y c2=y c1=z c2=z" {
		ctx.err("%v", v)
	}

	if s := ".test.8"; false {
	} else if v := ctx.get(s); v == nil {
		ctx.err(s)
	} else if s := v.String(); s == "" {
		ctx.err("%v", v)
	} else if len(strings.Fields(s)) != 9+9 {
		ctx.err("%v", v)
	} else if strings.Count(s, "a1=x a2=x") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "a1=y a2=y") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "a1=z a2=z") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "b1=x b2=x") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "b1=y b2=y") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "b1=z b2=z") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "=x c2=x") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "=y c2=y") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "=z c2=z") != 1 {
		ctx.err("%v", v)
	} else if s != "a1=x a2=x a1=y a2=y a1=z a2=z b1=x b2=x b1=y b2=y b1=z b2=z =x c2=x =y c2=y =z c2=z" {
		ctx.err("%v", v)
	}

	if s := ".test.9"; false {
	} else if v := ctx.get(s); v == nil {
		ctx.err(s)
	} else if s := v.String(); s == "" {
		ctx.err("%v", v)
	} else if len(strings.Fields(s)) != 9+9 {
		ctx.err("%v", v)
	} else if strings.Count(s, "a1=x a2=x") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "a1=y a2=y") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "a1=z a2=z") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "b1=x b2=x") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "b1=y b2=y") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "b1=z b2=z") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "c1=x =x") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "c1=y =y") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "c1=z =z") != 1 {
		ctx.err("%v", v)
	} else if s != "a1=x a2=x a1=y a2=y a1=z a2=z b1=x b2=x b1=y b2=y b1=z b2=z c1=x =x c1=y =y c1=z =z" {
		ctx.err("%v", v)
	}

	if s := ".test.10"; false {
	} else if v := ctx.get(s); v == nil {
		ctx.err(s)
	} else if s := v.String(); s == "" {
		ctx.err("%v", v)
	} else if len(strings.Fields(s)) != 9+9 {
		ctx.err("%v", v)
	} else if strings.Count(s, "a1=x1 a2=x2") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "a1=y1 a2=y2") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "a1= a2=z2") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "b1=x1 b2=x2") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "b1=y1 b2=y2") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "b1= b2=z2") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "c1=x1 =x2") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "c1=y1 =y2") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "c1= =z2") != 1 {
		ctx.err("%v", v)
	} else if s != "a1=x1 a2=x2 a1=y1 a2=y2 a1= a2=z2 b1=x1 b2=x2 b1=y1 b2=y2 b1= b2=z2 c1=x1 =x2 c1=y1 =y2 c1= =z2" {
		ctx.err("%v", v)
	}
}

func testTemplateForeach(ctx *testcase) {
	if s := ".test.a"; false {
	} else if r := ctx.rule(s); r == nil {
		ctx.err(s)
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

	if s := ".test.b"; false {
	} else if r := ctx.rule(s); r == nil {
		ctx.err(s)
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

	if s := ".test.c"; false {
	} else if r := ctx.rule(s); r == nil {
		ctx.err(s)
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
}
