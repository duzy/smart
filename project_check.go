//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

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
			if d := p.resolveDef(ctx, "workspace"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, d).trace()
			} else if ts(d.value) != "{=path {=punct root} {=word Volumes} {=word workspace}}" {
				erro(ctx, "%v : %v", p, ts(d.value)).trace()
			}
			if d := p.resolveDef(ctx, "workout"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, d).trace()
			} else if ts(d.value) != "{=path {=punct root} {=word Volumes} {=word workout}}" {
				erro(ctx, "%v : %v", p, ts(d.value)).trace()
			} else {
				workout = __string(ctx, d)
			}
			if d := p.resolveDef(ctx, "workext"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, d).trace()
			} else if ts(d.value) != "{=path {=punct root} {=word Volumes} {=word workspace} {=word external}}" {
				erro(ctx, "%v : %v", p, ts(d.value)).trace()
			}
			if d := p.resolveDef(ctx, "rel.chop"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, d).trace()
			} else if ts(d.value) != "{=list {=path {=percpat {=null} {=percpat {=null} {=null}}} {=compound {=punct .} {=word smart}} {=word modules} {=punct tail}} {=path {=punct root} {=word Volumes} {=word workspace} {=word .smart} {=word modules} {=punct tail}} {=path {=punct root} {=word Volumes} {=word workspace} {=word .smart} {=punct tail}} {=path {=punct root} {=word Volumes} {=word workspace} {=punct tail}}}" {
				erro(ctx, "%v : %v", p, ts(d.value)).trace()
			}
			if d := p.resolveDef(ctx, "rel.remnant"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, d).trace()
			} else if ts(d.value) != "{=delegate {=builtin trim-prefix} {=list {=closure {=def rel.chop}}} {=list {=closure {=def /}}}}" {
				erro(ctx, "%v : %v", p, ts(d.value)).trace()
			} else if s := __string(ctx, d); s == "" {
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
			if d := p.resolveDef(ctx, "outtmp"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, d).trace()
			} else if ts(d.value) != "{=path {=punct root} {=word Volumes} {=word workout} {=closure {=def rel.remnant}}}" {
				erro(ctx, "%v : %v", p, ts(d.value)).trace()
			} else if d.value.String() != "/Volumes/workout/&(rel.remnant)" {
				erro(ctx, "%v : %v", p, d.value).trace()
			} else if s := __string(ctx, d); s == "" {
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
		if d := p.resolveDef(ctx, "variant"); d == nil || d.value == nil {
			erro(ctx, "%v : %v", p, d).trace()
		} else if ts(d.value) != "{=path {=word darwin} {=word arm64} {=word bootstrap}}" {
			erro(ctx, "%v : %v", p, ts(d)).trace()
		} else if __string(ctx, d) != "darwin/arm64/bootstrap" {
			erro(ctx, "%v : %v", p, __string(ctx, d)).trace()
		}
		if d := p.resolveDef(ctx, "variant.tag"); d == nil || d.value == nil {
			erro(ctx, "%v : %v", p, d).trace()
		} else if ts(d.value) != "{=word bootstrap}" {
			erro(ctx, "%v : %v", p, ts(d)).trace()
		} else if s := __string(ctx, d); s != "bootstrap" {
			erro(ctx, "%v : %v", p, s).trace()
		} else {
			tag = s
		}
		if d := p.resolveDef(ctx, "outtmp"); d == nil {
			erro(ctx, "outtmp is undefined").trace()
		} else if s, t := __string(ctx, d), "/"; s == t {
			erro(ctx, "outtmp: %s != %s", s, t).trace()
		}
		if t, d := p.tempdir(ctx); d == "" {
			erro(ctx, "empty tempdir (%v)", t).trace()
		}
		if f.fullname() == "/configuration.sm" {
			erro(ctx, "wrong fullname: %v %s", f, f.dir).trace()
		}
		if false && strings.HasPrefix(p.absPath, "/Volumes/workspace/") {
			if d := p.resolveDef(ctx, "workspace"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, d).trace()
			}
			if d := p.resolveDef(ctx, "workout"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, d).trace()
			} else {
				workout = __string(ctx, d)
			}
			if d := p.resolveDef(ctx, "workext"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, d).trace()
			} else if ts(d.value) == "" {
				erro(ctx, "%v : %v", p, ts(d)).trace()
			}
			if d := p.resolveDef(ctx, "rel.chop"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, d).trace()
			} else if s, t := __string(ctx, d), srcdir+"/"; s != t {
				erro(ctx, "%v : %s != %s", p, s, t).trace()
			}
			if d := p.resolveDef(ctx, "rel.remnant"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, ts(d)).trace()
			} else if ts(d.value) != "{=delegate {=builtin trim-prefix} {=list {=closure {=def rel.chop}}} {=list {=closure {=def /}}}}" {
				erro(ctx, "%v : %v", p, ts(d.value)).trace()
			} else if v := expand(_final(ctx),d.value); ts(v) != "{=path {=word testdata} {=word configuration}}" {
				erro(ctx, "%v : %v → %v : %s ; %v", p, d.value, v, srcdir, p.resolveDef(ctx, "/").value).trace()
			} else if s, t := __string(ctx, d), filepath.Join("testdata", "configuration"); s != t {
				erro(ctx, "%v : %s != %s", p, s, t).trace()
			} else {
				rel = s
			}
			if d := p.resolveDef(ctx, "target.triple"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, d).trace()
			} else if ts(d.value) != "{=delegate {=builtin join} {=list {=compound {=closure {=def target.arch}} {=closure {=compound {=word target} {=punct .} {=word sub}}}} {=closure {=def target.vendor}} {=closure {=def target.sys}} {=closure {=def target.abi}}} {=list {=flag {=null}}}}" {
				erro(ctx, "%v : %v", p, ts(d.value)).trace()
			} else {
				triple = __string(ctx, d)
			}
			if d := p.resolveDef(ctx, "target.out"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, d).trace()
			} else if ts(d.value) != "{=path {=punct root} {=word Volumes} {=word workout} {=closure {=def target.triple}} {=closure {=compound {=word variant} {=punct .} {=word tag}}}}" {
				erro(ctx, "%v : %v", p, ts(d.value)).trace()
			} else if s, t := __string(ctx, d), filepath.Join(workout, triple, tag); s != t {
				erro(ctx, "%s != %s", s, t).trace()
			} else {
				out = s
			}
			if d := p.resolveDef(ctx, "target.tmp"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, d).trace()
			} else if ts(d.value) != "{=path {=closure {=def target.out}} {=word tmp}}" {
				erro(ctx, "%v : %v", p, ts(d.value)).trace()
			} else if s, t := __string(ctx, d), filepath.Join(out, "tmp"); s != t {
				erro(ctx, "%s != %s", s, t).trace()
			} else {
				tmp = s
			}
			if d := p.resolveDef(ctx, "outtmp"); d == nil || d.value == nil {
				erro(ctx, "%v : %v", p, d).trace()
			} else if ts(d.value) != "{=path {=closure {=def target.tmp}} {=closure {=def rel.remnant}}}" {
				erro(ctx, "%v : %v", p, ts(d.value)).trace()
			} else if s, t := __string(ctx, d), filepath.Join(tmp, rel); s != t {
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
		if d := p.configure.resolveDef(ctx, "foo"); d == nil || d.value == nil{
			erro(ctx, "%v : %v", p, d).trace()
		} else if ts(d.value) != "{=self configure}" {
			erro(ctx, "%v : %v", p, ts(d.value)).trace()
		}
	}
}

func check_unmap_entries(ctx Context, p *project, _k any, _res *[]entry) {
	switch res := *_res; p.name {
	case "configure.base":
		switch x := _k.(type) {
		case *word:
		case flag:
			var c, y = p.entries.v[MINUS.String()]
			if !y {
				erro(ctx, "%v %v", p.name, _k).trace()
			}

			var ss = c._keys()
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

func check_unmap_files(ctx Context, p *project, _k any, _res *[]filemap_name) {
}

func tempdir_check(ctx Context, p *project, d *def, s string) {
	switch p.name {
	case "testdefaultconfigure":
		if d.name != "outtmp" {
			erro(ctx, "wrong tempdir: %v", d).trace()
		}
		if t := p.resolveDef(ctx, "target.tmp"); __string(ctx, t) == "" {
			erro(ctx, "wrong tempdir: %v ; %v", d, t).trace()
		}
		if s == "/" {
			erro(ctx, "wrong tempdir: %v → %s", d, s)

			var t *def

			t = p.resolveDef(ctx, "target.tmp")
			note(ctx, "%v → %s", t, __string(ctx, t))

			t = p.resolveDef(ctx, "target.out")
			note(ctx, "%v → %s", t, __string(ctx, t))

			t = p.resolveDef(ctx, "target.triple")
			note(ctx, "%v → %s", t, __string(ctx, t))

			t = p.resolveDef(ctx, "rel.remnant")
			note(ctx, "%v → %s", t, __string(ctx, t))

			t = p.resolveDef(ctx, "rel.chop")
			note(ctx, "%v → %s", t, __string(ctx, t))

			t = p.resolveDef(ctx, "variant.tag")
			note(ctx, "%v → %s", t, __string(ctx, t)).trace()
		}
	}
}

func tempfile_check(ctx Context, p *project, name, d string, f *file) {
	if f.dir != d {
		erro(ctx, "%v: %s != %s", p, f.dir, d).trace()
	}
	switch p.name {
	case "testdefaultconfigure":
		var t *def
		if t = p.resolveDef(ctx, "outtmp"); t == nil { // outtmp:=&(target.tmp)/&(rel.remnant)
			erro(ctx, "%s: %v : %s", p.name, d, f.dir).trace()
		} else if s, t1 := "&(target.tmp)/&(rel.remnant)", t.value.String(); t1 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t1, s).trace()
		} else if s, t2 := "/", __string(ctx, t); t2 == s {
			erro(ctx, "%s: %v : %s", p.name, d, t2).trace()
		} else if t2 != f.dir {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t2, f.dir).trace()
		}
		if t = p.resolveDef(ctx, "target.tmp"); t == nil { // target.tmp := &(target.out)/tmp
			erro(ctx, "%s: %v : %s", p.name, d, f.dir).trace()
		} else if s, t1 := "target.tmp:=&(target.out)/tmp", t.String(); t1 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t1, s).trace()
		}
		if t = p.resolveDef(ctx, "target.out"); t == nil { // &(target.out)/tmp
			erro(ctx, "%s: %v : %s", p.name, d, f.dir).trace()
		} else if s, t1 := "target.out:=/Volumes/workout/&(target.triple)/&(variant.tag)", t.String(); t1 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t1, s).trace()
		} else if s, t2 := "/Volumes/workout/arm64-apple-Darwin23.2.0-macho/bootstrap", __string(ctx, t); t2 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t2, s).trace()
		}
		if t = p.resolveDef(ctx, "target.triple"); t == nil { // target.tmp := &(target.out)/tmp
			erro(ctx, "%s: %v : %s", p.name, d, f.dir).trace()
		} else if s, t1 := "target.triple:=$(join &(target.arch)&(target.sub) &(target.vendor) &(target.sys) &(target.abi),-)", t.String(); t1 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t1, s).trace()
		} else if s, t2 := "arm64-apple-Darwin23.2.0-macho", __string(ctx, t); t2 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t2, s).trace()
		}
		if t = p.resolveDef(ctx, "target.arch"); t == nil {
			erro(ctx, "%s: %v : %s", p.name, d, f.dir).trace()
		} else if s, t1 := "target.arch:=arm64", t.String(); t1 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t1, s).trace()
		}
		if true {
			// ...
		} else if t = p.resolveDef(ctx, "target.sub"); t == nil {
			erro(ctx, "%s: %v : %s", p.name, d, f.dir).trace()
		} else if s, t1 := "target.sub:=", t.String(); t1 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t1, s).trace()
		}
		if t = p.resolveDef(ctx, "target.vendor"); t == nil {
			erro(ctx, "%s: %v : %s", p.name, d, f.dir).trace()
		} else if s, t1 := "target.vendor:=apple", t.String(); t1 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t1, s).trace()
		}
		if t = p.resolveDef(ctx, "target.sys"); t == nil {
			erro(ctx, "%s: %v : %s", p.name, d, f.dir).trace()
		} else if s, t1 := "target.sys:=&(uname.os)&(target.release)", t.String(); t1 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t1, s).trace()
		} else if s, t2 := "Darwin23.2.0", __string(ctx, t); t2 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t2, s).trace()
		}
		if t = p.resolveDef(ctx, "target.abi"); t == nil {
			erro(ctx, "%s: %v : %s", p.name, d, f.dir).trace()
		} else if s, t1 := "target.abi:=macho", t.String(); t1 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t1, s).trace()
		}
		if t = p.resolveDef(ctx, "rel.chop"); t == nil { // rel.chop ::= $(dir3 $/)/
			erro(ctx, "%s: %v : %s", p.name, d, f.dir).trace()
		} else if s, t1 := dirs(2, p.absPath)+"/", t.String(); t1 != "rel.chop::="+s {
			erro(ctx, "%s: %v : %s != rel.chop::=%s", p.name, d, t1, s).trace()
		} else if t2 := __string(ctx, t); t2 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t2, s).trace()
		}
		if t = p.resolveDef(ctx, "rel.remnant"); t == nil { // rel.remnant = $(trim-prefix &(rel.chop),&/)
			erro(ctx, "%s: %v : %s", p.name, d, f.dir).trace()
		} else if s, t1 := "rel.remnant=$(trim-prefix &(rel.chop),&/)", t.String(); t1 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t1, s).trace()
		} else if s, t2 := bases(p.absPath, 2), __string(ctx, t); t2 != s {
			erro(ctx, "%s: %v : %s != %s", p.name, d, t2, s).trace()
		}
	}
}

func unmap_check(ctx *uncache, c *valcache, key any) {
	switch _project(ctx).name {
	case "configure.base":
		switch x := key.(type) {
		case flag:
			var cc, y = c.v[MINUS.String()]
			if !y {
				if truly(ctx, unmap_uncheck_y{}) { break }
				errostack(ctx, 16, "%v %v %v", do(ctx, propUncache), x, c.ks(true)).trace()
			}

			var ss = cc._keys()
			if len(ss) == 0 { erro(ctx, "%v", x).trace() }

			var v = x.Value
			var k = v.String()
			if _, y = ss[k]; !y { erro(ctx, "%v", v).trace() }
		}
	}
	switch do(ctx, get_include_spec{}) {
	case "configure/.base/.template":
		note(pc(ctx,key), "%v %v", tv(key), c)
		note(pc(ctx,key), "%v", ts(ctx)).debug()
	}
}
