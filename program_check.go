//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	"fmt"
)

func (prog *program) checks() (_ func(Context, *Value)) {
	if prog.project.name == "testrules" {
		return (map[string]func(Context, *Value){
			"testdata/rule/0":                prog.check_rule_0,
			"testdata/rule/1":                prog.check_rule_1,
			"testdata/rule/shell/for-stdout": prog.check_shell_for_stdout,
		})[prog.project.spec]
	}
	return
}

func (prog *program) check_rule_0(ctx Context, result *Value) {
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
		if v := auto_get(ctx, "@"); ts(v) != "{=bareword rule0}" {
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, "<"); ts(v) != "{=bareword rule1}" {
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, ">"); ts(v) != "{=bareword rule1}" {
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, "^"); ts(v) != "{=list {=bareword rule1}}" {
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, "-"); ts(v) != "{=list {=bareword rule1} {=bareword rule1} {=flag {=null}} {=barecomp {=bareword x} {=bareword y} {=bareword z}}}" {
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
		if auto_get(ctx, "<") != auto_get(ctx, ">") {
			erro(ctx, "%v %v", ts(auto_get(ctx, "<")), ts(auto_get(ctx, ">"))).trace()
		}
        if x, y := (*result).(*list); !y {
            erro(ctx, "%v: %v", ent, ts(*result)).trace()
        } else if x.len() != 4 {
            erro(ctx, "%v: %v", ent, ts(*result)).trace()
        } else if s := ts(x.elems[0]); s != "{=bareword rule1}" {
            erro(ctx, "%v: %v", ent, s).trace()
        } else if s := ts(x.elems[1]); s != "{=bareword rule1}" {
            erro(ctx, "%v: %v", ent, s).trace()
        } else if s := ts(x.elems[2]); s != "{=flag {=null}}" {
            erro(ctx, "%v: %v", ent, s).trace()
        } else if s := ts(x.elems[3]); s != "{=barecomp {=bareword x} {=bareword y} {=bareword z}}" {
            erro(ctx, "%v: %v", ent, s).trace()
        }
        if ts(*result) != "{=list {=bareword rule1} {=bareword rule1} {=flag {=null}} {=barecomp {=bareword x} {=bareword y} {=bareword z}}}" {
            erro(ctx, "%v: %v, %v", ent, ts(*result), args).trace()
        }
    case "rule1":
        if len(args) != 1 {
            erro(ctx, "%v: %d %v", ent, len(args), ts(args)).trace()
        }
		if v := auto_get(ctx, "@"); ts(v) != "{=bareword rule1}" {
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, "<"); ts(v) != "{}" {
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, ">"); ts(v) != "{}" {
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
		if v, s := auto_get(ctx, "-"), "{=plain {=bareword rule0} {=bareword xyz}}"; ts(v) == s {
			if x, y := (*result).(*plain); !y {
				erro(ctx, "%v: %v", ent, ts(*result)).trace()
			} else if x.len() != 2 {
				erro(ctx, "%v: %v", ent, ts(*result)).trace()
			} else if s := ts(x.elems[0]); s != "{=bareword rule0}" {
				erro(ctx, "%v: %v", ent, tv(v)).trace()
			} else if s := ts(x.elems[1]); s != "{=bareword xyz}" {
				erro(ctx, "%v: %v", ent, tv(v)).trace()
			}
			if (*result).string(ctx) != "rule0 xyz" {
				erro(ctx, "%v: %v %v", ent, ts(*result), args).trace()
			}
		} else if ts(v) == "{=plain {=bareword xyz}}" {
			if x, y := (*result).(*plain); !y {
				erro(ctx, "%v: %v", ent, ts(*result)).trace()
			} else if x.len() != 1 {
				erro(ctx, "%v: %v", ent, ts(*result)).trace()
			} else if s = ts(x.elems[0]); s != "{=bareword xyz}" {
				erro(ctx, "%v: %v", ent, s).trace()
			}
		} else {
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
    }
}

func (prog *program) check_rule_1(ctx Context, result *Value) {
	var ent = _entry(ctx)
    if *result == nil {
        erro(ctx, "%v: nil result", ts(ent)).trace()
    }
}

func (prog *program) check_shell_for_stdout(ctx Context, result *Value) {
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
		if v := auto_get(ctx, "@"); ts(v) != "{=barecomp {=punctuation .} {=bareword test} {=punctuation .} {=decimal 0}}" {
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
			if ts(*result) != fmt.Sprintf("{=delegate {=builtin debug} {=list %s %s}}", ts(args[1]), ts(args[0])) {
				errostack(ctx, 3, "%v (%s %s)", ts(*result), ts(args[1]), ts(args[0])).trace()
			}
		case defExpand1:
			if ts(*result) != "{=null}" {
				errostack(ctx, 3, "%v (%s %s)", ts(*result), ts(args[1]), ts(args[0])).trace()
			}
		}
    case ".test1":
		if v := auto_get(ctx, "@"); ts(v) != "{=barecomp {=punctuation .} {=bareword test1}}" {
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, "<"); ts(v) != "{}" {
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
		if v := auto_get(ctx, ">"); ts(v) != "{}" {
			erro(at(ctx,v), "%v", ts(v)).trace()
		}
    case ".test2":
		if v := auto_get(ctx, "@"); ts(v) != "{=barecomp {=punctuation .} {=bareword test2}}" {
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
		if v := auto_get(ctx, "@"); ts(v) != "{=barecomp {=punctuation .} {=bareword test}}" {
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
				t := fmt.Sprintf("{=delegate {=builtin debug} {=list %s %s}}", ts(args[1]), ts(args[0]))
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
