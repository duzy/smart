//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	"regexp"
	"strings"
)

func (ctx *builtin_trimprefix) x_check_match(val, prefix Value, f bool, r any, m []string) {
    var v = val.string(ctx)
    switch prefix.String() {
    case "%%/.smart/modules/":
        if s := val.string(ctx); strings.Contains(s, "/.smart/modules/") {
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

func (ctx *builtin_trimprefix) x_check(prefix, val, res Value) {
    var pre, str, t = prefix.string(ctx), val.string(ctx), res.string(ctx)

    if strings.HasSuffix(pre, "/") && strings.HasPrefix(str, pre) && strings.HasPrefix(t, "/") {
        erro(ctx, "{=%s %v} {=%s %v} {=%s %v}", typeof(prefix), prefix, typeof(val), val, typeof(res), res).trace()
    }

	var proj = _project(ctx)

    if p := "/.smart/modules/"; pre == "%%"+p {
        var s = val.string(ctx)
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
				var r1 = cp[0].def(ctx, "/").value
				var r2 =  proj.def(ctx, "/").value
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

func (ctx *builtin_auto) check_res(ar []Value) {
    var a = ctx.evocation.a[1]
    if a.String() == "$(a)" && auto_find(ctx, "a") == nil {
        if x, y := a.(*list); !y || x.len() != 1 {
            erro(ctx, "%v", ts(a)).trace()
        } else if z, y := x.elems[0].(*delegate); !y {
            erro(ctx, "%v", ts(x.elems[0])).trace()
        } else if x, y := z.x.(*auto); !y {
            erro(ctx, "%v", ts(z.x)).trace()
        } else if x.name != "a" {
            erro(ctx, "%v", ts(x)).trace()
        }
        if len(ar) == 1 {
            if x, y := ar[0].(*list); y && x.len() == 0 {
                erro(ctx, "%v → %v", ts(a), ts(x)).trace()
            }
        }
    }
}

func (ctx *builtin_grep) check_res(rx *regexp.Regexp, text string, temp, val Value) {
	if false {
		note(ctx, "%40v → %s", temp.expand(_final(ctx)), val.string(ctx)).debug(2)
	}
	if d, y := ctx.defs["0"]; !y {
		erro(ctx, "%v %v %v %v", rx, text, temp, val).trace()
	} else if t := d.string(ctx); t != text {
		erro(ctx, "%v: %v: %s != %s", rx, d, t, text).trace()
	}
	switch rx.String() {
	case `.+?\.o`:
	case `(.+?)((-(?P<i>.+?))*)(\.)(?P<x>o)`:
		switch text {
		case "foo.o":
			if d, y := ctx.defs["1"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "foo", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if d, y := ctx.defs["2"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if d, y := ctx.defs["3"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if _, y := ctx.defs["4"]; y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			}
			if d, y := ctx.defs["i"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if d, y := ctx.defs["5"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := ".", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if _, y := ctx.defs["6"]; y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			}
			if d, y := ctx.defs["x"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "o", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if v := auto_get(ctx, "0"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "foo.o", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "1"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "foo", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "2"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "3"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "4"); v != nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			}
			if v := auto_get(ctx, "i"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "5"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := ".", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "6"); v != nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			}
			if v := auto_get(ctx, "x"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "o", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
		case "foo-x.o":
			if d, y := ctx.defs["1"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "foo", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if d, y := ctx.defs["2"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "-x", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if d, y := ctx.defs["3"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "-x", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if _, y := ctx.defs["4"]; y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			}
			if d, y := ctx.defs["i"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "x", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if d, y := ctx.defs["5"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := ".", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if _, y := ctx.defs["6"]; y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			}
			if d, y := ctx.defs["x"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "o", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if v := auto_get(ctx, "0"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "foo-x.o", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "1"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "foo", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "2"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "-x", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "3"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "-x", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "4"); v != nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			}
			if v := auto_get(ctx, "i"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "x", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "5"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := ".", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "6"); v != nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			}
			if v := auto_get(ctx, "x"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "o", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
		case "foo-x-y.o":
			if d, y := ctx.defs["1"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "foo", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if d, y := ctx.defs["2"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "-x-y", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if d, y := ctx.defs["3"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "-y", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if _, y := ctx.defs["4"]; y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			}
			if d, y := ctx.defs["i"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "y", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if d, y := ctx.defs["5"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := ".", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if _, y := ctx.defs["6"]; y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			}
			if d, y := ctx.defs["x"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "o", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if v := auto_get(ctx, "0"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "foo-x-y.o", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "1"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "foo", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "2"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "-x-y", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "3"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "-y", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "4"); v != nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			}
			if v := auto_get(ctx, "i"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "y", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "5"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := ".", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "6"); v != nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			}
			if v := auto_get(ctx, "x"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "o", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
		case "foo-x-y-z.o":
			if d, y := ctx.defs["1"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "foo", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if d, y := ctx.defs["2"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "-x-y-z", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if d, y := ctx.defs["3"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "-z", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if _, y := ctx.defs["4"]; y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			}
			if d, y := ctx.defs["i"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "z", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if d, y := ctx.defs["5"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := ".", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if _, y := ctx.defs["6"]; y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			}
			if d, y := ctx.defs["x"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "o", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if v := auto_get(ctx, "0"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "foo-x-y-z.o", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "1"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "foo", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "2"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "-x-y-z", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "3"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "-z", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "4"); v != nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			}
			if v := auto_get(ctx, "i"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "z", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "5"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := ".", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "6"); v != nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			}
			if v := auto_get(ctx, "x"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "o", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
		case "foobar.o":
			if d, y := ctx.defs["1"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "foobar", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if d, y := ctx.defs["2"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if d, y := ctx.defs["3"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if _, y := ctx.defs["4"]; y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			}
			if d, y := ctx.defs["i"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if d, y := ctx.defs["5"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := ".", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if _, y := ctx.defs["6"]; y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			}
			if d, y := ctx.defs["x"]; !y {
				erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "o", d.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, d, t, s).trace()
			}
			if v := auto_get(ctx, "0"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "foobar.o", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "1"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "foobar", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "2"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "3"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "4"); v != nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			}
			if v := auto_get(ctx, "i"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "5"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := ".", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
			if v := auto_get(ctx, "6"); v != nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			}
			if v := auto_get(ctx, "x"); v == nil {
				erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
			} else if s, t := "o", v.string(ctx); s != t {
				erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
			}
		default:
			erro(ctx, "%v %v %v %v %v", rx, text, temp, val, ctx.defs).trace()
		}
	case `^ *set\(LLVM_VERSION_MAJOR +([0-9]+) *\)`:
		if v := auto_get(ctx, "1"); v == nil {
			erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
		} else if s, t := "20", v.string(ctx); s != t {
			erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
		}
	case `^ *set\(LLVM_VERSION_MINOR +([0-9]+) *\)`:
		if v := auto_get(ctx, "1"); v == nil {
			erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
		} else if s, t := "0", v.string(ctx); s != t {
			erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
		}
	case `^ *set\(LLVM_VERSION_PATCH +([0-9]+) *\)`:
		if v := auto_get(ctx, "1"); v == nil {
			erro(ctx, "%v: %v; %v; %v; %v", rx, text, temp, val, ctx.defs).trace()
		} else if s, t := "0", v.string(ctx); s != t {
			erro(ctx, "%v: %v: %s != %s", rx, v, t, s).trace()
		}
	default:
		switch p := _project(ctx); p.name {
		case "llvm.Config":
		case "testvalue":
			erro(ctx, "%s: %v; %v; %v; %v; %v", p.name, rx, text, temp, val, ctx.defs).trace()
		default:
			note(ctx, "%s: %v; %v; %v; %v; %v", p.name, rx, text, temp, val, ctx.defs).debug()
		}
	}
}

func (ctx *builtin_foreach) check_v(v Value) {
	if len(ctx.evocation.a) == 0 { return }

	var t = ctx.evocation.a[1]
	if d, y := v.(*delegate); y && d.x.String() == "if" {
		if len(d.a) == 2 && strings.Contains(d.a[1].String(), "$_") {
			if l, y := t.(*list); y && l.len() == 1 {
				if d, y := l.elems[0].(*delegate); y && d.x.String() == "if" {
					if len(d.a) == 2 && strings.Contains(d.a[1].String(), "$_") {
						// tv(d.a[1].expand(_final(ctx)))
						erro(pc(ctx,t), "%s: %v %v", d.x, d.a[0], d.a[1])
					}
				}
			}
			errostack(pc(ctx,t), 3, "%s: %v %v", d.x, d.a[0], d.a[1]).debug(2)
		}
	}
}

func (ctx *builtin_if) a_check(i int, a, v Value, skip bool) {
	if p := _project(ctx); i == 1 && p.name == "llvm.Config" {
		if w := auto_get(ctx, "_"); w != nil && strings.Contains(v.String(), "$_") {
			errostack(pc(ctx,a), 5, "%v: %v %v; %s", w, a, v, _builtincalls(ctx)).debug(32)
		}
	}
}
