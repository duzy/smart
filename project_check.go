//
//  Copyright (C) 2012-2024, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	"fmt"
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
		if strings.HasPrefix(p.absPath, "/Volumes/workspace/") {
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
			} else if v := d.value.expand(final{ctx}); ts(v) != "{=path {=word testdata} {=word configuration}}" {
				erro(ctx, "%v : %v → %v ; %s ; %v", p, d.value, v, srcdir, p.def(ctx, "/").value).trace()
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

func unmap_check(ctx *unmap, c *valcache, key any) {
	switch p := _project(ctx); p.name {
	case "configure.base":
		var s = fmt.Sprintf("%s", key)
		if false && !truly(ctx, unmap_uncheck_y{}) {
			var b bool
			for _, a := range ctx.a {
				var t = fmt.Sprintf("%s", a)
				if b = s == t; b { break }
			}
			if !b { erro(ctx, "%v %v %v", tv(key), ctx.a, c).trace() }
		}
		switch x := key.(type) {
		case flag:
			var cc, y = c.puncs[MINUS]
			if !y {
				if truly(ctx, unmap_uncheck_y{}) { break }
				erro(ctx, "%v %v %v", cacheMapping(ctx), x, c.ks(true)).trace()
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
