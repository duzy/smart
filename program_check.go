//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func (prog *program) execute_check_0(ctx Context, ent entry, result *Value) {
    if *result == nil {
        erro(ctx, "%v: nil result", ts(ent)).debug()
        trace(ctx)
    }

    var args = try[[]Value](ctx, get_arguments{})

    switch ent.destiny().string(ctx) {
    case "rule0":
        if len(args) != 3 {
            erro(ctx, "%v: %d %v", ent, len(args), ts(args)).debug()
            trace(ctx)
        }
		if v := auto_get(ctx, "@"); ts(v) != "{=bareword rule0}" {
			erro(at(ctx,v), "%v", ts(v)).debug()
			trace(ctx)
		}
		if v := auto_get(ctx, "<"); ts(v) != "{=bareword rule1}" {
			erro(at(ctx,v), "%v", ts(v)).debug()
			trace(ctx)
		}
		if v := auto_get(ctx, ">"); ts(v) != "{=bareword rule1}" {
			erro(at(ctx,v), "%v", ts(v)).debug()
			trace(ctx)
		}
		if v := auto_get(ctx, "^"); ts(v) != "{=bareword rule1}" {
			erro(at(ctx,v), "%v", ts(v)).debug()
			trace(ctx)
		}
		if v := auto_get(ctx, "-"); ts(v) != "{=list {=bareword rule1} {=bareword rule1} {=flag {=null}} {=barecomp {=bareword x} {=bareword y} {=bareword z}}}" {
			erro(at(ctx,v), "%v", ts(v)).debug()
			trace(ctx)
		}
		if v := auto_get(ctx, "<-"); ts(v) != "{=bareword rule1}" {
			erro(at(ctx,v), "%v", ts(v)).debug()
			trace(ctx)
		}
		if auto_get(ctx, "<") != auto_get(ctx, ">") {
			erro(ctx, "%v %v", ts(auto_get(ctx, "<")), ts(auto_get(ctx, ">"))).debug()
			trace(ctx)
		}
        if x, y := (*result).(*list); !y {
            erro(ctx, "%v: %v", ent, ts(*result)).debug()
            trace(ctx)
        } else if x.len() != 4 {
            erro(ctx, "%v: %v", ent, ts(*result)).debug()
            trace(ctx)
        } else if s := ts(x.elems[0]); s != "{=bareword rule0}" {
            erro(ctx, "%v: %v", ent, s).debug()
            trace(ctx)
        } else if s := ts(x.elems[1]); s != "{=bareword rule1}" {
            erro(ctx, "%v: %v", ent, s).debug()
            trace(ctx)
        } else if s := ts(x.elems[2]); s != "{=flag {=null}}" {
            erro(ctx, "%v: %v", ent, s).debug()
            trace(ctx)
        } else if s := ts(x.elems[3]); s != "{=barecomp {=bareword x} {=bareword y} {=bareword z}}" {
            erro(ctx, "%v: %v", ent, s).debug()
            trace(ctx)
        }
        if (*result).String() != "-" {
            erro(ctx, "%v: %v, %v", ent, ts(*result), args).debug()
            trace(ctx)
        }
    case "rule1":
        if len(args) != 1 {
            erro(ctx, "%v: %d %v", ent, len(args), ts(args)).debug()
            trace(ctx)
        }
		if v := auto_get(ctx, "@"); ts(v) != "{=bareword rule1}" {
			erro(at(ctx,v), "%v", ts(v)).debug()
			trace(ctx)
		}
		if v := auto_get(ctx, "<"); ts(v) != "{}" {
			erro(at(ctx,v), "%v", ts(v)).debug()
			trace(ctx)
		}
		if v := auto_get(ctx, ">"); ts(v) != "{}" {
			erro(at(ctx,v), "%v", ts(v)).debug()
			trace(ctx)
		}
		if v := auto_get(ctx, "-"); ts(v) != "{=list {=bareword rule0} {=bareword xyz}}" {
			erro(at(ctx,v), "%v", ts(v)).debug()
			trace(ctx)
		}
        if x, y := (*result).(*list); !y {
            erro(ctx, "%v: %v", ent, ts(*result)).debug()
            trace(ctx)
        } else if x.len() != 2 {
            erro(ctx, "%v: %v", ent, ts(*result)).debug()
            trace(ctx)
        } else if s := ts(x.elems[0]); s != "{=bareword rule0}" {
            erro(ctx, "%v: %v", ent, s).debug()
            trace(ctx)
        } else if s := ts(x.elems[1]); s != "{=bareword xyz}" {
            erro(ctx, "%v: %v", ent, s).debug()
            trace(ctx)
        }
        if (*result).String() != "rule0 xyz" {
            erro(ctx, "%v: %v %v", ent, ts(*result), args).debug()
            trace(ctx)
        }
    }
}
