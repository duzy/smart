//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
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

// func (ctx *builtin_fullname) x_check(p *project, a Value, x fullname) {
// 	switch p.name {
// 	case "testllvmconfig":
// 		switch ss := a.String(); ss {
// 		case "llvm/Config/llvm-config.h":
// 			if d := p.resolveDef(ctx, "outinc"); d == nil {
// 				erro(ctx, "%v %v", a, x).trace()
// 			} else if s := d.string(ctx); s == "" {
// 				erro(ctx, "%v %v", a, d).trace()
// 			} else if t := x.Value.(*file) ; t.name != ss {
// 				erro(ctx, "%s != %s", t.name, ss).trace()
// 			} else if t.dir != s {
// 				erro(ctx, "%s != %s", t.dir, s).trace()
// 			}
// 		case "llvm/Config/llvm-config.h.cmake":
// 			if d := p.resolveDef(ctx, "srcinc"); d == nil {
// 				erro(ctx, "%v %v", a, x).trace()
// 			} else if s := d.string(ctx); s == "" {
// 				erro(ctx, "%v %v", a, d).trace()
// 			} else if t := x.Value.(*file) ; t.name != ss {
// 				erro(ctx, "%s != %s", t.name, ss).trace()
// 			} else if t.dir != s {
// 				erro(ctx, "%s != %s", t.dir, s).trace()
// 			}
// 		}
// 	}
// }
