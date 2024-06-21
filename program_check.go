//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	"fmt"
)

func (prog *program) execute_check_0(ctx Context, result *Value) {
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

func (prog *program) execute_check_1(ctx Context, result *Value) {
	var ent = _entry(ctx)
    if *result == nil {
        erro(ctx, "%v: nil result", ts(ent)).trace()
    }
}

func (prog *program) check_shell_for_stdout(ctx Context, result *Value) {
	var ent = _entry(ctx)

	if *result == nil {
		erro(ctx, "%v: nil result", ts(ent)).trace()
	} else if ts(*result) != "{=null}" {
		erro(ctx, "%v", ts(*result)).trace()
	}

    var args = try[[]Value](ctx, get_arguments{})

    switch ent.destiny().string(ctx) {
    case ".test":
        if len(args) != 2 {
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
		if v, t := auto_get(ctx, "-"), fmt.Sprintf("{=delegate {=builtin debug} {=list %s %s}}", ts(args[1]), ts(args[0])); ts(v) != t {
			erro(at(ctx,v), "%s != %s", ts(v), t).trace()
		}
	}
}
