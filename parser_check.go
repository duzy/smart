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

func (l unilo) files_check(ctx Context) {
	switch p := l.project; p.name {
	case "variant.target.base":
		if d := p.resolveDef(ctx, "variant.tag"); d == nil || d.value == nil {
			erro(ctx, "variant.tag is undefined").trace()
		}
		if d := p.resolveDef(ctx, "variant.name"); d == nil || d.value == nil {
			erro(ctx, "variant.name is undefined").trace()
		}
		if d := p.resolveDef(ctx, "prefix"); d == nil || d.value == nil {
			erro(ctx, "prefix is undefined").trace()
		}
		if d := p.resolveDef(ctx, "outtmp"); d == nil || d.value == nil {
			erro(ctx, "outtmp is undefined").trace()
		}
		if d := p.resolveDef(ctx, "outinc"); d == nil || d.value == nil {
			erro(ctx, "outinc is undefined").trace()
		}
		if d := p.resolveDef(ctx, "outobj"); d == nil || d.value == nil {
			erro(ctx, "outobj is undefined").trace()
		}
		if d := p.resolveDef(ctx, "outlib"); d == nil || d.value == nil {
			erro(ctx, "outlib is undefined").trace()
		}
		if d := p.resolveDef(ctx, "outbin"); d == nil || d.value == nil {
			erro(ctx, "outbin is undefined").trace()
		}
	case "app.base":
		if d := p.resolveDef(ctx, "variant.tag"); d == nil || d.value == nil {
			erro(ctx, "variant.tag is undefined").trace()
		}
		if d := p.resolveDef(ctx, "variant.name"); d == nil || d.value == nil {
			erro(ctx, "variant.name is undefined").trace()
		}
		if d := p.resolveDef(ctx, "outinc"); d == nil {
			erro(ctx, "outinc is undefined").trace()
		}
		if d := p.resolveDef(ctx, "outobj"); d == nil {
			erro(ctx, "outobj is undefined").trace()
		}
		if d := p.resolveDef(ctx, "outlib"); d == nil {
			erro(ctx, "outlib is undefined").trace()
		}
	case "lib.std":
		if d := p.resolveDef(ctx, "outinc"); d == nil {
			erro(ctx, "outinc is undefined").trace()
		}
		if d := p.resolveDef(ctx, "outobj"); d == nil {
			erro(ctx, "outobj is undefined").trace()
		}
		if d := p.resolveDef(ctx, "outlib"); d == nil {
			erro(ctx, "outlib is undefined").trace()
		}
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

func (l unilo) new_project_check_bases(ctx Context) {
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

func (l unilo) parse_file_check_new_project(ctx Context) {
	switch p := l.project; p.name {
	case "lib.std":
		if len(p.bases) != 1 {
			erro(ctx, "%v: wrong bases: %v", p, p.bases).trace()
		}
		if b := p.bases[0]; b.name != "app.base" {
			erro(ctx, "%v: wrong bases[0]", b).trace()
		}
		if d := p.resolveDef(ctx, "outinc"); d == nil || d.value == nil {
			erro(ctx, "%v: %v", p, d).trace()
		}
		if d := p.resolveDef(ctx, "outobj"); d == nil || d.value == nil {
			erro(ctx, "%v: %v", p, d).trace()
		}
		if d := p.resolveDef(ctx, "outlib"); d == nil || d.value == nil {
			erro(ctx, "%v: %v", p, d).trace()
		}
	}
}
