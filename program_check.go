//
//  Copyright (C) 2012-2024, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

func (prog *program) execute_check(ctx Context, result *Value) {
	switch prog.project.spec {
	case "testdata/rule/0": prog.execute_check_rule_0(ctx, result)
	case "testdata/rule/1": prog.execute_check_rule_1(ctx, result)
	case "testdata/rule/shell/for-stdout":
		prog.execute_check_shell_for_stdout(ctx, result)
	}
}

func (prog *program) execute_check_rule_0(ctx Context, result *Value) {
	var ent = _entry(ctx)
    if *result == nil {
        erro(ctx, "%v: nil result", ts(ent)).trace()
    }

    var args = try[[]Value](ctx, get_arguments{})

    switch ent.destiny().string(ctx) {
    case "rule0":
        if len(args) != 3 {
            erro(ctx, "%v: %d %v", ent, len(args), ts(args)).trace()
        }
		if v := auto_get(ctx, "@"); ts(v) != "{=word rule0}" {
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, "<"); ts(v) != "{=word rule1}" {
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, ">"); ts(v) != "{=word rule1}" {
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, "^"); ts(v) != "{=list {=word rule1}}" {
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, "-"); ts(v) != "{=list {=word rule1} {=word rule1} {=flag {=null}} {=compound {=word x} {=word y} {=word z}}}" {
			erro(at(ctx,v), "%v", ts(v)).trace()
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
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, "<"); ts(v) != "{}" {
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, ">"); ts(v) != "{}" {
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, "^"); ts(v) != "{}" {
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, "ARG1"); v == nil {
			erro(ctx, "ARG1").trace()
		} else if s := ts(v); s == "" {
			erro(ctx, "%v", v).trace()
		} else if t := "{=word xxyzz}"; s == t {
			if v := auto_get(ctx, "-"); v == nil {
				erro(ctx, "-").trace()
			} else if s, t := ts(v), "{=plain(text) {=word xxyzz}}"; s != t {
				erro(ctx, "%v: %s != %s", v, s, t).trace()
			}
			if x, y := (*result).(*plain); !y {
				erro(ctx, "%v: %v", ent, ts(*result)).trace()
			} else if x.name != "text" {
				erro(ctx, "%v: %v", ent, ts(*result)).trace()
			} else if x.len() != 1 {
				erro(ctx, "%v: %v", ent, ts(*result)).trace()
			} else if s := ts(x.elems[0]); s != "{=word xxyzz}" {
				erro(ctx, "%v: %v", ent, tv(x.elems[0])).trace()
			}
			if (*result).string(ctx) != "xxyzz" {
				erro(ctx, "%v: %v %v", ent, ts(*result), args).trace()
			}
		} else if t := "{=list {=word rule0} {=word xyz}}"; s == t {
			if v := auto_get(ctx, "-"); v == nil {
				erro(ctx, "-").trace()
			} else if s, t := ts(v), "{=plain(text) {=word rule0} {=word xyz}}"; s != t {
				erro(ctx, "%v: %s != %s", v, s, t).trace()
			}
			if x, y := (*result).(*plain); !y {
				erro(ctx, "%v: %v", ent, ts(*result)).trace()
			} else if x.name != "text" {
				erro(ctx, "%v: %v", ent, ts(*result)).trace()
			} else if x.len() != 2 {
				erro(ctx, "%v: %v", ent, ts(*result)).trace()
			} else if s := ts(x.elems[0]); s != "{=word rule0}" {
				erro(ctx, "%v: %v", ent, tv(x.elems[0])).trace()
			} else if s := ts(x.elems[1]); s != "{=word xyz}" {
				erro(ctx, "%v: %v", ent, tv(x.elems[0])).trace()
			}
			if (*result).string(ctx) != "rule0 xyz" {
				erro(ctx, "%v: %v %v", ent, ts(*result), args).trace()
			}
		} else {
			erro(ctx, "%v: %s != %s", v, s, t).trace()
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

    var args = try[[]Value](ctx, get_arguments{})
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
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, "<"); ts(v) != "{}" {
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, ">"); ts(v) != "{}" {
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, "-"); ts(v) != ts(*result) {
			errostack(at(ctx,v), 3, "%s != %s", ts(v), ts(*result)).trace()
		}
		if v := auto_get(ctx, "-"); v != *result {
			errostack(at(ctx,v), 3, "%s != %s", ts(v), ts(*result)).trace()
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
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, "<"); ts(v) != "{}" {
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, ">"); ts(v) != "{}" {
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
    case ".test2":
		if v := auto_get(ctx, "@"); ts(v) != "{=compound {=punct .} {=word test2}}" {
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, "<"); ts(v) != "{}" {
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, ">"); ts(v) != "{}" {
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
    case ".test":
        if len(args) != 2 { // NOTE: always use two args in this test case
            erro(ctx, "%v: %d %v", ent, len(args), ts(args)).trace()
        }
		if v := auto_get(ctx, "@"); ts(v) != "{=compound {=punct .} {=word test}}" {
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, "<"); ts(v) != "{}" {
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, ">"); ts(v) != "{}" {
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, "-"); ts(v) != ts(*result) {
			errostack(at(ctx,v), 3, "%s != %s", ts(v), ts(*result)).trace()
		}
		if v := auto_get(ctx, "-"); v != *result {
			errostack(at(ctx,v), 3, "%s != %s", ts(v), ts(*result)).trace()
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
	case 0, defExpand0, defExpand1:
	default:
		errostack(ctx, 5, "untested: %v: %v %s %s", ent.destiny(), o, ts(args), ts(*result)).trace()
	}
}
