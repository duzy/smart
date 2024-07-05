///
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func (p *parser) define_check(ctx Context, tok token, ident, value Value, d **def) {
	if *d == nil {
		erro(ctx, "%v %v %v", ident, tok, ts(value)).trace()
	} else if (*d).value == nil && value != nil {
		erro(ctx, "%v %v %v", ident, tok, ts(value)).trace()
	}
}

func (l unilo) parse_file_check(ctx Context, abs, rel, tmp string) {
	var p = l.project
	if p == nil {
		erro(ctx, "nil project").trace()
		return
	}

	switch p.name {
	case "general":
		if d := p.resolveDef(ctx, "workspace"); d == nil || d.value == nil {
			erro(ctx, "%v: %v %v", p, rel, d).trace()
		} else if s, t := d.value.string(ctx), dirs(3, abs); s != t {
			erro(ctx, "%v: %v %v", p, s, t).trace()
		}
		if d := p.resolveDef(ctx, "workout"); d == nil || d.value == nil {
			erro(ctx, "%v: %v %v", p, rel, d).trace()
		} else if s, t := d.value.string(ctx), dirs(4, abs)+"/workout"; s != t {
			erro(ctx, "%v: %v %v", p, s, t).trace()
		}
		if d := p.resolveDef(ctx, "workext"); d == nil || d.value == nil {
			erro(ctx, "%v: %v %v", p, rel, d).trace()
		} else if s, t := d.value.string(ctx), dirs(3, abs)+"/external"; s != t {
			erro(ctx, "%v: %v %v", p, s, t).trace()
		}
	case "variant.target.base":
		if len(p.bases) != 1 {
			erro(ctx, "%v: %v", p, p.bases).trace()
		}
		if p.bases[0].name != "general" {
			erro(ctx, "%v: %v", p, p.bases[0]).trace()
		}
		if p.resolveDef(ctx, "workout") != p.bases[0].resolveDef(ctx, "workout") {
			erro(ctx, "%v: workout", p).trace()
		}
		if d := p.resolveDef(ctx, "workout"); d == nil || d.value == nil {
			erro(ctx, "%v: %v %v", p, rel, d).trace()
		} else if s, t := d.value.string(ctx), dirs(6, abs)+"/workout"; s != t {
			erro(ctx, "%v: %v %v", p, s, t).trace()
		}
	case "variant.target":
		// TODO: ...
	}
}
