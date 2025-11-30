//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

import (
	"regexp"
	"strings"
	"fmt"
)

func (ctx *__trimprefix) check_match(val, prefix Value, f bool, r any, m []string) {
    var v = __string(ctx, val)
    switch prefix.String() {
    case "%%/.smart/modules/":
        if s := __string(ctx, val); strings.Contains(s, "/.smart/modules/") {
            if a, y := r.([]string); !y {
                erro(ctx, "%v %v, %v %v %v", prefix, val, f, r, m).trace()
            } else if len(a) < 4 || a[len(a)-1] != "" {
                erro(ctx, "%v %v, %v %v %v", prefix, val, f, r, m).trace()
            }
        }
    case "/**/testdata/":
        if f != false {
            erro(ctx, "%v : %v %v %v", tv(prefix), f, r, m).trace()
        }
        if x, y := r.([]string); !y || strings.TrimPrefix(v, joinpath(x...)) != "builtins/trimprefix" {
            erro(ctx, "%v : %v %v %v", tv(prefix), f, r, m).trace()
        }
        if len(m) != 1 {
            erro(ctx, "%v : %v %v %v", tv(prefix), f, r, m).trace()
        } else if strings.TrimPrefix(v, "/"+m[0]) != "/testdata/builtins/trimprefix" {
            note(ctx, "/%v", m[0])
            note(ctx, "%v", val)
            erro(ctx, "%v : %v %v %v", tv(prefix), f, r, m).trace()
        }
    }
}

func (ctx *__trimprefix) check(prefix, val, res Value) {
    var pre, str, t = __string(ctx, prefix), __string(ctx, val), __string(ctx, res)

    if strings.HasSuffix(pre, "/") && strings.HasPrefix(str, pre) && strings.HasPrefix(t, "/") {
        erro(ctx, "{=%s %v} {=%s %v} {=%s %v}", typeof(prefix), prefix, typeof(val), val, typeof(res), res).trace()
    }

	var proj = _project(ctx)

    if p := "/.smart/modules/"; pre == "%%"+p {
        var s = __string(ctx, val)
        if i := strings.Index(s, p); 0 < i && s[i+len(p):] != t {
            erro(ctx, "%v %s, %s != %s", prefix, s, t, s[i+len(p):]).trace()
        }

		// For trim-prefix as in (from 'general/do.smart')
		//   rel.remnant = $(trim-prefix &(rel.chop),&/)
		if x, y := do(ctx, evoke_def{"rel.remnant"}).(*def); y {
			if x.value.String() == "$(trim-prefix &(rel.chop),&/)" {
				var a = ctx.evocation.a[1]
				if x, y := a.(*list); !y || x.len() != 1 {
					erro(ctx, "not a list: %v", ts(a)).trace()
				} else {
					a = x.elems[0] // aka: &/
				}

				var cp = closure_projects(ctx)
				var r1 = cp[0].resolveDef(ctx, "/").value
				var r2 =  proj.resolveDef(ctx, "/").value
				if false { note(ctx, "%-22v: %-50v; %-20v, %v, %v", proj, a, res, r1, r2) }

				if ts(a) != ts(r1) {
					note(ctx, "%v: %v, %v", proj.name, cp, ts(a))
					note(ctx, "%v: %v, %v", proj.name, cp, ts(r1))
					note(ctx, "%v: %v, %v", proj.name, cp, ts(r2))
					erro(ctx, "%v != %v : %v", a, r1, res).trace()
				}

				if proj.name == cp[0].name {
					if ts(r1) != ts(r2) {
						erro(ctx, "%-22v: %-50v; %-20v, %v != %v", proj, a, res, r1, r2).trace()
					}
				}

				switch proj.name {
				case "general":
				case "configure.base":
				case "lib.c++.abi":
				case "lib.c++.inc":
				case "testdefaultconfigure":
				case "testdeftwoconfigure":
				case "testcustomconfigure":
				}
			}
		}

		if true {/*FIXME: fail for configures */} else
		if x, y := do(ctx, evoke_def{"outtmp"}).(*def); y {
			if x.value.String() == "&(target.tmp)/&(rel.remnant)" {
				switch proj.name {
				case "configure.base":
					if ts(res) != "{=path {=word configure} {=word .base}}" {
						erro(ctx, "%v : %v ; %v", proj.name, ts(res), x).trace()
					}
				}
			}
		}
    }
}

func (ctx *__auto) check(res []Value) {
	switch j := _project(ctx); j.spec {
	case "testdata/value/auto":
		ctx.check_value_auto(res)
	default:
		erro(ctx, "%v: %v", j.spec, res).trace()
	}
}
func (ctx *__auto) check_value_auto(res []Value) {
 	switch d, o := try[*def](ctx, origin_def{}), try[origin](ctx, get_origin{}); line_column(ctx) {
	case "6:11":
		if s, t := ts(res), `[{=list {6:29:null}}]`; s != t {
			errostack(ctx, 3, "%v | %s != %s", res, s, t).trace()
		}
	case "7:11":
		if s, t := ts(res), `[{=list {7:21 {6:9 {6:29:null}}}} {=list {7:29 {7:18:decimal 2}}}]`; s != t {
			errostack(ctx, 3, "%v | %s != %s", res, s, t).trace()
		}
	case "8:11":
		if s, t := ts(res), `[{=list {=list {8:21 {7:9 {7:21 {6:9 {6:29:null}}}}} {8:21 {7:9 {7:29 {7:18:decimal 2}}}}}}]`; s != t {
			errostack(ctx, 3, "%v | %s != %s", res, s, t).trace()
		}
	case "10:11":
		switch try[string](ctx,source{}) {
		case "value_test.go:643":
			if s, t := ts(res), `[{=list {10:30:null}}]`; s != t {
				errostack(ctx, 5, "%v | %v: %s != %s", d, res, s, t).trace()
			}
		case "value_test.go:645":
			if s, t := ts(res), `[{=list {10:30 {1:9:word x}}}]`; s != t {
				errostack(ctx, 5, "%v | %v: %s != %s", d, res, s, t).trace()
			}
		case "value_test.go:668", "value_test.go:687", "value_test.go:706":
			if s, t := ts(res), `[{=list {10:30 {11:18:decimal 2}}}]`; s != t {
				errostack(ctx, 5, "%v | %v: %s != %s", d, res, s, t).trace()
			}
		default:
			errostack(ctx, 10, "%v | %s", res, ts(res)).trace()
		}
	case "11:11":
		switch try[string](ctx,source{}) {
		case "value_test.go:668", "value_test.go:687", "value_test.go:706":
			if s, t := ts(res), `[{=list {11:21 {10:9 {10:30 {11:18:decimal 2}}}}} {=list {11:30 {11:18:decimal 2}}}]`; s != t {
				errostack(ctx, 5, "%v | %v: %s != %s", d, res, s, t).trace()
			}
		default:
			errostack(ctx, 5, "%v | %s", res, ts(res)).trace()
		}
	case "12:11":
		switch try[string](ctx,source{}) {
		case "value_test.go:668", "value_test.go:687", "value_test.go:706":
			if s, t := ts(res), `[{=list {=list {12:21 {11:9 {11:21 {10:9 {10:30 {11:18:decimal 2}}}}}} {12:21 {11:9 {11:30 {11:18:decimal 2}}}}}}]`; s != t {
				errostack(ctx, 5, "%v | %v: %s != %s", d, res, s, t).trace()
			}
		default:
			errostack(ctx, 5, "%v | %s", res, ts(res)).trace()
		}
	case "19:15":
		switch line_column(d) {
		case "19:11":
			switch try[string](ctx,source{}) {
			case "value_test.go:750":
				if s, t := ts(res), `[{=list {=compound {19:46:null} {=flag {=compound {19:52:null} {=flag {19:58 {19:33:decimal 3}}}}}}}]`; s != t {
					errostack(pc(ctx,d), 5, "%s: %v: %s != %s", d.name, res, s, t).trace()
				}
			case "value_test.go:752":
				if s, t := ts(res), `[{=list {=compound {19:46 {1:9:word x}} {=flag {=compound {19:52 {1:9:word y}} {=flag {19:58 {19:33:decimal 3}}}}}}}]`; s != t {
					errostack(pc(ctx,d), 5, "%s: %v: %s != %s", d.name, res, s, t).trace()
				}
			default:
				errostack(pc(ctx,d), 5, "%v: %s", res, ts(res)).trace()
			}
		case "21:11":
			switch try[string](ctx,source{}) {
			case "value_test.go:786", "value_test.go:788":
				if s, t := ts(res), `[{=list {=compound {19:46 {21:23:word x}} {=flag {=compound {19:52 {21:28:word y}} {=flag {19:58 {19:33:decimal 3}}}}}}}]`; s != t {
					errostack(pc(ctx,d), 5, "%s: %v: %s != %s", d.name, res, s, t).trace()
				}
			default:
				errostack(pc(ctx,d), 5, "%v: %s", res, ts(res)).trace()
			}
		case "22:10":
			switch try[string](ctx,source{}) {
			case "value_test.go:820", loader_src:
				if s, t := ts(res), `[{=list {=compound {19:46 {22:23:word x}} {=flag {=compound {19:52 {22:28:word y}} {=flag {19:58 {19:33:decimal 3}}}}}}}]`; s != t {
					errostack(pc(ctx,d), 5, "%s: %v: %s != %s", d.name, res, s, t).trace()
				}
			default:
				errostack(pc(ctx,d), 5, "%v: %s", res, ts(res)).trace()
			}
		case "23:11":
			switch try[string](ctx,source{}) {
			case "value_test.go:818":
				if s, t := ts(res), `[{=list {=compound {19:46:null} {=flag {=compound {19:52:null} {=flag {19:58 {19:33:decimal 3}}}}}}}]`; s != t {
					errostack(pc(ctx,d), 5, "%s: %v: %s != %s", d.name, res, s, t).trace()
				}
			case "value_test.go:820", "value_test.go:824":
				if s, t := ts(res), `[{=list {=compound {19:46 {1:9:word x}} {=flag {=compound {19:52 {1:9:word y}} {=flag {19:58 {19:33:decimal 3}}}}}}}]`; s != t {
					errostack(pc(ctx,d), 5, "%s: %v: %s != %s", d.name, res, s, t).trace()
				}
			default:
				errostack(pc(ctx,d), 5, "%v: %s", res, ts(res)).trace()
			}
		case "24:10":
			if s, t := ts(res), `[{=list {=compound {19:46:null} {=flag {=compound {19:52:null} {=flag {19:58 {19:33:decimal 3}}}}}}}]`; s != t {
				errostack(ctx, 3, "%v | %s != %s", res, s, t).trace()
			}
		case "26:9":
			if s, t := ts(res), `[{=list {=compound {19:46 {21:23:word x}} {=flag {=compound {19:52 {21:28:word y}} {=flag {19:58 {19:33:decimal 3}}}}}}}]`; s != t {
				errostack(ctx, 3, "%s: %v | %s != %s", line_column(d), res, s, t).trace()
			}
		case "27:9":
			if s, t := ts(res), `[{=list {=compound {19:46 {21:23:word x}} {=flag {=compound {19:52 {21:28:word y}} {=flag {19:58 {19:33:decimal 3}}}}}}}]`; s != t {
				errostack(ctx, 3, "%s: %v | %s != %s", line_column(d), res, s, t).trace()
			}
		case "28:9":
			if s, t := ts(res), `[{=list {=compound {19:46 {28:22:word a}} {=flag {=compound {19:52 {28:27:word b}} {=flag {19:58 {19:33:decimal 3}}}}}}}]`; s != t {
				errostack(ctx, 5, "%s: %v | %s != %s", line_column(d), res, s, t).trace()
			}
		default:
			errostack(pc(ctx,d), 3, "%v | %s", res, ts(res)).trace()
		}
	case "20:15":
		if s, t := ts(res), `[{=list {=compound {20:46:null} {=flag {=compound {20:52:null} {=flag {20:58 {20:33:decimal 3}}}}}}}]`; s != t {
			errostack(ctx, 3, "%v | %s != %s", res, s, t).trace()
		}
	case "21:15":
		if s, t := ts(res), `[{=list {=compound {21:36 {19:13 {19:46 {21:23:word x}}}} {21:36 {19:13 {=flag {=compound {19:52 {21:28:word y}} {=flag {19:58 {19:33:decimal 3}}}}}}} {=flag {=compound {21:48 {21:23:word x}} {21:53 {21:28:word y}} {21:58 {21:33:word z}}}}}}]`; s != t {
			errostack(ctx, 3, "%v | %s != %s", res, s, t).trace()
		}
	case "22:15":
		if s, t := ts(res), `[{=list {=compound {22:36 {19:13 {19:46 {22:23:word x}}}} {22:36 {19:13 {=flag {=compound {19:52 {22:28:word y}} {=flag {19:58 {19:33:decimal 3}}}}}}} {=flag {=compound {22:48 {22:23:word x}} {22:53 {22:28:word y}} {22:58 {22:33:word z}}}}}}]`; s != t {
			errostack(ctx, 3, "%v | %s != %s", res, s, t).trace()
		}
	case "23:15":
		switch try[string](ctx,source{}) {
		case "value_test.go:818":
			if s, t := ts(res), "[{=list {=compound {23:36 {19:13 {19:46:null}}} {23:36 {19:13 {=flag {=compound {19:52:null} {=flag {19:58 {19:33:decimal 3}}}}}}} {=flag {=compound {23:48:null} {23:53:null} {23:58:null}}}}}]"; s != t {
				errostack(ctx, 5, "%v: %s != %s", o, s, t).trace()
			}
		case "value_test.go:820", "value_test.go:824":
			if s, t := ts(res), "[{=list {=compound {23:36 {19:13 {19:46 {1:9:word x}}}} {23:36 {19:13 {=flag {=compound {19:52 {1:9:word y}} {=flag {19:58 {19:33:decimal 3}}}}}}} {=flag {=compound {23:48 {1:9:word x}} {23:53 {1:9:word y}} {23:58:null}}}}}]"; s != t {
				errostack(ctx, 5, "%v: %s != %s", o, s, t).trace()
			}
		case loader_src:
			switch line_column(d) {
			case "28:9":
				if s, t := ts(res), `[{=list {=compound {23:36 {19:13 {19:46 {28:22:word a}}}} {23:36 {19:13 {=flag {=compound {19:52 {28:27:word b}} {=flag {19:58 {19:33:decimal 3}}}}}}} {=flag {=compound {23:48 {28:22:word a}} {23:53 {28:27:word b}} {23:58 {28:32:word c}}}}}}]`; s != t {
					errostack(ctx, 3, "%v | %s != %s", res, s, t).trace()
				}
			default:
				errostack(ctx, 3, "%v | %v: %s", d, res, ts(res)).trace()
			}
		default:
			errostack(ctx, 5, "%v: %v: %s", d, res, ts(res)).trace()
		}
	case "24:15":
		if s, t := ts(res), `[{=list {=compound {24:36 {19:13 {19:46:null}}} {24:36 {19:13 {=flag {=compound {19:52:null} {=flag {19:58 {19:33:decimal 3}}}}}}} {=flag {=compound {24:48:null} {24:53:null} {24:58:null}}}}}]`; s != t {
			errostack(ctx, 3, "%v | %s != %s", res, s, t).trace()
		}
	case "27:14":
		if s, t := ts(res), `[{=list {27:35 {21:13 {=compound {21:36 {19:13 {19:46 {21:23:word x}}}} {21:36 {19:13 {=flag {=compound {19:52 {21:28:word y}} {=flag {19:58 {19:33:decimal 3}}}}}}} {=flag {=compound {21:48 {21:23:word x}} {21:53 {21:28:word y}} {21:58 {21:33:word z}}}}}}}}]`; s != t {
			errostack(ctx, 3, "%v | %s != %s", res, s, t).trace()
		}
	case "28:14":
		if s, t := ts(res), `[{=list {28:35 {23:13 {=compound {23:36 {19:13 {19:46 {28:22:word a}}}} {23:36 {19:13 {=flag {=compound {19:52 {28:27:word b}} {=flag {19:58 {19:33:decimal 3}}}}}}} {=flag {=compound {23:48 {28:22:word a}} {23:53 {28:27:word b}} {23:58 {28:32:word c}}}}}}}}]`; s != t {
			errostack(ctx, 3, "%v | %s != %s", res, s, t).trace()
		}
	default:
		errostack(ctx, 3, "%v: %v | %s", o, res, ts(res)).trace()
	}
}

func (ctx *__grep) check(rx *regexp.Regexp, text string, temp, val Value) {
	if d, y := ctx.defs["0"]; !y {
		erro(ctx, "%v %v %v %v", rx, text, temp, val).trace()
	} else if t := __string(ctx, d); t != text {
		erro(ctx, "%v: %v: %s != %s", rx, d, t, text).trace()
	}
	switch j := _project(ctx); j.name {
	case "llvm.Config":
		switch rx.String() {
		case `^ *set\((LLVM_VERSION_(MAJOR|MINOR|PATCH)) +([0-9]+) *\)`:
			if v := auto_get(ctx, "2"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else {
				switch __string(ctx,v) {
				case "MAJOR":
					if v := auto_get(ctx, "0"); v == nil {
						erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
					} else if s, t := "  set(LLVM_VERSION_MAJOR 20)", __string(ctx,v); s != t {
						erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
					}
					if v := auto_get(ctx, "3"); v == nil {
						erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
					} else if s, t := "20", __string(ctx,v); s != t {
						erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
					}
				case "MINOR":
					if v := auto_get(ctx, "0"); v == nil {
						erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
					} else if s, t := "  set(LLVM_VERSION_MINOR 0)", __string(ctx,v); s != t {
						erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
					}
					if v := auto_get(ctx, "3"); v == nil {
						erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
					} else if s, t := "0", __string(ctx,v); s != t {
						erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
					}
				case "PATCH":
					if v := auto_get(ctx, "0"); v == nil {
						erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
					} else if s, t := "  set(LLVM_VERSION_PATCH 0)", __string(ctx,v); s != t {
						erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
					}
					if v := auto_get(ctx, "3"); v == nil {
						erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
					} else if s, t := "0", __string(ctx,v); s != t {
						erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
					}
				default:
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				}
			}
		default:
			note(ctx, "%s: %v; %v; %v; %v; %v", j.name, rx, text, temp, val, ctx.defs).debug(2)
		}
	case "testvalue":
		switch rx.String() {
		case `^.+?\.o$`:
			switch text {
			case "foo.o":
				if v := auto_get(ctx, "0"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "foo.o", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
			case "foo-x.o":
				if v := auto_get(ctx, "0"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "foo-x.o", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
			case "foo-x-y.o":
				if v := auto_get(ctx, "0"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "foo-x-y.o", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
			case "foo-x-y-z.o":
				if v := auto_get(ctx, "0"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "foo-x-y-z.o", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
			case "foobar.o":
				if v := auto_get(ctx, "0"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "foobar.o", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
			default:
				note(ctx, "%v; %v; %v; %v; %v", rx, text, temp, val, ctx.defs).debug(2)
			}
		case `^(.+?)((-(?P<i>.+?))*)(\.)(?P<x>o)$`:
			switch text {
			case "foo.o":
				if d, y := ctx.defs["1"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "foo", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if d, y := ctx.defs["2"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if d, y := ctx.defs["3"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if _, y := ctx.defs["4"]; y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				}
				if d, y := ctx.defs["i"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if d, y := ctx.defs["5"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := ".", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if _, y := ctx.defs["6"]; y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				}
				if d, y := ctx.defs["x"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "o", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if v := auto_get(ctx, "0"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "foo.o", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "1"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "foo", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "2"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "3"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "4"); v != nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				}
				if v := auto_get(ctx, "i"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "5"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := ".", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "6"); v != nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				}
				if v := auto_get(ctx, "x"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "o", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
			case "foo-x.o":
				if d, y := ctx.defs["1"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "foo", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if d, y := ctx.defs["2"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "-x", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if d, y := ctx.defs["3"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "-x", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if _, y := ctx.defs["4"]; y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				}
				if d, y := ctx.defs["i"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "x", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if d, y := ctx.defs["5"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := ".", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if _, y := ctx.defs["6"]; y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				}
				if d, y := ctx.defs["x"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "o", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if v := auto_get(ctx, "0"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "foo-x.o", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "1"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "foo", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "2"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "-x", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "3"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "-x", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "4"); v != nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				}
				if v := auto_get(ctx, "i"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "x", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "5"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := ".", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "6"); v != nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				}
				if v := auto_get(ctx, "x"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "o", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
			case "foo-x-y.o":
				if d, y := ctx.defs["1"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "foo", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if d, y := ctx.defs["2"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "-x-y", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if d, y := ctx.defs["3"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "-y", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if _, y := ctx.defs["4"]; y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				}
				if d, y := ctx.defs["i"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "y", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if d, y := ctx.defs["5"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := ".", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if _, y := ctx.defs["6"]; y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				}
				if d, y := ctx.defs["x"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "o", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if v := auto_get(ctx, "0"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "foo-x-y.o", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "1"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "foo", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "2"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "-x-y", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "3"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "-y", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "4"); v != nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				}
				if v := auto_get(ctx, "i"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "y", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "5"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := ".", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "6"); v != nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				}
				if v := auto_get(ctx, "x"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "o", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
			case "foo-x-y-z.o":
				if d, y := ctx.defs["1"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "foo", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if d, y := ctx.defs["2"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "-x-y-z", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if d, y := ctx.defs["3"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "-z", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if _, y := ctx.defs["4"]; y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				}
				if d, y := ctx.defs["i"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "z", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if d, y := ctx.defs["5"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := ".", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if _, y := ctx.defs["6"]; y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				}
				if d, y := ctx.defs["x"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "o", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if v := auto_get(ctx, "0"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "foo-x-y-z.o", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "1"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "foo", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "2"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "-x-y-z", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "3"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "-z", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "4"); v != nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				}
				if v := auto_get(ctx, "i"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "z", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "5"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := ".", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "6"); v != nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				}
				if v := auto_get(ctx, "x"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "o", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
			case "foobar.o":
				if d, y := ctx.defs["1"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "foobar", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if d, y := ctx.defs["2"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if d, y := ctx.defs["3"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if _, y := ctx.defs["4"]; y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				}
				if d, y := ctx.defs["i"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if d, y := ctx.defs["5"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := ".", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if _, y := ctx.defs["6"]; y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				}
				if d, y := ctx.defs["x"]; !y {
					erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "o", __string(ctx, d); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
				}
				if v := auto_get(ctx, "0"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "foobar.o", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "1"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "foobar", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "2"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "3"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "4"); v != nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				}
				if v := auto_get(ctx, "i"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "5"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := ".", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
				if v := auto_get(ctx, "6"); v != nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				}
				if v := auto_get(ctx, "x"); v == nil {
					erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
				} else if s, t := "o", __string(ctx,v); s != t {
					erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
				}
			default:
				note(ctx, "%v; %v; %v; %v; %v", rx, text, temp, val, ctx.defs).debug(2)
			}
		default:
			note(ctx, "%v; %v; %v; %v; %v", rx, text, temp, val, ctx.defs).debug(2)
		}
	case "lib.unwind":
		switch rx.String() {
		case `^#define +_LIBUNWIND_VERSION +([0-9]+)`:
			if v := auto_get(ctx, "0"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := `#define _LIBUNWIND_VERSION 15000`, __string(ctx,v); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "1"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "15000", __string(ctx,v); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
		default:
			note(ctx, "%s: %v; %v; %v; %v; %v", j.name, rx, text, temp, val, ctx.defs).debug(2)
		}
	default:
		note(ctx, "%s: %v; %v; %v; %v; %v", j.name, rx, text, temp, val, ctx.defs).debug(2)
	}
}

func (ctx *__foreach) check(_values, _vals *[]Value) {
	switch j := _project(ctx); j.spec {
	case "testdata/value/placeholder":
		ctx.value_placeholder(*_values, *_vals)
	case "testdata/value/optional":
		ctx.value_optional(*_values, *_vals)
	case "testdata/value/bug_01":
		ctx.value_bug_01(*_values, *_vals)
	case "testdata/builtins/foreach":
		ctx.check__foreach(*_values, *_vals)
	}
}

func (ctx *__foreach) value_placeholder(values, vals []Value) {
	switch o := try[origin](ctx, get_origin{}); line_column(ctx) {
	case "4:11":
		switch try[string](ctx,source{}) {
		case "value_test.go:1911":
			if s, t := ts(vals), `[{4:31 {4:19:word a}} {4:31 {4:21:word b}} {4:31 {4:23:word c}} {4:31 {4:25:word d}} {4:31 {4:27:word e}} {4:31 {4:29:word f}}]`; s != t {
				errostack(ctx, 5, "%v | %s != %s", values, s, t).trace()
			}
		default:
			errostack(ctx, 5, "%v: %v | %v: %s", o, values, vals, ts(vals)).trace()
		}
	case "5:11":
		switch try[string](ctx,source{}) {
		case "value_test.go:1917":
			if s, t := ts(vals), `[]`; s != t {
				errostack(ctx, 5, "%v: %v | %s != %s", o, values, s, t).trace()
			}
		case "value_test.go:1903":
			if s, t := ts(vals), `[{5:46 {5:19 {1:9:word 1}}} {5:46 {5:22 {1:9:word 2}}} {5:46 {5:25 {1:9:word 3}}} {5:46 {5:28 {1:9:word 4}}} {5:46 {5:31 {1:9:word 5}}} {5:46 {5:34 {1:9:word 6}}} {5:46 {5:37 {1:9:word 7}}} {5:46 {5:40 {1:9:word 8}}} {5:46 {5:43 {1:9:word 9}}}]`; s != t {
				errostack(ctx, 5, "%v: %v | %s != %s", o, values, s, t).trace()
			}
		default:
			errostack(ctx, 5, "%v: %v | %v: %s", o, values, vals, ts(vals)).trace()
		}
	case "6:11":
		switch try[string](ctx,source{}) {
		case loader_src:
			if s, t := ts(vals), `[{6:31 {6:19:word a}} {6:31 {6:21:word b}} {6:31 {6:23:word c}} {6:31 {6:25:word d}} {6:31 {6:27:word e}} {6:31 {6:29:word f}}]`; s != t {
				errostack(ctx, 5, "%v: %v | %s != %s", o, values, s, t).trace()
			}
		default:
			errostack(ctx, 5, "%v: %v | %v: %s", o, values, vals, ts(vals)).trace()
		}
	case "7:11":
		switch try[string](ctx,source{}) {
		case loader_src:
			if s, t := ts(vals), `[]`; s != t {
				errostack(ctx, 5, "%v: %v | %s != %s", o, values, s, t).trace()
			}
		default:
			errostack(ctx, 5, "%v: %v | %v: %s", o, values, vals, ts(vals)).trace()
		}
	default:
		errostack(pc(ctx,values[0]), 5, "%v: %v | %v: %s", o, values, vals, ts(vals)).trace()
	}
}

func (ctx *__foreach) value_optional(values, vals []Value) {
	switch ctx.a[1].String() {
	case "$($_→bar?)":
		if truly(ctx, ex_def_1{}) {
			if vals != nil {
				errostack(ctx, 3, "%v ; %v", vals, auto_get(ctx, "_")).trace()
			}
		} else {
			if vals == nil || vals[0] == nil {
				errostack(ctx, 3, "nil ; %v", auto_get(ctx, "_")).trace()
			}
			if s, t := vals[0].String(), "$({{=project foo}}→bar?)"; s != t {
				errostack(ctx, 3, "%s != %s ; %v", s, t, auto_get(ctx, "_")).trace()
			}
		}
	}
}

func (ctx *__foreach) value_bug_01(values, vals []Value) {
}

func (ctx *__foreach) check__foreach(values, vals []Value) {
	var s string
	var d, _ = do(ctx, evoke_def{}).(*def)
	switch fmt.Sprintf("%v", ctx.a[1:]) {
	case "[x$_]":
		s += "["
		for i, v := range values {
			if i > 0 { s += " " }
			s += "x"
			if d == nil {
				s += v.String()
			} else {
				switch d.name {
				case ".test.1", ".test.21", ".test.22":
					s += v.String()
				case ".test.2", ".test.23":
					s += redis(v).String()
				default:
					erro(pc(ctx,d.value), "%v %v ; %v", values, vals, d).trace()
				}
			}
		}
		s += "]"
		if t := fmt.Sprintf("%v", vals); s != t {
			erro(pc(ctx,vals), "%s != %s ; %v", s, t, d).trace()
		}
	case "[&(.test.h)$_]":
		if d, _ := do(ctx, evoke_def{}).(*def); d == nil {
			erro(pc(ctx,vals[0]), "%v %v", values, vals).trace()
		} else {
			switch args := fmt.Sprintf("%v", values); d.name {
			case ".test.1", ".test.2":
				switch args {
				case "[a b c]":
					if truly(ctx, ex_closure{}) {
						s = "[-a -b -c]"
					} else {
						s = "[&(.test.h)a &(.test.h)b &(.test.h)c]"
					}
				default:
					erro(pc(ctx,vals[0]), "%s: %v %v", d.name, args, vals).trace()
				}
			default:
				erro(pc(ctx,vals[0]), "%s: %v %v", d.name, args, vals).trace()
			}
			if t := fmt.Sprintf("%v", vals); s != t {
				erro(pc(ctx,vals[0]), "%s: %s != %s", d.name, s, t).trace()
			}
		}
	case "[&(.test.xx)$_]":
		s += "["
		for i, v := range values {
			if i > 0 { s += " " }
			if !truly(ctx, ex_closure{}) {
				s += "&(.test.xx)"
			}
			if d == nil {
				s += v.String()
			} else {
				switch d.name {
				case ".test.23":
					s += redis(v).String()
				default:
					erro(pc(ctx,d.value), "%v %v ; %v", values, vals, d).trace()
				}
			}
		}
		s += "]"
		if t := fmt.Sprintf("%v", vals); s != t {
			erro(ctx, "%s != %s", s, t).trace()
		}
	case "[&(.test.$_.$(or $4,$3))]":
		j := _project(ctx)
		v3 := auto_get(ctx, "3")
		v4 := auto_get(ctx, "4")
		s += "["
		for i, v := range values {
			if i > 0 { s += " " }

			t := ".test."+v.String()
			if v4 != nil {
				t += v4.String()
			} else if v3 != nil {
				t += v3.String()
			}

			if d := j.resolveDef(ctx, t); d == nil {
				erro(ctx, "%v", t).trace()
			} else if truly(ctx, ex_closure{}) {
				s += ""
			} else /* if truly(ctx, ex_delegate{}) */ {
				s += "{&(.test.xx)}"
			}
		}
		s += "]"
		if t := fmt.Sprintf("%v", vals); s != t {
			erro(ctx, "%s != %s", s, t).trace()
		}
	case "[$(closure .test.$_)$1{}99]":
		j := _project(ctx)
		v1 := auto_get(ctx, "1")
		s += "["
		for i, v := range values {
			if i > 0 { s += " " }

			t := ".test."+v.String()
			if d := j.resolveDef(ctx, t); d == nil {
				erro(ctx, "%v", t).trace()
			} else if truly(ctx, ex_closure{}) {
				s += ""
			} else /* if truly(ctx, ex_delegate{}) */ {
				s += "{&(.test.xx)}"
			}
			s += v1.String()
			s += "99"
		}
		s += "]"
		if t := fmt.Sprintf("%v", vals); s != t {
			erro(ctx, "%s != %s", s, t).trace()
		}
	case "[&(.test.x $_)]":
		s += "["
		for i, v := range values {
			if i > 0 { s += " " }
			if truly(ctx, ex_closure{}) {
				s += ""
			} else /* if truly(ctx, ex_delegate{}) */ {
				s += "{&(.test.xx)}"
				s += v.String()
			}
		}
		s += "]"
		if t := fmt.Sprintf("%v", vals); s != t {
			erro(ctx, "%s != %s", s, t).trace()
		}
	case "[&(.test.$_)$1{}zz]":
		s += "["
		for i, v := range values {
			if i > 0 { s += " " }
			if truly(ctx, ex_closure{}) {
				s += ""
			} else /* if truly(ctx, ex_delegate{}) */ {
				s += "{&(.test.xx)}"
				s += v.String()
			}
		}
		s += "]"
		if t := fmt.Sprintf("%v", vals); s != t {
			erro(ctx, "%s != %s", s, t).trace()
		}
	default:
		erro(ctx, "%d %v %v %v", len(values), values, vals, ctx.a).trace()
	}
}

func (ctx *__foreach) check_v(v Value) {
	if len(ctx.evocation.a) == 0 { return }

	var t = ctx.evocation.a[1]
	if d, y := v.(*delegate); y && d.x.String() == "if" {
		if len(d.a) == 2 && strings.Contains(d.a[1].String(), "$_") {
			if l, y := t.(*list); y && l.len() == 1 {
				if d, y := l.elems[0].(*delegate); y && d.x.String() == "if" {
					if len(d.a) == 2 && strings.Contains(d.a[1].String(), "$_") {
						erro(pc(ctx,t), "%s: %v %v", d.x, d.a[0], d.a[1])
					}
				}
			}
			errostack(pc(ctx,t), 3, "%s: %v %v", d.x, d.a[0], d.a[1]).trace()
		}
	}
}

func (ctx *__if) a_check(i int, a, v Value) {
	if p := _project(ctx); i == 1 && p.name == "llvm.Config" {
		if w := auto_get(ctx, "_"); w != nil && strings.Contains(v.String(), "$_") {
			errostack(pc(ctx,a), 5, "%v: %v %v; %s", w, a, v, _builtincalls(ctx)).trace()
		}
	}
}
