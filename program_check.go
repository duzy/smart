//
//  Copyright (C) 2012-2024, Duzy Chan <code@extbit.io>, all rights reserverd.
//
package smart

import (
	"strings"
)

func (*plainint) evaluate_check(ctx Context, args, recipes []Value, p *plain) {
	if t, y := auto_get(ctx, "@").(*file); y {
		if strings.HasPrefix(t.name, ".configure/") && strings.HasSuffix(t.name, ".c") {
			for i, recipe := range recipes {
				if x, y := recipe.(*plainline); y && x.len() == 1 {
					if x, y := x.elems[0].(*delegate); y && len(x.a) == 2 {
						// if x.String() == `$(foreach $(INCLUDE),"#include $_\n")` {}
						if b, y := x.x.(*builtin); y && b.name == "foreach" {
							if d, y := p.elems[i].(*plainline).elems[0].(*delegate); y {
								if false && x.a[0].String() == `$(INCLUDE)` {
									note(ctx, "%v → %v", x.a[0], d.a[0]).debug()
								}
								if x.a[1].String() == `"#include $_\n"` {
									s := `{=list {=strcomp {=raw #include } {=delegate {=auto _}} {=escaped \n}}}`
									if t := ts(x.a[1]); s != t {
										erro(pc(ctx,x), "%v: %s != %s", x.a[1], t, s).trace()
									} else {
										note(pc(ctx,x), "%s %s %s", x, x.expand(ctx), ts(x))
										note(pc(ctx,x), "%s %s", auto_find(ctx, "_"), ts(ctx))
										note(pc(ctx,x), "%s : %v → %v", t, x.a[1], d.a[1]).debug(3)
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return
}

func (ctx *exec_ctx) sources_check(cc Context, i int, rv Value, s string) {
	var exe = _execution(ctx)
	if exe.proj != nil && exe.proj.name == "configure.base" {
		switch dest := _entry(ctx).destiny().string(ctx); dest {
		case "-header-c", "-header-c++":
		case "-library-c", "-library-c++":
			if strings.HasSuffix(ctx.targetName, ".log") {}

			if r, y := rv.(*recipe); !y || r.len() == 0 {
				erro(pc(ctx,rv), "%v. %v", i, rv).trace()
			} else {
				for i, e := range r.elems {
					switch ts(e) {
					case "{=delegate {=def x}}":
						if v := e.expand(cc); v == nil || ts(v) == "{=null}" || ts(e) == ts(v) {
							var s1, s2 = e.string(cc), e.string(ctx)
							erro(pc(ctx,e), "%d. %s → %s → %s (%s)", i, ts(e), ts(v), s1, s2)
							note(pc(ctx,rv), "%v,", ts(e.expand(ctx)))
							note(pc(ctx,rv), "%v,", ts(e.expand(fullfile_ctx{ctx})))
							note(pc(ctx,rv), "%v.", ts(e.expand(_final(ctx))))
							note(pc(ctx,rv), "%v", ts(rv)).trace()
						}
						if v := e.expand(_final(ctx)); v == nil || ts(v) == "{=null}" || ts(e) == ts(v) {
							var s1, s2 = e.string(cc), e.string(ctx)
							erro(pc(ctx,e), "%d. %s → %s → %s (%s)", i, ts(e), ts(v), s1, s2)
							note(pc(ctx,rv), "%v,", ts(e.expand(ctx)))
							note(pc(ctx,rv), "%v,", ts(e.expand(fullfile_ctx{ctx})))
							note(pc(ctx,rv), "%v;", ts(e.expand(_final(ctx))))
							note(pc(ctx,rv), "%v,", ts(e.expand(cc)))
							note(pc(ctx,rv), "%v,", ts(e.expand(fullfile_ctx{cc})))
							note(pc(ctx,rv), "%v.", ts(e.expand(_final(cc))))
							note(pc(ctx,rv), "%v", ts(rv)).trace()
						}
					}
				}
			}

			var v = rv.expand(cc)
			if r, y := v.(*recipe); !y || r.len() == 0 {
				erro(pc(ctx,rv), "%v. %v", i, v).trace()
			} else {
				for i, e := range r.elems {
					switch ts(e) {
					case "{=null}":
						erro(pc(ctx,e), "%d. %s", i, ts(e))
						note(pc(ctx,rv), "%v", ts(rv))
						note(pc(ctx,rv), "%v", ts(v)).trace()
					}
				}
			}
		}
	}
}

func (exe *execution) evaluate_check(ctx Context, i interpreter, args []Value, res Value) {
	if t, y := auto_get(ctx, "@").(*file); y {
		if strings.HasPrefix(t.name, ".configure/") {
			var fn = t.fullname()
			if x, y := res.(*plain); y && 0 < x.len() {
				if exe.language != x.name {
					erro(ctx, "%s != %s ; %v", exe.language, x.name, ts(res)).trace()
				}
				// $(foreach $(INCLUDE),"#include $_\n") → $(foreach {},"#include xxx\n")
				if x, y := x.elems[0].(*plainline); y && 0 < x.len() {
					if false && (x.String() == "{=plainline {}}" || ts(x) == "{=plainline {=null}}") {
						erro(pc(ctx,x), "%s %v %s", typeof(i), x, ts(x)).debug(2)
					}
					if x, y := x.elems[0].(*delegate); y && ts(x.x) == "{=builtin foreach}" && len(x.a) == 2 {
						if s, t := "{=list {=null}}", ts(x.a[0]); s == t {
							notestack(pc(ctx,fn), 3, "%v", res).debug()
							erro(ctx, "%v : %v : %s != %s", ts(x.x), x.a[0], t, s).trace()
						}
						if s, t := `{=strval {=raw #include } {=delegate {=auto _}} {=raw \n}}`, ts(x.a[1]); s != t {
							notestack(pc(ctx,fn), 3, "%v", res).debug()
							erro(ctx, "%v : %v : %s != %s", ts(x.x), x.a[1], t, s).trace()
						}
						for _, a := range x.a[1:] {
							if s, t := "{=delegate {=auto _}}", ts(a); !strings.Contains(t, s) {
								notestack(pc(ctx,fn), 2, "%v", res).debug()
								erro(ctx, "%v : %v %v : %s", ts(x.x), ts(x.a[0]), a, ts(t)).trace()
							}
						}
					}
				}
			}
			if s := typeof(i); s != `plainint` && t.stat(ctx) == nil {
				switch {
				case
					strings.HasSuffix(t.name, ".c"),
					strings.HasSuffix(t.name, ".c++"),
					strings.HasSuffix(t.name, ".log"):
					errostack(pc(ctx,fn), 6, "%s %v %v", s, exe.language, res).trace()
				}
			}
		}
	}
}

func (exe *execution) interpret_check(ctx Context, i interpreter, args []Value, res Value) {
}

func (prog *program) execute_check(ctx *execution, result *Value) {
	switch prog.project.name {
	case "configure.base":
		if x1, y := ctx.defs["TYPE"]; y {
			if x2, y := ctx.defs["TARGET"]; y {
				s := strings.ToUpper(x1.string(ctx))
				s  = strings.Replace(s, " ", "_",  -1)
				s  = strings.Replace(s, "*", "_P", -1)
				if t := x2.string(ctx); "SIZEOF_"+s != t && "ALIGNOF_"+s != t {
					erro(ctx, "%v : %s != %s", ctx.prerequisite, s, t).trace()
				}
				if a, y := ctx.prerequisite.(*argumented); false && y && a != nil {
					note(ctx, "%v %v %v", a, x1, x2).debug(2)
				}
			}
		}
	case "testdefaultconfigure":
		if t := _entry(ctx).destiny(); t == nil {
			erro(ctx, "%v: %v", _entry(ctx)).trace()
		} else {
			switch t.String() {
			case "FOO":
				var v = *result
				if v == nil {
					erro(ctx, "%v", t).trace()
				} else if d, y := v.(*def); !y {
					erro(ctx, "%v", t).trace()
				} else if v = d.value; v == nil {
					erro(ctx, "%v", d).trace()
				} else if v.String() != "{=self testdefaultconfigure}" {
					erro(ctx, "%v", d).trace()
				}
				if d := prog.project.finddef("FOO"); d == nil {
					erro(ctx, "%v", t).trace()
				} else if d.value == nil {
					erro(ctx, "%v ; %v", d, v).trace()
				} else if d.value != v {
					erro(ctx, "%v", d).trace()
				}
			}
		}
	}
	switch prog.project.spec {
	case "testdata/rule/0": prog.execute_check_rule_0(ctx, result)
	case "testdata/rule/1": prog.execute_check_rule_1(ctx, result)
	case "testdata/rule/shell/for-stdout":
		prog.execute_check_shell_for_stdout(ctx, result)
	}
}

func (prog *program) execute_check_0(ctx *execution) {
	switch prog.project.name {
	case "configure.base":
		switch t := prog.target(ctx); t.String() {
		case "-sizeof-c":
			for _, dep := range prog.depends {
				if x, y := dep.(*argumented); y && x.Value.String() == "$(name).c.x" {
					if len(x.args) != 3 {
						erro(ctx, "%v %v", t, x).trace()
					}
					for i, s := range []string{"$(TYPE)","$(INCLUDE)","$(LIB)"} {
						if x.args[i].String() != s {
							erro(ctx, "%v %v %v != %s", t, x.Value, x.args[i], s).trace()
						}
					}
				}
			}
		default:
			if false { note(ctx, "%v", t).debug() }
		}
	}
}

func (prog *program) execute_check_1(ctx *execution) {
	switch prog.project.name {
	case "configure.base":
		if t := _entry(ctx).destiny(); t == nil {
			erro(ctx, "%v: %v", _entry(ctx)).trace()
		} else if false {
			var s = _scope(ctx)
			switch t.String() {
			case "-compiles-c":
				if d := s.finddef("name"); d == nil {
					erro(ctx, "%v : name", t).trace()
				} else if d.value.String() != ".configure/compiles/$(TARGET)" {
					erro(ctx, "%v : %s", t, d.value).trace()
				}
			case "-library-c":
				if d := s.finddef("name"); d == nil {
					erro(ctx, "%v : name", t).trace()
				} else if d.value.String() != ".configure/library/$(TARGET)" {
					erro(ctx, "%v : %s", t, d.value).trace()
				}
				if d := s.finddef("s"); d == nil {
					erro(ctx, "%v : name", t).trace()
				} else if d.value.String() != "$(file .configure/$(ifdef $(FUNCTION),function,library)/$(TARGET).c)" {
					erro(ctx, "%v : %s", t, d.value).trace()
				}
				if d := s.finddef("x"); d == nil {
					erro(ctx, "%v : name", t).trace()
				} else if d.value.String() != "$(file $(s).x)" {
					erro(ctx, "%v : %s", t, d.value).trace()
				}
				if d := s.finddef("o"); d == nil {
					erro(ctx, "%v : name", t).trace()
				} else if d.value.String() != "$(file $(s).o)" {
					erro(ctx, "%v : %s", t, d.value).trace()
				}
			}
		}
	}
}

func (prog *program) execute_check_rule_0(ctx Context, result *Value) {
	var ent = _entry(ctx)
    if *result == nil {
        erro(ctx, "%v: nil result", ts(ent)).trace()
    }

    var args = try[[]Value](ctx, get_args{})

    switch ent.destiny().string(ctx) {
    case "rule0":
        if len(args) != 3 {
            erro(ctx, "%v: %d %v", ent, len(args), ts(args)).trace()
        }
		if v := auto_get(ctx, "@"); ts(v) != "{=word rule0}" {
			erro(ctx, "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, "<"); ts(v) != "{=word rule1}" {
			erro(ctx, "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, ">"); ts(v) != "{=word rule1}" {
			erro(ctx, "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, "^"); ts(v) != "{=list {=word rule1}}" {
			erro(ctx, "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, "-"); ts(v) != "{=list {=word rule1} {=word rule1} {=flag {=null}} {=compound {=word x} {=word y} {=word z}}}" {
			erro(ctx, "%v", ts(v)).trace()
		}
		if auto_get(ctx, "<") != auto_get(ctx, ">") {
			erro(ctx, "%v %v", ts(auto_get(ctx, "<")), ts(auto_get(ctx, ">"))).trace()
		}
        if x, y := (*result).(*list); !y {
            erro(ctx, "%v: %v", ent, ts(*result)).trace()
        } else if x.len() != 4 {
            erro(ctx, "%v: %v", ent, ts(*result)).trace()
        } else if s := ts(x.elems[0]); s != "{=word rule1}" {
            erro(ctx, "%v: %v", ent, s).trace()
        } else if s := ts(x.elems[1]); s != "{=word rule1}" {
            erro(ctx, "%v: %v", ent, s).trace()
        } else if s := ts(x.elems[2]); s != "{=flag {=null}}" {
            erro(ctx, "%v: %v", ent, s).trace()
        } else if s := ts(x.elems[3]); s != "{=compound {=word x} {=word y} {=word z}}" {
            erro(ctx, "%v: %v", ent, s).trace()
        }
        if ts(*result) != "{=list {=word rule1} {=word rule1} {=flag {=null}} {=compound {=word x} {=word y} {=word z}}}" {
            erro(ctx, "%v: %v, %v", ent, ts(*result), args).trace()
        }
    case "rule1":
		if len(args) != 1 {
			erro(ctx, "%v: %d %v", ent, len(args), ts(args)).trace()
		}
		if v := auto_get(ctx, "@"); ts(v) != "{=word rule1}" {
			erro(ctx, "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, "<"); ts(v) != "{}" {
			erro(ctx, "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, ">"); ts(v) != "{}" {
			erro(ctx, "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, "^"); ts(v) != "{}" {
			erro(ctx, "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, "ARG1"); v == nil {
			erro(ctx, "ARG1").trace()
		} else if s := ts(v); s == "" {
			erro(ctx, "%v", v).trace()
		} else if t := "{=word xxyzz}"; s == t {
			if v := auto_get(ctx, "-"); v == nil {
				erro(ctx, "-").trace()
			} else if s, t := ts(v), "{=plain(text) {=plainline {=word xxyzz}}}"; s != t {
				erro(ctx, "%v: %s != %s", v, s, t).trace()
			}
			if x, y := (*result).(*plain); !y {
				erro(ctx, "%v: %v", ent, ts(*result)).trace()
			} else if x.name != "text" {
				erro(ctx, "%v: %v", ent, ts(*result)).trace()
			} else if x.len() != 1 {
				erro(ctx, "%v: %v", ent, ts(*result)).trace()
			} else if s := ts(x.elems[0]); s != "{=plainline {=word xxyzz}}" {
				erro(ctx, "%v: %v", ent, s).trace()
			}
			if s, t := (*result).string(ctx), "xxyzz\n"; s != t {
				erro(ctx, "%v: %v: %s != %s", ent, ts((*result)), s, t).trace()
			}
		} else if t := "{=list {=word rule0} {=word xyz}}"; s == t {
			if v := auto_get(ctx, "-"); v == nil {
				erro(ctx, "-").trace()
			} else if s, t := ts(v), "{=plain(text) {=plainline {=list {=word rule0} {=word xyz}}}}"; s != t {
				erro(ctx, "%v: %s != %s", v, s, t).trace()
			}
			if x, y := (*result).(*plain); !y {
				erro(ctx, "%v: %v", ent, ts(*result)).trace()
			} else if x.name != "text" {
				erro(ctx, "%v: %v", ent, ts(*result)).trace()
			} else if x.len() != 1 {
				erro(ctx, "%v: %v", ent, ts(*result)).trace()
			} else if x, y := x.elems[0].(*plainline); !y {
				erro(ctx, "%v: %v", ent, ts(*result)).trace()
			} else if x.len() != 1 {
				erro(ctx, "%v: %v", ent, ts(*result)).trace()
			} else if x, y := x.elems[0].(*list); !y {
				erro(ctx, "%v: %v", ent, ts(*result)).trace()
			} else if x.len() != 2 {
				erro(ctx, "%v: %v", ent, ts(*result)).trace()
			} else if s := ts(x.elems[0]); s != "{=word rule0}" {
				erro(ctx, "%v: %v", ent, tv(x.elems[0])).trace()
			} else if s := ts(x.elems[1]); s != "{=word xyz}" {
				erro(ctx, "%v: %v", ent, tv(x.elems[0])).trace()
			}
			if s, t := (*result).string(ctx), "rule0 xyz\n"; s != t {
				erro(ctx, "%v: %v: %s != %s", ent, ts((*result)), s, t).trace()
			}
		} else {
			erro(ctx, "%v : %s != %s ; %s", v, s, t, v.string(ctx)).trace()
		}
    }
}

func (prog *program) execute_check_rule_1(ctx Context, result *Value) {
	var ent = _entry(ctx)
    if *result == nil {
        erro(ctx, "%v: nil result", ts(ent)).trace()
    }
}

func (prog *program) execute_check_shell_for_stdout(ctx Context, result *Value) {
	var ent = _entry(ctx)

	if *result == nil {
		erro(ctx, "%v: nil result", ts(ent)).trace()
	}

    var args = try[[]Value](ctx, get_args{})
	var o = try[origin](ctx, get_origin{})

    switch ent.destiny().string(ctx) {
    case ".test.0":
		if len(_execution(ctx).interpreted) == 0 {
            erro(ctx, "%v: not interpreted", ent).trace()
		}
        if len(args) != 2 {
            erro(ctx, "%v: %d %v", ent, len(args), ts(args)).trace()
        }
        if len(prog.params) != 2 {
            erro(ctx, "%v: %d %v", ent, len(prog.params), ts(prog.params)).trace()
        }
		if v := auto_get(ctx, "@"); ts(v) != "{=compound {=punct .} {=word test} {=punct .} {=decimal 0}}" {
			erro(ctx, "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, "<"); ts(v) != "{}" {
			erro(ctx, "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, ">"); ts(v) != "{}" {
			erro(ctx, "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, "-"); ts(v) != ts(*result) {
			errostack(ctx, 3, "%s != %s", ts(v), ts(*result)).trace()
		}
		if v := auto_get(ctx, "-"); v != *result {
			errostack(ctx, 3, "%s != %s", ts(v), ts(*result)).trace()
		}

		switch o {
		case defExpand0:
			if ts(*result) != sfmt("{=delegate {=builtin debug} {=list %s %s}}", ts(args[1]), ts(args[0])) {
				errostack(ctx, 3, "%v (%s %s)", ts(*result), ts(args[1]), ts(args[0])).trace()
			}
		case defExpand1:
			if ts(*result) != "{=null}" {
				errostack(ctx, 3, "%v (%s %s)", ts(*result), ts(args[1]), ts(args[0])).trace()
			}
		}
    case ".test1":
		if v := auto_get(ctx, "@"); ts(v) != "{=compound {=punct .} {=word test1}}" {
			erro(ctx, "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, "<"); ts(v) != "{}" {
			erro(ctx, "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, ">"); ts(v) != "{}" {
			erro(ctx, "%v", ts(v)).trace()
		}
    case ".test2":
		if v := auto_get(ctx, "@"); ts(v) != "{=compound {=punct .} {=word test2}}" {
			erro(ctx, "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, "<"); ts(v) != "{}" {
			erro(ctx, "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, ">"); ts(v) != "{}" {
			erro(ctx, "%v", ts(v)).trace()
		}
    case ".test":
        if len(args) != 2 { // NOTE: always use two args in this test case
            erro(ctx, "%v: %d %v", ent, len(args), ts(args)).trace()
        }
		if v := auto_get(ctx, "@"); ts(v) != "{=compound {=punct .} {=word test}}" {
			erro(ctx, "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, "<"); ts(v) != "{}" {
			erro(ctx, "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, ">"); ts(v) != "{}" {
			erro(ctx, "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, "-"); ts(v) != ts(*result) {
			errostack(ctx, 3, "%s != %s", ts(v), ts(*result)).trace()
		}
		if v := auto_get(ctx, "-"); v != *result {
			errostack(ctx, 3, "%s != %s", ts(v), ts(*result)).trace()
		}

		switch ts(args) {
		case "{=[Value] {=list {=delegate {=auto 1}}} {=list {=delegate {=auto 2}}}}":
			switch o {
			case defExpand0, defExpand1:
				if ts(*result) != "{=delegate {=builtin debug} {=list {=delegate {=auto 2}} {=delegate {=auto 1}}}}" {
					errostack(ctx, 3, ".test: %v, %s", ts(*result), ts(args)).trace()
				}
			}
		default:
			switch o {
			case defExpand0:
				t := sfmt("{=delegate {=builtin debug} {=list %s %s}}", ts(args[1]), ts(args[0]))
				if ts(*result) != t {
					errostack(ctx, 3, ".test: %v != %s, %s", ts(*result), t, ts(args)).trace()
				}
			case defExpand1:
				if ts(*result) != "{=null}" {
					errostack(ctx, 3, ".test: %v %v", ts(*result), ts(args)).trace()
				}
			}
		}
	}

	switch o {
	case defExpand0, defExpand1, 0:
	default:
		errostack(ctx, 5, "untested: %v: %v %s %s", ent.destiny(), o, ts(args), ts(*result)).trace()
	}
}
