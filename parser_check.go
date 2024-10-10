///
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	"path/filepath"
	"strings"
	"fmt"
)

func (l ul) expr_check(ctx Context, _x *Value) {
	x := *_x

	switch t := x.(type) {
	case *path:
		if false {
			if s := ts(t.elems) ; strings.Contains(s, "{=path ") {
				erro(ctx, "%v : nested path: %v", l.p.tok, s).trace()
			}
		}
		for _, e := range t.elems {
			if _, y := e.(*path); y {
				erro(ctx, "%v : nested path: %v : %v", l.p.tok, t, e).trace()
			}
		}
	case *globpat:
		for _, e := range t.elems {
			if _, y := e.(*path); y {
				erro(ctx, "%v : glob with path: %v : %v", l.p.tok, t, e).trace()
			}
		}
	case *percpat:
	case *regexpat:
	}

	filename := l.p.scanner.file.Name()
	is_module := func(name, s string) (res bool) {
		res = (name == "" || name == l.project.name) && (l.project.spec == s ||
			strings.HasSuffix(l.project.spec, filepath.Join("", s)) ||
			strings.HasSuffix(filename, filepath.Join("", s, "do.smart")))
		if false && !res && name != "" && name == l.project.name {
			note(ctx, "%s %s %s", l.project.name, l.project.spec, filename)
		}
		return
	}

	if l.project == nil {
		if false { erro(ctx, "nil project").trace() }
		return
	} else if false && l.project.name == "testvalue" {
		note(ctx, "%s %s %s", l.project.name, l.project.spec, filename)
	}

	switch {
	case is_module("testvalue","testdata/value"):
	case is_module("testvalue","testdata/value/closure"):
	case is_module("testvalue","testdata/value/glob"):
		switch t := x.(type) {
		case *globpat:
			if s := ts(t.elems) ; strings.Contains(s, "{=path ") {
				erro(ctx, "glob with path: %v", s).trace()
			}
		}
	case is_module("testbuiltins","testdata/assert"):
	case is_module("testbuiltins","testdata/pushcontext"):
	case is_module("testbuiltins","testdata/builtins/trimprefix"):
		switch x.String() {
		case  "**/testdata":
			if s, t := ts(x), "{=path {=globpat {=globmeta **}} {=word testdata}}"; s != t {
				erro(ctx, "%s != %s", s, t).trace()
			}
		case  "%%/testdata":
			if s, t := ts(x), "{=path {=percpat {=null} {=percpat {=null} {=null}}} {=word testdata}}"; s != t {
				erro(ctx, "%s != %s", s, t).trace()
			}
		case "/**/testdata":
			if s, t := ts(x), "{=path {=punct root} {=globpat {=globmeta **}} {=word testdata}}"; s != t {
				erro(ctx, "%s != %s", s, t).trace()
			}
		case "/%%/testdata":
			if s, t := ts(x), "{=path {=punct root} {=percpat {=null} {=percpat {=null} {=null}}} {=word testdata}}"; s != t {
				erro(ctx, "%s != %s", s, t).trace()
			}
		}
	case is_module("configure.base","configure/.base"):
		if s := x.String(); strings.HasPrefix(s, "configure.funcs.lib") && strings.HasSuffix(s, "?") {
			t := "{=cond {=compound {=word configure} {=punct .} {=word funcs} {=punct .} {=word lib} {=word "
			t += s[19:len(s)-1] + "}}}"
			if s := ts(x); s != t {
				erro(ctx, "%s != %s", s, t).trace()
			}
		}
	case is_module("variant.target.darwin","variant/.target/darwin"):
		switch x.String() {
		case "-sdk' '$(MacOSX.sdk)":
			t := "{=flag {=compound {=word sdk} {=strlit  } {=delegate {=def MacOSX.sdk}}}}"
			if s := ts(x); s != t {
				erro(ctx, "%s != %s", s, t).trace()
			}
		case "-L$(MacOSX.sdk)/usr/lib":
			t := "{=flag {=path {=compound {=word L} {=delegate {=def MacOSX.sdk}}} {=word usr} {=word lib}}}"
			if s := ts(x); s != t {
				erro(ctx, "%s != %s", s, t).trace()
			}
		}
	case is_module("llvm","llvm"):
		if s := x.String(); strings.HasPrefix(s, "llvm/%%") {
			if s != "llvm/%%(&(headers))" {
				erro(ctx, "%v %v", typeof(x), x).trace()
			} else if a, y := x.(*argumented); !y {
				erro(ctx, "%v %v", typeof(x), x).trace()
			} else if _, y := a.Value.(*path); !y {
				erro(ctx, "%v %v", typeof(a), a).trace()
			}
		} else if s == "$(srcinc)/llvm/%%" {
			t := "{=path {=delegate {=def srcinc}} {=word llvm} {=percpat {=null} {=percpat {=null} {=null}}}}"
			if s := ts(x); s != t {
				erro(ctx, "%v %v : %s != %s", typeof(x), x, s, t).trace()
			}
		}
	}
	return
}

func define_check(ctx Context, tok token, ident, value Value, _d **def) {
	var d = *_d
	if d == nil {
		erro(ctx, "%v %v %v", ident, tok, ts(value)).trace()
	} else if d.value == nil && !isNull(value) {
		erro(ctx, "%v %v %v", ident, tok, ts(value)).trace()
	}

	var p = _project(ctx)
	var flat_mode = truly(ctx, is_flat_mode{})
	var is_config = truly(ctx, is_config_mode{})

	if is_config && d.o != defConfig {
		erro(ctx, "%v : %v : %v", p, d.o, d).trace()
	}

	switch p.name {
	case "variant.target.base":
		if strings.HasPrefix(d.name, "std.") {}
	case "variant.target":
		if strings.HasPrefix(d.name, "lang.") {
			var lang = strings.TrimPrefix(d.name, "lang.")
			if x, y := langs_map[lang]; !y {
				erro(ctx, "wrong lang: %v : %v", lang, d).trace()
			} else if d.string(ctx) != x {
				erro(ctx, "wrong: %v :", lang, d).trace()
			}
		}
	case "testdefaultconfigure", "testdeftwoconfigure":
		if true || flat_mode {
			var s = _scope(ctx)
			if x, y := p.elems[d.name]; !y {
				erro(ctx, "%v : %v : %v : %v", s.comment, s.names(), d, d.o).trace()
			} else if x != d {
				erro(ctx, "%v : %v : %v : %v", s.comment, s.names(), d, d.o).trace()
			}
		}
		if d.name == "FOO" {
			if v, s := d.value, "{=self "+p.name+"}"; v == nil || v.String() != s {
				erro(ctx, "%v : %v != %s", d.o, d, s).trace()
			} else if ts(d.value) != s {
				erro(ctx, "%v : %v", d.o, ts(d.value)).trace()
			}
		}
	}
}

func def_idents_check(ctx Context, idents, value Value, defs []*def) {
	switch p := _project(ctx); p.name {
	case "variant.target":
		if len(defs) == 0 {
			erro(ctx, "{=%v %v} : %v", typeof(idents), idents, value).trace()
		}
		if x, y := idents.(*argumented); y {
			var pre = x.Value.string(ctx)
			var args = merge(x.args...)
			if a, b := len(defs), len(args); a != b {
				erro(ctx, "%v : %v ; %d != %d", idents, defs, a, b).trace()
			}
			switch pre {
			case "lang.":
				if ts(x.Value) != "{=compound {=word lang} {=punct .}}" {
					erro(ctx, "%v : %v : %v", idents, ts(x.Value), value).trace()
				}
				for i, d := range defs {
					if s := args[i].string(ctx); d.name != pre+s {
						erro(ctx, "%v : %s%s : %v", idents, pre, s, d).trace()
					}
				}
			}
		}
	}
}

func (l ul) files_check(ctx Context) {
	switch p := l.project; p.name {
	case "variant.target.base":
		if d := p.def(ctx, "variant.tag"); d == nil || d.value == nil {
			erro(ctx, "variant.tag is undefined").trace()
		}
		if d := p.def(ctx, "variant.name"); d == nil || d.value == nil {
			erro(ctx, "variant.name is undefined").trace()
		}
		if d := p.def(ctx, "prefix"); d == nil || d.value == nil {
			erro(ctx, "prefix is undefined").trace()
		}
		if d := p.def(ctx, "outtmp"); d == nil || d.value == nil {
			erro(ctx, "outtmp is undefined").trace()
		}
		if d := p.def(ctx, "outinc"); d == nil || d.value == nil {
			erro(ctx, "outinc is undefined").trace()
		}
		if d := p.def(ctx, "outobj"); d == nil || d.value == nil {
			erro(ctx, "outobj is undefined").trace()
		}
		if d := p.def(ctx, "outlib"); d == nil || d.value == nil {
			erro(ctx, "outlib is undefined").trace()
		}
		if d := p.def(ctx, "outbin"); d == nil || d.value == nil {
			erro(ctx, "outbin is undefined").trace()
		}
	case "variant.target":
		if d := p.def(ctx, "variant.tag"); d == nil || d.value == nil {
			erro(ctx, "variant.tag is undefined").trace()
		}
		if d := p.def(ctx, "variant.name"); d == nil || d.value == nil {
			erro(ctx, "variant.name is undefined").trace()
		}
	case "app.base":
		if d := p.def(ctx, "variant.tag"); d == nil || d.value == nil {
			erro(ctx, "variant.tag is undefined").trace()
		}
		if d := p.def(ctx, "variant.name"); d == nil || d.value == nil {
			erro(ctx, "variant.name is undefined").trace()
		}
		if d := p.def(ctx, "outinc"); d == nil {
			erro(ctx, "outinc is undefined").trace()
		}
		if d := p.def(ctx, "outobj"); d == nil {
			erro(ctx, "outobj is undefined").trace()
		}
		if d := p.def(ctx, "outlib"); d == nil {
			erro(ctx, "outlib is undefined").trace()
		}
	case "lib.std":
		if d := p.def(ctx, "outinc"); d == nil {
			erro(ctx, "outinc is undefined").trace()
		}
		if d := p.def(ctx, "outobj"); d == nil {
			erro(ctx, "outobj is undefined").trace()
		}
		if d := p.def(ctx, "outlib"); d == nil {
			erro(ctx, "outlib is undefined").trace()
		}
	}
}

func (l ul) files_check_2(ctx Context, path Value, patts, paths []Value, ms []filemap) {
	switch l.project.name {
	case "llvm.Config":
		if path == nil && paths != nil {
			erro(ctx, "wrong paths: %v", paths).trace()
		}

		var srcinc = l.project.def(ctx, "srcinc").string(ctx)
		var outinc = l.project.def(ctx, "outinc").string(ctx)
		for _, m := range ms {
			if x, y := m.pattern.(*file); y {
				var t = l.project.file(ctx, x.name)
				if t == nil {
					erro(ctx, "%s %s", x.name, ts(m.pattern)).trace()
				}
				if strings.HasSuffix(x.name, ".def") || strings.HasSuffix(x.name, ".h") {
					if m.paths == nil {
						if x.dir != outinc {
							erro(ctx, "%s != %s", x.dir, outinc).trace()
						}
					} else if s := m.paths[0].string(ctx); s != outinc {
						erro(ctx, "%s != %s", s, outinc).trace()
					}
					if t.dir != outinc {
						erro(ctx, "%s != %s", t.dir, outinc).trace()
					}
				} else if strings.HasSuffix(x.name, ".def.in") || strings.HasSuffix(x.name, ".h.cmake") {
					if m.paths == nil {
						if x.dir != srcinc {
							erro(ctx, "%s != %s", x.dir, srcinc).trace()
						}
					} else if s := m.paths[0].string(ctx); s != srcinc {
						erro(ctx, "%s != %s", s, srcinc).trace()
					}
					if t.dir != srcinc {
						erro(ctx, "%s != %s", t.dir, srcinc).trace()
					}
				}
			}
		}
	}
}

func (l ul) parse_file_check_1(ctx Context, abs, rel, tmp string) {
	var p = l.project
	if p == nil {
		if false { erro(ctx, "nil project").trace() }
		return
	}

	var workout string

	switch p.name {
	case "general":
		if len(p.bases) != 0 {
			erro(ctx, "%v: %v", p, p.bases).trace()
		}
		if d := p.def(ctx, "workspace"); d == nil || d.value == nil {
			erro(ctx, "%v: %v %v", p, rel, d).trace()
		} else if x, y := d.value.(*path); !y {
			erro(ctx, "%v: %v", p, tst{d.value}).trace()
		} else if s, t := x.string(ctx), dirs(3, abs); s != t {
			erro(ctx, "%v: %v %v", p, s, t).trace()
		}
		if d := p.def(ctx, "workout"); d == nil || d.value == nil {
			erro(ctx, "%v: %v %v", p, rel, d).trace()
		} else if s, t := d.string(ctx), dirs(4, abs)+"/workout"; s != t {
			erro(ctx, "%v: %v %v", p, s, t).trace()
		} else {
			workout = s
		}
		if d := p.def(ctx, "workext"); d == nil || d.value == nil {
			erro(ctx, "%v: %v %v", p, rel, d).trace()
		} else if s, t := d.string(ctx), dirs(3, abs)+"/external"; s != t {
			erro(ctx, "%v: %v %v", p, s, t).trace()
		}
		if d := p.def(ctx, "CTD"); d == nil || d.value == nil {
			erro(ctx, "%v: %v %v", p, rel, d).trace()
		} else if false && ts(d.value) != "{=closure {=word outtmp}}" {
			erro(ctx, "%v: %v", p, d).trace()
		} else if s := d.string(ctx); false && s != filepath.Join(workout, "general") {
			erro(ctx, "%v: %v %v", p, s, d).trace()
		}
	case "variant.target.base":
		if len(p.bases) != 1 {
			erro(ctx, "%v: %v", p, p.bases).trace()
		}
		if p.bases[0].name != "general" {
			erro(ctx, "%v: %v", p, p.bases[0]).trace()
		}
		if p.def(ctx, "workout") != p.bases[0].def(ctx, "workout") {
			erro(ctx, "%v: workout", p).trace()
		}
		if d := p.def(ctx, "workout"); d == nil || d.value == nil {
			erro(ctx, "%v: %v %v", p, rel, d).trace()
		} else if s, t := d.value.string(ctx), dirs(6, abs)+"/workout"; s != t {
			erro(ctx, "%v: %v %v", p, s, t).trace()
		}
	case "variant.target":
		// TODO: ...
	}
}

func (l ul) parse_file_check_new_project(ctx Context) {
	switch p := l.project; p.name {
	case "lib.std":
		if len(p.bases) != 1 {
			erro(ctx, "%v: wrong bases: %v", p, p.bases).trace()
		}
		if b := p.bases[0]; b.name != "app.base" {
			erro(ctx, "%v: wrong bases[0]", b).trace()
		}
		if d := p.def(ctx, "outinc"); d == nil || d.value == nil {
			erro(ctx, "%v: %v", p, d).trace()
		}
		if d := p.def(ctx, "outobj"); d == nil || d.value == nil {
			erro(ctx, "%v: %v", p, d).trace()
		}
		if d := p.def(ctx, "outlib"); d == nil || d.value == nil {
			erro(ctx, "%v: %v", p, d).trace()
		}
	}
}

func (l ul) parse_file_check_2(ctx Context, filename string) {
	var p = l.project
	if p == nil {
		erro(ctx, "nil project").trace()
		return
	}

	var workout string

	switch p.name {
	case "general":
		if d := p.def(ctx, "workout"); d == nil || d.value == nil {
			erro(ctx, "%v: %v", p, d).trace()
		} else {
			workout = d.string(ctx)
		}
		if d := p.def(ctx, "CTD"); d == nil || d.value == nil {
			erro(ctx, "%v: %v", p, d).trace()
		} else if false && ts(d.value) != "{=closure {=word outtmp}}" {
			erro(ctx, "%v: %v", p, d).trace()
		} else if s := d.string(ctx); false && s != filepath.Join(workout, "general") {
			erro(ctx, "%v: %v %v", p, s, d).trace()
		}
	}

	switch filepath.Base(filename) {
	case ".autoload.declared": l.parse_file_check_autoload_declared(ctx, p)
	case ".autoload.appendix": l.parse_file_check_autoload_appendix(ctx, p)
	case ".configure": l.parse_file_check_dot_configure(ctx, p)
	case "do.smart": l.parse_file_check_do_smart(ctx, p)
	}
}

func (l ul) parse_file_check_autoload_declared(ctx Context, p *project) {
	switch p.name {
	case "testdefaultconfigure":
		if len(p.configs) != 0 {
			erro(ctx, "wrong configs: %v", p.configs).trace()
		}
	}
}

func (l ul) parse_file_check_autoload_appendix(ctx Context, p *project) {
	switch p.name {
	case "app.base":
	case "configure.base":
	}
}

func (l ul) parse_file_check_dot_configure(ctx Context, p *project) {
	switch p.name {
	case "llvm.Config":
		// TODO: check LLVM_VERSION
		// TODO: check configure.package
		// TODO: check configure.version
	}
}

func (l ul) parse_file_check_do_smart(ctx Context, p *project) {
	switch p.name {
	case "testdefaultconfigure":
		if len(p.configs) != 1 {
			erro(ctx, "wrong configs: %v", p.configs).trace()
		}
		if p.configs[0].String() != "FOO" {
			erro(ctx, "wrong configs[0]: %v", p.configs[0]).trace()
		}
	case "testcustomconfigure":
		var configs = []string{"FOO1","FOO2","FOO3","FOO4","FOO5"}
		if len(p.configs) != len(configs) {
			erro(ctx, "wrong configs: %v", p.configs).trace()
		}
		for i, s := range configs {
			if p.configs[i].String() != s {
				erro(ctx, "wrong configs[%d]: %v", i, p.configs[i]).trace()
			}
		}
	case "testdeftwoconfigure":
		if len(p.configs) != 1 {
			erro(ctx, "wrong configs: %v", p.configs).trace()
		}
		if p.configs[0].String() != "FOO" {
			erro(ctx, "wrong configs[0]: %v", p.configs[0]).trace()
		}
	}
}

func (l ul) project_check_bases(ctx Context) {
	switch p := l.project; p.name {
	case "lib.std":
		if len(p.bases) != 1 {
			erro(ctx, "%v: wrong bases: %v", p, p.bases).trace()
		}
		if b := p.bases[0]; b.name != "app.base" {
			erro(ctx, "%v: wrong bases[0]", b).trace()
		}
	}
}

func (l ul) braced_str_check(ctx Context, elems []Value, res *Value) {
	fn := l.p.scanner.file.Name()

	if strings.HasSuffix(fn, "/configure/.base/.template") {
		for _, v := range elems {
			if indeterminate(ctx, v) {
				erro(pc(ctx,v), "indeterminate str: %v", ts(v)).trace()
			}
		}

		if _, y := (*res).(*strlit); !y {
			erro(pc(ctx, *res), "%v", ts(*res)).trace()
		}
	}
}

func (l ul) braced_word_check(ctx Context, elems []Value, res *Value) {
	fn := l.p.scanner.file.Name()

	if strings.HasSuffix(fn, "/configure/.base/.template") {
		for _, v := range elems {
			if indeterminate(ctx, v) {
				erro(pc(ctx,v), "indeterminate str: %v", ts(v)).trace()
			}
		}

		if _, y := (*res).(*word); !y {
			erro(pc(ctx, *res), "%v", ts(*res)).trace()
		}
	}
}

func (l ul) rule_check(ctx Context, targets []Value, res *Value) {
	for _, target := range targets {
		var v Value
		switch t := target.(type) {
		case *argumented: v = t.Value
		default: v = target
		}
		if v != nil && indeterminate(ctx, v) {
			erro(ctx, "indeterminate: %v : %v", v, ts(v)).trace()
		}
	}

	fn := l.p.scanner.file.Name()

	if strings.HasSuffix(fn, "/configure/.base/.template") {
		target := targets[0]
		t := target.string(ctx)
		if strings.HasPrefix(t, "HAVE_FUN_") || strings.HasPrefix(t, "HAVE_SYM_") {
			name := strings.TrimPrefix(t, "HAVE_")
			if e := l.project.unmap_entries(ctx, target, nil); len(e) != 1 {
				erro(pc(ctx,target), "%v : no such entry : %s", tv(target), t)
				note(pc(ctx,target), "%v", &l.project.entries).trace()
			} else if e := l.project._entries(ctx, target); len(e) != 1 {
				erro(pc(ctx,target), "%v : no such entry : %s", tv(target), t)
				note(pc(ctx,target), "%v", &l.project.entries).trace()
			} else if x, y := e[0].(rule_name); !y {
				erro(pc(ctx,target), "%v %v", ts(target), tv(e[0])).trace()
			} else if len(x.program) == 1 {
				if p := x.program[0]; len(p.recipes) == 1 {
					// Checking for recipe:
					//   $(or $(HAVE_FUN_$(uppercase $_)),$(HAVE_SYM_$(uppercase $_)))
					var t = fmt.Sprintf("$(or $(HAVE_FUN_%s),$(HAVE_SYM_%s))", name, name)
					if r := p.recipes[0]; r.String() != t {
						erro(pc(ctx,target), "%v %v", ts(target), tv(r)).trace()
					}
				}
			}
		}
	}

	switch l.project.name {
	case "configure.base":
		t := targets[0].String()
		switch t {
		case "-library-c":
		}

	case "lib.c++.inc":
	case "llvm.Config":
		if strings.HasSuffix(fn, "/llvm/Config/.configure") {
			target := targets[0]
			if e := l.project.unmap_entries(ctx, target, nil); len(e) != 1 {
				erro(ctx, "%v", target).trace()
			}
			if e := l.project._entries(ctx, target); len(e) != 1 {
				erro(ctx, "%v", target).trace()
			}
			switch target.String() {
			case "'LLVM_VERSION_INFO'":
			case "'LLVM_VERSION_MAJOR'":
			case "'LLVM_VERSION_MINOR'":
			case "'LLVM_VERSION_PATCH'":
			case "'LLVM_VERSION":
			}
		}
	}
}

func (l ul) rule_check_targets(ctx Context, targets []Value) {
	switch l.project.name {
	case "lib.c++.inc":
		if len(targets) != 1 {
			erro(ctx, "%v", targets).trace()
		} else {
			l.rule_check_target(ctx, targets[0])
		}
	}
}

func (l ul) rule_check_target(ctx Context, target Value) {
	if indeterminate(ctx, target) {
		erro(ctx, "%v : %v", target, ts(target)).trace()
	}
}

func (l ul) codeblock_check(ctx Context, op token, vars map[string]Value) {
	if op == DEF {
		erro(ctx, "wrong codeblock op: %v", op).trace()
	}
	switch l.project.name {
	case "configure.base":
		// note(ctx, "%v: %v : %v", l.project.name, op, l.project.spec); flush(ctx)
	case "variant.target.base":
		// note(ctx, "%v: %v : %v", l.project.name, op, l.project.spec); flush(ctx)
	case "variant.target":
		switch op {
		case FOR: //note(ctx, "%v: %v : %v", l.project.name, op, l.project.spec); flush(ctx)
		case FOREACH: //note(ctx, "%v: %v : %v", l.project.name, op, l.project.spec); flush(ctx)
		case LPAREN: // aka call
		}
	case "lib.c++.inc":
		if op == FOR {
			if x, y := vars["feature"]; !y {
				erro(ctx, "no 'feature' : %v : %v", x, ts(x)).trace()
			} else if w, y := x.(*word); !y {
				erro(ctx, "wrong feature : %v : %v", x, ts(x)).trace()
			} else if !strings.HasPrefix(w.s, "LIBCXX_ENABLE_") {
				erro(ctx, "wrong name : %v", w).trace()
			}
		}
	}
}
