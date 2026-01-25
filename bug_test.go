//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func testBug_01(ctx *testcase) {
	s := "okay"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else {
		var v Value
		func () {
			defer func () {
				if x, y := recover().(trace_evoke_loop_err); y {
					debug(ctx, "%s", __string(ctx, x.Value), trace{})
				}
			} ()
			v = expand(_final(ctx), d.value)
		} ()
		if v == nil {
			ctx.err("%v", d)
		}

		func () {
			defer func () {
				if x, y := recover().(trace_evoke_loop_err); y {
					debug(ctx, "%s", __string(ctx, x.Value), trace{})
				}
			} ()
			v = ctx.val(d, defExpand1, "a", "b", "c", "d")
		} ()
		if v == nil {
			ctx.err("%v", d)
		} else if s, t := ts(v), "{=list {=compound {=word x} {=punct .} {=word a}} {=compound {=word x} {=punct .} {=word b}} {=compound {=word y} {=punct .} {=word a}} {=compound {=word y} {=punct .} {=word b}} {=compound {=word z} {=punct .} {=word a}} {=compound {=word z} {=punct .} {=word b}}}"; s != t {
			note(ctx, "%s", s)
			note(ctx, "%s", t)
			ctx.err("%s", v)
		}
	}

	s = "bug_0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(bug_0.1 $1,$2)"; s != t {
		ctx.err("%v : %s != %s", d, s, t)
	} else {
		var v Value
		func () {
			defer func () {
				if x, y := recover().(trace_evoke_loop_err); y {
					debug(ctx, "%s", __string(ctx, x.Value), trace{})
				}
			} ()
			v = expand(trace_evoke_loop{_final(ctx)},d.value)
		} ()
		if v == nil {
			ctx.err("%v", d)
		}

		func () {
			defer func () {
				if x, y := recover().(trace_evoke_loop_err); y {
					debug(ctx, "%s", __string(ctx, x.Value), trace{})
				}
			} ()
			v = ctx.val(d, defExpand1, "a", "b", "c", "d")
		} ()
		if v == nil {
			ctx.err("%v", d)
		} else if s, t := ts(v), "{=list {=closure {=compound {=word x} {=punct .} {=word a}}} {=closure {=compound {=word x} {=punct .} {=word b}}} {=closure {=compound {=word y} {=punct .} {=word a}}} {=closure {=compound {=word y} {=punct .} {=word b}}} {=closure {=compound {=word z} {=punct .} {=word a}}} {=closure {=compound {=word z} {=punct .} {=word b}}}}"; s != t {
			note(ctx, "%s", s)
			note(ctx, "%s", t)
			ctx.err("%s", v)
		}
	}

	s = "bug_1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(bug_1.1 $1,$2)"; s != t {
		ctx.err("%v : %s != %s", d, s, t)
	} else {
		var e, v Value
		func () {
			defer func () {
				if x, y := recover().(trace_evoke_loop_err); y { e = x.Value }
			} ()
			v = expand(trace_evoke_loop{_final(ctx)},d.value)
		} ()
		if s, t := ts(e), "{=def bug_1.1}"; s != t {
			ctx.err("expecting evocation loop: %s != %s", s, t)
		}
		if v != nil {
			ctx.err("%v → %v", d, v)
		}
	}

	s = "flags"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s := ts(v); s != "{=delegate {=def .flags} {=list {=delegate {=auto 1}}} {=list {=delegate {=auto 2}}}}" {
		ctx.err("%v : %v", d, s)
	} else {
		var e, v Value
		func () {
			defer func () {
				if x, y := recover().(trace_evoke_loop_err); y { e = x.Value }
			} ()
			v = expand(trace_evoke_loop{_final(ctx)},d.value)
		} ()
		if s, t := ts(e), "{=def .flags}"; s != t {
			ctx.err("expecting evocation loop: %s != %s", s, t)
		}
		if v != nil {
			ctx.err("%v → %v", d, v)
		}

		func () {
			defer func () {
				if x, y := recover().(trace_evoke_loop_err); y { e = x.Value }
			} ()
			a, _ := va(ctx, []string{"a", "b", "c", "d"}).(*list)
			v = evoke(trace_evoke_loop{_final(ctx)}, d, nil, a.elems)
		} ()
		if s, t := ts(e), "{=def .flags}"; s != t {
			ctx.err("expecting evocation loop: %s != %s", s, t)
		}
		if v != nil {
			ctx.err("%v → %v", d, v)
		}
	}
}
