//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func testValues4(ctx *testcase) {
	s := ".test.*"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v", d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := "D.c(-unique) D.c++(-unique) I.c(-unique) I.c++(-unique)", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := v.string(src(ctx,d)); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 4 {
		ctx.err("%d, %v", l.len(), tst{l})
	} else if _, y := l.elems[0].(*argumented); !y || l.elems[0].String() != "D.c(-unique)" {
		ctx.err("%v", tst{l.elems[0]})
	} else if _, y := l.elems[1].(*argumented); !y || l.elems[1].String() != "D.c++(-unique)" {
		ctx.err("%v", tst{l.elems[1]})
	} else if _, y := l.elems[2].(*argumented); !y || l.elems[2].String() != "I.c(-unique)" {
		ctx.err("%v", tst{l.elems[2]})
	} else if _, y := l.elems[3].(*argumented); !y || l.elems[3].String() != "I.c++(-unique)" {
		ctx.err("%v", tst{l.elems[3]})
	}

	s = ".test.D.c.0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v", tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {8:31 {=compound {6:12:word c} {8:36:punct .} {8:39 {5:11:word D}}}} {17:16:delegate {17:18:builtin value} {=list {17:24:closure {=compound {17:26:punct .} {17:27:word test} {17:31:punct .} {17:32:word x}}}}} {19:16:delegate {19:18:builtin value} {=list {=compound {19:26:punct .} {19:27:word test} {19:31:punct .} {19:32:word v}}}} {25:16:closure {25:18:builtin value} {=list {25:24:delegate {23:9:def .test.x}}}} {39:16:delegate {37:15:def .test.foreach} {=list {39:32:delegate {37:19:auto 1}}} {=list {39:35:closure {=compound {39:37:punct .} {39:38:word test} {39:42:punct .} {39:43:word none}}}}} {=group {39:51:delegate {37:19:auto 1}}} {41:16:delegate {41:18:builtin foreach} {=list {41:26:delegate {37:19:auto 1}}} {=list {41:29:closure {=compound {41:31:punct .} {41:32:word test} {41:36:punct .} {41:37:word x} {41:38:punct .} {41:39:delegate {41:40:auto _}}}}}} {=group {41:45:delegate {37:19:auto 1}}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.String(), "c.D $(value &(.test.x)) $(value .test.v) &(value $(.test.x)) $(.test.foreach $1,&(.test.none)) ($1) $(foreach $1,&(.test.x.$_)) ($1)"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.string(src(ctx,d)), "c.D xx xx xx"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 8 {
		ctx.err("%d, %v", l.len(), tst{l})
	}

	s = ".test.D.c.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v", tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {9:31 {=compound {6:12:word c} {9:36:punct .} {9:39 {5:11:word D}}}} {18:16 {18:18:null}} {20:16 {20:18:null}} {26:16:closure {26:18:builtin value} {=list {26:24 {=compound {23:12:punct .} {23:13:word test} {23:17:punct .} {23:18:word v}}}}} {40:16 {=group {37:18 {40:32:null}}}} {=group {40:51:null}} {42:16 {42:18:null}} {=group {42:45:null}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.String(), "c.D {} {} &(value .test.v) ({}) ({}) {} ({})"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.string(src(ctx,d)), "c.D xx () () ()"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 8 {
		ctx.err("%d, %v: %v", l.len(), l, tst{l})
	}

	s = ".test.D.c++.0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v", d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "c++.D"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if t := v.string(src(ctx,d)); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.D.c++.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v", d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "c++.D"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if t := v.string(src(ctx,d)); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.I.c.0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v", tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {8:31 {=compound {6:12:word c} {8:36:punct .} {8:39 {5:13:word I}}}} {28:16:closure {28:18:builtin value} {=list {28:24:closure {23:9:def .test.x}}}} {30:16:closure {30:18:builtin value} {=list {30:24:delegate {23:9:def .test.x}}}} {32:16:delegate {32:18:builtin value} {=list {32:24:closure {23:9:def .test.x}}}} {34:16:delegate {34:18:builtin value} {=list {34:24:delegate {23:9:def .test.x}}}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.String(), "c.I &(value &(.test.x)) &(value $(.test.x)) $(value &(.test.x)) $(value $(.test.x))"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.string(src(ctx,d)), "c.I xx xx xx xx"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 5 {
		ctx.err("%d, %v", l.len(), tst{l})
	}

	s = ".test.I.c.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v", tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {9:31 {=compound {6:12:word c} {9:36:punct .} {9:39 {5:13:word I}}}} {29:16:closure {29:18:builtin value} {=list {29:24:closure {23:9:def .test.x}}}} {31:16:closure {31:18:builtin value} {=list {31:24 {=compound {23:12:punct .} {23:13:word test} {23:17:punct .} {23:18:word v}}}}} {33:16 {22:12:word xx}} {35:16 {22:12:word xx}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.String(), "c.I &(value &(.test.x)) &(value .test.v) xx xx"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.string(src(ctx,d)), "c.I xx xx xx xx"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 5 {
		ctx.err("%d, %v", l.len(), tst{l})
	}


	s = ".test.I.c++.0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v", tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{8:31 {=compound {6:14:word c++} {8:36:punct .} {8:39 {5:13:word I}}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.String(), "c++.I"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.string(src(ctx,d)), "c++.I"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.I.c++.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v", tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{9:31 {=compound {6:14:word c++} {9:36:punct .} {9:39 {5:13:word I}}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.String(), "c++.I"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.string(src(ctx,d)), "c++.I"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.and.x.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v", d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
		//
		//
	} else if s, t := "x1", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.and.x.2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v", d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
		//
		//
	} else if s, t := "x2", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.and.y.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v", d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
		//
		//
	} else if s, t := "y1", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.and.y.2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v", d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
		//
		//
	} else if s, t := "y2", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}
}
