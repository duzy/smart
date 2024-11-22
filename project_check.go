//
//  Copyright (C) 2012-2024, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	"strings"
	"path/filepath"
)

func (p *project) configuration_sm_check(ctx Context, f *file) {
	var srcdir = dirs(1, get_filename(1))
	var workout, triple, out, rel, tag, tmp string
	switch p.name {
	case "general":
		if p.configure != nil {
			erro(ctx, "%v : wrong configure", p).trace()
		}
		if !strings.HasSuffix(p.absPath, ".smart/modules/general") {
			erro(ctx, "%v : %v", p, p.absPath).trace()
		}
		if !strings.HasSuffix(p.spec, ".smart/modules/general") {
			erro(ctx, "%v : %v", p, p.spec).trace()
		}
		if strings.HasPrefix(p.absPath, "/Volumes/workspace/") {
			if d := p.def(ctx, "workspace"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, d).trace()
			} else if ts(d.value) != "{=path {=punct root} {=word Volumes} {=word workspace}}" {
				erro(ctx, "%v : %v", p, ts(d.value)).trace()
			}
			if d := p.def(ctx, "workout"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, d).trace()
			} else if ts(d.value) != "{=path {=punct root} {=word Volumes} {=word workout}}" {
				erro(ctx, "%v : %v", p, ts(d.value)).trace()
			} else {
				workout = d.string(ctx)
			}
			if d := p.def(ctx, "workext"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, d).trace()
			} else if ts(d.value) != "{=path {=punct root} {=word Volumes} {=word workspace} {=word external}}" {
				erro(ctx, "%v : %v", p, ts(d.value)).trace()
			}
			if d := p.def(ctx, "rel.chop"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, d).trace()
			} else if ts(d.value) != "{=list {=path {=percpat {=null} {=percpat {=null} {=null}}} {=compound {=punct .} {=word smart}} {=word modules} {=punct tail}} {=path {=punct root} {=word Volumes} {=word workspace} {=word .smart} {=word modules} {=punct tail}} {=path {=punct root} {=word Volumes} {=word workspace} {=word .smart} {=punct tail}} {=path {=punct root} {=word Volumes} {=word workspace} {=punct tail}}}" {
				erro(ctx, "%v : %v", p, ts(d.value)).trace()
			}
			if d := p.def(ctx, "rel.remnant"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, d).trace()
			} else if ts(d.value) != "{=delegate {=builtin trim-prefix} {=list {=closure {=def rel.chop}}} {=list {=closure {=def /}}}}" {
				erro(ctx, "%v : %v", p, ts(d.value)).trace()
			} else if s := d.string(ctx); s == "" {
				erro(ctx, "%v : %v", p, ts(d)).trace()
			} else {
				switch proj := _project(ctx); proj.name {
				case "testdefaultconfigure":
					if t := filepath.Join("testdata", "configuration"); s != t {
						erro(ctx, "%v : %v : %v : %s != %s", p, proj, ts(d), s, t).trace()
					}
				case "testdeftwoconfigure":
					if t := filepath.Join("testdata", "configuration", "two"); s != t {
						erro(ctx, "%v : %v : %v : %s != %s", p, proj, ts(d), s, t).trace()
					}
				}
			}
			if d := p.def(ctx, "outtmp"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, d).trace()
			} else if ts(d.value) != "{=path {=punct root} {=word Volumes} {=word workout} {=closure {=def rel.remnant}}}" {
				erro(ctx, "%v : %v", p, ts(d.value)).trace()
			} else if d.value.String() != "/Volumes/workout/&(rel.remnant)" {
				erro(ctx, "%v : %v", p, d.value).trace()
			} else if s := d.string(ctx); s == "" {
				erro(ctx, "%v : %v", p, ts(d)).trace()
			} else {
				switch proj := _project(ctx); proj.name {
				case "testdefaultconfigure":
					if t := filepath.Join(workout, "testdata", "configuration"); s != t {
						erro(ctx, "%v : %v : %v : %s != %s", p, proj, ts(d), s, t).trace()
					}
				case "testdeftwoconfigure":
					if t := filepath.Join(workout, "testdata", "configuration", "two"); s != t {
						erro(ctx, "%v : %v : %v : %s != %s", p, proj, ts(d), s, t).trace()
					}
				}
			}
		}
	case "configure.base":
		if p.configure != nil {
			erro(ctx, "%v : wrong configure", p).trace()
		}
	case "lib.c++.abi":
		if p.configure == nil {
			erro(ctx, "%v : nil configure", p).trace()
		}
		if p.configure.name != "configure.base" {
			erro(ctx, "%v : %v", p, p.configure).trace()
		}
	case "lib.c++.inc":
		if p.configure == nil {
			erro(ctx, "%v : nil configure", p).trace()
		}
		if p.configure.name != "configure.base" {
			erro(ctx, "%v : %v", p, p.configure).trace()
		}
	case "lib.c++":
		if p.configure != nil {
			erro(ctx, "%v : wrong configure", p).trace()
		}
	case "lib.std":
		if p.configure != nil {
			erro(ctx, "%v : wrong configure", p).trace()
		}
	case "lib.unwind":
		if p.configure != nil {
			erro(ctx, "%v : wrong configure", p).trace()
		}
	case "testdefaultconfigure":
		if p.configure == nil {
			erro(ctx, "%v : nil configure", p).trace()
		}
		if p.configure.name != "configure" {
			erro(ctx, "%v : %v", p, p.configure).trace()
		}
		if !strings.HasSuffix(p.configure.absPath, ".smart/modules/configure") {
			erro(ctx, "%v : %v", p, p.configure.absPath).trace()
		}
		if !strings.HasSuffix(p.configure.spec, ".smart/modules/configure") {
			erro(ctx, "%v : %v", p, p.configure.spec).trace()
		}
		if d := p.def(ctx, "variant"); d == nil || d.value == nil {
			erro(ctx, "%v : %v", p, d).trace()
		} else if ts(d.value) != "{=path {=word darwin} {=word arm64} {=word bootstrap}}" {
			erro(ctx, "%v : %v", p, ts(d)).trace()
		} else if d.string(ctx) != "darwin/arm64/bootstrap" {
			erro(ctx, "%v : %v", p, d.string(ctx)).trace()
		}
		if d := p.def(ctx, "variant.tag"); d == nil || d.value == nil {
			erro(ctx, "%v : %v", p, d).trace()
		} else if ts(d.value) != "{=word bootstrap}" {
			erro(ctx, "%v : %v", p, ts(d)).trace()
		} else if s := d.string(ctx); s != "bootstrap" {
			erro(ctx, "%v : %v", p, s).trace()
		} else {
			tag = s
		}
		if d := p.def(ctx, "outtmp"); d == nil {
			erro(ctx, "outtmp is undefined").trace()
		} else if s, t := d.string(ctx), "/"; s == t {
			erro(ctx, "outtmp: %s != %s", s, t).trace()
		}
		if t, d := p.tempdir(ctx); d == "" {
			erro(ctx, "empty tempdir (%v)", t).trace()
		}
		if f.fullname() == "/configuration.sm" {
			erro(ctx, "wrong fullname: %v %s", f, f.dir).trace()
		}
		if false && strings.HasPrefix(p.absPath, "/Volumes/workspace/") {
			if d := p.def(ctx, "workspace"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, d).trace()
			}
			if d := p.def(ctx, "workout"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, d).trace()
			} else {
				workout = d.string(ctx)
			}
			if d := p.def(ctx, "workext"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, d).trace()
			} else if ts(d.value) == "" {
				erro(ctx, "%v : %v", p, ts(d)).trace()
			}
			if d := p.def(ctx, "rel.chop"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, d).trace()
			} else if s, t := d.string(ctx), srcdir+"/"; s != t {
				erro(ctx, "%v : %s != %s", p, s, t).trace()
			}
			if d := p.def(ctx, "rel.remnant"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, ts(d)).trace()
			} else if ts(d.value) != "{=delegate {=builtin trim-prefix} {=list {=closure {=def rel.chop}}} {=list {=closure {=def /}}}}" {
				erro(ctx, "%v : %v", p, ts(d.value)).trace()
			} else if v := d.value.expand(_final(ctx)); ts(v) != "{=path {=word testdata} {=word configuration}}" {
				erro(ctx, "%v : %v → %v : %s ; %v", p, d.value, v, srcdir, p.def(ctx, "/").value).trace()
			} else if s, t := d.string(ctx), filepath.Join("testdata", "configuration"); s != t {
				erro(ctx, "%v : %s != %s", p, s, t).trace()
			} else {
				rel = s
			}
			if d := p.def(ctx, "target.triple"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, d).trace()
			} else if ts(d.value) != "{=delegate {=builtin join} {=list {=compound {=closure {=def target.arch}} {=closure {=compound {=word target} {=punct .} {=word sub}}}} {=closure {=def target.vendor}} {=closure {=def target.sys}} {=closure {=def target.abi}}} {=list {=flag {=null}}}}" {
				erro(ctx, "%v : %v", p, ts(d.value)).trace()
			} else {
				triple = d.string(ctx)
			}
			if d := p.def(ctx, "target.out"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, d).trace()
			} else if ts(d.value) != "{=path {=punct root} {=word Volumes} {=word workout} {=closure {=def target.triple}} {=closure {=compound {=word variant} {=punct .} {=word tag}}}}" {
				erro(ctx, "%v : %v", p, ts(d.value)).trace()
			} else if s, t := d.string(ctx), filepath.Join(workout, triple, tag); s != t {
				erro(ctx, "%s != %s", s, t).trace()
			} else {
				out = s
			}
			if d := p.def(ctx, "target.tmp"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, d).trace()
			} else if ts(d.value) != "{=path {=closure {=def target.out}} {=word tmp}}" {
				erro(ctx, "%v : %v", p, ts(d.value)).trace()
			} else if s, t := d.string(ctx), filepath.Join(out, "tmp"); s != t {
				erro(ctx, "%s != %s", s, t).trace()
			} else {
				tmp = s
			}
			if d := p.def(ctx, "outtmp"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, d).trace()
			} else if ts(d.value) != "{=path {=closure {=def target.tmp}} {=closure {=def rel.remnant}}}" {
				erro(ctx, "%v : %v", p, ts(d.value)).trace()
			} else if s, t := d.string(ctx), filepath.Join(tmp, rel); s != t {
				erro(ctx, "%s != %s (%s)", s, t, rel).trace()
			}
		}
	case "testdeftwoconfigure":
		if p.configure == nil {
			erro(ctx, "%v : nil configure", p).trace()
		}
		if p.configure.name != "configure" {
			erro(ctx, "%v : %v", p, p.configure).trace()
		}
		if !strings.HasSuffix(p.configure.absPath, ".smart/modules/configure") {
			erro(ctx, "%v : %v", p, p.configure.absPath).trace()
		}
		if !strings.HasSuffix(p.configure.spec, ".smart/modules/configure") {
			erro(ctx, "%v : %v", p, p.configure.spec).trace()
		}
	case "testcustomconfigure":
		if p.configure == nil {
			erro(ctx, "%v : nil configure", p).trace()
		}
		if p.configure.name != "configure" {
			erro(ctx, "%v : %v", p, p.configure).trace()
		}
		if !strings.HasSuffix(p.configure.absPath, "testdata/configuration/custom/configure") {
			erro(ctx, "%v : %v", p, p.configure.absPath).trace()
		}
		if !strings.HasSuffix(p.configure.spec, "testdata/configuration/custom/configure") {
			erro(ctx, "%v : %v", p, p.configure.spec).trace()
		}
		if d := p.configure.def(ctx, "foo"); d == nil || d.value == nil{
			erro(ctx, "%v : %v", p, d).trace()
		} else if ts(d.value) != "{=self configure}" {
			erro(ctx, "%v : %v", p, ts(d.value)).trace()
		}
	}
}

func (p *project) unmap_entries_check(ctx Context, _k any, _res *[]entry) {
	switch res := *_res; p.name {
	case "configure.base":
		switch x := _k.(type) {
		case *word:
		case flag:
			var c, y = p.entries.puncs[MINUS]
			if !y {
				erro(ctx, "%v %v", p.name, _k).trace()
			}

			var ss = c.keys()
			if len(ss) == 0 {
				erro(ctx, "%v %v", p.name, _k).trace()
			}

			var v = x.Value
			var k = v.String()
			if _, y = ss[k]; y {
				if res == nil {
					erro(ctx, "%v: %v %v", p.name, k, c.ks(true)).trace()
				} else if x.String() != res[0].String() {
					erro(ctx, "%v: %v != %v", p.name, v, res[0]).trace()
				}
			} else {
				erro(ctx, "%v: %v", p.name, v).trace()
			}
		default:
			// note(ctx, "%v: %v %v", p.name, ts(x), res).debug()
		}
	}
}

func (p *project) tempdir_check(ctx Context, d *def, s string) {
	switch p.name {
	case "testdefaultconfigure":
		if d.name != "outtmp" {
			erro(ctx, "wrong tempdir: %v", d).trace()
		}
		if t := p.def(ctx, "target.tmp"); t.string(ctx) == "" {
			erro(ctx, "wrong tempdir: %v ; %v", d, t).trace()
		}
		if s == "/" {
			erro(ctx, "wrong tempdir: %v → %s", d, s)

			var t *def

			t = p.def(ctx, "target.tmp")
			note(ctx, "%v → %s", t, t.string(ctx))

			t = p.def(ctx, "target.out")
			note(ctx, "%v → %s", t, t.string(ctx))

			t = p.def(ctx, "target.triple")
			note(ctx, "%v → %s", t, t.string(ctx))

			t = p.def(ctx, "rel.remnant")
			note(ctx, "%v → %s", t, t.string(ctx))

			t = p.def(ctx, "rel.chop")
			note(ctx, "%v → %s", t, t.string(ctx))

			t = p.def(ctx, "variant.tag")
			note(ctx, "%v → %s", t, t.string(ctx)).trace()
		}
	}
}

func (p *project) tempfile_check(ctx Context, name, d string, f *file) {
	if f.dir != d {
		erro(ctx, "%v: %s != %s", p, f.dir, d).trace()
	}
	switch p.name {
	case "testdefaultconfigure":
		var t *def
		if t = p.def(ctx, "outtmp"); t == nil { // outtmp:=&(target.tmp)/&(rel.remnant)
			erro(ctx, "%s: %v : %s", p.name, d, f.dir).trace()
		} else if s, t1 := "&(target.tmp)/&(rel.remnant)", t.value.String(); t1 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t1, s).trace()
		} else if s, t2 := "/", t.string(ctx); t2 == s {
			erro(ctx, "%s: %v : %s", p.name, d, t2).trace()
		} else if t2 != f.dir {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t2, f.dir).trace()
		}
		if t = p.def(ctx, "target.tmp"); t == nil { // target.tmp := &(target.out)/tmp
			erro(ctx, "%s: %v : %s", p.name, d, f.dir).trace()
		} else if s, t1 := "target.tmp:=&(target.out)/tmp", t.String(); t1 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t1, s).trace()
		}
		if t = p.def(ctx, "target.out"); t == nil { // &(target.out)/tmp
			erro(ctx, "%s: %v : %s", p.name, d, f.dir).trace()
		} else if s, t1 := "target.out:=/Volumes/workout/&(target.triple)/&(variant.tag)", t.String(); t1 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t1, s).trace()
		} else if s, t2 := "/Volumes/workout/arm64-apple-Darwin23.2.0-macho/bootstrap", t.string(ctx); t2 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t2, s).trace()
		}
		if t = p.def(ctx, "target.triple"); t == nil { // target.tmp := &(target.out)/tmp
			erro(ctx, "%s: %v : %s", p.name, d, f.dir).trace()
		} else if s, t1 := "target.triple:=$(join &(target.arch)&(target.sub) &(target.vendor) &(target.sys) &(target.abi),-)", t.String(); t1 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t1, s).trace()
		} else if s, t2 := "arm64-apple-Darwin23.2.0-macho", t.string(ctx); t2 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t2, s).trace()
		}
		if t = p.def(ctx, "target.arch"); t == nil {
			erro(ctx, "%s: %v : %s", p.name, d, f.dir).trace()
		} else if s, t1 := "target.arch:=arm64", t.String(); t1 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t1, s).trace()
		}
		if true {
			// ...
		} else if t = p.def(ctx, "target.sub"); t == nil {
			erro(ctx, "%s: %v : %s", p.name, d, f.dir).trace()
		} else if s, t1 := "target.sub:=", t.String(); t1 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t1, s).trace()
		}
		if t = p.def(ctx, "target.vendor"); t == nil {
			erro(ctx, "%s: %v : %s", p.name, d, f.dir).trace()
		} else if s, t1 := "target.vendor:=apple", t.String(); t1 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t1, s).trace()
		}
		if t = p.def(ctx, "target.sys"); t == nil {
			erro(ctx, "%s: %v : %s", p.name, d, f.dir).trace()
		} else if s, t1 := "target.sys:=&(uname.os)&(target.release)", t.String(); t1 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t1, s).trace()
		} else if s, t2 := "Darwin23.2.0", t.string(ctx); t2 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t2, s).trace()
		}
		if t = p.def(ctx, "target.abi"); t == nil {
			erro(ctx, "%s: %v : %s", p.name, d, f.dir).trace()
		} else if s, t1 := "target.abi:=macho", t.String(); t1 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t1, s).trace()
		}
		if t = p.def(ctx, "rel.chop"); t == nil { // rel.chop ::= $(dir3 $/)/
			erro(ctx, "%s: %v : %s", p.name, d, f.dir).trace()
		} else if s, t1 := dirs(2, p.absPath)+"/", t.String(); t1 != "rel.chop::="+s {
			erro(ctx, "%s: %v : %s != rel.chop::=%s", p.name, d, t1, s).trace()
		} else if t2 := t.string(ctx); t2 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t2, s).trace()
		}
		if t = p.def(ctx, "rel.remnant"); t == nil { // rel.remnant = $(trim-prefix &(rel.chop),&/)
			erro(ctx, "%s: %v : %s", p.name, d, f.dir).trace()
		} else if s, t1 := "rel.remnant=$(trim-prefix &(rel.chop),&/)", t.String(); t1 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t1, s).trace()
		} else if s, t2 := bases(2, p.absPath), t.string(ctx); t2 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t2, s).trace()
		}
	}
}

func unmap_check(ctx *unmap, c *valcache, key any) {
	switch p := _project(ctx); p.name {
	case "configure.base":
		switch x := key.(type) {
		case flag:
			var cc, y = c.puncs[MINUS]
			if !y {
				if truly(ctx, unmap_uncheck_y{}) { break }
				errostack(ctx, 16, "%v %v %v", cacheMapping(ctx), x, c.ks(true)).trace()
			}

			var ss = cc.keys()
			if len(ss) == 0 { erro(ctx, "%v", x).trace() }

			var v = x.Value
			var k = v.String()
			if _, y = ss[k]; !y { erro(ctx, "%v", v).trace() }
		}
	}

	spec, _ := do(ctx, get_include_spec{}).(string)

	switch spec {
	case "configure/.base/.template":
		if false {
			note(pc(ctx,key), "%v %v", tv(key), c)
			note(pc(ctx,key), "%v", ts(ctx)).debug()
		}
	}
}
