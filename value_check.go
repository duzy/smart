//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	"reflect"
	"strings"
	"strconv"
	"path/filepath"
	"time"
	"fmt"
)

func prefix_check(ctx Context, x, y Value, _res *Value) {
	switch j := _project(ctx); j.spec {
	case "testdata/builtins/addprefix":
		prefix___addprefix(ctx, x, y, *_res)
	case "testdata/builtins/addsuffix":
		prefix___addsuffix(ctx, x, y, *_res)
	}
}
func prefix___addprefix(ctx Context, x, y Value, res Value) {
	if x == nil || y == nil {
		erro(pc(ctx,x), "%v %v : %v %v → %v", x, y, ts(x), ts(y), ts(res)).trace()
	}
	sx := x.String()
	sy := y.String()
	xy := sx+" "+sy
	switch xy {
	case "-std= foo", "-std= bar":
		if s, t := ts(x), "{=pair {=flag {=word std}}={=none}}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", x, s, t).trace()
		}
		if s, t := ts(y), fmt.Sprintf("{=word %v}", y); s != t {
			erro(pc(ctx,y), "%s: %s != %s", y, s, t).trace()
		}
		if s, t := ts(res), fmt.Sprintf("{=pair {=flag {=word std}}=%s}", ts(y)); s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case "-foo= bar":
		if s, t := ts(x), "{=pair {=flag {=word foo}}={=none}}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", x, s, t).trace()
		}
		if s, t := ts(y), fmt.Sprintf("{=word %v}", y); s != t {
			erro(pc(ctx,y), "%s: %s != %s", y, s, t).trace()
		}
		if s, t := ts(res), fmt.Sprintf("{=pair {=flag {=word foo}}=%s}", ts(y)); s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case "-foo= {&(none)}":
		if s, t := ts(x), "{=pair {=flag {=word foo}}={=none}}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", x, s, t).trace()
		}
		if s, t := ts(y), "{=disjunction {=closure {=word none}}}"; s != t {
			erro(pc(ctx,y), "%s: %s != %s", y, s, t).trace()
		}
		if s, t := ts(res), fmt.Sprintf("{=pair {=flag {=word foo}}=%s}", ts(y)); s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case " foo", " bar", " ax", " ay", " az", " bx", " by", " bz":
		if s, t := ts(x), "{=none}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", y, s, t).trace()
		}
		if s, t := ts(y), fmt.Sprintf("{=word %v}", y); s != t {
			erro(pc(ctx,y), "%s: %s != %s", y, s, t).trace()
		}
		if s, t := ts(res), fmt.Sprintf("%s", ts(y)); s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case " {&(none)}":
		if s, t := ts(x), "{=none}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", x, s, t).trace()
		}
		if s, t := ts(y), "{=disjunction {=closure {=word none}}}"; s != t {
			erro(pc(ctx,y), "%s: %s != %s", y, s, t).trace()
		}
		if s, t := ts(res), fmt.Sprintf("%s", ts(y)); s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case " {&(.test.$1)}":
		if s, t := ts(x), "{=none}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", x, s, t).trace()
		}
		if s, t := ts(y), "{=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}"; s != t {
			erro(pc(ctx,y), "%s: %s != %s", y, s, t).trace()
		}
		if s, t := ts(res), ts(y); s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case " {&(.test.a)}":
		if s, t := ts(x), "{=none}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", x, s, t).trace()
		}
		if s, t := ts(y), "{=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}}"; s != t {
			erro(pc(ctx,y), "%s: %s != %s", y, s, t).trace()
		}
		if s, t := ts(res), ts(y); s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case " {&(.test.b)}":
		if s, t := ts(x), "{=none}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", x, s, t).trace()
		}
		if s, t := ts(y), "{=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word b}}}}"; s != t {
			erro(pc(ctx,y), "%s: %s != %s", y, s, t).trace()
		}
		if s, t := ts(res), ts(y); s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case "std= {&(.test.$1)}":
		if s, t := ts(x), "{=pair {=word std}={=none}}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", x, s, t).trace()
		}
		if s, t := ts(y), "{=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}"; s != t {
			erro(pc(ctx,y), "%s: %s != %s", y, s, t).trace()
		}
		if s, t := ts(res), fmt.Sprintf("{=pair {=word std}=%s}", ts(y)); s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case "std= {&(.test.a)}":
		if s, t := ts(x), "{=pair {=word std}={=none}}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", x, s, t).trace()
		}
		if s, t := ts(y), "{=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}}"; s != t {
			erro(pc(ctx,y), "%s: %s != %s", y, s, t).trace()
		}
		if s, t := ts(res), fmt.Sprintf("{=pair {=word std}=%s}", ts(y)); s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case "std= {&(.test.b)}":
		if s, t := ts(x), "{=pair {=word std}={=none}}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", x, s, t).trace()
		}
		if s, t := ts(y), "{=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word b}}}}"; s != t {
			erro(pc(ctx,y), "%s: %s != %s", y, s, t).trace()
		}
		if s, t := ts(res), fmt.Sprintf("{=pair {=word std}=%s}", ts(y)); s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case "std= ax", "std= ay", "std= az", "std= bx", "std= by", "std= bz":
		if s, t := ts(x), "{=pair {=word std}={=none}}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", x, s, t).trace()
		}
		if s, t := ts(y), fmt.Sprintf("{=word %s}", y); s != t {
			erro(pc(ctx,y), "%s: %s != %s", y, s, t).trace()
		}
		if s, t := ts(res), fmt.Sprintf("{=pair {=word std}=%s}", ts(y)); s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case "foo bar":
		if s, t := ts(x), fmt.Sprintf("{=word %v}", x); s != t {
			erro(pc(ctx,x), "%s: %s != %s", x, s, t).trace()
		}
		if s, t := ts(y), fmt.Sprintf("{=word %v}", y); s != t {
			erro(pc(ctx,y), "%s: %s != %s", y, s, t).trace()
		}
		if s, t := ts(res), fmt.Sprintf("{=compound %s %s}", ts(x), ts(y)); s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case "ax =ax", "ax =ay", "ax =az", "ax =bx", "ax =by", "ax =bz":
	case "ay =ax", "ay =ay", "ay =az", "ay =bx", "ay =by", "ay =bz":
	case "az =ax", "az =ay", "az =az", "az =bx", "az =by", "az =bz":
	case "bx =ax", "bx =ay", "bx =az", "bx =bx", "bx =by", "bx =bz":
	case "by =ax", "by =ay", "by =az", "by =bx", "by =by", "by =bz":
	case "bz =ax", "bz =ay", "bz =az", "bz =bx", "bz =by", "bz =bz":
	case "foo =ax", "foo =ay", "foo =az", "foo =bx", "foo =by", "foo =bz":
	case "foo =xxx", "bar =xxx", "ax =xxx", "ay =xxx", "az =xxx", "bx =xxx", "by =xxx", "bz =xxx":
		if s, t := ts(x), fmt.Sprintf("{=word %v}", x); s != t {
			erro(pc(ctx,x), "%v: %s != %s", x, s, t).trace()
		}
		if s, t := ts(y), "{=pair {=none}={=word xxx}}"; s != t {
			erro(pc(ctx,y), "%v: %s != %s", y, s, t).trace()
		}
		if s, t := ts(res), fmt.Sprintf("{=pair {=word %v}={=word xxx}}", x); s != t {
			erro(pc(ctx,x), "%v: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case "{&(.test.$1)} ={&(.test.$1)}":
		if s, t := ts(x), "{=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}"; s != t {
			erro(pc(ctx,x), "%v: %s != %s", x, s, t).trace()
		}
		if s, t := ts(y), "{=pair {=none}={=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}}"; s != t {
			erro(pc(ctx,y), "%v: %s != %s", y, s, t).trace()
		}
		if s, t := ts(res), "{=pair {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}={=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}}"; s != t {
			erro(pc(ctx,x), "%v: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case "{&(.test.a)} ={&(.test.a)}":
		if s, t := ts(res), "{=pair {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}}={=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case "{&(.test.a)} ={&(.test.b)}":
		if s, t := ts(res), "{=pair {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}}={=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word b}}}}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case "{&(.test.b)} ={&(.test.b)}":
		if s, t := ts(res), "{=pair {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word b}}}}={=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word b}}}}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case "{&(.test.b)} ={&(.test.a)}":
		if s, t := ts(res), "{=pair {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word b}}}}={=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case "{&(.test.$1)} =xxx":
		if s, t := ts(x), "{=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}"; s != t {
			erro(pc(ctx,x), "%v: %s != %s", x, s, t).trace()
		}
		if s, t := ts(y), "{=pair {=none}={=word xxx}}"; s != t {
			erro(pc(ctx,y), "%v: %s != %s", y, s, t).trace()
		}
		if s, t := ts(res), "{=pair {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}={=word xxx}}"; s != t {
			erro(pc(ctx,x), "%v: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case "{&(.test.a)} =xxx":
		if s, t := ts(x), "{=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}}"; s != t {
			erro(pc(ctx,x), "%v: %s != %s", x, s, t).trace()
		}
		if s, t := ts(y), "{=pair {=none}={=word xxx}}"; s != t {
			erro(pc(ctx,y), "%v: %s != %s", y, s, t).trace()
		}
		if s, t := ts(res), "{=pair {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}}={=word xxx}}"; s != t {
			erro(pc(ctx,x), "%v: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case "{&(.test.b)} =xxx":
		if s, t := ts(x), "{=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word b}}}}"; s != t {
			erro(pc(ctx,x), "%v: %s != %s", x, s, t).trace()
		}
		if s, t := ts(y), "{=pair {=none}={=word xxx}}"; s != t {
			erro(pc(ctx,y), "%v: %s != %s", y, s, t).trace()
		}
		if s, t := ts(res), "{=pair {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word b}}}}={=word xxx}}"; s != t {
			erro(pc(ctx,x), "%v: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case "foo {&(none)}":
		if s, t := ts(x), fmt.Sprintf("{=word %v}", x); s != t {
			erro(pc(ctx,x), "%s: %s != %s", x, s, t).trace()
		}
		if s, t := ts(y), "{=disjunction {=closure {=word none}}}"; s != t {
			erro(pc(ctx,y), "%s: %s != %s", y, s, t).trace()
		}
		if s, t := ts(res), fmt.Sprintf("{=compound %s {=disjunction {=closure {=word none}}}}", ts(x)); s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case "foo ={&(.test.$1)}":
		if s, t := ts(res), "{=pair {=word foo}={=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case "foo ={&(.test.a)}":
		if s, t := ts(res), "{=pair {=word foo}={=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case "foo ={&(.test.b)}":
		if s, t := ts(res), "{=pair {=word foo}={=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word b}}}}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case "fo -":
		if s, t := ts(res), "{=compound {=word fo} {=flag {=null}}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case ". test":
		if s, t := ts(res), "{=compound {=punct .} {=word test}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case ".test .":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case ".test. $1":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case ".test. a":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=word a}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case ".test. b":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=word b}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case ".test. c":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=word c}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case ".test. {}":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=null}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case ".test.a .", ".test.a. x", ".test.a.x .", ".test.a.x. 1", ".test.a.x.1 .", ".test.a.x.1. y", ".test.a.x.1.y .", ".test.a.x.1.y. 0", ".test.a.x.1.y.0 .", ".test.a.x.1.y.0. z":
	case ".test.b .", ".test.b. x", ".test.b.x .", ".test.b.x. 1", ".test.b.x.1 .", ".test.b.x.1. y", ".test.b.x.1.y .", ".test.b.x.1.y. 0", ".test.b.x.1.y.0 .", ".test.b.x.1.y.0. z":
	case                                           ".test.b.x. 2", ".test.b.x.2 .", ".test.b.x.2. y", ".test.b.x.2.y .", ".test.b.x.2.y. 0", ".test.b.x.2.y.0 .", ".test.b.x.2.y.0. z":
	case ".test.c .", ".test.c. x", ".test.c.x .", ".test.c.x. 1" , ".test.c.x.1 .", ".test.c.x.1. y", ".test.c.x.1.y .", ".test.c.x.1.y. 0", ".test.c.x.1.y.0 .", ".test.c.x.1.y.0. z":
	case                                           ".test.c.x. 2" , ".test.c.x.2 .", ".test.c.x.2. y", ".test.c.x.2.y .", ".test.c.x.2.y. 0", ".test.c.x.2.y.0 .", ".test.c.x.2.y.0. z":
	case                                           ".test.c.x. 3" , ".test.c.x.3 .", ".test.c.x.3. y", ".test.c.x.3.y .", ".test.c.x.3.y. 0", ".test.c.x.3.y.0 .", ".test.c.x.3.y.0. z":
	case ".test.$1 .":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}} {=punct .}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case ".test.$1. x":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}} {=punct .} {=word x}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case ".test.$1.x .":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}} {=punct .} {=word x} {=punct .}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case ".test.$1.x. $2":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}} {=punct .} {=word x} {=punct .} {=delegate {=auto 2}}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case ".test.$1.x.$2 .":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}} {=punct .} {=word x} {=punct .} {=delegate {=auto 2}} {=punct .}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case ".test.$1.x.$2. y":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}} {=punct .} {=word x} {=punct .} {=delegate {=auto 2}} {=punct .} {=word y}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case ".test.$1.x.$2.y .":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}} {=punct .} {=word x} {=punct .} {=delegate {=auto 2}} {=punct .} {=word y} {=punct .}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case ".test.$1.x.$2.y. $3":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}} {=punct .} {=word x} {=punct .} {=delegate {=auto 2}} {=punct .} {=word y} {=punct .} {=delegate {=auto 3}}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case ".test.$1.x.$2.y.$3 .":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}} {=punct .} {=word x} {=punct .} {=delegate {=auto 2}} {=punct .} {=word y} {=punct .} {=delegate {=auto 3}} {=punct .}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case ".test.$1.x.$2.y.$3. z":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}} {=punct .} {=word x} {=punct .} {=delegate {=auto 2}} {=punct .} {=word y} {=punct .} {=delegate {=auto 3}} {=punct .} {=word z}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case "fo- {&(.test.$1.x.$2.y.$3.z)}":
		if s, t := ts(res), "{=compound {=word fo} {=flag {=null}} {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}} {=punct .} {=word x} {=punct .} {=delegate {=auto 2}} {=punct .} {=word y} {=punct .} {=delegate {=auto 3}} {=punct .} {=word z}}}}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case "fo- {&(.test.a.x.1.y.0.z)}":
		if truly(ctx, ex_closure{}) {
			if s, t := ts(res), "{=compound {=word fo} {=flag {=null}} {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a} {=punct .} {=word x} {=punct .} {=word 1} {=punct .} {=word y} {=punct .} {=word 0} {=punct .} {=word z}}}}}"; s != t {
				erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
			}
		} else {
			if s, t := ts(res), "{=compound {=word fo} {=flag {=null}} {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a} {=punct .} {=word x} {=punct .} {=word 1} {=punct .} {=word y} {=punct .} {=word 0} {=punct .} {=word z}}}}}"; s != t {
				erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
			}
		}
	case "fo- {&(.test.a.x.2.y.0.z)}":
		if s, t := ts(res), "{=compound {=word fo} {=flag {=null}} {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a} {=punct .} {=word x} {=punct .} {=word 2} {=punct .} {=word y} {=punct .} {=word 0} {=punct .} {=word z}}}}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case "fo- {&(.test.a.x.3.y.0.z)}":
		if s, t := ts(res), "{=compound {=word fo} {=flag {=null}} {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a} {=punct .} {=word x} {=punct .} {=word 3} {=punct .} {=word y} {=punct .} {=word 0} {=punct .} {=word z}}}}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case "fo- {&(.test.b.x.1.y.0.z)}", "fo- {&(.test.b.x.2.y.0.z)}", "fo- {&(.test.b.x.3.y.0.z)}":
	case "fo- {&(.test.c.x.1.y.0.z)}", "fo- {&(.test.c.x.2.y.0.z)}", "fo- {&(.test.c.x.3.y.0.z)}":
	case "fo- ax", "fo- ay", "fo- az", "fo- bx", "fo- by", "fo- bz", "fo- cx", "fo- cy", "fo- cz", "fo- dx", "fo- dy", "fo- dz", "fo- ex", "fo- ey", "fo- ez", "fo- fx", "fo- fy", "fo- fz":
	default:
		erro(pc(ctx,x), "%s: %s %s: %v %v → %v", xy, typeof(x), typeof(y), ts(x), ts(y), ts(res)).trace()
	}
}
func prefix___addsuffix(ctx Context, x, y Value, res Value) {
	if x == nil || y == nil {
		erro(pc(ctx,x), "%v %v : %v %v → %v", x, y, ts(x), ts(y), ts(res)).trace()
	}
	sx := x.String()
	sy := y.String()
	xy := sx+" "+sy
	switch xy {
	case ". test":
		if s, t := ts(res), "{=compound {=punct .} {=word test}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case ".test .":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	case ".test. a":
	case ".test. b":
	case ".test. c":
	case ".test. $1":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}"; s != t {
			erro(pc(ctx,x), "%s: %s %s: %s != %s", res, typeof(x), typeof(y), s, t).trace()
		}
	default:
		erro(pc(ctx,x), "%s : %v %v → %v", xy, ts(x), ts(y), ts(res)).trace()
	}
}

func suffix_check(ctx Context, x, y Value, _res *Value) {
	switch j := _project(ctx); j.spec {
	case "testdata/builtins/addprefix":
		suffix___addprefix(ctx, x, y, *_res)
	case "testdata/builtins/addsuffix":
		suffix___addsuffix(ctx, x, y, *_res)
	}
}
func suffix___addprefix(ctx Context, x, y Value, res Value) {
	if x == nil || y == nil {
		erro(pc(ctx,x), "%v %v : %v %v → %v", x, y, ts(x), ts(y), ts(res)).trace()
	}
	sx := x.String()
	sy := y.String()
	xy := sx+" "+sy
	switch xy {
	case " foo", " bar", " ax", " ay", " az", " bx", " by", " bz":
		if s, t := ts(res), fmt.Sprintf("{=word %s}", y); s != t {
			erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
		}
	case " {&(.test.$1)}":
		if s, t := ts(res), "{=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
		}
	case " {&(.test.a)}":
		if s, t := ts(res), "{=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
		}
	case " {&(.test.b)}":
		if s, t := ts(res), "{=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word b}}}}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
		}
	default:
		erro(pc(ctx,x), "%s : %v %v → %v", xy, ts(x), ts(y), ts(res)).trace()
	}
}
func suffix___addsuffix(ctx Context, x, y Value, res Value) {
	if x == nil || y == nil {
		erro(pc(ctx,x), "%v %v : %v %v → %v", x, y, ts(x), ts(y), ts(res)).trace()
	}
	sx := x.String()
	sy := y.String()
	xy := sx+" "+sy
	switch xy {
	case " foo", " bar":
		if s, t := ts(res), fmt.Sprintf("{=word %s}", sy); s != t {
			erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
		}
	case "=xxx foo", "=xxx bar":
		if s, t := ts(res), fmt.Sprintf("{=pair {=word %s}={=word xxx}}", sy); s != t {
			erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
		}
	case " {&(.test.$1)}":
		if s, t := ts(res), "{=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
		}
	case "=xxx {&(.test.$1)}":
		if s, t := ts(res), "{=pair {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}={=word xxx}}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
		}
	default:
		erro(pc(ctx,x), "%s : %v %v → %v", xy, ts(x), ts(y), ts(res)).trace()
	}
}
func compose_check(ctx Context, x, y Value, _res *Value) {
	switch j := _project(ctx); j.spec {
	case "testdata/builtins/addprefix":
		compose___addprefix(ctx, x, y, *_res)
	case "testdata/builtins/addsuffix":
		compose___addsuffix(ctx, x, y, *_res)
	}
}
func compose___addprefix(ctx Context, x, y Value, res Value) {
	if x == nil || y == nil {
		erro(pc(ctx,x), "%v %v : %v %v → %v", x, y, ts(x), ts(y), ts(res)).trace()
	}
	sx := x.String()
	sy := y.String()
	xy := sx+" "+sy
	switch xy {
	case "fo -":
		if s, t := ts(res), "{=compound {=word fo} {=flag {=null}}}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
		}
	case ". test":
		if s, t := ts(res), "{=compound {=punct .} {=word test}}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
		}
	case ".test .":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .}}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
		}
	case ".test. a", ".test. b",  ".test. c":
		if s, t := ts(res), fmt.Sprintf("{=compound {=punct .} {=word test} {=punct .} %s}", ts(y)); s != t {
			erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
		}
	case ".test. $1":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
		}
	case ".test.$1 .":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}} {=punct .}}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
		}
	case ".test.$1. x":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}} {=punct .} {=word x}}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
		}
	case ".test.$1.x .":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}} {=punct .} {=word x} {=punct .}}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
		}
	case ".test.$1.x. $2":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}} {=punct .} {=word x} {=punct .} {=delegate {=auto 2}}}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
		}
	case ".test.$1.x.$2 .":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}} {=punct .} {=word x} {=punct .} {=delegate {=auto 2}} {=punct .}}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
		}
	case ".test.$1.x.$2. y":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}} {=punct .} {=word x} {=punct .} {=delegate {=auto 2}} {=punct .} {=word y}}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
		}
	case ".test.$1.x.$2.y .":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}} {=punct .} {=word x} {=punct .} {=delegate {=auto 2}} {=punct .} {=word y} {=punct .}}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
		}
	case ".test.$1.x.$2.y. $3":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}} {=punct .} {=word x} {=punct .} {=delegate {=auto 2}} {=punct .} {=word y} {=punct .} {=delegate {=auto 3}}}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
		}
	case ".test.$1.x.$2.y.$3 .":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}} {=punct .} {=word x} {=punct .} {=delegate {=auto 2}} {=punct .} {=word y} {=punct .} {=delegate {=auto 3}} {=punct .}}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
		}
	case ".test.$1.x.$2.y.$3. z":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}} {=punct .} {=word x} {=punct .} {=delegate {=auto 2}} {=punct .} {=word y} {=punct .} {=delegate {=auto 3}} {=punct .} {=word z}}"; s != t {
			erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
		}
	case "-std= foo", "-std= bar":
		if s, t := ts(res), fmt.Sprintf("{=pair {=flag {=word std}}=%s}", ts(y)); s != t {
			erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
		}
	case "-foo= bar":
		if s, t := ts(res), fmt.Sprintf("{=pair {=flag {=word foo}}=%s}", ts(y)); s != t {
			erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
		}
	case "-foo= {&(none)}":
		if truly(ctx, ex_closure{}) {
			if s, t := ts(res), "{}"; s != t {
				erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
			}
		} else {
			if s, t := ts(res), "{=pair {=flag {=word foo}}={=disjunction {=closure {=word none}}}}"; s != t {
				erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
			}
		}
	case "foo bar":
		if s, t := ts(res), fmt.Sprintf("{=compound {=word foo} %s}", ts(y)); s != t {
			erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
		}
	case "foo =xxx", "bar =xxx", "ax =xxx", "ay =xxx", "az =xxx", "bx =xxx", "by =xxx", "bz =xxx":
		if s, t := ts(res), fmt.Sprintf("{=pair %s={=word xxx}}", ts(x)); s != t {
			erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
		}
	case "{&(.test.$1)} =xxx":
		if v := auto_get(ctx, "1"); v == nil {
			if truly(ctx, ex_closure{}) {
				if s, t := ts(res), "{=pair {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}={=word xxx}}"; s != t {
					erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), "{=pair {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}={=word xxx}}"; s != t {
					erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
				}
			}
		} else if truly(ctx, ex_closure{}) {
			if s, t := ts(res), fmt.Sprintf("{=pair {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} %s}}}={=word xxx}}", ts(v)); s != t {
				erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
			}
		} else {
			if s, t := ts(res), fmt.Sprintf("{=pair {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} %s}}}={=word xxx}}", ts(v)); s != t {
				erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
			}
		}
	case "{&(.test.a)} =xxx":
		if truly(ctx, ex_closure{}) {
			if s, t := ts(res), "{=list {=pair {=word ax}={=word xxx}} {=pair {=word ay}={=word xxx}} {=pair {=word az}={=word xxx}}}"; s != t {
				erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
			}
		} else {
			if s, t := ts(res), "{=pair {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}}={=word xxx}}"; s != t {
				erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
			}
		}
	case "{&(.test.b)} =xxx":
		if truly(ctx, ex_closure{}) {
			if s, t := ts(res), "{=list {=pair {=word bx}={=word xxx}} {=pair {=word by}={=word xxx}} {=pair {=word bz}={=word xxx}}}"; s != t {
				erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
			}
		} else {
			if s, t := ts(res), "{=pair {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word b}}}}={=word xxx}}"; s != t {
				erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
			}
		}
	case "foo {&(none)}":
		if truly(ctx, ex_closure{}) {
			if s, t := ts(res), "{=compound {=word foo} {=disjunction {=closure {=word none}}}}"; s != t {
				erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
			}
		} else if truly(ctx, ex_delegate{}) {
			if s, t := ts(res), "{=compound {=word foo} {=disjunction {=closure {=word none}}}}"; s != t {
				erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
			}
		} else {
			if s, t := ts(res), "{=compound {=word foo} {=disjunction {=closure {=word none}}}}"; s != t {
				erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
			}
		}
	case "foo {=&(.test.$1)}":
		if v := auto_get(ctx, "1"); v == nil {
			if s, t := ts(res), "{=compound {=word foo} {=disjunction {=pair {=none}={=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}}}"; s != t {
				erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
			}
		} else if truly(ctx, ex_closure{}) {
			if s, t := ts(res), "{=compound {=word foo} {=disjunction {=closure {=word none}}}}"; s != t {
				erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
			}
		} else if truly(ctx, ex_delegate{}) {
			if s, t := ts(res), "{=compound {=word foo} {=disjunction {=closure {=word none}}}}"; s != t {
				erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
			}
		} else {
			if s, t := ts(res), "{=compound {=word foo} {=disjunction {=closure {=word none}}}}"; s != t {
				erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
			}
		}
	case "std= {&(.test.$1)}", "std= {&(.test.a)}", "std= {&(.test.b)}", "std= {&(.test.c)}", "std= ax", "std= ay", "std= az", "std= bx", "std= by", "std= bz", "std= cx", "std= cy", "std= cz":
		if s, t := ts(res), fmt.Sprintf("{=pair {=word std}=%s}", ts(y)); s != t {
			erro(pc(ctx,x), "%s: %s != %s", res, s, t).trace()
		}
	case ".test.a .", ".test.a. x", ".test.a.x .", ".test.a.x. 1", ".test.a.x.1 .", ".test.a.x.1. y", ".test.a.x.1.y .", ".test.a.x.1.y. 0", ".test.a.x.1.y.0 .", ".test.a.x.1.y.0. z":
	case ".test.b .", ".test.b. x", ".test.b.x .", ".test.b.x. 1", ".test.b.x.1 .", ".test.b.x.1. y", ".test.b.x.1.y .", ".test.b.x.1.y. 0", ".test.b.x.1.y.0 .", ".test.b.x.1.y.0. z":
	case                                           ".test.b.x. 2", ".test.b.x.2 .", ".test.b.x.2. y", ".test.b.x.2.y .", ".test.b.x.2.y. 0", ".test.b.x.2.y.0 .", ".test.b.x.2.y.0. z":
	case ".test.c .", ".test.c. x", ".test.c.x .", ".test.c.x. 1", ".test.c.x.1 .", ".test.c.x.1. y", ".test.c.x.1.y .", ".test.c.x.1.y. 0", ".test.c.x.1.y.0 .", ".test.c.x.1.y.0. z":
	case                                           ".test.c.x. 2", ".test.c.x.2 .", ".test.c.x.2. y", ".test.c.x.2.y .", ".test.c.x.2.y. 0", ".test.c.x.2.y.0 .", ".test.c.x.2.y.0. z":
	case                                           ".test.c.x. 3", ".test.c.x.3 .", ".test.c.x.3. y", ".test.c.x.3.y .", ".test.c.x.3.y. 0", ".test.c.x.3.y.0 .", ".test.c.x.3.y.0. z":
	default:
		erro(pc(ctx,x), "%s : %v %v → %v", xy, ts(x), ts(y), ts(res)).trace()
	}
}
func compose___addsuffix(ctx Context, x, y Value, res Value) {
	if x == nil || y == nil {
		erro(pc(ctx,x), "%v %v : %v %v → %v", x, y, ts(x), ts(y), ts(res)).trace()
	}
	sx := x.String()
	sy := y.String()
	xy := sx+" "+sy
	switch xy {
	default:
		erro(pc(ctx,x), "%s : %v %v → %v", xy, ts(x), ts(y), ts(res)).trace()
	}
}

func ex_check(ctx Context, p, _x Value, _a, _o []Value, _l token, _cl bool, res, t, x *Value, a, o *[]Value) {
	if false && truly(ctx, ex_def_1{}) {
		if s := p.String(); "$(name?)" == s {
			note(pc(ctx,_x), "p=%v, x=%v→%v", ts(p), ts(_x), ts(*x))
			notestack(pc(ctx,_x), 1, "r=%v", ts(*res)).debug(16)
		}
	}

	switch _x.(type) {
	case *builtin, *def, *project, self:
		if s, t := ts(_x), ts(*x); s != t {
			errostack(pc(ctx,p), 3, "%v → %v → %v", s, t, *res).trace()
		}
	}

	if *res == nil {
		if a, y := _x.(*auto); y {
			if d := auto_find(ctx, a.name); d != nil {
				errostack(pc(ctx,p), 10, "%v : %v → %v", ts(p), ts(_x), ts(*x)).trace()
			}
		}
		if !_cl && (x == nil || *x == nil) {
			errostack(pc(ctx,p), 3, "%v : %v → %v", ts(p), ts(_x), ts(*x)).trace()
		}
	}

	if j := _project(ctx); j != nil {
		switch j.name {
		case "configure.base":
			ex_check_configure_base(ctx, p, _x, _a, _o, _l, _cl, res, t, x, a, o)
		}
		switch j.spec {
		case "testdata/value":
			ex_check_value(ctx, p, _x, *x, *res, *o, *a)
		case "testdata/value/2":
			ex_check_value_2(ctx, p, _x, *x, *o, *a, *res)
		case "testdata/value/4":
			ex_check_value_4(ctx, p, _x, _o, _a, res, x, o, a)
		case "testdata/value/closure":
			ex_check_value_closure(ctx, p, _x, _o, _a, res, x, o, a)
		case "testdata/value/optional":
			ex_check_value_optional(ctx, p, _x, _o, _a, res, x, o, a)
		case "testdata/value/bug_01":
			ex_check_value_bug_01(ctx, p, _x, _o, _a, *res, *x, *o, *a)
		case "testdata/builtins/foreach":
			ex_check_builtins_foreach(ctx, p, _x, _o, _a, *res, *x, *o, *a)
		case "testdata/rule/shell/for-stdout":
			ex_check_rule_shell_forstdout(ctx, p, _x, _o, _a, res, x, a)
		}
	}
}

func ex_check_configure_base(ctx Context, p, _x Value, _a, _o []Value, _l token, _cl bool, res, t, x *Value, a, o *[]Value) {
	if at := ts(auto_get(ctx, "@")); strings.HasPrefix(at, "{=file .configure/library/HAVE_LIB") {
		switch s := p.String(); s {
		case `$(foreach $(INCLUDE),"#include $_\n")`:
			if v := (*res).String(); strings.HasPrefix(v, `$(foreach {},"#include`) {
				note(ctx, "%v → %v ; %v %v %v", p, *res, _cl, truly(ctx, ex_closure{}), truly(ctx, ex_delegate{}))
				erro(pc(ctx,p), "%s : %s → %s ; %v", at, s, v, *a).trace()
			}
		}
	}
	if ent := _entry(ctx); ent != nil {
		switch ent.destiny().string(ctx) {
		case "-compiles-c", "-library-c", "-symbol-c":
			if truly(ctx, is_exec{}) {
				switch p.String() {
				case "$(file $(name).c)", "$(file $(name).c++)", "$(file $(name).log)":
					if _, y := (*res).(*file); !y {
						errostack(ctx, 8, "not a file: %v: %v → %v", p, ts(p), ts(*res)).trace()
					}
				case "$<", "$>", "$(file $(s).x)", "$(file $(s).o)":
					if _, y := (*res).(fullfile); !y {
						errostack(ctx, 8, "not a fullfile: %v: %v → %v", p, ts(p), ts(*res)).trace()
					}
				}
			}
			if truly(ctx, is_modify{}) {
				ex_check_configure_base_library_c(ctx, p, _x, _a, _o, _l, res, a, o)
			}
		}
	}
}

func ex_check_configure_base_library_c(ctx Context, p, _x Value, _a, _o []Value, _l token, res *Value, a, o *[]Value) {
	var kind string
	var t = auto_find(ctx, "TARGET")
	var d = auto_find(ctx, "FUNCTION")
	if d != nil && !isTrivial(d.value) {
		kind = "function"
	} else {
		kind = "library"
	}
	switch p.String() {
	case "$(ifdef FUNCTION,function,library)":
		if (*res).String() != kind {
			erro(ctx, "%v", *res).trace()
		}
	case "$(file .configure/$(ifdef FUNCTION,function,library)/$(TARGET).c)":
		if t == nil || isTrivial(t.value) {
			erro(ctx, "%v", *res).trace()
		} else if s := t.string(ctx); s == "" {
			erro(ctx, "%v", t.value).trace()
		} else if x, y := (*res).(*file); !y {
			erro(ctx, "%v", *res).trace()
		} else if t := filepath.Join(".configure", kind, s+".c"); t != x.name {
			erro(ctx, "%s != %s", x.name, t).trace()
		}
	case "$(file $(s).x)":
		if t == nil || isTrivial(t.value) {
			erro(ctx, "%v", *res).trace()
		} else if s := t.string(ctx); s == "" {
			erro(ctx, "%v %v", t.value, s).trace()
		} else {
			if len(*a) != 1 {
				erro(ctx, "%v", *a).trace()
			} else if l, y := (*a)[0].(*list); !y {
				erro(ctx, "%v %v %v", typeof(*res), *res, ts((*a)[0])).trace()
			} else if x, y := l.elems[0].(*compound); !y {
				erro(ctx, "%v %v %v", typeof(*res), *res, ts(l.elems[0])).trace()
			} else if f, y := x.elems[0].(*file); !y {
				erro(ctx, "%v %v %v", typeof(*res), *res, ts(x.elems[0])).trace()
			} else if t := filepath.Join(".configure", kind, s+".c"); t != f.name {
				erro(ctx, "%s != %s", f.name, t).trace()
			}
			if x, y := (*res).(*file); !y {
				erro(ctx, "%v %v", typeof(*res), *res).trace()
			} else if t := filepath.Join(".configure", kind, s+".c.x"); t != x.name {
				erro(ctx, "%s != %s", x.name, t).trace()
			}
		}
	case "$(file $(s).o)":
		if t == nil || isTrivial(t.value) {
			erro(ctx, "%v", *res).trace()
		} else if s := t.string(ctx); s == "" {
			erro(ctx, "%v %v", t.value, s).trace()
		} else {
			if len(*a) != 1 {
				erro(ctx, "%v", *a).trace()
			} else if l, y := (*a)[0].(*list); !y {
				erro(ctx, "%v %v %v", typeof(*res), *res, ts((*a)[0])).trace()
			} else if x, y := l.elems[0].(*compound); !y {
				erro(ctx, "%v %v %v", typeof(*res), *res, ts(l.elems[0])).trace()
			} else if f, y := x.elems[0].(*file); !y {
				erro(ctx, "%v %v %v", typeof(*res), *res, ts(x.elems[0])).trace()
			} else if t := filepath.Join(".configure", kind, s+".c"); t != f.name {
				erro(ctx, "%s != %s", f.name, t).trace()
			}
			if x, y := (*res).(*file); !y {
				erro(ctx, "%v %v", typeof(*res), *res).trace()
			} else if t := filepath.Join(".configure", kind, s+".c.o"); t != x.name {
				erro(ctx, "%s != %s", x.name, t).trace()
			}
		}
	}
}

func ex_check_value(ctx Context, p, _x, x, res Value, o, a []Value) {
	switch p.String() {
	case `$(quote a\,b\,c,x\,y\,z)`:
		if s, t := res.String(), `{=quote a\,b\,c x\,y\,z}`; s != t {
			erro(ctx, "%s != %s : %s", s, t, ts(res))
			note(ctx, "%v", ts(p))
			note(ctx, "_x=%v", ts(_x))
			note(ctx, " x=%v", ts(x)).trace()
		}
	}
}

func ex_check_value_optional(ctx Context, p, _x Value, _o, _a []Value, _res, x *Value, o, a *[]Value) {
	var res = *_res

	if truly(ctx, ex_def_1{}) {
		switch p.String() {
		case "$(name?)":
			if "{=null}" != ts(res) {
				errostack(pc(ctx,_x), 1, "p=%v, x=%v→%v, r=%v %s", ts(p), ts(_x), ts(*x), ts(res), res).trace()
			}
		}
	}

	switch ps := p.String(); ps {
	case "$(foo)":
		if s, t := ts(res), "{=project foo}"; s != t {
			erro(pc(ctx,res), "%s: %s != %s; %v → %v", ps, s, t, _x, *x).trace()
		}
	case "$(name?)":
		if s, t := ts(res), "{=null}"; s != t {
			erro(pc(ctx,res), "%s: %s != %s; %v → %v", ps, s, t, _x, *x).trace()
		}
	case "$({=project foo}→name?)":
		if s, t := ts(res), "{=self foo}"; s != t {
			erro(pc(ctx,res), "%s: %s != %s; %v → %v", ps, s, t, _x, *x).trace()
		}
	case "$({=project foo}→baz?)":
		if res != nil {
			if s, t := ts(res), "{=null}"; s != t {
				erro(pc(ctx,res), "%s: %s != %s; %v → %v", ps, s, t, _x, *x).trace()
			}
		}
	case "$(fo?→bar)":
		if res != nil {
			if s, t := ts(res), "{=null}"; s != t {
				erro(pc(ctx,res), "%s: %s != %s; %v → %v", ps, s, t, _x, *x).trace()
			}
		}
	case "$(fo?→bar?)":
		if res != nil {
			if s, t := ts(res), "{=null}"; s != t {
				erro(pc(ctx,res), "%s: %s != %s; %v → %v", ps, s, t, _x, *x).trace()
			}
		}
	case "$({=project foo}→name→xxxx?)":
		if res != nil {
			if s, t := ts(res), "{=null}"; s != t {
				erro(pc(ctx,res), "%s: %s != %s; %v → %v", ps, s, t, _x, *x).trace()
			}
		}
	case "$({=project foo}→name→item?)":
		if s, t := ts(res), "{=answer yes}"; s != t {
			erro(pc(ctx,res), "%s: %s != %s; %v → %v", ps, s, t, _x, *x).trace()
		}
	case "$({=project foo}→name?→item?)":
		if s, t := ts(res), "{=answer yes}"; s != t {
			erro(pc(ctx,res), "%s: %s != %s; %v → %v", ps, s, t, _x, *x).trace()
		}
	case "$({=project foo}?→name?→item?)":
		if s, t := ts(res), "{=answer yes}"; s != t {
			erro(pc(ctx,res), "%s: %s != %s; %v → %v", ps, s, t, _x, *x).trace()
		}
	case "$($_→name)":
		if false {
			if s, v := ts(res), auto_get(ctx, "_"); v == nil {
				if t := "{=delegate {=arrow {=delegate {=auto _}}→{=word name}}}"; s != t {
					erro(pc(ctx,res), "%s: %s != %s; %v → %v", ps, s, t, _x, *x).trace()
				}
			} else if t := "{=self foo}"; s != t {
				erro(pc(ctx,res), "%s: %s != %s; %v → %v", ps, s, t, _x, *x).trace()
			}
		}
	case "$($_→name?)":
		if false {
			if s, v := ts(res), auto_get(ctx, "_"); v == nil {
				if t := "{=delegate {=cond {=arrow {=delegate {=auto _}}→{=word name}}}}"; s != t {
					erro(pc(ctx,res), "%s: %s != %s; %v → %v", ps, s, t, _x, *x).trace()
				}
			} else if t := "{=self foo}"; s != t {
				erro(pc(ctx,res), "%s: %s != %s; %v → %v", ps, s, t, _x, *x).trace()
			}
		}
	}

	switch s := _x.String(); s {
	case "$_→name":
		if s, t := ts(_x), "{=arrow {=delegate {=auto _}}→{=word name}}"; s != t {
			erro(pc(ctx,_x), "%s: %v != %s", p, s, t).trace()
		} else if s, v := ts(*x), auto_get(ctx, "_"); v == nil {
			if s != t { erro(ctx, "%s: %s != %s", p, s, t).trace() }
		} else if t := "{=arrow {=project foo}→{=word name}}"; s != t {
			erro(pc(ctx,_x), "%s: %v != %s; %s", p, s, t, res).trace()
		} else if s, t := ts(res), "{=self foo}"; s != t {
			erro(pc(ctx,_x), "%s: %v != %s; %s", p, s, t, res).trace()
		}
	case "$_→name?":
		if s, t := ts(_x), "{=cond {=arrow {=delegate {=auto _}}→{=word name}}}"; s != t {
			erro(pc(ctx,_x), "%s: %v != %s", p, s, t).trace()
		} else if s, v := ts(*x), auto_get(ctx, "_"); v == nil {
			if s != t { erro(ctx, "%s: %s != %s", p, s, t).trace() }
		} else if t := "{=arrow {=project foo}→{=word name}}"; s != t {
			erro(pc(ctx,_x), "%s: %v != %s", p, s, t).trace()
		}
	case "$_→bar?":
		if s, t := ts(_x), "{=cond {=arrow {=delegate {=auto _}}→{=word bar}}}"; s != t {
			erro(pc(ctx,_x), "%s: %v != %s", p, s, t).trace()
		} else if s, v := ts(*x), auto_get(ctx, "_"); v == nil {
			if s != t { erro(ctx, "%s: %s != %s", p, s, t).trace() }
		} else if t := "{=arrow {=project foo}→{=word bar}}"; s != t {
			erro(pc(ctx,_x), "%s: %v != %s", p, s, t).trace()
		}
	}
}

func ex_check_value_closure(ctx Context, p, _x Value, _o, _a []Value, res, x *Value, o, a *[]Value) {
	if truly(ctx, ex_closure{}) {
		switch v := *res ; p.String() {
		case "&(foo.pre)":
			if s := do(ctx, get_scope{}); s == nil {
				erro(pc(ctx,p), "%v → %v : nil scope", p, *res).trace()
			}
			if cs := do(ctx, get_closure_scopes{}); cs == nil {
				erro(pc(ctx,p), "%v → %v : nil closure scopes ; %v", p, *res, do(ctx, get_scope{})).trace()
			}
			if cs := closure_scopes(ctx); cs == nil {
				erro(pc(ctx,p), "%v → %v : nil closure scopes ; %v", p, *res, do(ctx, get_closure_scopes{})).trace()
			}
			if cp := closure_projects(ctx); cp == nil {
				erro(pc(ctx,p), "%v → %v : nil closure projects", p, *res).trace()
			}
			if o := closure_resolve(ctx, "foo.pre"); o == nil {
				erro(pc(ctx,p), "%v → %v", p, *res).trace()
			}
			if ts(v) != "{=word foo}" {
				erro(pc(ctx,v), "%v → %v", p, *res).trace()
			}
		case "&(foo.pos)":
			if o := closure_resolve(ctx, "foo.pos"); o == nil {
				erro(pc(ctx,p), "%v → %v", p, *res).trace()
			}
			if ts(v) != "{=word foo}" {
				erro(pc(ctx,v), "%v → %v", p, *res).trace()
			}
		case "&(&(foo.tail))":
			if ts(v) != "{=word foo}" {
				erro(pc(ctx,v), "%v → %v", p, *res).trace()
			}
		}
	}
}

func ex_check_value_2(ctx Context, p, _x, x Value, o, a []Value, res Value) {
	if false && p.String() == "$(string x $(closure &(.test.x)) y $(&(.test.x) aa,bb) c)" {
		note(pc(ctx,_x), "p=%v, x=%v→%v", ts(p), ts(_x), ts(x))
		notestack(pc(ctx,_x), 1, "r=%v", ts(res)).debug(16)
	}
}

func ex_check_value_4(ctx Context, p, _x Value, _o, _a []Value, res, x *Value, o, a *[]Value) {
	switch s := ts(_x); s {
	case "{=compound {=punct .} {=word test} {=punct .} {=word foreach}}":
		if s, t := ts(*x), "{=def .test.foreach}"; s != t {
			erro(ctx, "%s : %s != %s → %s", p, s, t, *res).trace()
		}
	case "{=compound {=punct .} {=word test} {=punct .} {=word v}}":
		if s, t := ts(*x), "{=def .test.v}"; s != t {
			erro(ctx, "%s : %s != %s → %s", p, s, t, *res).trace()
		}
	case "{=compound {=punct .} {=word test} {=punct .} {=word x}}":
		switch p.String() {
		case "&(.test.x)":
			if _project(ctx).def(ctx, ".test.x") == nil {
				if t := ts(*x); s != t {
					erro(pc(ctx,p), "%s : %s != %s → %s", p, s, t, *res).trace()
				}
				if s, t := ts(*res), "{=closure "+ts(_x)+"}"; s != t {
					erro(pc(ctx,p), "%s : %s != %s → %s", p, s, t, *res).trace()
				}
			} else if truly(ctx, ex_closure{}) {
				if s, t := ts(*x), "{=def .test.x}"; s != t {
					erro(pc(ctx,p), "%s : %s != %s → %s", p, s, t, *res).trace()
				}
				if s, t := ts(*res), "{=compound {=punct .} {=word test} {=punct .} {=word v}}"; s != t {
					erro(pc(ctx,p), "%s : %s != %s → %s", p, s, t, *res).trace()
				}
			}
		}
	}
	switch s := p.String() ; s {
	case "&(.test.x)":
		if truly(ctx, ex_closure{}) {
			if s, t := (*res).String(), ".test.v"; s != t {
				erro(ctx, "%v → %v → %v", s, t, *res).trace()
			}
		} else {
			if t := (*res).String(); s != t {
				erro(ctx, "%v → %v → %v", s, t, *res).trace()
			}
		}
	case "&(value .test.v)":
		if s, t := ts(_x), ts(*x); s != t {
			erro(ctx, "%v → %v → %v", s, t, *res).trace()
		}
	case "$(value &(.test.x))":
		if s, t := ts(_x), ts(*x); s != t {
			erro(ctx, "%v → %v → %v", s, t, *res).trace()
		}
		if truly(ctx, ex_closure{}) {
			if s, t := (*res).String(), "xx"; s != t {
				erro(ctx, "%v → %v → %v", s, t, *res).trace()
			}
		} else {
			if t := (*res).String(); s != t {
				erro(ctx, "%v → %v → %v", s, t, *res).trace()
			}
		}
	}
}

func ex_check_builtins_foreach(ctx Context, p, _x Value, _o, _a []Value, res, x Value, o, a []Value) {
	switch p.String() {
	case "&(.test.h)":
		note(ctx, "%v %v %v", p, tv(x), res).debug()
	}
}

func ex_check_value_bug_01(ctx Context, p, _x Value, _o, _a []Value, res, x Value, o, a []Value) {
	switch s := p.String() ; s {
	case "$1":
		if false && truly(ctx, ex_closure{}) {
			x1, x2, x3, x4 := do(ctx, evoke_x{}), do(ctx, evoke_def{}), do(ctx, evoke_def{"bug_0.1"}), do(ctx, evoke_def{"bug_0.2"})
			s1, s2, s3, s4 := ts(x1), ts(x2), ts(x3), ts(x4)
			s0 := res.String()

			if s0 == s && s1 == "{=def 1}" && x1 == x2 && x3 != nil && x4 != nil {
				erro(ctx, "%s %s %s %s", s1, s2, s3, s4).trace() // %s → %s
			}

			if s1 == "{=def 1}" {
				note(ctx, "%s → %s : %s : %s : %s : 1=%s", s, s0, s1, s2, s3, x1.(*def).value).debug()
			} else if xv := x; ts(xv) == "{=def 1}" {
				note(ctx, "%s → %s : %s : %s : %s : x=%s", s, s0, s1, s2, s3, xv.(*def).value).debug()
			} else {
				note(ctx, "%s → %s : %s : %s : %s : %s", s, s0, s1, s2, s3, ts(x)).debug()
			}
		}

	case "&($2.$_)":
		if x, y := do(ctx, evoke_def{}).(*def); y {
			var v1, v2, v3 = auto_get(ctx, "1"), auto_get(ctx, "2"), auto_get(ctx, "3")
			var s1, s2, s3 = ts(v1), ts(v2), ts(v3)
			switch x.name {
			case ".flag":
				if r, t := res.String(), "&("+v2.String()+".$_)"; r != t {
					erro(ctx, "%s → %s != %s : %v, %v, %v", s, r, t, s1, s2, s3).trace()
				}
			case "bug_0.2":
				if x.value.String() != "$(foreach(-unique) $(foreach $1,&($2.$_)),$_)" {
					erro(ctx, "%v → %v : %v", s, res, x).trace()
				}
				if s2 == s3 {
					if r, t := res.String(), "&("+v2.String()+".$_)"; r != t {
						erro(ctx, "%s → %s != %s : %v", s, r, t, s1).trace()
					}
				} else {
					if t := res.String(); s != t {
						erro(ctx, "%s → %s : %v, %v, %v", s, t, s1, s2, s3).trace()
					}
				}
			}
		} else if _, y := do(ctx, evoke_builtin{"foreach"}).(*builtin); y && false {
			erro(ctx, "%v → %v : %v : %v", s, res, ts(auto_get(ctx, "_")), do(ctx, evoke_def{})).trace()
		} else if s, t := "&($2.{$1})", res.String(); s != t {
			erro(ctx, "%v → %v : %v : %v", s, t, ts(auto_get(ctx, "_")), ts(do(ctx, evoke_x{}))).trace()
		} else if v := auto_get(ctx, "_"); v == nil {
			erro(ctx, "%v → %v : %v : %v", s, t, ts(auto_get(ctx, "_")), ts(do(ctx, evoke_x{}))).trace()
		} else if s, t := "{=disjunction {=delegate {=auto 1}}}", ts(v); s != t {
			erro(ctx, "%v → %v : %v : %v", s, t, ts(auto_get(ctx, "_")), ts(do(ctx, evoke_x{}))).trace()
		} else if false {
			erro(ctx, "%v : %v : %v", s, auto_get(ctx, "_"), ts(do(ctx, evoke_x{}))).trace()
		}

	case "$(foreach $1,$2.$_)":
		var v1, v2, v3 = auto_get(ctx, "1"), auto_get(ctx, "2"), auto_get(ctx, "3")
		var s1, s2, s3 = ts(v1), ts(v2), ts(v3)
		var s0 string
		if s1 == "{=list {=delegate {=auto 1}} {=delegate {=auto 2}}}" {
			for _, s0 = range []string{"x", "y", "z"} {
				if s2 == "{=word "+s0+"}" && s3 == "{=word "+s0+"}" {
					if r, t := res.String(), s0+".{$1} "+s0+".{$2}"; s == r {
						erro(ctx, "%v → %v != %v; %v, %v, %v", s, r, t, v1, v2, v3).trace()
					} else if r != t {
						erro(ctx, "%v → %v != %v; %v, %v, %v", s, r, t, v1, v2, v3).trace()
					}
					break
				}
			}
		}
		if v0 := auto_get(ctx, "_"); v0 == nil && s0 == "" {
			if s, t := "$2.{$1}", res.String(); s != t {
				note(ctx, "1: %v → %v", v1, s1)
				note(ctx, "2: %v → %v", v2, s2)
				note(ctx, "3: %v → %v", v3, s3)
				note(ctx, "%v → %v", _a, a)
				errostack(ctx, 8, "%s != %s; %v", s, t, v0).trace()
			}
		}

	case "$(foreach(-unique) $(foreach $1,$2.$_),$_)":
		var v1, v2, v3 = auto_get(ctx, "1"), auto_get(ctx, "2"), auto_get(ctx, "3")
		var s1, s2, s3 = ts(v1), ts(v2), ts(v3)
		var s0 string
		if s1 == "{=list {=delegate {=auto 1}} {=delegate {=auto 2}}}" {
			for _, s0 = range []string{"x", "y", "z"} {
				if s2 == "{=word "+s0+"}" && s3 == "{=word "+s0+"}" {
					if r, t := res.String(), "{"+s0+".{$1}} {"+s0+".{$2}}"; s == r {
						errostack(pc(ctx,p), 3, "%v → %v != %v", s, r, t).trace()
					} else if r != t {
						errostack(pc(ctx,p), 3, "%v → %v != %v", s, r, t).trace()
					}
					break
				}
			}
		}
		if v0 := auto_get(ctx, "_"); v0 == nil && s0 == "" {
			if s, t := "{$2.{$1}}", res.String(); s != t {
				erro(ctx, "%v → %v : %v → %v : %v, %v, %v", s, t, _a, a, s1, s2, s3).trace()
			}
		}

	case "$(foreach x y z,$(okay.2 $1 $2,$_,$_))":
		var v1, v2 = auto_get(ctx, "1"), auto_get(ctx, "2")
		var s1, s2 = ts(v1), ts(v2)
		if x, y := do(ctx, evoke_def{"okay.1"}).(*def); y {
			if s1 == "{=list {=delegate {=auto 1}}}" && s2 == "{=list {=delegate {=auto 2}}}" {
				erro(ctx, "%v → %v : %v → %v : %v", s, res, _a, a, do(ctx, evoke_x{})).trace()
			} else {
				erro(ctx, "%v → %v : %v %v : %v → %v", s, x, s1, s2, _a, a).trace()
			}
		} else if s, t := res.String(), "$(okay.2 $1 $2,x,x)? $(okay.2 $1 $2,y,y)? $(okay.2 $1 $2,z,z)?"; s != t {
			if s1 != "{}" || s2 != "{}" {
				erro(ctx, "%v → %v : %v %v : %v", s, t, s1, s2, do(ctx, evoke_x{})).trace()
			} else {
				erro(ctx, "%v → %v : %v → %v : %v", s, t, _a, a, do(ctx, evoke_x{})).trace()
			}
		} else if s, t := ts(res), "{=list {=condval {=delegate {=def okay.2} {=list {=delegate {=auto 1}} {=delegate {=auto 2}}} {=list {=word x}} {=list {=word x}}}} {=condval {=delegate {=def okay.2} {=list {=delegate {=auto 1}} {=delegate {=auto 2}}} {=list {=word y}} {=list {=word y}}}} {=condval {=delegate {=def okay.2} {=list {=delegate {=auto 1}} {=delegate {=auto 2}}} {=list {=word z}} {=list {=word z}}}}}"; s != t {
			if s1 != "{}" || s2 != "{}" {
				erro(ctx, "%v → %v : %v %v : %v", s, t, s1, s2, do(ctx, evoke_x{})).trace()
			} else {
				erro(ctx, "%v → %v : %v → %v : %v", s, t, _a, a, do(ctx, evoke_x{})).trace()
			}
		}

	case "$(okay.2 $1 $2,$_,$_)":
		if v := auto_get(ctx, "_"); v == nil {
			if t := res.String(); s != t {
				erro(ctx, "%v → %v : %v", s, t, do(ctx, evoke_x{})).trace()
			}
		} else {
			if s, t := res.String(), fmt.Sprintf("$(okay.2 $1 $2,%s,%s)", v, v); s != t {
				erro(ctx, "%v → %v : %v", s, t, do(ctx, evoke_x{})).trace()
			}
		}

	case "$(okay.1 $1,$2)":
		if truly(ctx, ex_delegate{}) {
			if s, t := res.String(), "{x.{$1 $2}} {y.{$1 $2}} {z.{$1 $2}}"; s != t {
				erro(ctx, "%v != %v : %v → %v : %v", s, t, _a, a, do(ctx, evoke_x{})).trace()
			} else {
				var v1, v2, v3 = auto_get(ctx, "1"), auto_get(ctx, "2"), auto_get(ctx, "3")
				if v1 != nil || v2 != nil || v3 != nil {
					erro(ctx, "%v : %v %v %v : %v", s, v1, v2, v2, do(ctx, evoke_x{})).trace()
				}
			}
		} else if t := res.String(); s != t {
			erro(ctx, "%v → %v : %v → %v : %v", s, t, _a, a, do(ctx, evoke_x{})).trace()
		}
		if s, t := ts(_a), "{=[]Value {=list {=delegate {=auto 1}}} {=list {=delegate {=auto 2}}}}"; s != t {
			erro(ctx, "%v → %v : %v → %v : %v", s, t, _a, a, do(ctx, evoke_x{})).trace()
		} else if s := ts(a); s != t {
			erro(ctx, "%v → %v : %v → %v : %v", s, t, _a, a, do(ctx, evoke_x{})).trace()
		}
	}
}

func ex_check_rule_shell_forstdout(ctx Context, p, _x Value, _o, _a []Value, res, x *Value, a *[]Value) {
	var o = try[origin](ctx, get_origin{})

	switch p.String() {
	case "${.test $1,$2}":
		if a1, a2 := auto_get(ctx, "1"), auto_get(ctx, "2") ; a1 != nil && a2 != nil {
			if ts(*a) != sfmt("{=[]Value {=list %s} {=list %s}}", ts(a1), ts(a2)) {
				errostack(ctx, 5, "%s %s: %s, %s ; %v, %v", typeof(_x), typeof(*x), ts(a1), ts(a2), ts(_a), ts(*a)).trace()
			}
			switch o {
			case defExpand0:
				if ts(*res) != sfmt("{=delegate {=builtin debug} {=list %s %s}}", ts(a1), ts(a2)) {
					errostack(ctx, 3, "%s %s, %s", ts(a1), ts(a2), ts(*res)).trace()
				}
			case defExpand1:
				if ts(*res) != "{=null}" {
					errostack(ctx, 3, "%s, %s", ts(*a), ts(*res)).trace()
				}
			}
		} else if truly(ctx, ex_delegate{}) {
			if ts(*res) != "{=delegate {=builtin debug} {=list {=delegate {=auto 2}} {=delegate {=auto 1}}}}" {
				errostack(ctx, 3, "%v: %s", p, ts(*res)).trace()
			}
		} else {
			erro(ctx, "%v: %s %s ; %s", p, ts(a1), ts(a2), ts(*res)).trace()
		}
	case "${.test a,b}":
		switch o {
		case defExpand0:
			if ts(*res) != "{=delegate {=builtin debug} {=list {=word b} {=word a}}}" {
				errostack(ctx, 3, "%v", ts(*res)).trace()
			}
		case defExpand1:
			if ts(*res) != "{=null}" {
				errostack(ctx, 3, "%s", ts(*res)).trace()
			}
		}
	case "$(.test.v3 a,b)":
		switch o {
		case defExpand1:
			if ts(*res) != "{=null}" {
				errostack(ctx, 3, "%s", ts(*res)).trace()
			}
		}
	case "$(debug $(line) $(str))":
		if ts(_a) != "{=[]Value {=list {=delegate {=auto line}} {=delegate {=auto str}}}}" {
			erro(ctx, "%v", ts(_a)).trace()
		}

		if a := _automatic(ctx); a == nil {
			errostack(ctx, 5, "%v", ts(*res)).trace()
		} else {
			keys := reflect.ValueOf(a.defs).MapKeys()

			if x1, y := a.defs["1"]; !y {
				errostack(ctx, 5, "%v; %v", keys, ts(*res)).trace()
			} else if x2, y := a.defs["str"]; !y {
				errostack(ctx, 5, "%v; %v", keys, ts(*res)).trace()
			} else if x1 != x2 {
				errostack(ctx, 5, "%v != %v; %v", x1, x2, ts(*res)).trace()
			}

			if x1, y := a.defs["2"]; !y {
				errostack(ctx, 5, "%v; %v", keys, ts(*res)).trace()
			} else if x2, y := a.defs["line"]; !y {
				errostack(ctx, 5, "%v; %v", keys, ts(*res)).trace()
			} else if x1 != x2 {
				errostack(ctx, 5, "%v != %v; %v", x1, x2, ts(*res)).trace()
			}
		}
		if a, b := auto_get(ctx, "1"), auto_get(ctx, "str"); a != b {
			errostack(ctx, 5, "%v != %v; %v", a, b, ts(*res)).trace()
		}
		if a, b := auto_get(ctx, "2"), auto_get(ctx, "line"); a != b {
			errostack(ctx, 5, "%v != %v; %v", a, b, ts(*res)).trace()
		}

		switch ts(*a) {
		case "{=[]Value {=list {=delegate {=auto 2}} {=delegate {=auto 1}}}}":
			switch o {
			case defExpand0, defExpand1:
				if ts(*res) != "{=delegate {=builtin debug} {=list {=delegate {=auto 2}} {=delegate {=auto 1}}}}" {
					errostack(ctx, 5, "%v", ts(*res)).trace()
				}
			}
		case "{=[]Value {=list {=word b} {=word a}}}":
			switch o {
			case defExpand0:
				if ts(*res) != "{=delegate {=builtin debug} {=list {=word b} {=word a}}}" {
					errostack(ctx, 5, "%v", ts(*res)).trace()
				}
			case defExpand1:
				if ts(*res) != "{}" {
					errostack(ctx, 5, "%v", ts(*res)).trace()
				}
			}
		case "{=[]Value}":
			var t = []Value{auto_get(ctx, "1"), auto_get(ctx, "2")}
			if ts(t) != "{=[]Value {} {}}" {
				errostack(ctx, 5, "%v %v", ts(*x), ts(t)).trace()
			}
			if ts(*res) != "{}" {
				errostack(ctx, 5, "%v", ts(*res)).trace()
			}
		case
			`{=[]Value {=list {=decimal 1} {=strlit test one\n}}}`,
			`{=[]Value {=list {=decimal 2} {=strlit test two\n}}}`,
			`{=[]Value {=list {=decimal 3} {=strlit test thr\n}}}`:
			if ts(*res) != "{}" {
				errostack(ctx, 5, "%v", ts(*res)).trace()
			}
		default:
			errostack(ctx, 5, "untested: %v, %s, %s", o, ts(*a), ts(*res)).trace()
		}
	}

	switch o {
	case defExpand0, defExpand1, 0:
	default:
		errostack(ctx, 5, "untested: %v %s %s", o, ts(_a), ts(*res)).trace()
	}
}

func expand_check(ctx Context, e, v Value) {
	if e == nil {
		erro(ctx, "nil a").trace()
	}
	if v == nil {
		erro(ctx, "nil b").trace()
	}
	if a, b := e.cmp(ctx, v), v.cmp(ctx, e); a != b {
		note(ctx, "cmp.a: %v", e)
		note(ctx, "cmp.b: %v", v)
		erro(pc(ctx,v), "cmp(%s, %s) → (%v, %v)", typeof(e), typeof(v), a, b).trace()
	}
}

func equal_check(x Context, a, b Value, _res *bool) {
	switch j := _project(x); j.spec {
	case "testdata/value/auto":
		equal_check_value_auto(x, a, b, *_res)
	}
}
func equal_check_value_auto(x Context, a, b Value, res bool) {
	if res {
		if ts(a) != ts(b) {
			erro(pc(x,a), "%v != %v : %v != %v", a, b, ts(a), ts(b)).trace()
		}
	} else {
		if a == nil || b == nil {
			erro(pc(x,a), "%v ⇔ %v", a, b).trace()
		}
		if ts(a) == ts(b) {
			erro(pc(x,a), "%v != %v : %v != %v", a, b, ts(a), ts(b)).trace()
		}
	}
}

func (p *none) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *null) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p negative) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *escaped) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *boolean) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *float) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *integer) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *binary) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *datetime) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *url) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *word) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *arrow) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && v != nil {
		if v == nil {
			errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
		}
		if p.String() == v.String() {
			if x, y := v.(*arrow); y {
				note(ctx, "%v %v, %v", ts(p.o), ts(x.o), p.o.cmp(ctx, x.o))
				note(ctx, "%v %v, %v", ts(p.s), ts(x.s), p.s.cmp(ctx, x.s))
			}
			errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
		}
	}
}

func (p cond) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p disjunction) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && ts(p) == ts(v) {
		errostack(pc(ctx,p), 3, "%v, %s == %s, %s ⇔ %s", res, p, v, ts(p), ts(v)).trace()
	} else if x, y := v.(disjunction); y {
		if p.Value.cmp(ctx, x.Value) != res {
			errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p.Value), ts(x.Value)).trace()
		}
	}
}
func (p disjunction) expand_check(ctx Context, _res *Value) {
	if j := _project(ctx); j != nil {
		switch j.spec {
		case "testdata/builtins/addprefix":
			p.expand___addprefix(ctx, *_res)
		case "testdata/builtins/addsuffix":
			p.expand___addsuffix(ctx, *_res)
		case "testdata/builtins/foreach":
			p.expand___foreach(ctx, *_res)
		}
	}
}
func (p disjunction) expand___addprefix(ctx Context, res Value) {
	switch s0 := p.Value.String(); s0 {
	case "$1":
		if v := auto_get(ctx, "1"); truly(ctx, ex_closure{}) {
			if v == nil {
				if s, t := ts(res), "{=delegate {=auto 1}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace() // where: truly(ctx, identity{})
				}
			} else {
				if s, t := ts(res), "{=delegate {=auto 1}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace() // where: truly(ctx, identity{})
				}
			}
		} else if truly(ctx, ex_delegate{}) {
			if v == nil {
				if res != nil {
					erro(pc(ctx,p), "%v", res).trace()
				}
			} else {
				if s, t := ts(res), fmt.Sprintf("{=closure {=compound {=punct .} {=word test} {=punct .} %s}}", ts(v)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			if s, t := ts(res), "{=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "$2":
		if v := auto_get(ctx, "2"); truly(ctx, ex_closure{}) {
			if v == nil {
				if s, t := ts(res), "{=delegate {=auto 2}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace() // where: truly(ctx, identity{})
				}
			} else {
				if s, t := ts(res), "{=delegate {=auto 2}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace() // where: truly(ctx, identity{})
				}
			}
		} else if truly(ctx, ex_delegate{}) {
			if v == nil {
				if res != nil {
					erro(pc(ctx,p), "%v", res).trace()
				}
			} else {
				if s, t := ts(res), fmt.Sprintf("{=closure {=compound {=punct .} {=word test} {=punct .} %s}}", ts(v)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			if s, t := ts(res), "{=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 2}}}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "$3":
		if v := auto_get(ctx, "3"); truly(ctx, ex_closure{}) {
			if v == nil {
				if s, t := ts(res), "{=delegate {=auto 3}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace() // where: truly(ctx, identity{})
				}
			} else {
				if s, t := ts(res), "{=delegate {=auto 3}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace() // where: truly(ctx, identity{})
				}
			}
		} else if truly(ctx, ex_delegate{}) {
			if v == nil {
				if res != nil {
					erro(pc(ctx,p), "%v", res).trace()
				}
			} else {
				if s, t := ts(res), fmt.Sprintf("{=closure {=compound {=punct .} {=word test} {=punct .} %s}}", ts(v)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			if s, t := ts(res), "{=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 3}}}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "&(none)":
		if truly(ctx, ex_closure{}) {
			if res != nil {
				erro(pc(ctx,p), "%v %s", res, ts(res)).trace()
			}
		} else {
			if s, t := ts(res), "{=closure {=word none}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "&(.test.$1)":
		if v := auto_get(ctx, "1"); truly(ctx, ex_closure{}) {
			if v == nil {
				if s, t := ts(res), "{=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace() // where: truly(ctx, identity{})
				}
			} else {
				switch v.String() {
				case "a":
					if s, t := ts(res), "{=list {=word ax} {=word ay} {=word az}}"; s != t {
						erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
					}
				case "b":
					if s, t := ts(res), "{=list {=word bx} {=word by} {=word bz}}"; s != t {
						erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
					}
				case "c":
					if s, t := ts(res), "{=list {=word cx} {=word cy} {=word cz}}"; s != t {
						erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
					}
				default:
					erro(pc(ctx,p), "%s: %s", v, res).trace()
				}
			}
		} else if truly(ctx, ex_delegate{}) {
			if v == nil {
				if res != nil {
					erro(pc(ctx,p), "%v", res).trace()
				}
			} else {
				if s, t := ts(res), fmt.Sprintf("{=closure {=compound {=punct .} {=word test} {=punct .} %s}}", ts(v)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			if s, t := ts(res), "{=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "&(.test.{$1})":
		if v := auto_get(ctx, "1"); truly(ctx, ex_closure{}) {
			if v == nil {
				if s, t := ts(res), "{=closure {=compound {=punct .} {=word test} {=punct .} {=disjunction {=delegate {=auto 1}}}}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace() // where: truly(ctx, identity{})
				}
			} else {
				switch v.String() {
				case "a":
					if s, t := ts(res), "{=list {=word ax} {=word ay} {=word az}}"; s != t {
						erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
					}
				case "b":
					if s, t := ts(res), "{=list {=word bx} {=word by} {=word bz}}"; s != t {
						erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
					}
				case "c":
					if s, t := ts(res), "{=list {=word cx} {=word cy} {=word cz}}"; s != t {
						erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
					}
				default:
					erro(pc(ctx,p), "%s: %s", v, res).trace()
				}
			}
		} else if truly(ctx, ex_delegate{}) {
			if v == nil {
				if res != nil {
					erro(pc(ctx,p), "%v", res).trace()
				}
			} else {
				if s, t := ts(res), fmt.Sprintf("{=closure {=compound {=punct .} {=word test} {=punct .} %s}}", ts(v)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			if s, t := ts(res), "{=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "&(.test.a)":
		if s, t := ts(res), "{=list {=word ax} {=word ay} {=word az}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "&(.test.b)":
		if s, t := ts(res), "{=list {=word bx} {=word by} {=word bz}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "&(.test.$1.x.$2.y.$3.z)":
		var v1 = auto_get(ctx, "1")
		var v2 = auto_get(ctx, "2")
		var v3 = auto_get(ctx, "3")
		if v1 != nil && v2 != nil && v3 != nil {
			if truly(ctx, ex_closure{}) {
				if s, t := ts(res), "{=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace() // where: truly(ctx, identity{})
				}
			} else if truly(ctx, ex_delegate{}) {
				if s, t := ts(res), fmt.Sprintf("{=closure {=compound {=punct .} {=word test} {=punct .} %s}}", ts(v1)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				erro(pc(ctx,p), "%v: %s ; %v %v %v", res, ts(res), v1, v2, v3).trace()

			}
		} else if s, t := ts(res), "{=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}} {=punct .} {=word x} {=punct .} {=delegate {=auto 2}} {=punct .} {=word y} {=punct .} {=delegate {=auto 3}} {=punct .} {=word z}}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "&(.test.a.x.1.y.0.z)":
		if truly(ctx, ex_closure{}) {
			if s, t := ts(res), "{=list {=word ax} {=word ay} {=word az}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		} else {
			if s, t := ts(res), "{=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a} {=punct .} {=word x} {=punct .} {=word 1} {=punct .} {=word y} {=punct .} {=word 0} {=punct .} {=word z}}}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "&(.test.a.x.2.y.0.z)":
		if truly(ctx, ex_closure{}) {
			if s, t := ts(res), "{}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		} else {
			if s, t := ts(res), "{=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a} {=punct .} {=word x} {=punct .} {=word 2} {=punct .} {=word y} {=punct .} {=word 0} {=punct .} {=word z}}}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "&(.test.a.x.3.y.0.z)":
		if truly(ctx, ex_closure{}) {
			if s, t := ts(res), "{}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		} else {
			if s, t := ts(res), "{=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a} {=punct .} {=word x} {=punct .} {=word 3} {=punct .} {=word y} {=punct .} {=word 0} {=punct .} {=word z}}}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "&(.test.b.x.1.y.0.z)":
	case "&(.test.b.x.2.y.0.z)":
	case "&(.test.b.x.3.y.0.z)":
	case "&(.test.c.x.1.y.0.z)":
	case "&(.test.c.x.2.y.0.z)":
	case "&(.test.c.x.3.y.0.z)":
	default:
		erro(pc(ctx,p), "%s: %v %s", s0, res, ts(res)).trace()
	}
}
func (p disjunction) expand___addsuffix(ctx Context, res Value) {
	switch s0 := p.Value.String(); s0 {
	case "$1":
		if v := auto_get(ctx, "1"); truly(ctx, ex_closure{}) {
			if v == nil {
				if s, t := ts(res), "{=delegate {=auto 1}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), "{=delegate {=auto 1}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else if truly(ctx, ex_delegate{}) {
			if v == nil {
				if res != nil {
					erro(pc(ctx,p), "%v", res).trace()
				}
			} else {
				if s, t := ts(res), fmt.Sprintf("{=closure {=compound {=punct .} {=word test} {=punct .} %s}}", ts(v)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			if s, t := ts(res), "{=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "&(.test.$1)":
		if v := auto_get(ctx, "1"); truly(ctx, ex_closure{}) {
			if v == nil {
				if s, t := ts(res), "{=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), "{=delegate {=auto 1}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else if truly(ctx, ex_delegate{}) {
			if v == nil {
				if res != nil {
					erro(pc(ctx,p), "%v", res).trace()
				}
			} else {
				if s, t := ts(res), fmt.Sprintf("{=closure {=compound {=punct .} {=word test} {=punct .} %s}}", ts(v)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			if s, t := ts(res), "{=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		}
	default:
		erro(pc(ctx,p), "%v: %v %s", s0, res, ts(res)).trace()
	}
}
func (p disjunction) expand___foreach(ctx Context, res Value) {
	switch s0 := p.String(); s0 {
	case "&(.test.h)$_":
		if u := auto_get(ctx, "_"); u == nil {
			erro(pc(ctx,p), "%s : %v", s0, ts(res)).trace()
		} else {
			if s, t := res.String(), "&(.test.h){"+u.String()+"}"; s != t {
				erro(pc(ctx,p), "%s != %s : %v, %v", s, t, ts(u), ts(res)).trace()
			}
		}
	case "{&(.test.h){$1}}":
		if a := auto_get(ctx, "1"); a == nil {
			if s := res.String(); s != s0 {
				erro(pc(ctx,p), "%s != %s : %v", s, s0, ts(res)).trace()
			}
		} else {
			var t = "{"
			for i, v := range merge(a) {
				if 0 < i { t += " " }
				t += "&(.test.h)" + v.String()
			}
			t += "}"
			if s := res.String(); s != t {
				erro(pc(ctx,p), "%s != %s : %v, %v", s, t, a, ts(res)).trace()
			}
		}
	default:
		// note(ctx, "%v %v %v", p, indeterminate(ctx, p.Value), res).debug()
	}
}

func (p *qualword) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *raw) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v; %v ⇔ %v", res, p, v, ts(p), ts(v)).trace()
	}
}

func (p *strlit) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *strval) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *globmeta) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *globrange) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *argumented) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *recipe) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *barefile) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *arrow) evoke_check(ctx Context, _o, _s, _res *Value) {
	switch j := _project(ctx); j.spec {
	case "testdata/value/optional":
		p.evoke_check_value_optional(ctx, j, *_o, *_s, *_res)
	}
}
func (p *arrow) evoke_check_value_optional(ctx Context, j *project, o, s, res Value) {
	switch ps := p.String(); ps {
	case "foo→name":
		if ts(o) != "{=project foo}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word name}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(res) != "{=self foo}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
	case "foo→name→item":
		if ts(o) != "{=self foo}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word item}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(res) != "{=answer yes}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
	case "foo→baz":
		if ts(o) != "{=word foo}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word baz}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if res != nil {
			if ts(res) != "{=null}" {
				erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
			}
		}
	case "foo→bar":
		if ts(o) != "{=word foo}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word bar}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if res != nil {
			if ts(res) != "{=}" {
				erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
			}
		}
	case "fo→bar":
		if ts(o) != "{=word fo}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word bar}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(res) != "{=null}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
	case "fo?→bar":
		if ts(o) != "{=cond {=word fo}}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word bar}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(res) != "{=arrow {=cond {=word fo}}→{=word bar}}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
	case "$_→name":
		if v := auto_get(ctx, "_"); v == nil {
			if o != p.o {
				erro(ctx, "%v != %v", o, p.o).trace()
			} else if false {
				note(ctx, "%v %v %v %v", p, o, s, res).debug()
			}
		} else if s, t := "{=project foo}", ts(v); s != t {
			erro(ctx, "%v != %v", s, t).trace()
		} else {
			note(ctx, "%v %v %v %v", p, o, s, res).debug()
		}
	case "$_→bar":
		if v := auto_get(ctx, "_"); v == nil {
			if o != p.o {
				erro(ctx, "%v != %v", o, p.o).trace()
			} else if false {
				note(ctx, "%v %v %v %v", p, o, s, res).debug()
			}
		} else if s, t := "{=project foo}", ts(v); s != t {
			erro(ctx, "%v != %v", s, t).trace()
		} else {
			note(ctx, "%v %v %v %v", p, o, s, res).debug()
		}
	case "{=project foo}→name":
		if ts(o) != "{=project foo}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word name}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(res) != "{=self foo}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		} else if false {
			note(pc(ctx,p), "%v %v→%v %v", p, o, s, res).debug(6)
		}
	case "{=project foo}→bar":
		if ts(o) != "{=project foo}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word bar}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if res != nil {
			if ts(res) != "{=}" {
				erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
			}
		}
	case "{=project foo}→baz":
		if ts(o) != "{=project foo}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word baz}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if res != nil {
			if ts(res) != "{=null}" {
				erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
			}
		}
	case "{=project foo}→name→xxxx":
		if ts(o) != "{=self foo}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word xxxx}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if res != nil {
			if ts(res) != "{=null}" {
				erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
			}
		}
	case "{=project foo}→name→item":
		if ts(o) != "{=self foo}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word item}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(res) != "{=answer yes}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
	default:
		erro(ctx, "%s: %s; %s→%s; %s", ps, ts(p), ts(o), ts(s), ts(res)).trace()
	}
}

func (p *arrow) expand_check(ctx Context, _o, _s, _res *Value) {
	var res = *_res
	if res == nil { return }

	switch j := _project(ctx); j.spec {
	case "testdata/value/optional":
		p.expand__value_optional(ctx, j, *_o, *_s, res)
	}

	if equal(ctx, p, res) {
		if truly(ctx, ex_condless{}) {
			if _, y := res.(cond); y {
				erro(ctx, "%v → %v", p, res).trace()
			}

			var po = p.o; if x, y := po.(cond); y { po = x.Value }
			var ps = p.s; if x, y := ps.(cond); y { ps = x.Value }
			if s, t := ts(*_o), ts(po); s != t {
				erro(ctx, "%s != %s", s, t).trace()
			}
			if s, t := ts(*_s), ts(ps); s != t {
				erro(ctx, "%s != %s", s, t).trace()
			}

			if !is_cond(p.o) && !is_cond(p.s) {
				if s, t := res.String(), p.String(); s != t {
					erro(ctx, "%v → %v : %s != %s", p, res, s, t).trace()
				}
				if s, t := ts(res), ts(p); s != t {
					erro(ctx, "%v → %v : %s != %s", p, res, s, t).trace()
				}
			}
		} else {
			if s, t := ts(*_o), ts(p.o); s != t {
				erro(ctx, "%s != %s : %v %v", s, t, is_cond(*_o), is_cond(p.o)).trace()
			}
			if s, t := ts(*_s), ts(p.s); s != t {
				erro(ctx, "%s != %s : %v %v", s, t, is_cond(*_s), is_cond(p.s)).trace()
			}
			if s, t := res.String(), p.String(); s != t {
				erro(ctx, "%v → %v : %s != %s", p, res, s, t).trace()
			}
			if s, t := ts(res), ts(p); s != t {
				erro(ctx, "%v → %v : %s != %s", p, res, s, t).trace()
			}
		}
	} else {
		if s, t := res.String(), p.String(); s == t {
			erro(ctx, "%v → %v : %s == %s", p, res, s, t).trace()
		}
		if s, t := ts(res), ts(p); s == t {
			erro(ctx, "%v → %v : %s == %s", p, res, s, t).trace()
		}
		if false && *_s != nil {
			if *_o != nil {
				if x, y := (*_o).(cond); y {
					erro(ctx, "%v", ts(x)).trace()
				}
			}
			if *_s != nil {
				if x, y := (*_s).(cond); y {
					erro(ctx, "%v", ts(x)).trace()
				}
			}
		}
	}
}
func (p *arrow) expand__value_optional(ctx Context, proj *project, o, s, res Value) {
	switch ps := p.String(); ps {
	case "foo→name":
		if s, t := ts(o), "{=word foo}"; s != t {
			erro(pc(ctx,p), "%s: %s; %s != %s; %s", p, ts(p), s, t, ts(res)).trace()
		}
		if s, t := ts(s), "{=word name}"; s != t {
			erro(pc(ctx,p), "%s: %s; %s != %s; %s", p, ts(p), s, t, ts(res)).trace()
		}
		if s, t := ts(res), "{=arrow {=word foo}→{=word name}}"; s != t {
			erro(pc(ctx,p), "%s: %s; %s != %s; %s", p, ts(p), s, t, ts(res)).trace()
		}
	case "foo→name→item":
		if s, t := ts(o), "{=arrow {=word foo}→{=word name}}"; s != t {
			erro(pc(ctx,p), "%s: %s; %s != %s; %s", p, ts(p), s, t, ts(res)).trace()
		}
		if s, t := ts(s), "{=word item}"; s != t {
			erro(pc(ctx,p), "%s: %s; %s != %s; %s", p, ts(p), s, t, ts(res)).trace()
		}
		if s, t := ts(res), "{=arrow {=arrow {=word foo}→{=word name}}→{=word item}}"; s != t {
			erro(pc(ctx,p), "%s: %s; %s != %s; %s", p, ts(p), s, t, ts(res)).trace()
		}
	case "foo?→name":
		if s, t := ts(o), "{=project foo}"; s != t {
			erro(pc(ctx,p), "%s: %s; %s != %s; %s", p, ts(p), s, t, ts(res)).trace()
		}
		if s, t := ts(s), "{=word name}"; s != t {
			erro(pc(ctx,p), "%s: %s; %s != %s; %s", p, ts(p), s, t, ts(res)).trace()
		}
		if s, t := ts(res), "{=self foo}"; s != t {
			erro(pc(ctx,p), "%s: %s; %s != %s; %s", p, ts(p), s, t, ts(res)).trace()
		}
	case "foo?→name?→item":
		if s, t := ts(o), "{=self foo}"; s != t {
			erro(pc(ctx,p), "%s: %s; %s != %s; %s", p, ts(p), s, t, ts(res)).trace()
		}
		if s, t := ts(s), "{=word item}"; s != t {
			erro(pc(ctx,p), "%s: %s; %s != %s; %s", p, ts(p), s, t, ts(res)).trace()
		}
		if s, t := ts(res), "{=answer yes}"; s != t {
			erro(pc(ctx,p), "%s: %s; %s != %s; %s", p, ts(p), s, t, ts(res)).trace()
		}
	case "foo→baz":
		if s, t := ts(o), "{=word foo}"; s != t {
			erro(pc(ctx,p), "%s: %s; %s != %s; %s", p, ts(p), s, t, ts(res)).trace()
		}
		if s, t := ts(s), "{=word baz}"; s != t {
			erro(pc(ctx,p), "%s: %s; %s != %s; %s", p, ts(p), s, t, ts(res)).trace()
		}
		if s, t := ts(res), "{=arrow {=word foo}→{=word baz}}"; s != t {
			erro(pc(ctx,p), "%s: %s; %s != %s; %s", p, ts(p), s, t, ts(res)).trace()
		}
	case "foo→bar":
		if s, t := ts(o), "{=word foo}"; s != t {
			erro(pc(ctx,p), "%s: %s; %s != %s; %s", p, ts(p), s, t, ts(res)).trace()
		}
		if s, t := ts(s), "{=word bar}"; s != t {
			erro(pc(ctx,p), "%s: %s; %s != %s; %s", p, ts(p), s, t, ts(res)).trace()
		}
		if s, t := ts(res), "{=arrow {=word foo}→{=word bar}}"; s != t {
			erro(pc(ctx,p), "%s: %s; %s != %s; %s", p, ts(p), s, t, ts(res)).trace()
		}
	case "fo→bar", "fo?→bar":
		if s, t := ts(o), "{=word fo}"; s != t {
			erro(pc(ctx,p), "%s: %s; %s != %s; %s", p, ts(p), s, t, ts(res)).trace()
		}
		if s, t := ts(s), "{=word bar}"; s != t {
			erro(pc(ctx,p), "%s: %s; %s != %s; %s", p, ts(p), s, t, ts(res)).trace()
		}
		if s, t := ts(res), "{=arrow {=word fo}→{=word bar}}"; s != t {
			erro(pc(ctx,p), "%s: %s; %s != %s; %s", p, ts(p), s, t, ts(res)).trace()
		}
	case "$_→name":
		if v := auto_get(ctx, "_"); v == nil {
			if o != p.o {
				erro(ctx, "%v != %v", o, p.o).trace()
			}
		} else {
			if s, t := ts(v), "{=project foo}"; s != t {
				erro(ctx, "%v != %v", s, t).trace()
			}
			if s, t := ts(o), "{=project foo}"; s != t {
				erro(pc(ctx,p), "%s; %s != %s; %s", ts(p), s, t, ts(res)).trace()
			}
			if s, t := ts(s), "{=word name}"; s != t {
				erro(pc(ctx,p), "%s; %s != %s; %s", ts(p), s, t, ts(res)).trace()
			}
			if s, t := ts(res), "{=self foo}"; s != t { // "{=arrow {=project foo}→{=word name}}"
				erro(pc(ctx,p), "%s; %s != %s; %s", ts(p), s, t, ts(res)).trace()
			}
		}
	case "$_→bar":
		if v := auto_get(ctx, "_"); v == nil {
			if o != p.o {
				erro(ctx, "%v != %v", o, p.o).trace()
			}
		} else if s0, t := ts(v), "{=project foo}"; s0 != t {
			erro(ctx, "%v != %v", s0, t).trace()
		} else {
			if s, t := ts(o), "{=project foo}"; s != t {
				erro(pc(ctx,p), "%s; %s != %s; %s", ts(p), s, t, ts(res)).trace()
			}
			if s, t := ts(s), "{=word bar}"; s != t {
				erro(pc(ctx,p), "%s; %s != %s; %s", ts(p), s, t, ts(res)).trace()
			}
			if s, t := ts(res), "{=arrow {=project foo}→{=word bar}}"; s != t {
				erro(pc(ctx,p), "%s; %s != %s; %s", ts(p), s, t, ts(res)).trace()
			}
		}
	case "{=project foo}→name":
		if s, t := ts(o), "{=project foo}"; s != t {
			erro(pc(ctx,p), "%s: %s != %s; %s", p, s, t, ts(res)).trace()
		}
		if s, t := ts(s), "{=word name}"; s != t {
			erro(pc(ctx,p), "%s: %s != %s; %s", p, s, t, ts(res)).trace()
		}
		if truly(ctx, ex_delegate{}) {
			if s, t := ts(res), "{=self foo}"; s != t {
				erro(pc(ctx,p), "%s: %s != %s; %s", p, s, t, ts(res)).trace()
			}
		} else {
			if s, t := ts(res), "{=arrow {=project foo}→{=word name}}"; s != t {
				erro(pc(ctx,p), "%s: %s != %s; %s", p, s, t, ts(res)).trace()
			}
		}
	case "{=project foo}→bar":
		if s, t := ts(o), "{=project foo}"; s != t {
			erro(pc(ctx,p), "%s: %s != %s; %s", p, s, t, ts(res)).trace()
		}
		if s, t := ts(s), "{=word bar}"; s != t {
			erro(pc(ctx,p), "%s: %s != %s; %s", p, s, t, ts(res)).trace()
		}
		if s, t := ts(res), "{=arrow {=project foo}→{=word bar}}"; s != t {
			erro(pc(ctx,p), "%s: %s != %s; %s", p, s, t, ts(res)).trace()
		}
	case "{=project foo}→baz":
		if s, t := ts(o), "{=project foo}"; s != t {
			erro(pc(ctx,p), "%s: %s != %s; %s", p, s, t, ts(res)).trace()
		}
		if s, t := ts(s), "{=word baz}"; s != t {
			erro(pc(ctx,p), "%s: %s != %s; %s", p, s, t, ts(res)).trace()
		}
		if s, t := ts(res), "{=arrow {=project foo}→{=word baz}}"; s != t {
			erro(pc(ctx,p), "%s: %s != %s; %s", p, s, t, ts(res)).trace()
		}
	case "{=project foo}→name→xxxx":
		if s, t := ts(o), "{=arrow {=project foo}→{=word name}}"; s != t {
			erro(pc(ctx,p), "%s; %s != %s; %s", ts(p), s, t, ts(res)).trace()
		}
		if s, t := ts(s), "{=word xxxx}"; s != t {
			erro(pc(ctx,p), "%s; %s != %s; %s", ts(p), s, t, ts(res)).trace()
		}
		if s, t := ts(res), "{=arrow {=arrow {=project foo}→{=word name}}→{=word xxxx}}"; s != t {
			erro(pc(ctx,p), "%s: %s != %s; %s", p, s, t, ts(res)).trace()
		}
	case "{=project foo}→name→item":
		if s, t := ts(o), "{=self foo}"; s != t {
			erro(pc(ctx,p), "%s; %s != %s; %s", ts(p), s, t, ts(res)).trace()
		}
		if s, t := ts(s), "{=word item}"; s != t {
			erro(pc(ctx,p), "%s; %s != %s; %s", ts(p), s, t, ts(res)).trace()
		}
		if s, t := ts(res), "{=answer yes}"; s != t {
			erro(pc(ctx,p), "%s: %s != %s; %s", p, s, t, ts(res)).trace()
		}
	case "{=project foo}→name?→item":
		if s, t := ts(o), "{=self foo}"; s != t {
			erro(pc(ctx,p), "%s; %s != %s; %s", ts(p), s, t, ts(res)).trace()
		}
		if s, t := ts(s), "{=word item}"; s != t {
			erro(pc(ctx,p), "%s; %s != %s; %s", ts(p), s, t, ts(res)).trace()
		}
		if s, t := ts(res), "{=answer yes}"; s != t {
			erro(pc(ctx,p), "%s: %s != %s; %s", p, s, t, ts(res)).trace()
		}
	default:
		erro(pc(ctx,o), "%v %v %v %v", p, o, s, res).debug()
	}
}

func (p *compound) cmp_check(ctx Context, v Value, _res *cmpres) {
    if res := *_res; res != cmpEqual && ts(p) == ts(v) {
        note(ctx, "%s, %s", p, ts(p))
        note(ctx, "%s, %s", v, ts(v))
        errostack(pc(ctx,v), 3, "%v, %v ; %v == %v", res, p==v, p, v).trace()
    }
}
func (p *compound) expand_check(ctx Context, _res *Value) {
	if j := _project(ctx); j != nil {
		switch j.name {
		case "configure.base":
			p.expand__configure_base(ctx, *_res)
		}
		switch j.spec {
		case "testdata/value/1":
			p.expand__value_1(ctx, *_res); return
		case "testdata/value/2":
			p.expand__value_2(ctx, *_res); return
		case "testdata/value/3":
			p.expand__value_3(ctx, *_res); return
		case "testdata/value/4":
			p.expand__value_4(ctx, *_res); return
		case "testdata/value/5":
			p.expand__value_5(ctx, *_res); return
		case "testdata/builtins/foreach":
			p.expand____foreach(ctx, *_res)
		}
	}

	var res = *_res
	if res == nil {
		if false { errostack(pc(ctx,p), 3, "%v : %v", p, ts(p)).trace() }
		return
	}
	if false && /* p.expandable(ctx) && */ equal(ctx, p, res) {
		if s := p.String(); strings.Contains(s, "$_") {
			if r := res.String(); res == p || r == s || strings.Contains(r, "$_") {
				if d := auto_find(ctx, "_"); d != nil {
					note(ctx, "%v", ts(d))
					note(ctx, "%v → %v", ts(p), ts(res))
					erro(ctx, "%v", ts(ctx)).trace()
				}
			}
		}
	}
}
func (p *compound) expand__configure_base(ctx Context, res Value) {
	if d, y := do(ctx, evoke_def{}).(*def); y {
		switch p.String() {
		case "-fautolink$(or $4)":
			if s, t := res.String(), "-fautolinkxar"; s == t {
				errostack(pc(ctx,p), 3, "%v: %v: %v", d.name, p, ts(res)).trace()
			}
		}
	} else {
		erro(pc(ctx,p), "%v → %v", p, ts(res)).trace()
	}
}
func (p *compound) expand__value_1(ctx Context, res Value) {
	switch p.String() {
	case "$(.test.foo)foobar":
		if truly(ctx, ex_delegate{}) {
			if s, t := ts(res), "{=flag {=word foobar}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		} else {
			erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
		}
	case "$(.test.foo)bar":
		if truly(ctx, ex_delegate{}) {
			if s, t := ts(res), "{=flag {=word foobar}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		} else {
			erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
		}
	case "-foobar":
		if s, t := ts(res), "{=flag {=word foobar}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case ".test.foo":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=word foo}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	default:
		erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
	}
}
func (p *compound) expand__value_2(ctx Context, res Value) {
	switch p.String() {
	case ".test.ab":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=word ab}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case ".test.ba":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=word ba}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case ".test.x":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=word x}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case ".test.0":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=decimal 0}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case ".test.1":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=decimal 1}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case ".test.2":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=decimal 2}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case ".test.s0":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=word s0}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case ".test.s1":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=word s1}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case ".test.s2":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=word s2}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case ".test.s3":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=word s3}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case ".test.t1":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=word t1}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case ".test.t2":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=word t2}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case ".test.t3":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=word t3}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case ".test.t4":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=word t4}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case ".test.t5":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=word t5}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case ".test.t6":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=word t6}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "foo_ab-$1-$2":
		if v1, v2 := auto_get(ctx, "1"), auto_get(ctx, "2"); v1 == nil || v2 == nil {
			if truly(ctx, ex_delegate{}) {
				if s, t := ts(res), "{=compound {=word foo_ab} {=flag {=compound {=null} {=flag {=null}}}}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), fmt.Sprintf("{=compound %s %s}", ts(v1), ts(v2)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else if _, y := v1.(*null); y {
			if s, t := ts(res), fmt.Sprintf("{=compound {=word foo_ab} {=flag {=flag %s}}}", ts(v2)); s != t {
				erro(pc(ctx,p), "%s: %s != %s ; %v", res, s, t, v1).trace()
			}
		} else {
			var s1 string
			if x, y := v1.(*compound); y {
				for _, v := range x.elems {
					if s1 != "" { s1 += " " }
					s1 += ts(v)
				}
			} else {
				s1 = ts(v1)
			}
			if s, t := ts(res), fmt.Sprintf("{=compound {=word foo_ab} {=flag {=compound %s {=flag %s}}}}", s1, ts(v2)); s != t {
				erro(pc(ctx,p), "%s: %s != %s ; %v %v", res, s, t, v2, v1).trace()
			}
		}
	case "foo_ba-$2-$1":
		if v1, v2 := auto_get(ctx, "1"), auto_get(ctx, "2"); v1 == nil || v2 == nil {
			if truly(ctx, ex_delegate{}) {
				if s, t := ts(res), "{=compound {=word foo_ba} {=flag {=compound {=null} {=flag {=null}}}}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), fmt.Sprintf("{=compound %s %s}", ts(v2), ts(v1)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else if _, y := v2.(*null); y {
			if s, t := ts(res), fmt.Sprintf("{=compound {=word foo_ba} {=flag {=flag %s}}}", ts(v1)); s != t {
				erro(pc(ctx,p), "%s: %s != %s ; %v", res, s, t, v1).trace()
			}
		} else {
			var s2 string
			if x, y := v2.(*compound); y {
				for _, v := range x.elems {
					if s2 != "" { s2 += " " }
					s2 += ts(v)
				}
			} else {
				s2 = ts(v2)
			}
			if s, t := ts(res), fmt.Sprintf("{=compound {=word foo_ba} {=flag {=compound %s {=flag %s}}}}", s2, ts(v1)); s != t {
				erro(pc(ctx,p), "%s: %s != %s ; %v %v", res, s, t, v2, v1).trace()
			}
		}
	case "foo_ab-a-b":
		if s, t := ts(res), "{=compound {=word foo_ab} {=flag {=compound {=word a} {=flag {=word b}}}}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "foo_ba-b-a":
		if s, t := ts(res), "{=compound {=word foo_ba} {=flag {=compound {=word b} {=flag {=word a}}}}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "foo_ab-aa-bb":
		if s, t := ts(res), "{=compound {=word foo_ab} {=flag {=compound {=word a} {=word a} {=flag {=compound {=word b} {=word b}}}}}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "foo_ba-bb-aa":
		if s, t := ts(res), "{=compound {=word foo_ab} {=flag {=compound {=word b} {=word b} {=flag {=compound {=word a} {=word a}}}}}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "foo_ab-ab-ba":
		if s, t := ts(res), "{=compound {=word foo_ab} {=flag {=compound {=word a} {=word b} {=flag {=compound {=word b} {=word a}}}}}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	// case "foo_ab--":
	// 	if s, t := ts(res), "{=compound {=word foo_ab} {=flag {=flag {=null}}}}"; s != t {
	// 		erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
	// 	}
	case "foo_ab-{}-":
		if s, t := ts(res), "{=compound {=word foo_ab} {=flag {=compound {=null} {=flag {=null}}}}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "foo_ab-{}{}-{}{}":
		if s, t := ts(res), "{=compound {=word foo_ab} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	// case "foo_ba--":
	// 	if s, t := ts(res), "{=compound {=word foo_ba} {=flag {=flag {=null}}}}"; s != t {
	// 		erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
	// 	}
	case "foo_ba-{}-":
		if s, t := ts(res), "{=compound {=word foo_ba} {=flag {=compound {=null} {=flag {=null}}}}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "foo_ba-{}{}-{}{}":
		if s, t := ts(res), "{=compound {=word foo_ba} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "a-b":
		if s, t := ts(res), "{=compound {=word a} {=flag {=word b}}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "b-a":
		if s, t := ts(res), "{=compound {=word b} {=flag {=word a}}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "aa-bb":
		if s, t := ts(res), "{=compound {=word a} {=word a} {=flag {=compound {=word b} {=word b}}}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "bb-aa":
		if s, t := ts(res), "{=compound {=word b} {=word b} {=flag {=compound {=word a} {=word a}}}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "ab-ba":
		if s, t := ts(res), "{=compound {=word a} {=word b} {=flag {=compound {=word b} {=word a}}}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "aa":
		if s, t := ts(res), "{=compound {=word a} {=word a}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "bb":
		if s, t := ts(res), "{=compound {=word b} {=word b}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "ab":
		if s, t := ts(res), "{=compound {=word a} {=word b}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "ba":
		if s, t := ts(res), "{=compound {=word b} {=word a}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "-":
		if s, t := ts(res), "{=compound {=flag {=null}}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "$1$1":
		if v := auto_get(ctx, "1"); v == nil {
			if truly(ctx, ex_delegate{}) {
				if s, t := ts(res), "{=compound {=null} {=null}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), fmt.Sprintf("{=compound %s %s}", ts(v), ts(v)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			if truly(ctx, ex_delegate{}) {
				if s, t := ts(res), fmt.Sprintf("{=compound %s %s}", ts(v), ts(v)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), "{=compound {=delegate {=auto 1}} {=delegate {=auto 1}}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		}
	case "$2$2":
		if v := auto_get(ctx, "2"); v == nil {
			if truly(ctx, ex_delegate{}) {
				if s, t := ts(res), "{=compound {=null} {=null}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), fmt.Sprintf("{=compound %s %s}", ts(v), ts(v)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			if truly(ctx, ex_delegate{}) {
				if s, t := ts(res), fmt.Sprintf("{=compound %s %s}", ts(v), ts(v)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), "{=compound {=delegate {=auto 2}} {=delegate {=auto 2}}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		}
	case "$1$2":
		if v1, v2 := auto_get(ctx, "1"), auto_get(ctx, "2"); v1 == nil || v2 == nil {
			if truly(ctx, ex_delegate{}) {
				if s, t := ts(res), "{=compound {=null} {=null}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), fmt.Sprintf("{=compound %s %s}", ts(v1), ts(v2)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			if truly(ctx, ex_delegate{}) {
				if s, t := ts(res), fmt.Sprintf("{=compound %s %s}", ts(v1), ts(v2)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), "{=compound {=delegate {=auto 1}} {=delegate {=auto 2}}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		}
	case "$2$1":
		if v1, v2 := auto_get(ctx, "1"), auto_get(ctx, "2"); v1 == nil || v2 == nil {
			if truly(ctx, ex_delegate{}) {
				if s, t := ts(res), "{=compound {=null} {=null}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), fmt.Sprintf("{=compound %s %s}", ts(v2), ts(v1)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			if truly(ctx, ex_delegate{}) {
				if s, t := ts(res), fmt.Sprintf("{=compound %s %s}", ts(v2), ts(v1)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), "{=compound {=delegate {=auto 2}} {=delegate {=auto 1}}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		}
	case "$1-$2":
		if v1, v2 := auto_get(ctx, "1"), auto_get(ctx, "2"); v1 == nil || v2 == nil {
			if truly(ctx, ex_delegate{}) {
				if s, t := ts(res), "{=compound {=null} {=flag {=null}}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), fmt.Sprintf("{=compound %s %s}", ts(v1), ts(v2)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else if _, y := v1.(*null); y {
			if s, t := ts(res), fmt.Sprintf("{=flag %s}", ts(v2)); s != t {
				erro(pc(ctx,p), "%s: %s != %s ; %v", res, s, t, v1).trace()
			}
		} else {
			var s1 string
			if x, y := v1.(*compound); y {
				for _, v := range x.elems {
					if s1 != "" { s1 += " " }
					s1 += ts(v)
				}
			} else {
				s1 = ts(v1)
			}
			if s, t := ts(res), fmt.Sprintf("{=compound %s {=flag %s}}", s1, ts(v2)); s != t {
				erro(pc(ctx,p), "%s: %s != %s ; %v %v", res, s, t, v2, v1).trace()
			}
		}
	case "$2-$1":
		if v1, v2 := auto_get(ctx, "1"), auto_get(ctx, "2"); v1 == nil || v2 == nil {
			if truly(ctx, ex_delegate{}) {
				if s, t := ts(res), "{=compound {=null} {=flag {=null}}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), fmt.Sprintf("{=compound %s %s}", ts(v2), ts(v1)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else if _, y := v2.(*null); y {
			if s, t := ts(res), fmt.Sprintf("{=flag %s}", ts(v1)); s != t {
				erro(pc(ctx,p), "%s: %s != %s ; %v", res, s, t, v1).trace()
			}
		} else {
			var s2 string
			if x, y := v2.(*compound); y {
				for _, v := range x.elems {
					if s2 != "" { s2 += " " }
					s2 += ts(v)
				}
			} else {
				s2 = ts(v2)
			}
			if s, t := ts(res), fmt.Sprintf("{=compound %s {=flag %s}}", s2, ts(v1)); s != t {
				erro(pc(ctx,p), "%s: %s != %s ; %v %v", res, s, t, v2, v1).trace()
			}
		}
	case "{}{}":
		if s, t := ts(res), "{=compound {=null} {=null}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "{}-":
		if s, t := ts(res), "{=compound {=null} {=flag {=null}}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "{}{}-{}{}":
		if s, t := ts(res), "{=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	default:
		errostack(pc(ctx,p), 6, "%v → %v : %v", p, res, ts(res)).trace()
	}
}
func (p *compound) expand__value_3(ctx Context, res Value) {
	switch p.String() {
	case ".test":
		if s, t := ts(res), "{=compound {=punct .} {=word test}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	default:
		erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
	}
}
func (p *compound) expand__value_4(ctx Context, res Value) {
	switch p.String() {
	case "c.D":
		if s, t := ts(res), "{=compound {=word c} {=punct .} {=word D}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "c++.D":
		if s, t := ts(res), "{=compound {=word c++} {=punct .} {=word D}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "c.I":
		if s, t := ts(res), "{=compound {=word c} {=punct .} {=word I}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "c++.I":
		if s, t := ts(res), "{=compound {=word c++} {=punct .} {=word I}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "D.c":
		if s, t := ts(res), "{=compound {=word D} {=punct .} {=word c}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "D.c++":
		if s, t := ts(res), "{=compound {=word D} {=punct .} {=word c++}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "I.c":
		if s, t := ts(res), "{=compound {=word I} {=punct .} {=word c}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "I.c++":
		if s, t := ts(res), "{=compound {=word I} {=punct .} {=word c++}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case ".test.D.c.0":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=word D} {=punct .} {=word c} {=punct .} {=decimal 0}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case ".test.D.c.1":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=word D} {=punct .} {=word c} {=punct .} {=decimal 1}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case ".test.D.c++.0":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=word D} {=punct .} {=word c++} {=punct .} {=decimal 0}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case ".test.D.c++.1":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=word D} {=punct .} {=word c++} {=punct .} {=decimal 1}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case ".test.I.c.0":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=word I} {=punct .} {=word c} {=punct .} {=decimal 0}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case ".test.I.c.1":
		if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=word I} {=punct .} {=word c} {=punct .} {=decimal 1}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case ".test.I.c++.0":
	case ".test.I.c++.1":
	case ".test.and.x.1":
	case ".test.and.x.2":
	case ".test.and.y.1":
	case ".test.and.y.2":
	case ".test.foreach":
	case ".test.none":
	case ".test.x.a":
	case ".test.x.b":
	case ".test.x.c":
	case ".test.x.$_":
	case ".test.x":
	case ".test.v":
	default:
		erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
	}
}
func (p *compound) expand__value_5(ctx Context, res Value) {
	switch p.String() {
	case "z-$1":
		if truly(ctx, ex_delegate{}) {
			if v := auto_get(ctx, "1"); v == nil {
				if s, t := ts(res), "{=compound {=word z} {=flag {=null}}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				t := fmt.Sprintf("{=compound {=word z} {=flag %s}}", ts(v))
				if s := ts(res); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			if s, t := ts(res), "{=compound {=word z} {=flag {=null}}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "z-a":
		if s, t := ts(res), "{=compound {=word z} {=flag {=word a}}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "z-":
		if s, t := ts(res), "{=compound {=word z} {=flag {=null}}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case ".test.x0":
	case ".test.x1":
	case ".test.y0":
	case ".test.y1":
	case ".test.z0":
	case ".test.z1":
	case ".test.0":
	case ".test.1":
	default:
		erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
	}
}
func (p *compound) expand____foreach(ctx Context, res Value) {
	switch p.String() {
	case ".test.1", ".test.2", ".test.21", ".test.22", ".test.23", ".test.3", ".test.4", ".test.5", ".test.6", ".test.61", ".test.7":
	case ".test.h", ".test.x", ".test.xx", ".test.x.a", ".test.x.b", ".test.y", ".test.y.a", ".test.y.b", ".test.z", ".test.foo", ".test.bar":
	case ".test.$_":
		if u := auto_get(ctx, "_"); u == nil {
			if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=delegate {=auto _}}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		} else {
			if s, t := ts(res), fmt.Sprintf("{=compound {=punct .} {=word test} {=punct .} %s}", ts(u)); s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		}
	case ".test.$_.$(or $4,$3)":
		if u, v3, v4 := auto_get(ctx, "_"), auto_get(ctx, "3"), auto_get(ctx, "4"); u == nil {
			if s, t := ts(res), "{=compound {=punct .} {=word test} {=punct .} {=delegate {=auto _}} {=punct .} {=delegate {=builtin or} {=list {=delegate {=auto 4}}} {=list {=delegate {=auto 3}}}}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		} else if v4 != nil {
			if s, t := ts(res), fmt.Sprintf("{=compound {=punct .} {=word test} {=punct .} %s {=punct .} %s}", ts(u), ts(v4)); s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		} else if v3 != nil {
			if s, t := ts(res), fmt.Sprintf("{=compound {=punct .} {=word test} {=punct .} %s {=punct .} %s}", ts(u), ts(v3)); s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		} else {
			erro(pc(ctx,p), "%v → %v, %v", p, res, ts(res)).trace()
		}
	case "&(.test.$_.$(or $4,$3))":
		if u, v3, v4 := auto_get(ctx, "_"), auto_get(ctx, "3"), auto_get(ctx, "4"); u != nil && v3 != nil && v4 != nil {
			erro(pc(ctx,p), "%v → %v, %v", p, res, ts(res)).trace()
		} else {
			erro(pc(ctx,p), "%v → %v, %v", p, res, ts(res)).trace()
		}
	case "$(closure .test.$_)$1{}99":
		if u, v1 := auto_get(ctx, "_"), auto_get(ctx, "1"); u != nil && v1 != nil {
			if truly(ctx, ex_closure{}) {
				if s, t := ts(res), "{=null}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else if truly(ctx, ex_delegate{}) {
				switch v1.String() {
				case "foo bar":
					if s, t := ts(v1), "{=list {=word foo} {=word bar}}"; s != t {
						erro(pc(ctx,p), "%v: %s != %s", v1, s, t).trace()
					}
					switch u.String() {
					case "foo":
						if s, t := ts(res), "{=list {=compound {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word foo}}}} {=word foo}} {=compound {=word bar} {=decimal 99}}}"; s != t {
							erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
						}
					case "bar":
						if s, t := ts(res), "{=list {=compound {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word bar}}}} {=word foo}} {=compound {=word bar} {=decimal 99}}}"; s != t {
							erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
						}
					default:
						erro(pc(ctx,p), "%v, %v → %v: %s", u, v1, res, ts(res)).trace()
					}
				default:
					erro(pc(ctx,p), "%v, %v → %v: %s", u, v1, res, ts(res)).trace()
				}
			} else {
				erro(pc(ctx,p), "%v, %v → %v: %s", u, v1, res, ts(res)).trace()
			}
		} else {
			if s, t := ts(res), "{=compound {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word xx}}}} {=word c}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "&(.test.$_)$1{}88":
		if u, v1 := auto_get(ctx, "_"), auto_get(ctx, "1"); u != nil && v1 != nil {
			if truly(ctx, ex_closure{}) {
				if s, t := ts(res), "{}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
				if res != nil {
					erro(pc(ctx,p), "%v %v → %v", u, v1, res).trace()
				}
			} else if truly(ctx, ex_delegate{}) {
				switch v1.String() {
				case "foo bar":
					if s, t := ts(v1), "{=list {=word foo} {=word bar}}"; s != t {
						erro(pc(ctx,p), "%v: %s != %s", v1, s, t).trace()
					}
					switch u.String() {
					case "foo":
						if s, t := ts(res), "{=list {=compound {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word foo}}}} {=word foo}} {=compound {=word bar} {=decimal 88}}}"; s != t {
							erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
						}
					case "bar":
						if s, t := ts(res), "{=list {=compound {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word bar}}}} {=word foo}} {=compound {=word bar} {=decimal 88}}}"; s != t {
							erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
						}
					default:
						erro(pc(ctx,p), "%v, %v → %v: %s", u, v1, res, ts(res)).trace()
					}
				default:
					erro(pc(ctx,p), "%v, %v → %v: %s", u, v1, res, ts(res)).trace()
				}
			} else {
				erro(pc(ctx,p), "%v, %v → %v: %s", u, v1, res, ts(res)).trace()
			}
		} else {
			if s, t := ts(res), "{=compound {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto _}}}} {=delegate {=def 1}} {=decimal 88}}"; s != t {
				note(pc(ctx,p), "%v", v1)
				note(pc(ctx,p), "%v", u)
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "&(.test.z)$_{}88":
		if u := auto_get(ctx, "_"); u == nil {
			erro(pc(ctx,p), "%v", res).trace()
		} else if truly(ctx, ex_closure{}) {
			if s, t := ts(res), fmt.Sprintf("{=compound {=word w} %s {=decimal 88}}", ts(u)); s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		} else {
			if s, t := ts(res), "{=compound {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=decimal 88}}}} {=word c}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "&(.test.h)$_":
		if u := auto_get(ctx, "_"); u == nil {
			erro(pc(ctx,p), "%v", res).trace()
		} else if d, _ := do(ctx, evoke_def{}).(*def); d == nil {
			erro(pc(ctx,p), "%v : %s", res, ts(res)).trace()
		} else {
			var t string
			switch d.name {
			case ".test.1":
				if truly(ctx, ex_closure{}) {
					t = fmt.Sprintf("{=flag %s}", ts(u))
				} else {
					t = fmt.Sprintf("{=compound {=closure {=compound {=punct .} {=word test} {=punct .} {=word h}}} %s}", ts(u))
				}
			case ".test.2":
				if truly(ctx, ex_closure{}) {
					t = fmt.Sprintf("{=flag %s}", ts(u))
				} else {
					t = fmt.Sprintf("{=compound {=closure {=def .test.h}} %s}", ts(u))
				}
			default:
				erro(pc(ctx,p), "%s: %v : %s", d.name, res, ts(res)).trace()
			}
			if s := ts(res); s != t {
				note(pc(ctx,p), "ineq: %s", s)
				note(pc(ctx,p), "ineq: %s", t)
				erro(pc(ctx,p), "%v : %v", u, res).trace()
			}
		}
	case "&(.test.xx)$_":
		if u := auto_get(ctx, "_"); u == nil {
			erro(pc(ctx,p), "%v", res).trace()
		} else if truly(ctx, ex_closure{}) {
			if s, t := ts(res), fmt.Sprintf("{=compound {} %s}", ts(u)); s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		} else {
			if s, t := ts(res), fmt.Sprintf("{=compound {=closure {=compound {=punct .} {=word test} {=punct .} {=word xx}}} %s}", ts(u)); s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "&(.test.xx)a":
	case "&(.test.xx)b":
	case "&(.test.xx)c":
		if truly(ctx, ex_closure{}) {
			if s, t := ts(res), "{=word c}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		} else {
			if s, t := ts(res), "{=compound {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word xx}}}} {=word c}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "x$_":
		if u := auto_get(ctx, "_"); u == nil {
			if s, t := ts(res), "{=compound {=word x} {=disjunction {=delegate {=auto _}}}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		} else {
			switch u.String() {
			case "{&(.test.h)}a":
			case "{&(.test.h)}b":
			case "{&(.test.h)}c":
				if truly(ctx, ex_closure{}) {
					if s, t := ts(res), "{=flag {=word c}}"; s != t {
						erro(pc(ctx,p), "%v: %s != %s ; %v", res, s, t, u).trace()
					}
				} else {
					if s, t := ts(res), "{=compound {=word x} {=disjunction {=closure {=def .test.h}}} {=word c}}"; s != t {
						erro(pc(ctx,p), "%v: %s != %s ; %v", res, s, t, u).trace()
					}
				}
			case "{&(.test.xx)}a":
			case "{&(.test.xx)}b":
			case "{&(.test.xx)}c":
				if truly(ctx, ex_closure{}) {
					if s, t := ts(res), "{=word c}"; s != t {
						erro(pc(ctx,p), "%v: %s != %s ; %v", res, s, t, u).trace()
					}
				} else {
					if s, t := ts(res), "{=compound {=word x} {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word xx}}}} {=word c}}"; s != t {
						erro(pc(ctx,p), "%v: %s != %s ; %v", res, s, t, u).trace()
					}
				}
			case "a", "aa", "b", "bb", "c", "cc", "q", "p", "-", "-a", "-b", "-c":
			default:
				erro(pc(ctx,p), "%v: %s → %v: %s", u, ts(u), res, ts(res)).trace()
			}
		}
	case "x{&(.test.h)}a":
	case "x{&(.test.h)}b":
	case "x{&(.test.h)}c":
		if truly(ctx, ex_closure{}) {
			if s, t := ts(res), "{=compound {=word x} {=flag {=word c}}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		} else {
			if s, t := ts(res), "{=compound {=disjunction {=closure {=def .test.h}}} {=word c}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "&(.test.h)a":
	case "&(.test.h)b":
	case "&(.test.h)c":
		if truly(ctx, ex_closure{}) {
			if s, t := ts(res), "{=flag {=word c}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		} else {
			if s, t := ts(res), "{=compound {=disjunction {=closure {=def .test.h}}} {=word c}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "{&(.test.h)}a":
	case "{&(.test.h)}b":
	case "{&(.test.h)}c":
		if d, _ := do(ctx, evoke_def{".test.2"}).(*def); d == nil {
			erro(pc(ctx,p), "%v : %s", res, ts(res)).trace()
		} else if truly(ctx, ex_closure{}) {
			if s, t := ts(res), "{=flag {=word c}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		} else {
			if s, t := ts(res), "{=compound {=disjunction {=closure {=def .test.h}}} {=word c}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "{&(.test.xx)}a":
	case "{&(.test.xx)}b":
	case "{&(.test.xx)}c":
		if truly(ctx, ex_closure{}) {
			if s, t := ts(res), "{=null}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		} else {
			if s, t := ts(res), "{=compound {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word xx}}}} {=word c}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "{&(.test.foo)}foo":
		if truly(ctx, ex_closure{}) {
			if s, t := ts(res), "{=null}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		} else {
			if s, t := ts(res), "{=compound {=disjunction {=closure {=def .test.foo}}} {=word foo}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "{&(.test.bar)}foo":
		if truly(ctx, ex_closure{}) {
			if s, t := ts(res), "{=null}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		} else {
			if s, t := ts(res), "{=compound {=disjunction {=closure {=def .test.foo}}} {=word foo}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "x{&(.test.xx)}a":
	case "x{&(.test.xx)}b":
	case "x{&(.test.xx)}c":
		if truly(ctx, ex_closure{}) {
			if s, t := ts(res), "{=null}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		} else {
			if s, t := ts(res), "{=compound {=word x} {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word xx}}}} {=word c}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "xq", "xp", "x-", "x-a", "x-b", "x-c":
		if s, t := ts(res), fmt.Sprintf("{=compound {=word x} %s}", ts(p.elems[1])); s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "barzz":
		if s, t := ts(res), "{=compound {=word bar} {=word zz}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "bar99":
		if s, t := ts(res), "{=compound {=word bar} {=decimal 99}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "xaa":
	case "xbb":
	case "xcc":
		if s, t := ts(res), "{=compound {=word x} {=word cc}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "{}a":
	case "{}b":
	case "{}c":
		if s, t := ts(res), "{=compound {=word c}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	default:
		erro(pc(ctx,p), "%v → %v %v", p, res, ts(res)).trace()
	}
}

func (p cond) expand_check(ctx Context, v Value, _res *Value) {
	switch j := _project(ctx); j.spec {
	case "testdata/value/optional":
		p.expand__value_optional(ctx, v, *_res)
	}

	var res = *_res
	if v == nil {
		if res == nil {
			if false { erro(ctx, "%v", p).trace() }
			return
		}
		if !isNull(res) {
			erro(pc(ctx,res), "%v → %v", p, ts(res)).trace()
			return
		}
	} else if !truly(ctx, ex_closure{}) {
		if !equal(ctx, v, p.Value) && p.Value.String() == v.String() {
			note(ctx, "%v → %v → %v", p.Value, v, res)
			note(ctx, "%-20v : %v", p.Value, ts(p.Value))
			note(ctx, "%-20v : %v", v,       ts(v))
			note(ctx, "%-20v : %v", res,     ts(res))
			errostack(pc(ctx,res), 3, "%v", p).trace()
		}
	}
	if is_cond(v) {
		note(ctx, "%v → %v → %v", p.Value, v, res)
		note(ctx, "%-20v : %v", p.Value, ts(p.Value))
		note(ctx, "%-20v : %v", v,       ts(v))
		note(ctx, "%-20v : %v", res,     ts(res))
		errostack(pc(ctx,res), 3, "%v", p).trace()
	}
}
func (p cond) expand__value_optional(ctx Context, v Value, res Value) {
	switch s1 := p.Value.String(); s1 {
	case "name":
		if s, t := ts(v), "{=word name}"; s != t {
			erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
		}
		if s, t := ts(res), "{=word name}"; s != t {
			erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
		}
		return
	case "{=project foo}→name":
		if truly(ctx, ex_arrow{}) {
			if s, t := ts(v), "{=self foo}"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
			if s, t := ts(res), "{=self foo}"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
		} else {
			if s, t := ts(v), "{=project foo}→name"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
			if s, t := ts(res), "{=project foo}→name?"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
		}
		return
	case "{=project foo}→name→xxxx":
		if truly(ctx, ex_arrow{}) {
			if s, t := ts(v), "{}"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
			if s, t := ts(res), "{}"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
		} else {
			if s, t := ts(v), "{=project foo}→name→xxxx"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
			if s, t := ts(res), "{=project foo}→name→xxxx?"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
		}
		return
	case "{=project foo}→name→item":
		if truly(ctx, ex_arrow{}) {
			if s, t := ts(v), "{=answer yes}"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
			if s, t := ts(res), "{=answer yes}"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
		} else {
			if s, t := ts(v), "{=project foo}→name→item"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
			if s, t := ts(res), "{=project foo}→name→item?"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
		}
		return
	case "{=project foo}→name?→item":
		if truly(ctx, ex_arrow{}) {
			if s, t := ts(v), "{=answer yes}"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
			if s, t := ts(res), "{=answer yes}"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
		} else {
			if s, t := ts(v), "{=project foo}→name→item"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
			if s, t := ts(res), "{=project foo}→name→item"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
		}
		return
	case "{=project foo}→baz":
		if truly(ctx, ex_arrow{}) {
			if s, t := ts(v), "{}"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
			if s, t := ts(res), "{}"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
		} else {
			if s, t := ts(v), "{=project foo}→baz"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
			if s, t := ts(res), "{=project foo}→baz?"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
		}
		return
	case "foo":
		if false {
			if s, t := ts(v), "{=word foo}"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
			if s, t := ts(res), "{=self foo}"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
		}
	case "foo?→name":
		if s, t := ts(v), "{=self foo}"; s != t {
			erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
		}
		if s, t := ts(res), "{=self foo}"; s != t {
			erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
		}
	case "foo?→name?→item":
		if s, t := ts(v), "{=answer yes}"; s != t {
			erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
		}
		if s, t := ts(res), "{=answer yes}"; s != t {
			erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
		}
	case "fo":
		if s, t := ts(v), "{=word fo}"; s != t {
			erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
		}
		if s, t := ts(res), "{=word fo}"; s != t {
			erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
		}
		return
	case "fo?→bar":
		switch {
		case truly(ctx, ex_def_0{}):
			erro(pc(ctx,p), "%s %s", ts(v), ts(res)).trace()
		case truly(ctx, ex_def_1{}, ex_arrow{}):
			if s, t := ts(v), "{}"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
			if s, t := ts(res), "{}"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
		case false:
			if s, t := ts(v), "fo?→bar"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
			if s, t := ts(res.String), "fo?→bar?"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
		default:
			erro(pc(ctx,p), "%s %s", ts(v), ts(res)).trace()
		}
		return
	case "{=self foo}":
		if s, t := ts(v), "{=self foo}"; s != t {
			erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
		}
		if s, t := ts(res), "{=self foo}?"; s != t {
			erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
		}
		return
	case "$_→name":
		switch {
		case truly(ctx, ex_def_0{}):
			erro(pc(ctx,p), "%s %s", ts(v), ts(res)).debug()
		case truly(ctx, ex_def_1{}):
			if s, t := ts(v), "{=self foo}"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
			if s, t := ts(res), "{=self foo}"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
		default:
			if s, t := ts(v), "{=arrow {=delegate {=auto _}}→{=word name}}"; s != t {
				erro(pc(ctx,p), "%s != %s : %s", s, t, ts(res)).trace()
			}
			if s, t := ts(res), "{=arrow {=delegate {=auto _}}→{=word name}}"; s != t {
				erro(pc(ctx,p), "%s != %s", s, t).trace()
			}
		}
		return
	case "$_→bar":
		switch {
		case truly(ctx, ex_def_0{}):
			erro(pc(ctx,p), "%s %s", ts(v), ts(res)).debug()
		case truly(ctx, ex_def_1{}):
			if s, t := ts(v), "{}"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
			if s, t := ts(res), "{}"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
		default:
			if s, t := ts(v), "{=arrow {=delegate {=auto _}}→{=word bar}}"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
			if s, t := ts(res), "{=arrow {=delegate {=auto _}}→{=word bar}}"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, ts(v), ts(res)).trace()
			}
		}
		return
	default:
		errostack(pc(ctx,p.Value), 1, "%v %v %v:%v", s1, tv(v), tv(res), ts(res)).debug(10)
	}

	if s1 := ts(p); s1 == fmt.Sprintf("{=cond {=word %s}}", v) {
		if s2 := ts(v); s2 != fmt.Sprintf("{=word %s}", v) {
			erro(pc(ctx,p), "%s != %s ; %s", s1, s2, ts(res)).trace()
		}
		if s2 := ts(res); false && s1 != s2 {
			erro(pc(ctx,p), "%s != %s ; %s", s1, s2, ts(res)).trace()
		}
	}

	if truly(ctx, ex_arrow{}) {
		if s, t1 := ts(p.Value), "{=arrow {=project foo}→{=word name}}"; s == t1 && false {
			if s, t2 := ts(res), "{=def name}"; s != t2 {
				erro(pc(ctx,p), "%v: %v → %v", p, t1, t2).trace()
			} else if d, y := res.(*def); !y {
				erro(pc(ctx,p), "%v: %v → %v", p, t1, t2).trace()
			} else if s, t2 := ts(d.value), "{=self foo}"; s != t2 {
				erro(pc(ctx,p), "%v: %v → %v", p, t1, t2).trace()
			}
		}
	} else {
		if s, t1 := ts(p.Value), "{=arrow {=project foo}→{=word name}}"; s == t1 && false {
			if s, t2 := ts(res), "{=def name}"; s != t2 {
				erro(pc(ctx,p), "%v: %v → %v", p, t1, t2).trace()
			} else if d, y := res.(*def); !y {
				erro(pc(ctx,p), "%v: %v → %v", p, t1, t2).trace()
			} else if s, t2 := ts(d.value), "{=self foo}"; s != t2 {
				erro(pc(ctx,p), "%v: %v → %v", p, t1, t2).trace()
			}
		} else {
			errostack(pc(ctx,p.Value), 1, "%v %v %v", p, v, res).debug(10)
		}
	}
}

func (p *path) cmp_check(ctx Context, v Value, _res *cmpres) {
    if res := *_res; res != cmpEqual && p.String() == v.String() {
        note(ctx, "%v, %v, %v", p, p==v, res)
        note(ctx, "%v", ts(p))
        note(ctx, "%v", ts(v))
        errostack(ctx, 3, "%v", ts(ctx)).trace()
    }
}
func (p *path) match2_check(ctx Context, srcs []string, full *bool, res, stems *[]string) {
	var s, t = joinpath(srcs...), p.string(ctx)

	if strings.HasPrefix(s, t) && *res == nil {
		note(ctx, "%v →", p)
		note(ctx, "%v →", s)
		note(ctx, "→ %v %v %v", full, res, stems)
		errostack(ctx, 3, "%v", ts(ctx)).trace()
	}

	switch t {
	case "%%/.smart/modules/":
		if strings.Contains(s, "/.smart/modules/") {
			if a := *res; len(a) < 4 || a[len(a)-1] != "" {
				errostack(ctx, 3, "%v : %v %v %v", s, *full, *res, *stems).trace()
			}
		}
	}
	return
}

func (p *strcomp) cmp_check(ctx Context, v Value, _res *cmpres) {
    if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(ctx, 3, "%v, %v ⇔ %v, %v ⇔ %v", res, p, v, ts(p), ts(v)).trace()
	}
}

func (p *punct) cmp_check(ctx Context, v Value, _res *cmpres) {
    if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(ctx, 3, "%v, %v ⇔ %v, %v ⇔ %v", res, p, v, ts(p), ts(v)).trace()
    }
}

func (p fullname) cmp_check(ctx Context, v Value, _res *cmpres) {
    if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(ctx, 3, "%v, %v ⇔ %v, %v ⇔ %v", res, p, v, ts(p), ts(v)).trace()
    }
}

func (p *list) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual {
		if s, t := ts(p), ts(v) ; s == t {
			errostack(ctx, 3, "%v, %v ⇔ %v", res, s, t).trace()
		} else if false && p.String() == v.String() {
			errostack(ctx, 3, "%v, %v ⇔ %v", res, s, t).trace()
		}
	}
	return
}
func (p *list) expand_check(ctx Context, d bool, _res *Value) {
	var res = *_res

	if j := _project(ctx); j != nil {
		switch j.spec {
		case "testdata/value":
			p.expand__value(ctx, d, res); return
		case "testdata/value/1":
			p.expand__value_1(ctx, d, res); return
		case "testdata/value/2":
			p.expand__value_2(ctx, d, res); return
		case "testdata/value/3":
			p.expand__value_3(ctx, d, res); return
		case "testdata/value/4":
			p.expand__value_4(ctx, d, res); return
		case "testdata/value/5":
			p.expand__value_5(ctx, d, res); return
		case "testdata/value/6":
			p.expand__value_6(ctx, d, res); return
		case "testdata/value/7":
			p.expand__value_7(ctx, d, res); return
		case "testdata/value/8":
			p.expand__value_8(ctx, d, res); return
		case "testdata/value/9":
			p.expand__value_9(ctx, d, res); return
		case "testdata/value/10":
			p.expand__value_10(ctx, d, res); return
		case "testdata/value/11":
			p.expand__value_11(ctx, d, res); return
		case "testdata/value/12":
			p.expand__value_12(ctx, d, res); return
		case "testdata/value/13":
			p.expand__value_13(ctx, d, res); return
		case "testdata/value/auto":
			p.expand__value_auto(ctx, d, res)
		case "testdata/value/optional":
			p.expand__value_optional(ctx, d, res)
		case "testdata/value/placeholder":
			p.expand__value_placeholder(ctx, d, res)
		case "testdata/builtins/addprefix":
			p.expand___addprefix(ctx, d, *_res)
		}
	}

	return

	if t := res.(*list); d {
		if s, t := ts(p), ts(res); s == t {
			note(pc(ctx,p), "%s", s)
			note(pc(ctx,p), "%s", t)
			errostack(pc(ctx,p), 3, "%s == %s", s, t).trace()
		}
		if s, t := ts(p.elems), ts(t.elems); s == t {
			errostack(pc(ctx,p), 3, "%s == %s", s, t).trace()
		}
		if p == res || p.String() == res.String() {
			errostack(pc(ctx,p), 3, "%v == %v ; %v", p, res, p==res).trace()
		}
	} else {
		if s, t := ts(p), ts(res); s != t {
			note(pc(ctx,p), "%s", s)
			note(pc(ctx,p), "%s", t)
			errostack(pc(ctx,p), 3, "%s != %s", p, res).trace()
		}
		if s, t := ts(p.elems), ts(t.elems); s != t {
			errostack(pc(ctx,p), 3, "%s != %s", s, t).trace()
		}
		if p != res || p.String() != res.String() {
			errostack(pc(ctx,p), 3, "%v != %v ; %v", p, res, p!=res).trace()
		}
	}
	return
}
func (p *list) expand__value(ctx Context, d bool, res Value) {
	switch p.String() {
	case "a b c":
		if s, t := ts(res), "{=list {=word a} {=word b} {=word c}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "1 2 3":
		if s, t := ts(res), "{=list {=decimal 1} {=decimal 2} {=decimal 3}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case `a\,b\,c`:
		if s, t := ts(res), `{=list {=compound {=word a} {=escaped \,} {=word b} {=escaped \,} {=word c}}}`; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case `x\,y\,z`:
		if s, t := ts(res), `{=list {=compound {=word x} {=escaped \,} {=word y} {=escaped \,} {=word z}}}`; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case `{=defscapture configure.types.<atomic.h> {0:configure.types.<atomic.h>} {1:<atomic.h>} {2:atomic.h} {3:}} {=defscapture configure.types."atomic.h" {0:configure.types."atomic.h"} {1:"atomic.h"} {2:} {3:atomic.h}}`:
		if s, t := ts(res), `{=list {=defscapture {=word configure.types.<atomic.h>} {0:{=word configure.types.<atomic.h>}} {1:{=word <atomic.h>}} {2:{=word atomic.h}} {3:{=word}}} {=defscapture {=word configure.types."atomic.h"} {0:{=word configure.types."atomic.h"}} {1:{=word "atomic.h"}} {2:{=word}} {3:{=word atomic.h}}}}`; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case `{=defscapture configure.types."atomic.h" {0:configure.types."atomic.h"} {1:"atomic.h"} {2:} {3:atomic.h}} {=defscapture configure.types.<atomic.h> {0:configure.types.<atomic.h>} {1:<atomic.h>} {2:atomic.h} {3:}}`:
		if s, t := ts(res), `{=list {=defscapture {=word configure.types."atomic.h"} {0:{=word configure.types."atomic.h"}} {1:{=word "atomic.h"}} {2:{=word}} {3:{=word atomic.h}}} {=defscapture {=word configure.types.<atomic.h>} {0:{=word configure.types.<atomic.h>}} {1:{=word <atomic.h>}} {2:{=word atomic.h}} {3:{=word}}}}`; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case `{=regex ^.+?\.o$$}`:
		if s, t := ts(res), `{=list {=regex ^.+?\.o$$}}`; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case `{=regex ^(.+?)((-(?P<i>.+?))*)(\.)(?P<x>o)$$}`:
		if s, t := ts(res), `{=list {=regex ^(.+?)((-(?P<i>.+?))*)(\.)(?P<x>o)$$}}`; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case `$0`:
		if truly(ctx, ex_delegate{}) {
			if v := auto_get(ctx, "0"); v == nil {
				if s, t := ts(res), ``; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), fmt.Sprintf(`{=list %s}`, ts(v)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			erro(pc(ctx,p), "%s: %s", res, ts(res)).trace()
		}
	case `$0 $1 $2 $3 $(i) $5 $(x)`:
		if truly(ctx, ex_delegate{}) {
			v0 := auto_get(ctx, "0")
			v1 := auto_get(ctx, "1")
			v2 := auto_get(ctx, "2")
			v3 := auto_get(ctx, "3")
			vi := auto_get(ctx, "i")
			v5 := auto_get(ctx, "5")
			vx := auto_get(ctx, "x")
			if v0 == nil || v1 == nil || v2 == nil || v3 == nil || vi == nil || v5 == nil || vx == nil {
				if s, t := ts(res), ``; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), fmt.Sprintf(`{=list %s %s %s %s %s %s %s}`, ts(v0), ts(v1), ts(v2), ts(v3), ts(vi), ts(v5), ts(vx)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			erro(pc(ctx,p), "%s: %s", res, ts(res)).trace()
		}
	case "$//test.txt":
		if truly(ctx, ex_delegate{}) {
			t := fmt.Sprintf("{=list {=path {=punct root} {=word Volumes} {=word workspace} {=word go} {=word src} {=word extbit.io} {=word smart} {=word testdata} {=word value} {=compound {=word test} {=punct .} {=word txt}}}}")
			if s := ts(res); s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		} else {
			erro(pc(ctx,p), "%s: %s", res, ts(res)).trace()
		}
	default:
		erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
	}
}
func (p *list) expand__value_1(ctx Context, d bool, res Value) {
	switch p.String() {
	case "$(.test.foo)foobar":
		if truly(ctx, ex_delegate{}) {
			if s, t := ts(res), "{=list {=flag {=word foobar}}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		} else {
			erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
		}
	case "$(.test.foo)bar":
		if truly(ctx, ex_delegate{}) {
			if s, t := ts(res), "{=list {=flag {=word foobar}}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		} else {
			erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
		}
	case "-foobar":
		if s, t := ts(res), "{=list {=flag {=word foobar}}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	default:
		erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
	}
}
func (p *list) expand__value_2(ctx Context, d bool, res Value) {
	var j = _project(ctx)
	var test_x Value
	if d := j.def(ctx, ".test.x"); d != nil { test_x = d.value }
	switch p.String() {
	case "&(.test.x)":
		if truly(ctx, ex_def_2{}) {
			switch s := ts(res); s {
			case "{=list {=compound {=punct .} {=word test} {=punct .} {=word ab}}}":
			case "{=list {=compound {=punct .} {=word test} {=punct .} {=word ba}}}":
			case "{=list}":
			default:
				erro(pc(ctx,p), "%s: %s", res, s).trace()
			}
			switch {
			case nil == cast[*__closure](ctx):
			case nil == cast[*__call](ctx):
			}
		} else {
			switch s := ts(res); s {
			case "{=list {=closure {=compound {=punct .} {=word test} {=punct .} {=word x}}}}":
			case "{=list {=closure {=def .test.x}}}":
			default:
				erro(pc(ctx,p), "%s: %s", res, s).trace()
			}
		}
	case "1 &(.test.ba) 10 2 foo_ba-{}{}-{}{} 20":
		if truly(ctx, defExpand2) {
			switch s := ts(res); s {
			case "{=list {=list {=decimal 1} {=compound {=word foo_ba} {=flag {=compound {=null} {=flag {=null}}}}} {=decimal 10} {=decimal 2} {=compound {=word foo_ba} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 20}}}":
			case        "{=list {=decimal 1} {=compound {=word foo_ba} {=flag {=compound {=null} {=flag {=null}}}}} {=decimal 10} {=decimal 2} {=compound {=word foo_ba} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 20}}":
			default:
				erro(pc(ctx,p), "%v: %s", res, s).trace()
			}
		} else if truly(ctx, defExpand1) {
			switch s := ts(res); s {
			case "{=list {=list {=decimal 1} {=closure {=def .test.ba}} {=decimal 10} {=decimal 2} {=compound {=word foo_ba} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 20}}}":
			case        "{=list {=decimal 1} {=closure {=def .test.ba}} {=decimal 10} {=decimal 2} {=compound {=word foo_ba} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 20}}":
			default:
				erro(pc(ctx,p), "%v: %s", res, s).trace()
			}
		} else {
			switch s := ts(res); s {
			default:
				erro(pc(ctx,p), "%v: %s", res, s).trace()
			}
		}
	case "1 &(.test.ab) 10 2 foo_ab-{}{}-{}{} 20 3 foo_ab-{}{}-{}{} foo_ab-{}{}-{}{} 4":
		if truly(ctx, defExpand2) {
			switch s := ts(res); s {
			default:
				erro(pc(ctx,p), "%v: %s", res, s).trace()
			}
		} else if truly(ctx, defExpand1) {
			switch s := ts(res); s {
			case "{=list {=decimal 1} {=closure {=def .test.ab}} {=decimal 10} {=decimal 2} {=compound {=word foo_ab} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 20} {=decimal 3} {=compound {=word foo_ab} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=compound {=word foo_ab} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 4}}":
			default:
				erro(pc(ctx,p), "%v: %s", res, s).trace()
			}
		} else {
			switch s := ts(res); s {
			default:
				erro(pc(ctx,p), "%v: %s", res, s).trace()
			}
		}
	case "1 &(.test.ab) 10 2 foo_ab-{}{}-{}{} 20 3 foo_ab-{}{}-{}{} foo_ab-{}{}-{}{} 4 .":
		if truly(ctx, defExpand2) {
			switch s := ts(res); s {
			default:
				erro(pc(ctx,p), "%v: %s", res, s).trace()
			}
		} else if truly(ctx, defExpand1) {
			switch s := ts(res); s {
			case "{=list {=list {=decimal 1} {=closure {=def .test.ab}} {=decimal 10} {=decimal 2} {=compound {=word foo_ab} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 20} {=decimal 3} {=compound {=word foo_ab} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=compound {=word foo_ab} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 4}} {=punct .}}":
			default:
				erro(pc(ctx,p), "%v: %s", res, s).trace()
			}
		} else {
			switch s := ts(res); s {
			default:
				erro(pc(ctx,p), "%v: %s", res, s).trace()
			}
		}
	case "1 &(&(.test.x)) 10 2 20":
		if truly(ctx, defExpand2) {
			switch s := ts(res); s {
			case "{=list {=list {=decimal 1} {=compound {=word foo_ab} {=flag {=compound {=null} {=flag {=null}}}}} {=decimal 10} {=decimal 2} {=decimal 20}}}":
			case        "{=list {=decimal 1} {=compound {=word foo_ab} {=flag {=compound {=null} {=flag {=null}}}}} {=decimal 10} {=decimal 2} {=decimal 20}}":
			default:
				erro(pc(ctx,p), "%v: %s", res, s).trace()
			}
		} else if truly(ctx, defExpand1) {
			switch s := ts(res); s {
			case "{=list {=list {=decimal 1} {=closure {=compound {=punct .} {=word test} {=punct .} {=word ab}}} {=decimal 10} {=decimal 2} {=decimal 20}}}":
			case        "{=list {=decimal 1} {=closure {=compound {=punct .} {=word test} {=punct .} {=word ab}}} {=decimal 10} {=decimal 2} {=decimal 20}}":
			default:
				erro(pc(ctx,p), "%v: %s", res, s).trace()
			}
		} else {
			switch s := ts(res); s {
			default:
				erro(pc(ctx,p), "%v: %s", res, s).trace()
			}
		}
		if x, y := do(ctx, evoke_def{".test.t1"}).(*def); y {
			if s, t := ts(x.value), "{=list {=list {=decimal 1} {=closure {=closure {=compound {=punct .} {=word test} {=punct .} {=word x}}}} {=decimal 10} {=decimal 2} {=decimal 20}}}"; s != t {
				erro(pc(ctx,p), "%s: %s != %s", x.value, s, t).trace()
			}
		} else {
			erro(pc(ctx,p), "%v → %v : %v ; %v", p, res, ts(res), do(ctx, evoke_def{})).trace()
		}
	case "1 $(closure &(.test.x)) 11":
		if truly(ctx, defExpand2) {
			switch s := ts(res); s {
			default:
				erro(pc(ctx,p), "%v: %s", res, s).trace()
			}
		} else if truly(ctx, defExpand1) {
			switch s := ts(res); s {
			default:
				erro(pc(ctx,p), "%v: %s", res, s).trace()
			}
		} else {
			switch s := ts(res); s {
			default:
				erro(pc(ctx,p), "%v: %s", res, s).trace()
			}
		}
	case "1 &(&(.test.x)) 11":
		if truly(ctx, defExpand2) {
			switch s := ts(res); s {
			default:
				erro(pc(ctx,p), "%v: %s", res, s).trace()
			}
		} else if truly(ctx, defExpand1) {
			switch s := ts(res); s {
			default:
				erro(pc(ctx,p), "%v: %s", res, s).trace()
			}
		} else {
			switch s := ts(res); s {
			default:
				erro(pc(ctx,p), "%v: %s", res, s).trace()
			}
		}
	case "1 &(&(.test.x)) 11 2 21": // .test.s0 ::=
		if x, y := do(ctx, evoke_def{".test.1"}).(*def); y {
			if s, t := ts(x.value), "{=list {=decimal 1} {=closure {=closure {=compound {=punct .} {=word test} {=punct .} {=word x}}}} {=decimal 11} {=decimal 2} {=decimal 21}}"; s != t {
				erro(pc(ctx,p), "%s: %s != %s", x.value, s, t).trace()
			}
			if truly(ctx, defExpand2) {
				if s, t := ts(res), "{=list {=decimal 1} {=decimal 11} {=decimal 2} {=decimal 21}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				switch s := ts(res); s {
				case "{=list {=list {=decimal 1} {=closure {=compound {=punct .} {=word test} {=punct .} {=word ab}}} {=decimal 11} {=decimal 2} {=decimal 21}}}":
				default:
					erro(pc(ctx,p), "%v: %s", res, s).trace()
				}
			}
		} else {
			erro(pc(ctx,p), "%v → %v : %v ; %v", p, res, ts(res), do(ctx, evoke_def{})).trace()
		}
	case "1 &(&(.test.x)) 11 2 21 3 foo_ba-{}{}-{}{} foo_ba-{}{}-{}{} 4":
		switch {
		case truly(ctx, ex_closure{}):
			if s, t := ts(res), "{=list {=decimal 1} {=closure {=compound {=punct .} {=word test} {=punct .} {=word ab}}} {=decimal 11} {=decimal 2} {=decimal 21} {=decimal 3} {=compound {=word foo_ba} {=flag {=flag {=null}}}} {=compound {=word foo_ba} {=flag {=flag {=null}}}} {=decimal 4}}"; s != t {
				errostack(pc(ctx,p), 6, "%v: %s != %s", res, s, t).trace()
			}
		case truly(ctx, ex_delegate{}):
			if s, t := ts(res), "{=list {=decimal 1} {=closure {=compound {=punct .} {=word test} {=punct .} {=word ab}}} {=decimal 11} {=decimal 2} {=decimal 21} {=decimal 3} {=compound {=word foo_ba} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=compound {=word foo_ba} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 4}}"; s != t {
				errostack(pc(ctx,p), 6, "%v: %s != %s", res, s, t).trace()
			}
		default:
			if s, t := ts(res), "{=list {=decimal 1} {=closure {=def .test.ab}} {=decimal 10} {=decimal 2} {=compound {=word foo_ab} {=flag {=flag {=null}}}} {=decimal 20} {=decimal 3} {=compound {=word foo_ab} {=flag {=flag {=null}}}} {=compound {=word foo_ab} {=flag {=flag {=null}}}} {=decimal 4}}"; s != t {
				errostack(pc(ctx,p), 6, "%v: %s != %s", res, s, t).trace()
			}
		}
	case "1 &(&(.test.x)) 12 2 22":
		if truly(ctx, ex_closure{}) {
			if s, t := ts(res), "{=list {=decimal 1} {=closure {=compound {=punct .} {=word test} {=punct .} {=word ab}}} {=decimal 11} {=decimal 2} {=decimal 21}}"; s != t {
				errostack(pc(ctx,p), 6, "%v: %s != %s", res, s, t).trace()
			}
		} else if truly(ctx, ex_delegate{}) {
			if s, t := ts(res), "{=list {=decimal 1} {=closure {=compound {=punct .} {=word test} {=punct .} {=word ab}}} {=decimal 11} {=decimal 2} {=decimal 21}}"; s != t {
				errostack(pc(ctx,p), 6, "%v: %s != %s", res, s, t).trace()
			}
		} else {
			if s, t := ts(res), "{=list {=decimal 1} {=closure {=def .test.ab}} {=decimal 10} {=decimal 2} {=compound {=word foo_ab} {=flag {=flag {=null}}}} {=decimal 20}}"; s != t {
				errostack(pc(ctx,p), 6, "%v: %s != %s", res, s, t).trace()
			}
		}
	case "1 $(closure &(.test.x)) 10 2 $(&(.test.x) $1$1,$2$2) 20 3 $(call &(.test.x),$1$2,$2$1) $(call(-closure) &(.test.x),$1$2,$2$1) 4 $3":
		if _, y := do(ctx, evoke_def{".test.0"}).(*def); y {
			if v1, v2 := auto_get(ctx, "1"), auto_get(ctx, "2"); v1 == nil || v2 == nil {
				if truly(ctx, ex_delegate{}) {
					if s, t := ts(res), "{=list {=decimal 1} {=closure {=def .test.ab}} {=decimal 10} {=decimal 2} {=compound {=word foo_ab} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 20} {=decimal 3} {=compound {=word foo_ab} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=compound {=word foo_ab} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 4}}"; s != t {
						errostack(pc(ctx,p), 6, "%v: %s != %s", res, s, t).trace()
					}
				} else {
					if s, t := ts(res), "{=list {=decimal 1} {=closure {=def .test.ab}} {=decimal 10} {=decimal 2} {=compound {=word foo_ab} {=flag {=flag {=null}}}} {=decimal 20} {=decimal 3} {=compound {=word foo_ab} {=flag {=flag {=null}}}} {=compound {=word foo_ab} {=flag {=flag {=null}}}} {=decimal 4}}"; s != t {
						errostack(pc(ctx,p), 6, "%v: %s != %s", res, s, t).trace()
					}
				}
			} else {
				if truly(ctx, ex_delegate{}) {
					switch fmt.Sprintf("%s %s", v1, v2) {
					case "a b":
						if s, t := ts(res), "{=list {=decimal 1} {=closure {=def .test.ab}} {=decimal 10} {=decimal 2} {=compound {=word foo_ab} {=flag {=compound {=word a} {=word a} {=flag {=compound {=word b} {=word b}}}}}} {=decimal 20} {=decimal 3} {=compound {=word foo_ab} {=flag {=compound {=word a} {=word b} {=flag {=compound {=word b} {=word a}}}}}} {=compound {=word foo_ab} {=flag {=compound {=word a} {=word b} {=flag {=compound {=word b} {=word a}}}}}} {=decimal 4} {=word c}}"; s != t {
							erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
						}
					default:
						erro(pc(ctx,p), "%s %s : %v", v1, v2, res).trace()
					}
				} else {
					if s, t := ts(res), "{=list {=compound {=delegate {=auto 1}} {=delegate {=auto 2}}}}"; s != t {
						erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
					}
				}
			}
		} else {
			errostack(pc(ctx,p), 6, "%v → %v : %v", p, res, ts(res)).trace()
		}
	case "1 11 2 21":
		if s, t := ts(res), "{=list {=decimal 1} {=decimal 11} {=decimal 2} {=decimal 21}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
		if x, y := do(ctx, evoke_def{".test.s0"}).(*def); y {
			if s, t := ts(x.value), "{=list {=list {=decimal 1} {=decimal 11} {=decimal 2} {=decimal 21}} {=word s0}}"; s != t {
				erro(pc(ctx,p), "%s: %s != %s", x.value, s, t).trace()
			}
			// if cast[*__finalize](ctx) == nil && !truly(ctx, is_braced{}) {
			// 	erro(pc(ctx,p), "%s ; %s", x.value, res).trace()
			// } else if !(x.o == defExpand2 || truly(ctx, defExpand2)) {
			// 	erro(pc(ctx,p), "%v → %v : %v ; %v", p, res, ts(res), do(ctx, get_origin{})).trace()
			// }
		} else if false {
			erro(pc(ctx,p), "%v → %v : %v ; %v", p, res, ts(res), do(ctx, evoke_def{})).trace()
		}
	case "1 11 2 21 s0":
		switch s := ts(res); s {
		case "{=list {=list {=decimal 1} {=decimal 11} {=decimal 2} {=decimal 21}} {=word s0}}":
		case "{=list {=decimal 1} {=decimal 11} {=decimal 2} {=decimal 21} {=word s0}}":
		default:
			erro(pc(ctx,p), "%v: %s", res, s).trace()
		}
		if x, y := do(ctx, evoke_def{".test.s0"}).(*def); y {
			if s, t := ts(x.value), "{=list {=list {=decimal 1} {=decimal 11} {=decimal 2} {=decimal 21}} {=word s0}}"; s != t {
				erro(pc(ctx,p), "%s: %s != %s", x.value, s, t).trace()
			}
		} else if false {
			erro(pc(ctx,p), "%v → %v : %v ; %v", p, res, ts(res), do(ctx, evoke_def{})).trace()
		}
	case "1 11 2 21 s0 s1":
		switch s := ts(res); s {
		case "{=list {=list {=decimal 1} {=decimal 11} {=decimal 2} {=decimal 21} {=word s0}} {=word s1}}":
		default:
			erro(pc(ctx,p), "%v: %s", res, s).trace()
		}
		if x, y := do(ctx, evoke_def{".test.s0"}).(*def); y {
			if s, t := ts(x.value), "{=list {=list {=decimal 1} {=decimal 11} {=decimal 2} {=decimal 21}} {=word s0}}"; s != t {
				erro(pc(ctx,p), "%s: %s != %s", x.value, s, t).trace()
			}
		} else if false {
			erro(pc(ctx,p), "%v → %v : %v ; %v", p, res, ts(res), do(ctx, evoke_def{})).trace()
		}
	case "1 12 2 22 3 foo_ba-{}{}-{}{} foo_ba-{}{}-{}{} 4":
		switch s := ts(res); s {
		case "{=list {=decimal 1} {=decimal 12} {=decimal 2} {=decimal 22} {=decimal 3} {=compound {=word foo_ba} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=compound {=word foo_ba} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 4}}":
		default:
			erro(pc(ctx,p), "%v: %s", res, s).trace()
		}
	case "1 $(closure &(.test.x)) 10 2 $(&(.test.x) $1$1,$2$2) 20":
		if x, y := do(ctx, evoke_def{".test.0"}).(*def); y {
			if s, t := ts(x.value), "{=list {=decimal 1} {=delegate {=builtin closure} {=list {=closure {=compound {=punct .} {=word test} {=punct .} {=word x}}}}} {=decimal 10} {=decimal 2} {=delegate {=closure {=compound {=punct .} {=word test} {=punct .} {=word x}}} {=list {=compound {=delegate {=auto 1}} {=delegate {=auto 1}}}} {=list {=compound {=delegate {=auto 2}} {=delegate {=auto 2}}}}} {=decimal 20}}"; s != t {
				erro(pc(ctx,p), "%s: %s != %s", x.value, s, t).trace()
			}
			if truly(ctx, ex_closure{}) {
				switch s := ts(res); s {
				// case "{=list {=decimal 1} {=closure {=def .test.ba}} {=decimal 10} {=decimal 2} {=compound {=word foo_ba} {=flag {=flag {=null}}}} {=decimal 20}}":
				// case "{=list {=decimal 1} {=compound {=word foo_ba} {=flag {=flag {=null}}}} {=decimal 10} {=decimal 2} {=compound {=word foo_ba} {=flag {=flag {=null}}}} {=decimal 20}}":
				case "{=list {=decimal 1} {=closure {=def .test.ba}} {=decimal 10} {=decimal 2} {=compound {=word foo_ba} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 20}}":
				default:
					erro(pc(ctx,p), "%s: %s; %v", res, s, test_x).trace()
				}
			} else if truly(ctx, ex_delegate{}) {
				switch s := ts(res); s {
				// case "{=list {=decimal 1} {=closure {=def .test.ba}} {=decimal 10} {=decimal 2} {=compound {=word foo_ba} {=flag {=flag {=null}}}} {=decimal 20}}":
				// case "{=list {=decimal 1} {=closure {=closure {=compound {=punct .} {=word test} {=punct .} {=word x}}}} {=decimal 10} {=decimal 2} {=decimal 20}}":
				case "{=list {=decimal 1} {=closure {=def .test.ba}} {=decimal 10} {=decimal 2} {=compound {=word foo_ba} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 20}}":
				case "{=list {=decimal 1} {=closure {=closure {=compound {=punct .} {=word test} {=punct .} {=word x}}}} {=decimal 10} {=decimal 2} {=decimal 20}}":
				default:
					erro(pc(ctx,p), "%s: %s; %v", res, s, test_x).trace()
				}
			} else {
				switch s := ts(res); s {
				case "{=list {=decimal 1} {=closure {=closure {=compound {=punct .} {=word test} {=punct .} {=word x}}}} {=decimal 10} {=decimal 2} {=decimal 20}}":
				default:
					erro(pc(ctx,p), "%s: %s; %v", res, s, test_x).trace()
				}
			}
		} else {
			erro(pc(ctx,p), "%v → %v : %v ; %v", p, res, ts(res), do(ctx, evoke_def{})).trace()
		}
	case "$(.test.s0)":// → 1 11 2 21 s0
		if cast[*__finalize](ctx) == nil {
			erro(pc(ctx,p), "%s", res).trace()
		} else if truly(ctx, defExpand2) {
			if s, t := ts(res), "{=list {=list {=list {=decimal 1} {=decimal 11} {=decimal 2} {=decimal 21}} {=word s0}}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		} else {
			erro(pc(ctx,p), "%v → %v : %v ; %v", p, res, ts(res), do(ctx, get_origin{})).trace()
		}
	case "$(.test.0) . $3":
		if truly(ctx, defExpand2) {
			switch s := ts(res); s {
			case "{=list {=list {=decimal 1} {=closure {=def .test.ab}} {=decimal 10} {=decimal 2} {=compound {=word foo_ab} {=flag {=flag {=null}}}} {=decimal 20} {=decimal 3} {=compound {=word foo_ab} {=flag {=flag {=null}}}} {=compound {=word foo_ab} {=flag {=flag {=null}}}} {=decimal 4}} {=punct .} {=word x}}":
			default:
				erro(pc(ctx,p), "%v: %s", res, s).trace()
			}
		} else if truly(ctx, defExpand1) {
			switch s := ts(res); s {
			case "{=list {=list {=decimal 1} {=closure {=def .test.ab}} {=decimal 10} {=decimal 2} {=compound {=word foo_ab} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 20} {=decimal 3} {=compound {=word foo_ab} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=compound {=word foo_ab} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 4}} {=punct .} {=word x}}":
			default:
				erro(pc(ctx,p), "%v: %s", res, s).trace()
			}
		} else {
			switch s := ts(res); s {
			default:
				erro(pc(ctx,p), "%v: %s", res, s).trace()
			}
		}
	case "$1$1":
		if v := auto_get(ctx, "1"); v == nil {
			if truly(ctx, ex_delegate{}) {
				if s, t := ts(res), "{=list {=compound {=null} {=null}}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), fmt.Sprintf("{=compound %s %s}", ts(v), ts(v)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			if truly(ctx, ex_delegate{}) {
				if s, t := ts(res), fmt.Sprintf("{=list {=compound %s %s}}", ts(v), ts(v)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), "{=list {=compound {=delegate {=auto 1}} {=delegate {=auto 1}}}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		}
	case "$2$2":
		if v := auto_get(ctx, "2"); v == nil {
			if truly(ctx, ex_delegate{}) {
				if s, t := ts(res), "{=list {=compound {=null} {=null}}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), fmt.Sprintf("{=compound %s %s}", ts(v), ts(v)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			if truly(ctx, ex_delegate{}) {
				if s, t := ts(res), fmt.Sprintf("{=list {=compound %s %s}}", ts(v), ts(v)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), "{=list {=compound {=delegate {=auto 2}} {=delegate {=auto 2}}}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		}
	case "$1$2":
		if v1, v2 := auto_get(ctx, "1"), auto_get(ctx, "2"); v1 == nil || v2 == nil {
			if truly(ctx, ex_delegate{}) {
				if s, t := ts(res), "{=list {=compound {=null} {=null}}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), fmt.Sprintf("{=compound %s %s}", ts(v1), ts(v2)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			if truly(ctx, ex_delegate{}) {
				if s, t := ts(res), fmt.Sprintf("{=list {=compound %s %s}}", ts(v1), ts(v2)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), "{=list {=compound {=delegate {=auto 1}} {=delegate {=auto 2}}}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		}
	case "$2$1":
		if v1, v2 := auto_get(ctx, "1"), auto_get(ctx, "2"); v1 == nil || v2 == nil {
			if truly(ctx, ex_delegate{}) {
				if s, t := ts(res), "{=list {=compound {=null} {=null}}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), fmt.Sprintf("{=compound %s %s}", ts(v2), ts(v1)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			if truly(ctx, ex_delegate{}) {
				if s, t := ts(res), fmt.Sprintf("{=list {=compound %s %s}}", ts(v2), ts(v1)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), "{=list {=compound {=delegate {=auto 2}} {=delegate {=auto 1}}}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		}
	case "aa":
		if s, t := ts(res), "{=list {=compound {=word a} {=word a}}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "bb":
		if s, t := ts(res), "{=list {=compound {=word b} {=word b}}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "ab":
		if s, t := ts(res), "{=list {=compound {=word a} {=word b}}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "ba":
		if s, t := ts(res), "{=list {=compound {=word b} {=word a}}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case "{}{}":
		if s, t := ts(res), "{=list {=compound {=null} {=null}}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	default:
		errostack(pc(ctx,p), 6, "%v → %v : %v", p, res, ts(res)).trace()
	}
}
func (p *list) expand__value_3(ctx Context, d bool, res Value) {
	switch p.String() {
	default:
		erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
	}
}
func (p *list) expand__value_4(ctx Context, d bool, res Value) {
	switch p.String() {
	case "&(.test.none)":
		if truly(ctx, ex_closure{}) {
			switch s := ts(res); s {
			case "{=list}":
			default:
				erro(pc(ctx,p), "%v: %s", res, s).trace()
			}
		} else if truly(ctx, ex_delegate{}) {
			switch s := ts(res); s {
			case "{=list {=closure {=compound {=punct .} {=word test} {=punct .} {=word none}}}}":
			default:
				erro(pc(ctx,p), "%v: %s", res, s).trace()
			}
		} else {
			switch s := ts(res); s {
			case "{=list {=closure {=compound {=punct .} {=word test} {=punct .} {=word none}}}}":
			default:
				erro(pc(ctx,p), "%v: %s", res, s).trace()
			}
		}
	case "&(.test.x)":
		if truly(ctx, ex_closure{}) {
			switch s := ts(res); s {
			case "{=list {=compound {=punct .} {=word test} {=punct .} {=word v}}}":
			default:
				erro(pc(ctx,p), "%v: %s", res, s).trace()
			}
		} else if truly(ctx, ex_delegate{}) {
			switch s := ts(res); s {
			case "{=list {=closure {=def .test.x}}}":
			case "{=list {=closure {=compound {=punct .} {=word test} {=punct .} {=word x}}}}":
			default:
				erro(pc(ctx,p), "%v: %s", res, s).trace()
			}
		} else {
			switch s := ts(res); s {
			default:
				erro(pc(ctx,p), "%v: %s", res, s).trace()
			}
		}
	case "$(.test.x)":
		if truly(ctx, ex_delegate{}) {
			if s, t := ts(res), "{=list {=compound {=punct .} {=word test} {=punct .} {=word v}}}"; s != t {
				erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
			}
		} else {
			erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
		}
	case "-unique":
		if s, t := ts(res), "{=list {=flag {=word unique}}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	case ".test.D.c.0":
	case ".test.D.c.1":
	case ".test.D.c++.0":
	case ".test.D.c++.1":
	case ".test.I.c.0":
	case ".test.I.c.1":
	case ".test.I.c++.0":
	case ".test.I.c++.1":
	case ".test.foreach":
	case ".test.none":
	case ".test.and.x.1":
	case ".test.and.x.2":
	case ".test.and.y.1":
	case ".test.and.y.2":
	case ".test.x":
	case ".test.v":
	case "$1":
	default:
		erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
	}
}
func (p *list) expand__value_5(ctx Context, d bool, res Value) {
	switch p.String() {
	case "$1":
	default:
		erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
	}
}
func (p *list) expand__value_6(ctx Context, d bool, res Value) {
	switch p.String() {
	case "x-$1":
		if truly(ctx, ex_delegate{}) {
			if v := auto_get(ctx, "1"); v == nil {
				if s, t := ts(res), "{=list {=compound {=word x} {=flag {=null}}}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				t := fmt.Sprintf("{=list {=compound {=word x} {=flag %s}}}", ts(v))
				if s := ts(res); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
		}
	case "y-$1":
		if truly(ctx, ex_delegate{}) {
			if v := auto_get(ctx, "1"); v == nil {
				if s, t := ts(res), "{=list {=compound {=word y} {=flag {=null}}}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				t := fmt.Sprintf("{=list {=compound {=word y} {=flag %s}}}", ts(v))
				if s := ts(res); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
		}
	case "a":
		if s, t := ts(res), "{=list {=word a}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	default:
		erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
	}
}
func (p *list) expand__value_7(ctx Context, d bool, res Value) {
	switch p.String() {
	case "$1":
		if truly(ctx, ex_delegate{}) {
			if v := auto_get(ctx, "1"); v == nil {
				if s, t := ts(res), "{=list}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), fmt.Sprintf("{=list %s}", ts(v)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
		}
	case "$2":
		if truly(ctx, ex_delegate{}) {
			if v := auto_get(ctx, "2"); v == nil {
				if s, t := ts(res), "{=list}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), fmt.Sprintf("{=list %s}", ts(v)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
		}
	case "x$1":
		if truly(ctx, ex_delegate{}) {
			if v := auto_get(ctx, "1"); v == nil {
				if s, t := ts(res), "{=list}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), fmt.Sprintf("{=list {=compound {=word x} %s}}", ts(v)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
		}
	case "x$2":
		if truly(ctx, ex_delegate{}) {
			if v := auto_get(ctx, "2"); v == nil {
				if s, t := ts(res), "{=list}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), fmt.Sprintf("{=list {=compound {=word x} %s}}", ts(v)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
		}
	case "y$1":
		if truly(ctx, ex_delegate{}) {
			if v := auto_get(ctx, "1"); v == nil {
				if s, t := ts(res), "{=list}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				switch v.String() {
				case "xa":
					if s, t := ts(res), "{=list {=compound {=word y} {=word x} {=word a}}}"; s != t {
						erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
					}
				default:
					erro(pc(ctx,p), "%v → %v : %v", v, res, ts(res)).trace()
				}
			}
		} else {
			erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
		}
	case "y$2":
		if truly(ctx, ex_delegate{}) {
			if v := auto_get(ctx, "2"); v == nil {
				if s, t := ts(res), "{=list}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				switch v.String() {
				case "xb":
					if s, t := ts(res), "{=list {=compound {=word y} {=word x} {=word b}}}"; s != t {
						erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
					}
				default:
					erro(pc(ctx,p), "%v → %v : %v", v, res, ts(res)).trace()
				}
			}
		} else {
			erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
		}
	case "a":
		if s, t := ts(res), "{=list {=word a}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	default:
		erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
	}
}
func (p *list) expand__value_8(ctx Context, d bool, res Value) {
	switch p.String() {
	default:
		erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
	}
}
func (p *list) expand__value_9(ctx Context, d bool, res Value) {
	switch p.String() {
	case ".$1":
		if truly(ctx, ex_delegate{}) {
			if v := auto_get(ctx, "1"); v == nil {
				if s, t := ts(res), "{=list}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), fmt.Sprintf("{=list {=compound {=punct .} %s}}", ts(v)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
		}
	default:
		erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
	}
}
func (p *list) expand__value_10(ctx Context, d bool, res Value) {
	switch p.String() {
	default:
		erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
	}
}
func (p *list) expand__value_11(ctx Context, d bool, res Value) {
	switch p.String() {
	case ".v1 .v2":
		if s, t := ts(res), "{=list {=word .v1} {=word .v2}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	default:
		erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
	}
}
func (p *list) expand__value_12(ctx Context, d bool, res Value) {
	switch p.String() {
	case "$1":
		if truly(ctx, ex_delegate{}) {
			if v := auto_get(ctx, "1"); v == nil {
				if s, t := ts(res), "{=list}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), fmt.Sprintf("{=list %s}", ts(v)); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
		}
	case ".$1":
		if truly(ctx, ex_delegate{}) {
			if v := auto_get(ctx, "1"); v == nil {
				if s, t := ts(res), "{=list}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else if _, y := v.(*null); y {
				if s, t := ts(res), "{=list {=compound {=punct .} {=null}}}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				switch v.String() {
				case "w":
					if s, t := ts(res), "{=list {=compound {=punct .} {=word w}}}"; s != t {
						erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
					}
				default:
					erro(pc(ctx,p), "%v → %v : %v", v, res, ts(res)).trace()
				}
			}
		} else {
			erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
		}
	default:
		erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
	}
}
func (p *list) expand__value_13(ctx Context, d bool, res Value) {
	switch p.String() {
	default:
		erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
	}
}
func (p *list) expand__value_auto(ctx Context, d bool, res Value) {
	switch ps := p.String(); ps {
	case "$1 $2 $3 $4 $5 $6 $7 $8 $9":
		if false { note(pc(ctx,p), "%v %v", truly(ctx, ex_delegate{}), res).debug() }
		if truly(ctx, ex_def_1{}, ex_def_2{}) {
			var t string
			if truly(ctx, ex_delegate{}) {
				for i := 1; i <= 9; i += 1 {
					if a := auto_get(ctx, strconv.Itoa(i)); a != nil {
						if 1 < i { t += " " }
						t += a.String()
					} else if false {
						t += fmt.Sprintf("$%d", i)
					} else if false {
						if 1 < i { t += " " }
						t += "{}"
					}
				}
			}
			if t == "" { t = "{}" }
			if s := res.String(); s != t {
				errostack(pc(ctx,p), 3, "%s != %s, %s", s, t, ts(res)).trace()
			}
		} else {
			if s, t := res.String(), ps; s != t {
				errostack(pc(ctx,p), 3, "%s != %s, %s", s, t, ts(res)).trace()
			}
		}
	}
}
func (p *list) expand__value_optional(ctx Context, d bool, res Value) {
}
func (p *list) expand__value_placeholder(ctx Context, d bool, res Value) {
	switch p.String() {
	case "$_":
		if a := auto_get(ctx, "_"); a == nil {
			erro(ctx, "%v %v", do(ctx, evoke_x{}), res).debug()
		} else if false {
			var t = a.String()
			if s := res.String(); s != t {
				erro(ctx, "%s != %s : %s", s, t, ts(res)).trace()
			}
		}
	}
}
func (p *list) expand___addprefix(ctx Context, d bool, res Value) {
	switch p.String() {
	case "-std=":
		if s, t := ts(res), "{=list {=pair {=flag {=word std}}={=none}}}"; s != t {
			erro(pc(ctx,p), "%s: %s != %s", res, s, t).trace()
		}
	case "-foo=":
		if s, t := ts(res), "{=list {=pair {=flag {=word foo}}={=none}}}"; s != t {
			erro(pc(ctx,p), "%s: %s != %s", res, s, t).trace()
		}
	case "std=":
		if s, t := ts(res), "{=list {=pair {=word std}={=none}}}"; s != t {
			erro(pc(ctx,p), "%s: %s != %s", res, s, t).trace()
		}
	case "fo-":
		if s, t := ts(res), "{=list {=compound {=word fo} {=flag {=null}}}}"; s != t {
			erro(pc(ctx,p), "%s: %s != %s", res, s, t).trace()
		}
	case "foo":
		if s, t := ts(res), "{=list {=word foo}}"; s != t {
			erro(pc(ctx,p), "%s: %s != %s", res, s, t).trace()
		}
	case "foo bar":
		if s, t := ts(res), "{=list {=word foo} {=word bar}}"; s != t {
			erro(pc(ctx,p), "%s: %s != %s", res, s, t).trace()
		}
	case "&(.test.$1)":
		if v := auto_get(ctx, "1"); v == nil {
			if s, t := ts(res), "{=list {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}"; s != t {
				erro(pc(ctx,p), "%s: %s != %s (delegate=%v, closure=%v)", res, s, t, truly(ctx, ex_delegate{}), truly(ctx, ex_closure{})).trace()
			}
		} else if j := _project(ctx); truly(ctx, ex_closure{}) {
			t := "{=list {=list"
			for _, v := range merge(v) {
				if d := j.def(ctx, fmt.Sprintf(".test.%s", v)); d == nil {
					erro(pc(ctx,p), "%s ; %v %s", ts(v), res, ts(res)).trace()
				} else if v := d.value; v != nil {
					for _, v := range merge(v) {
						if t != "" { t += " " }
						t += ts(v)
					}
				}
			}
			t += "}}"
			if s := ts(res); s != t {
				erro(pc(ctx,p), "%v: %v: %s != %s ; %s", p, res, s, t, d).trace()
			}
		} else if truly(ctx, ex_delegate{}) {
			if vs := merge(v); len(vs) > 1 {
				t := "{=list {=list"
				for _, v := range vs {
					t += fmt.Sprintf(" {=closure {=compound {=punct .} {=word test} {=punct .} %s}}", ts(v))
				}
				t += "}}"
				if s := ts(res); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else if len(vs) == 1 {
				t := fmt.Sprintf("{=list {=closure {=compound {=punct .} {=word test} {=punct .} %s}}}", ts(v))
				if s := ts(res); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), "{=list}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			if s, t := ts(res), "{=list {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}"; s != t {
				erro(pc(ctx,p), "%s: %s != %s", res, s, t).trace()
			}
		}
	case "{&(.test.{$1})}":
		if v := auto_get(ctx, "1"); v == nil {
			if s, t := ts(res), "{=list {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=disjunction {=delegate {=auto 1}}}}}}"; s != t {
				erro(pc(ctx,p), "%s: %s != %s (delegate=%v, closure=%v)", res, s, t, truly(ctx, ex_delegate{}), truly(ctx, ex_closure{})).trace()
			}
		} else if j := _project(ctx); truly(ctx, ex_closure{}) {
			t := "{=list {=list"
			for _, v := range merge(v) {
				if d := j.def(ctx, fmt.Sprintf(".test.%s", v)); d == nil {
					erro(pc(ctx,p), "%s ; %v %s", ts(v), res, ts(res)).trace()
				} else if v := d.value; v != nil {
					for _, v := range merge(v) {
						if t != "" { t += " " }
						t += ts(v)
					}
				}
			}
			t += "}}"
			if s := ts(res); s != t {
				erro(pc(ctx,p), "%v: %v: %s != %s ; %s", p, res, s, t, d).trace()
			}
		} else if truly(ctx, ex_delegate{}) {
			if vs := merge(v); len(vs) > 1 {
				t := "{=list {=list"
				for _, v := range vs {
					t += fmt.Sprintf(" {=closure {=compound {=punct .} {=word test} {=punct .} %s}}", ts(v))
				}
				t += "}}"
				if s := ts(res); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else if len(vs) == 1 {
				t := fmt.Sprintf("{=list {=closure {=compound {=punct .} {=word test} {=punct .} %s}}}", ts(v))
				if s := ts(res); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), "{=list}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			if s, t := ts(res), "{=list {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}"; s != t {
				erro(pc(ctx,p), "%s: %s != %s", res, s, t).trace()
			}
		}
	case "=&(.test.$1)":
		if v := auto_get(ctx, "1"); v == nil {
			if s, t := ts(res), "{=list {=pair {=none}={=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}}"; s != t {
				erro(pc(ctx,p), "%s: %s != %s (delegate=%v, closure=%v)", res, s, t, truly(ctx, ex_delegate{}), truly(ctx, ex_closure{})).trace()
			}
		} else if j := _project(ctx); truly(ctx, ex_closure{}) {
			t := "{=list {=list"
			for _, v := range merge(v) {
				if d := j.def(ctx, fmt.Sprintf(".test.%s", v)); d == nil {
					erro(pc(ctx,p), "%s ; %v %s", ts(v), res, ts(res)).trace()
				} else if v := d.value; v != nil {
					for _, v := range merge(v) {
						if t != "" { t += " " }
						t += fmt.Sprintf("{=pair {=none}=%s}", ts(v))
					}
				}
			}
			t += "}}"
			if s := ts(res); s != t {
				erro(pc(ctx,p), "%v: %v: %s != %s ; %v", p, res, s, t, d).trace()
			}
		} else if truly(ctx, ex_delegate{}) {
			if vs := merge(v); len(vs) > 1 {
				t := "{=list {=list"
				for _, v := range vs {
					t += fmt.Sprintf(" {=pair {=none}={=closure {=compound {=punct .} {=word test} {=punct .} %s}}}", ts(v))
				}
				t += "}}"
				if s := ts(res); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else if len(vs) == 1 {
				t := fmt.Sprintf("{=list {=pair {=none}={=closure {=compound {=punct .} {=word test} {=punct .} %s}}}}", ts(v))
				if s := ts(res); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), "{=list}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			if s, t := ts(res), "{=list {=pair {=none}={=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}}"; s != t {
				erro(pc(ctx,p), "%s: %s != %s", res, s, t).trace()
			}
		}
	case "=&(.test.{$1})":
		if v := auto_get(ctx, "1"); v == nil {
			if s, t := ts(res), "{=list {=pair {=none}={=closure {=compound {=punct .} {=word test} {=punct .} {=disjunction {=delegate {=auto 1}}}}}}}"; s != t {
				erro(pc(ctx,p), "%s: %s != %s (delegate=%v, closure=%v)", res, s, t, truly(ctx, ex_delegate{}), truly(ctx, ex_closure{})).trace()
			}
		} else if j := _project(ctx); truly(ctx, ex_closure{}) {
			t := "{=list {=list"
			for _, v := range merge(v) {
				if d := j.def(ctx, fmt.Sprintf(".test.%s", v)); d == nil {
					erro(pc(ctx,p), "%s ; %v %s", ts(v), res, ts(res)).trace()
				} else if v := d.value; v != nil {
					for _, v := range merge(v) {
						if t != "" { t += " " }
						t += ts(v)
					}
				}
			}
			t += "}}"
			if s := ts(res); s != t {
				erro(pc(ctx,p), "%v: %v: %s != %s ; %s", p, res, s, t, d).trace()
			}
		} else if truly(ctx, ex_delegate{}) {
			if vs := merge(v); len(vs) > 1 {
				t := "{=list {=list"
				for _, v := range vs {
					t += fmt.Sprintf(" {=closure {=compound {=punct .} {=word test} {=punct .} %s}}", ts(v))
				}
				t += "}}"
				if s := ts(res); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else if len(vs) == 1 {
				t := fmt.Sprintf("{=list {=closure {=compound {=punct .} {=word test} {=punct .} %s}}}", ts(v))
				if s := ts(res); s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				if s, t := ts(res), "{=list}"; s != t {
					erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
				}
			}
		} else {
			if s, t := ts(res), "{=list {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}"; s != t {
				erro(pc(ctx,p), "%s: %s != %s", res, s, t).trace()
			}
		}
	case "=&(.test.a) =&(.test.b)":
		switch ts(res) {
		case "{=list {=list {=pair {=none}={=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}} {=pair {=none}={=closure {=compound {=punct .} {=word test} {=punct .} {=word b}}}}}}":
		case        "{=list {=pair {=none}={=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}} {=pair {=none}={=closure {=compound {=punct .} {=word test} {=punct .} {=word b}}}}}":
		default:
			erro(pc(ctx,p), "%s: %s", res, ts(res)).trace()
		}
	case "foo &(.test.$1)":
		if v := auto_get(ctx, "1"); v == nil {
			if s, t := ts(res), "{=list {=word foo} {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}"; s != t {
				erro(pc(ctx,p), "%s: %s != %s (delegate=%v, closure=%v)", res, s, t, truly(ctx, ex_delegate{}), truly(ctx, ex_closure{})).trace()
			}
		} else if j := _project(ctx); truly(ctx, ex_closure{}) {
			t := "{=list {=word foo} {=list"
			for _, v := range merge(v) {
				if d := j.def(ctx, fmt.Sprintf(".test.%s", v)); d == nil {
					erro(pc(ctx,p), "%s ; %v %s", ts(v), res, ts(res)).trace()
				} else if v := d.value; v != nil {
					for _, v := range merge(v) {
						if t != "" { t += " " }
						t += ts(v)
					}
				}
			}
			t += "}}"
			if s := ts(res); s != t {
				erro(pc(ctx,p), "%s: %s != %s", res, s, t).trace()
			}
		} else if truly(ctx, ex_delegate{}) {
			t := "{=list {=word foo} {=list"
			for _, v := range merge(v) {
				t += fmt.Sprintf(" {=closure {=compound {=punct .} {=word test} {=punct .} %s}}", ts(v))
			}
			t += "}}"
			if s := ts(res); s != t {
				erro(pc(ctx,p), "%s: %s != %s", res, s, t).trace()
			}
		} else {
			if s, t := ts(res), "{=list {=word foo} {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}"; s != t {
				erro(pc(ctx,p), "%s: %s != %s", res, s, t).trace()
			}
		}
	case "&(.test.a)":
		if truly(ctx, ex_closure{}) {
			if s, t := ts(res), "{=list {=word ax} {=word ay} {=word az}}"; s != t {
				erro(pc(ctx,p), "%s: %s != %s (delegate=%v, closure=%v)", res, s, t, truly(ctx, ex_delegate{}), truly(ctx, ex_closure{})).trace()
			}
		} else {
			if s, t := ts(res), "{=list {=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}}"; s != t {
				erro(pc(ctx,p), "%s: %s != %s (delegate=%v, closure=%v)", res, s, t, truly(ctx, ex_delegate{}), truly(ctx, ex_closure{})).trace()
			}
		}
	case "bar &(none)":
		if truly(ctx, ex_closure{}) {
			if s, t := ts(res), "{=list {=word bar}}"; s != t {
				erro(pc(ctx,p), "%s: %s != %s", res, s, t).trace()
			}
		} else {
			if s, t := ts(res), "{=list {=word bar} {=closure {=word none}}}"; s != t {
				erro(pc(ctx,p), "%s: %s != %s", res, s, t).trace()
			}
		}
	case "ax ay az":
		if s, t := ts(res), "{=list {=word ax} {=word ay} {=word az}}"; s != t {
			erro(pc(ctx,p), "%s: %s != %s", res, s, t).trace()
		}
	case "bx by bz":
	case "cx cy cz":
	case "dx dy dz":
	case "ex ey ez":
	case "fx fy fz":
	case "a b", "a b c", "1 2 3", "test null":
	case "=xxx":
		if s, t := ts(res), "{=list {=pair {=none}={=word xxx}}}"; s != t {
			erro(pc(ctx,p), "%s: %s != %s", res, s, t).trace()
		}
	case "=ax =ay =az =bx =by =bz":
		switch ts(res) {
		case "{=list {=list {=pair {=none}={=word ax}} {=pair {=none}={=word ay}} {=pair {=none}={=word az}} {=pair {=none}={=word bx}} {=pair {=none}={=word by}} {=pair {=none}={=word bz}}}}":
		case        "{=list {=pair {=none}={=word ax}} {=pair {=none}={=word ay}} {=pair {=none}={=word az}} {=pair {=none}={=word bx}} {=pair {=none}={=word by}} {=pair {=none}={=word bz}}}":
		default:
			erro(pc(ctx,p), "%s: %s", res, ts(res)).trace()
		}
	case "&(.test.$1.x.$2.y.$3.z)":
		var v1 = auto_get(ctx, "1")
		var v2 = auto_get(ctx, "2")
		var v3 = auto_get(ctx, "3")
		if v1 != nil && v2 != nil && v3 != nil {
			var j = _project(ctx)
			var t, n string
			for _, _v1 := range merge(v1) {
				for _, _v2 := range merge(v2) {
					for _, _v3 := range merge(v3) {
						s := fmt.Sprintf("{=compound {=punct .} {=word test} {=punct .} %s {=punct .} {=word x} {=punct .} %s {=punct .} {=word y} {=punct .} %s {=punct .} {=word z}}", ts(_v1), ts(_v2), ts(_v3))
						n += " " + s
						if truly(ctx, ex_closure{}) {
							s := fmt.Sprintf(".test.%s.x.%s.y.%s.z", _v1, _v2, _v3)
							if d := j.def(ctx, s); d != nil {
								for _, v := range merge(d.value) {
									t += " " + ts(v)
								}
							}
						} else if truly(ctx, ex_delegate{}) {
							t += fmt.Sprintf(" {=closure %s}", s)
						}
					}
				}
			}
			if s, t := ts(res), "{=list {=list"+t+"}}"; s != t {
				note(pc(ctx,p), "%s", s)
				note(pc(ctx,p), "%s", t)
				erro(pc(ctx,p), "%v", res).trace()
			}
		} else if s, t := ts(res), "{=list {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}} {=punct .} {=word x} {=punct .} {=delegate {=auto 2}} {=punct .} {=word y} {=punct .} {=delegate {=auto 3}} {=punct .} {=word z}}}}"; s != t {
			erro(pc(ctx,p), "%v: %s != %s", res, s, t).trace()
		}
	default:
		erro(pc(ctx,p), "%s: %s, %s", p, res, ts(res)).trace()
	}
}

func (p *delegate) cmp_check(ctx Context, v Value, _res *cmpres) {
    if res := *_res; res != cmpEqual && /* p.String() == v.String() */ts(p) == ts(v) {
        erro(pc(ctx,p), "%v, %v ⇔ %v, %v ⇔ %v", res, p, v, ts(p), ts(v)).trace()
    }
}
func (p *delegate) expand_check(ctx Context, x Value, y bool, _o, _a *[]Value, _res *Value) {
	if j := _project(ctx); j != nil {
		switch j.name {
		case "configure.base":
			p.expand__configure_base(ctx, *_res)
		}
		switch j.spec {
		case "testdata/value":
			p.expand__value(ctx, j, *_a, *_res)
		case "testdata/value/auto":
			p.expand__value_auto(ctx, j, *_o, *_a, *_res)
		case "testdata/value/placeholder":
			p.expand__value_placeholder(ctx, j, *_a, *_res)
		case "testdata/value/2":
			p.expand__value_2(ctx, *_res)
		case "testdata/value/4":
			p.expand__value_4(ctx, *_res)
		case "testdata/value/closure":
			p.expand__value_closure(ctx, x, *_res)
		case "testdata/value/optional":
			p.expand__value_optional(ctx, *_res)
		case "testdata/value/bug_01":
			p.expand__value_bug_01(ctx, *_o, *_a, *_res)
		case "testdata/builtins/addprefix":
			p.expand___addprefix(ctx, *_res)
		case "testdata/builtins/foreach":
			p.expand___foreach(ctx, *_res)
		case "testdata/rule/shell/for-stdout":
			p.expand_rule_shell_forstdout(ctx, *_res)
		}
	}
}
func (p *delegate) expand__configure_base(ctx Context, res Value) {
	if at := ts(auto_get(ctx, "@")); strings.HasPrefix(at, "{=file .configure/library/HAVE_LIB") {
		switch s := p.String(); s {
		case `$(foreach $(INCLUDE),"#include $_\n")`:
			if v := res.String(); strings.HasPrefix(v, `$(foreach {},"#include`) {
				note(ctx, "%v → %v ; %v %v", p, res, truly(ctx, ex_closure{}), truly(ctx, ex_delegate{}))
				erro(pc(ctx,p), "%s : %s → %s", at, s, v).trace()
			}
		}
	}
	if ent := _entry(ctx); ent != nil {
		switch ent.destiny().string(ctx) {
		case "-compiles-c", "-library-c", "-symbol-c":
			if truly(ctx, is_exec{}) {
				switch p.String() {
				case "$(file $(name).c)", "$(file $(name).c++)", "$(file $(name).log)":
					if _, y := res.(*file); !y {
						errostack(ctx, 8, "not a file: %v: %v → %v", p, ts(p), ts(res)).trace()
					}
				case "$<", "$>", "$(file $(s).x)", "$(file $(s).o)":
					if _, y := res.(fullfile); !y {
						errostack(ctx, 8, "not a fullfile: %v: %v → %v", p, ts(p), ts(res)).trace()
					}
				}
			}
			if truly(ctx, is_modify{}) {
				p.expand__configure_base_library_c(ctx, res)
			}
		}
	}
}
func (p *delegate) expand__configure_base_library_c(ctx Context, res Value) {
	var kind string
	var t = auto_find(ctx, "TARGET")
	var d = auto_find(ctx, "FUNCTION")
	if d != nil && !isTrivial(d.value) {
		kind = "function"
	} else {
		kind = "library"
	}
	switch p.String() {
	case "$(ifdef FUNCTION,function,library)":
		if (res).String() != kind {
			erro(pc(ctx,p), "%v", res).trace()
		}
	case "$(file .configure/$(ifdef FUNCTION,function,library)/$(TARGET).c)":
		if t == nil || isTrivial(t.value) {
			erro(pc(ctx,p), "%v", res).trace()
		} else if s := t.string(ctx); s == "" {
			erro(pc(ctx,p), "%v", t.value).trace()
		} else if x, y := (res).(*file); !y {
			erro(pc(ctx,p), "%v", res).trace()
		} else if t := filepath.Join(".configure", kind, s+".c"); t != x.name {
			erro(pc(ctx,p), "%s != %s", x.name, t).trace()
		}
	case "$(file $(s).x)":
		if t == nil || isTrivial(t.value) {
			erro(pc(ctx,p), "%v", res).trace()
		} else if s := t.string(ctx); s == "" {
			erro(pc(ctx,p), "%v %v", t.value, s).trace()
		} else {
			if x, y := (res).(*file); !y {
				erro(pc(ctx,p), "%v %v", typeof(res), res).trace()
			} else if t := filepath.Join(".configure", kind, s+".c.x"); t != x.name {
				erro(pc(ctx,p), "%s != %s", x.name, t).trace()
			}
		}
	case "$(file $(s).o)":
		if t == nil || isTrivial(t.value) {
			erro(pc(ctx,p), "%v", res).trace()
		} else if s := t.string(ctx); s == "" {
			erro(pc(ctx,p), "%v %v", t.value, s).trace()
		} else {
			if x, y := (res).(*file); !y {
				erro(pc(ctx,p), "%v %v", typeof(res), res).trace()
			} else if t := filepath.Join(".configure", kind, s+".c.o"); t != x.name {
				erro(pc(ctx,p), "%s != %s", x.name, t).trace()
			}
		}
	}
}
func (p *delegate) expand__value(ctx Context, j *project, a []Value, res Value) {
	switch p.String() {
	case `$(quote a\,b\,c,x\,y\,z)`:
		if s, t := res.String(), `{=quote a\,b\,c x\,y\,z}`; s != t {
			erro(pc(ctx,p), "%s != %s", s, t)
			note(ctx, "%v", ts(p))
			note(ctx, "%v", ts(res)).trace()
		}
	case fmt.Sprintf(`$(grep {=regex ^.+?\.o$$},$0,%s/test.txt)`, j.absPath):
		note(ctx, "%v", p)
		note(ctx, "%v", res).debug(3)
	}
}
func (p *delegate) expand__value_auto(ctx Context, j *project, o, a []Value, res Value) {
	var ps = p.String()
	switch ps {
	case "$(closure foobar)":
		if s, t := res.String(), "&(foobar)"; s != t {
			erro(pc(ctx,p), "%s != %s : %s", s, t, ts(res)).trace()
		}
		if s, t := ts(res), "{=closure {=word foobar}}"; s != t {
			erro(pc(ctx,p), "%s != %s", s, t).trace()
		}
	case "$(auto $(a))":
		// if o := do(ctx, evoke_def{"val1"}); o == nil {
		// if o := do(ctx, evoke_def{"val10"}); o == nil {
		if o := auto_get(ctx, "a"); o == nil {
			if s, t := res.String(), "{}"; s != t { // $(a)
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, do(ctx, evoke_x{}), ts(res)).trace()
			}
			if s, t := ts(res), "{=null}"; s != t { // {=delegate {=auto a}}
				erro(pc(ctx,p), "%s != %s", s, t).trace()
			}
		} else {
			if s, t := res.String(), o.String(); s != t {
				erro(pc(ctx,p), "%s != %s : %s", s, t, ts(res)).trace()
			}
			if s, t := ts(res), ts(o); s != t {
				erro(pc(ctx,p), "%s != %s", s, t).trace()
			}
		}
	case "$(auto(a=2) $(val1),$(a))":
		if truly(ctx, ex_def_1{}, ex_def_2{}) {
			if s, t := res.String(), "2"; s != t {
				erro(pc(ctx,p), "%s != %s : %s", s, t, ts(res)).trace()
			}
			if s, t := ts(res), "{=decimal 2}"; s != t {
				erro(pc(ctx,p), "%s != %s", s, t).trace()
			}
		} else {
			if s, t := res.String(), "{} 2"; s != t {
				erro(pc(ctx,p), "%s != %s : %s", s, t, ts(res)).trace()
			}
			if s, t := ts(res), "{=list {=null} {=decimal 2}}"; s != t {
				erro(pc(ctx,p), "%s != %s", s, t).trace()
			}
		}
	case "$(auto(a=2) $(val10),$(a))":
		if o := auto_get(ctx, "a"); o == nil {
			if s, t := res.String(), "2 2"; s != t {
				erro(pc(ctx,p), "%s != %s : %v : %s", s, t, p, ts(res)).trace()
			}
			if s, t := ts(res), "{=list {=decimal 2} {=decimal 2}}"; s != t {
				erro(pc(ctx,p), "%s != %s", s, t).trace()
			}
		} else {
			if s, t := res.String(), "2 2"; s != t {
				erro(pc(ctx,p), "%s != %s : %s", s, t, ts(res)).trace()
			}
			if s, t := ts(res), "{=list {=decimal 2} {=decimal 2}}"; s != t {
				erro(pc(ctx,p), "%s != %s", s, t).trace()
			}
		}
	case "$(auto(a=3) $(val2))":
		if truly(ctx, ex_def_1{}, ex_def_2{}) {
			if s, t := res.String(), "2"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, p, ts(res)).trace()
			}
			if s, t := ts(res), "{=decimal 2}"; s != t {
				erro(pc(ctx,p), "%s != %s", s, t).trace()
			}
		} else {
			if s, t := res.String(), "{} 2"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, p, ts(res)).trace()
			}
			if s, t := ts(res), "{=list {=null} {=decimal 2}}"; s != t {
				erro(pc(ctx,p), "%s != %s", s, t).trace()
			}
		}
	case "$(auto(a=3) $(val20))":
		if truly(ctx, ex_def_1{}, ex_def_2{}) {
			if s, t := res.String(), "2 2"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, p, ts(res)).trace()
			}
			if s, t := ts(res), "{=list {=decimal 2} {=decimal 2}}"; s != t {
				erro(pc(ctx,p), "%s != %s", s, t).trace()
			}
		} else {
			erro(pc(ctx,p), "%s : %s", p, ts(res)).trace()
		}
	case "$(val3)":
		if truly(ctx, ex_def_1{}, ex_def_2{}) {
			if s, t := res.String(), "2"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, p, ts(res)).trace()
			}
			if s, t := ts(res), "{=decimal 2}"; s != t {
				erro(pc(ctx,p), "%s != %s", s, t).trace()
			}
		} else {
			erro(pc(ctx,p), "%s : %s", p, ts(res)).trace()
		}
	case "$(val30)":
		if truly(ctx, ex_def_1{}, ex_def_2{}) {
			if s, t := res.String(), "2 2"; s != t {
				erro(pc(ctx,p), "%s != %s : %s : %s", s, t, p, ts(res)).trace()
			}
			if s, t := ts(res), "{=list {=decimal 2} {=decimal 2}}"; s != t {
				erro(pc(ctx,p), "%s != %s", s, t).trace()
			}
		} else {
			erro(pc(ctx,p), "%s : %s", p, ts(res)).trace()
		}
	default:
		for i := 1; i <= 9; i += 1 {
			if s := strconv.Itoa(i); ps == "$"+s {
				if t := auto_get(ctx, s); t != nil {
					if res.cmp(ctx, t) != cmpEqual {
						erro(pc(ctx,p), "%d, %s != %s", i, res, t).trace()
					}
				}
			}
		}
	}
}
func (p *delegate) expand__value_placeholder(ctx Context, j *project, a []Value, res Value) {
	var ps = p.String()
	switch ps {
	case "$(foreach a b c d e f,$_)":
		if s, t := res.String(), "a b c d e f"; s != t {
			erro(pc(ctx,p), "%s != %s : %s", s, t, ts(res)).trace()
		}
	case "$(foreach $1 $2 $3 $4 $5 $6 $7 $8 $9,$_)":
		if truly(ctx, ex_delegate{}) {
			if do(ctx, evoke_def{"val3"}) != nil {
				var t string
				for i := 1; i <= 9; i += 1 {
					if a := auto_get(ctx, strconv.Itoa(i)); a != nil {
						if 1 < i { t += " " }
						t += a.String()
					}
				}
				if s := res.String(); s != t {
					erro(pc(ctx,p), "%s != %s : %s", s, t, ts(res)).trace()
				}
			} else if false {
				note(ctx, "%v %v %v", p, do(ctx, evoke_x{}), res).debug()
			}
		} else {
			if s, t := res.String(), "{$1} {$2} {$3} {$4} {$5} {$6} {$7} {$8} {$9}"; s != t {
				erro(pc(ctx,p), "%s != %s : %s", s, t, ts(res)).trace()
			}
		}
	}
}
func (p *delegate) expand__value_2(ctx Context, res Value) {
}
func (p *delegate) expand__value_4(ctx Context, res Value) {
}
func (p *delegate) expand__value_closure(ctx Context, x, res Value) {
}
func (p *delegate) expand__value_optional(ctx Context, res Value) {
	switch ps := p.String(); ps {
	case "$({=yes})":
		if s, t := ts(res), "{=answer yes}"; s != t {
			erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
		}
	case "$(foo)":
		if s, t := ts(res), "{=project foo}"; s != t {
			erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
		}
	case "$(name?)":
		if s, t := ts(res), "{}"; s != t {
			erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
		}
	case "$(fo?→bar)":
		if res != nil {
			if s, t := ts(res), "{=null}"; s != t {
				erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
			}
		}
	case "$(fo?→bar?)":
		if res != nil {
			if s, t := ts(res), "{=null}"; s != t {
				erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
			}
		}
	case "$({=self foo})":
		if s, t := ts(res), "{=self foo}"; s != t {
			erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
		}
	case "$({=project foo})":
		if s, t := ts(res), "{=project foo}"; s != t {
			erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
		}
	case "$({=project foo}→name?)":
		if s, t := ts(res), "{=self foo}"; s != t {
			erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
		}
	case "$({=project foo}→baz?)":
		if res != nil {
			if s, t := ts(res), "{=null}"; s != t {
				erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
			}
		}
	case "$({=project foo}→name→xxxx?)":
		if res != nil {
			if s, t := ts(res), "{=null}"; s != t {
				erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
			}
		}
	case "$({=project foo}→name→item?)":
		if s, t := ts(res), "{=answer yes}"; s != t {
			erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
		}
	case "$({=project foo}→name?→item?)":
		if s, t := ts(res), "{=answer yes}"; s != t {
			erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
		}
	case "$({=project foo}?→name?→item?)":
		if s, t := ts(res), "{=answer yes}"; s != t {
			erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
		}
	case "$($_→name)":
		if s, v := ts(res), auto_get(ctx, "_"); v == nil {
			if t := "{=delegate {=arrow {=delegate {=auto _}}→{=word name}}}"; s != t {
				erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
			}
		} else if t := "{=self foo}"; s != t {
			erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
		}
	case "$($_→name?)":
		if s, v := ts(res), auto_get(ctx, "_"); v == nil {
			if t := "{=delegate {=cond {=arrow {=delegate {=auto _}}→{=word name}}}}"; s != t {
				erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
			}
		} else if t := "{=self foo}"; s != t {
			erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
		}
	case "$($_→bar?)":
		if s, v := ts(res), auto_get(ctx, "_"); v == nil {
			if t := "{=delegate {=cond {=arrow {=delegate {=auto _}}→{=word bar}}}}"; s != t {
				erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
			}
		} else if t := "{=null}"; s != t {
			note(ctx, "%v", truly(ctx, evoke_builtin{"foreach"})).debug()
			erro(pc(ctx,res), "%s: %s != %s ; %v", ps, s, t, tv(v)).trace()
		}
	case "$_":
		if s := ts(res); truly(ctx, ex_def_1{}) {
			if v := auto_get(ctx, "_"); v == nil {
				if t := "{=null}"; s != t {
					erro(pc(ctx,res), "%s: %s != %s ; %v", ps, s, t, tv(v)).trace()
				}
			} else {
				if t := ts(v); s != t {
					erro(pc(ctx,res), "%s: %s != %s ; %v", ps, s, t, tv(v)).trace()
				}
			}
		} else if _, y := do(ctx, identity{}).(*identity_ctx); y {
			if t := "{}"; s != t || res != nil {
				erro(pc(ctx,p), "%s != %s", s, t).trace()
			}
		} else if t := "{=null}"; s != t {
			erro(pc(ctx,p), "%s != %s ; %v", s, t, do(ctx, evoke_x{})).trace()
		}
	case "$(foreach $({=project foo}),$($_→name))":
		if s := ts(res); truly(ctx, ex_def_1{}) {
			if v := auto_get(ctx, "_"); v == nil {
				if t := "{=self foo}"; s != t {
					erro(pc(ctx,res), "%s: %s != %s ; %v", ps, s, t, tv(v)).trace()
				}
			} else {
				if t := ts(v); s != t {
					erro(pc(ctx,res), "%s: %s != %s ; %v", ps, s, t, tv(v)).trace()
				}
			}
		} else if t := "{=delegate {=arrow {=delegate {=auto _}}→{=word name}}}"; s != t {
			erro(pc(ctx,p), "%s: %s != %s", ps, s, t).trace()
		}
	case "$(foreach $({=project foo}),$($_→name?))":
		if s := ts(res); truly(ctx, ex_def_1{}) {
			if v := auto_get(ctx, "_"); v == nil {
				if t := "{=self foo}"; s != t {
					erro(pc(ctx,res), "%s: %s != %s ; %v", ps, s, t, tv(v)).trace()
				}
			} else {
				if t := ts(v); s != t {
					erro(pc(ctx,res), "%s: %s != %s ; %v", ps, s, t, tv(v)).trace()
				}
			}
		} else if t := "{=delegate {=cond {=arrow {=delegate {=auto _}}→{=word name}}}}"; s != t {
			erro(pc(ctx,p), "%s: %s != %s", ps, s, t).trace()
		}
	case "$(foreach $({=project foo}),$($_→bar?))":
		if s := ts(res); truly(ctx, ex_def_1{}) {
			if v := auto_get(ctx, "_"); v == nil {
				if t := "{=null}"; s != t {
					erro(pc(ctx,res), "%s: %s != %s ; %v", ps, s, t, tv(v)).trace()
				}
			} else {
				if t := ts(v); s != t {
					erro(pc(ctx,res), "%s: %s != %s ; %v", ps, s, t, tv(v)).trace()
				}
			}
		} else if t := "{=delegate {=cond {=arrow {=delegate {=auto _}}→{=word bar}}}}"; s != t {
			erro(pc(ctx,p), "%s: %s != %s", ps, s, t).trace()
		}
	default:
		erro(pc(ctx,p), "%v %v", ps, res).trace()
	}

	if truly(ctx, ex_def_1{}) {
		switch p.x.String() {
		case "foo":
			if s, t := ts(res), "{=project foo}"; s != t {
				errostack(pc(ctx,p.x), 1, "%v: res=%v : %v", p, res, ts(res)).trace()
			}
		case "name?":
			if !isNull(res) {
				errostack(pc(ctx,p.x), 1, "%v: res=%v : %v", p, res, ts(res)).trace()
			}
		case "fo?→bar":
			if res != nil {
				if s, t := ts(res), "{=null}"; s != t {
					erro(pc(ctx,p.x), "%s != %s : %s", s, t, ts(res)).trace()
				}
			}
		case "fo?→bar?":
			if res != nil {
				if s, t := ts(res), "{=null}"; s != t {
					erro(pc(ctx,p.x), "%s != %s : %s", s, t, ts(res)).trace()
				}
			}
		case "{=project foo}→name?":
			if s, t := ts(res), "{=self foo}"; s != t {
				erro(pc(ctx,p.x), "%s != %s : %s", s, t, ts(res)).trace()
			}
		case "{=project foo}→baz?":
			if res != nil {
				if s, t := ts(res), "{=null}"; s != t {
					erro(pc(ctx,p.x), "%s != %s : %s", s, t, ts(res)).trace()
				}
			}
		case "{=project foo}→name→xxxx?":
			if res != nil {
				if s, t := ts(res), "{=null}"; s != t {
					erro(pc(ctx,p.x), "%s != %s : %s", s, t, ts(res)).trace()
				}
			}
		case "{=project foo}→name→item?":
			if s, t := ts(res), "{=answer yes}"; s != t {
				erro(pc(ctx,p.x), "%s != %s : %s", s, t, ts(res)).trace()
			}
		case "{=project foo}→name?→item?":
			if s, t := ts(res), "{=answer yes}"; s != t {
				erro(pc(ctx,p.x), "%s != %s : %s", s, t, ts(res)).trace()
			}
		case "{=project foo}?→name?→item?":
			if s, t := ts(res), "{=answer yes}"; s != t {
				erro(pc(ctx,p.x), "%s != %s : %s", s, t, ts(res)).trace()
			}
		}
	}
}
func (p *delegate) expand__value_bug_01(ctx Context, o, a []Value, res Value) {
	switch s := p.String() ; s {
	case "$(foreach $1,$2.$_)":
		var v1, v2, v3 = auto_get(ctx, "1"), auto_get(ctx, "2"), auto_get(ctx, "3")
		var s1, s2, s3 = ts(v1), ts(v2), ts(v3)
		if x, y := do(ctx, evoke_def{"okay.2"}).(*def); !y {
			erro(pc(ctx,p), "%v, %v, %v ; %v ; %v", s1, s2, s3, x, res).trace()
		}
		if x, y := do(ctx, evoke_def{"okay.1"}).(*def); !y {
			erro(pc(ctx,p), "%v, %v, %v ; %v ; %v", s1, s2, s3, x, res).trace()
		}

		var s0 string
		switch s1 {
		case "{=list {=delegate {=auto 1}} {=delegate {=auto 2}}}":
			for _, s0 = range []string{"x", "y", "z"} {
				if s2 == "{=word "+s0+"}" && s3 == "{=word "+s0+"}" {
					if r, t := res.String(), s0+".{$1} "+s0+".{$2}"; s == r {
						erro(pc(ctx,p), "%v → %v != %v; %v, %v, %v", s, r, t, v1, v2, v3).trace()
					} else if r != t {
						erro(pc(ctx,p), "%v → %v != %v; %v, %v, %v", s, r, t, v1, v2, v3).trace()
					}
					break
				}
			}
			switch s2+s3 {
			case "xx", "yy", "zz":
			default:
				erro(pc(ctx,p), "%s, %s, %s ; %v ; %v", s1, s2, s3, ts(a), ts(res)).trace()
			}
		case "{=null}":
			if s, t := ts(res), "{=null}"; s != t {
				erro(pc(ctx,p), "%s, %s, %s ; %v ; %v", s1, s2, s3, ts(a), ts(res)).trace()
			}
		case "{=list {=word a} {=word b}}":
			var s = ts(auto_get(ctx, "_"))
			var t = fmt.Sprintf("{=list {=compound %s {=punct .} {=word a}} {=compound %s {=punct .} {=word b}}}", s, s)
			if s = ts(res); s != t {
				erro(pc(ctx,p), "%s, %s, %s; %s ; %s", s1, s2, s3, ts(a), ts(res)).trace()
			}
		default:
			erro(pc(ctx,p), "%s, %s, %s ; %v ; %v", s1, s2, s3, ts(a), ts(res)).trace()
		}

		if v0 := auto_get(ctx, "_"); v0 == nil && s0 == "" {
			if s, t := "$2.{$1}", res.String(); s != t {
				note(ctx, "1: %v → %v", v1, s1)
				note(ctx, "2: %v → %v", v2, s2)
				note(ctx, "3: %v → %v", v3, s3)
				errostack(ctx, 8, "%s != %s; %v", s, t, v0).trace()
			}
		}

	case "$(foreach(-unique) $(foreach $1,$2.$_),$_)":
		var v1, v2, v3 = auto_get(ctx, "1"), auto_get(ctx, "2"), auto_get(ctx, "3")
		var s1, s2, s3 = ts(v1), ts(v2), ts(v3)
		var s0 string
		if s1 == "{=list {=delegate {=auto 1}} {=delegate {=auto 2}}}" {
			for _, s0 = range []string{"x", "y", "z"} {
				if s2 == "{=word "+s0+"}" && s3 == "{=word "+s0+"}" {
					if r, t := res.String(), "{"+s0+".{$1}} {"+s0+".{$2}}"; s == r {
						errostack(pc(ctx,p), 3, "%v → %v != %v", s, r, t).trace()
					} else if r != t {
						errostack(pc(ctx,p), 3, "%v → %v != %v", s, r, t).trace()
					}
					break
				}
			}
		}
		if v0 := auto_get(ctx, "_"); v0 == nil && s0 == "" {
			if s, t := "{$2.{$1}}", res.String(); s != t {
				erro(pc(ctx,p), "%v != %v : %v, %v, %v", s, t, s1, s2, s3).trace()
			}
		}

	case "$(foreach x y z,$(okay.2 $1 $2,$_,$_))":
		var v1, v2 = auto_get(ctx, "1"), auto_get(ctx, "2")
		if isNull(v1) && isNull(v2) {
			if s, t := ts(res), "{=null}"; s != t {
				erro(pc(ctx,p), "%v != %v : %v : %v %v", s, t, a, ts(v1), ts(v2)).trace()
			}
		} else {
			t := "{=list"
			for _, k := range []string{"x", "y", "z"} {
				for _, v := range merge(v1, v2) {
					t += fmt.Sprintf(" {=compound {=word %s} {=punct .} %s}", k, ts(v))
				}
			}
			t += "}"
			if s := ts(res); s != t {
				note(ctx, "%s", s)
				note(ctx, "%s", t)
				erro(pc(ctx,p), "%v : %v %v", a, ts(v1), ts(v2)).trace()
			}
		}

	case "$(okay.2 $1 $2,$_,$_)":
		if v1, v2 := auto_get(ctx, "1"), auto_get(ctx, "2"); isNull(v1) && isNull(v2) {
			if s, t := ts(res), "{=null}"; s != t {
				erro(pc(ctx,p), "%v != %v : %v : %v %v", s, t, a, ts(v1), ts(v2)).trace()
			}
		} else {
			var s = ts(auto_get(ctx, "_"))
			var t = fmt.Sprintf("{=list {=compound %s {=punct .} %s} {=compound %s {=punct .} %s}}", s, ts(v1), s, ts(v2))
			if s = ts(res); s != t {
				erro(pc(ctx,p), "%s != %s : %v : %v %v", s, t, a, ts(v1), ts(v2)).trace()
			}
		}

	case "$(okay.1 $1,$2)":
		if truly(ctx, ex_delegate{}) {
			var v1, v2 = auto_get(ctx, "1"), auto_get(ctx, "2")
			if isNull(v1) && isNull(v2) {
				if s, t := ts(res), "{=null}"; s != t {
					erro(pc(ctx,p), "%s != %s : %v : %v %v", s, t, a, ts(v1), ts(v2)).trace()
				}
			} else {
				t := "{=list"
				for _, k := range []string{"x", "y", "z"} {
					for _, v := range merge(v1, v2) {
						t += fmt.Sprintf(" {=compound {=word %s} {=punct .} %s}", k, ts(v))
					}
				}
				t += "}"
				if s := ts(res); s != t {
					note(ctx, "%s", s)
					note(ctx, "%s", t)
					erro(pc(ctx,p), "%v : %v %v", res, ts(v1), ts(v2)).trace()
				}
			}
		} else if t := res.String(); s != t {
			erro(pc(ctx,p), "%v → %v : %v", s, t, do(ctx, evoke_x{})).trace()
		}

	default:
		if false { erro(pc(ctx,p), "%v → %v", s, res).trace() }
	}
}
func (p *delegate) expand___addprefix(ctx Context, res Value) {
}
func (p *delegate) expand___foreach(ctx Context, res Value) {
	// switch _, parsing := do(ctx, get_parser{}).(*parser); p.String() {}
	if d := do(ctx, evoke_def{".test.7"}); d != nil {
		switch p.String() {
		case "$_", "$1":
		case "$(foreach $1,&(.test.z)$_{}zz)":
			v1, s, t := auto_get(ctx, "1"), ts(res), "{=list"
			for _, v := range merge(v1) {
				switch {
				case truly(ctx, ex_closure{}):
					t += fmt.Sprintf(" {=compound {=word w} %s {=word zz}}", ts(v))
				case truly(ctx, ex_delegate{}):
					t += fmt.Sprintf(" {=compound {=closure {=compound {=punct .} {=word test} {=punct .} {=word z}}} %s {=word zz}}", ts(v))
				}
			}
			if t += "}"; s != t {
				note(pc(ctx,p.x), "%s", s)
				note(pc(ctx,p.x), "%s", t)
				erro(pc(ctx,p.x), "%v ; %v", res, v1).trace()
			}
		case "$(foreach a $(foreach $1,&(.test.z)$_{}zz) b,x$_)":
			v1, s, t := auto_get(ctx, "1"), ts(res), "{=list {=compound {=word x} {=word a}}"
			for _, v := range merge(v1) {
				switch {
				case truly(ctx, ex_closure{}):
					t += fmt.Sprintf(" {=compound {=word x} {=compound {=word w} %s {=word zz}}}", ts(v))
				case truly(ctx, ex_delegate{}):
					t += fmt.Sprintf(" {=compound {=word x} {=compound {=closure {=compound {=punct .} {=word test} {=punct .} {=word z}}} %s {=word zz}}}", ts(v))
				}
			}
			if t += " {=compound {=word x} {=word b}}}"; s != t {
				note(pc(ctx,p.x), "%s", s)
				note(pc(ctx,p.x), "%s", t)
				erro(pc(ctx,p.x), "%v ; %v", res, v1).trace()
			}
		default:
			erro(pc(ctx,p.x), "%v → %v ; %v", p, res, d).trace()
		}
	}
}
func (p *delegate) expand_rule_shell_forstdout(ctx Context, res Value) {
	var o = try[origin](ctx, get_origin{})

	switch p.String() {
	case "${.test $1,$2}":
		if a1, a2 := auto_get(ctx, "1"), auto_get(ctx, "2") ; a1 != nil && a2 != nil {
			switch o {
			case defExpand0:
				if ts(res) != sfmt("{=delegate {=builtin debug} {=list %s %s}}", ts(a1), ts(a2)) {
					errostack(ctx, 3, "%s %s, %s", ts(a1), ts(a2), ts(res)).trace()
				}
			case defExpand1:
				if ts(res) != "{=null}" {
					errostack(ctx, 3, "%s", ts(res)).trace()
				}
			}
		} else if truly(ctx, ex_delegate{}) {
			if ts(res) != "{=delegate {=builtin debug} {=list {=delegate {=auto 2}} {=delegate {=auto 1}}}}" {
				errostack(ctx, 3, "%v: %s", p, ts(res)).trace()
			}
		} else {
			erro(pc(ctx,p), "%v: %s %s ; %s", p, ts(a1), ts(a2), ts(res)).trace()
		}
	case "${.test a,b}":
		switch o {
		case defExpand0:
			if ts(res) != "{=delegate {=builtin debug} {=list {=word b} {=word a}}}" {
				errostack(ctx, 3, "%v", ts(res)).trace()
			}
		case defExpand1:
			if ts(res) != "{=null}" {
				errostack(ctx, 3, "%s", ts(res)).trace()
			}
		}
	case "$(.test.v3 a,b)":
		switch o {
		case defExpand1:
			if ts(res) != "{=null}" {
				errostack(ctx, 3, "%s", ts(res)).trace()
			}
		}
	case "$(debug $(line) $(str))":
		if ts(p.a) != "{=[]Value {=list {=delegate {=auto line}} {=delegate {=auto str}}}}" {
			erro(pc(ctx,p), "%v", ts(p.a)).trace()
		}

		if a := _automatic(ctx); a == nil {
			errostack(ctx, 5, "%v", ts(res)).trace()
		} else {
			keys := reflect.ValueOf(a.defs).MapKeys()

			if x1, y := a.defs["1"]; !y {
				errostack(ctx, 5, "%v; %v", keys, ts(res)).trace()
			} else if x2, y := a.defs["str"]; !y {
				errostack(ctx, 5, "%v; %v", keys, ts(res)).trace()
			} else if x1 != x2 {
				errostack(ctx, 5, "%v != %v; %v", x1, x2, ts(res)).trace()
			}

			if x1, y := a.defs["2"]; !y {
				errostack(ctx, 5, "%v; %v", keys, ts(res)).trace()
			} else if x2, y := a.defs["line"]; !y {
				errostack(ctx, 5, "%v; %v", keys, ts(res)).trace()
			} else if x1 != x2 {
				errostack(ctx, 5, "%v != %v; %v", x1, x2, ts(res)).trace()
			}
		}
		if a, b := auto_get(ctx, "1"), auto_get(ctx, "str"); a != b {
			errostack(ctx, 5, "%v != %v; %v", a, b, ts(res)).trace()
		}
		if a, b := auto_get(ctx, "2"), auto_get(ctx, "line"); a != b {
			errostack(ctx, 5, "%v != %v; %v", a, b, ts(res)).trace()
		}

		switch ts(p.a) {
		case "{=[]Value {=list {=delegate {=auto 2}} {=delegate {=auto 1}}}}":
			switch o {
			case defExpand0, defExpand1:
				if ts(res) != "{=delegate {=builtin debug} {=list {=delegate {=auto 2}} {=delegate {=auto 1}}}}" {
					errostack(ctx, 5, "%v", ts(res)).trace()
				}
			}
		case "{=[]Value {=list {=word b} {=word a}}}":
			switch o {
			case defExpand0:
				if ts(res) != "{=delegate {=builtin debug} {=list {=word b} {=word a}}}" {
					errostack(ctx, 5, "%v", ts(res)).trace()
				}
			case defExpand1:
				if ts(res) != "{}" {
					errostack(ctx, 5, "%v", ts(res)).trace()
				}
			}
		case "{=[]Value}":
			var t = []Value{auto_get(ctx, "1"), auto_get(ctx, "2")}
			if ts(t) != "{=[]Value {} {}}" {
				errostack(ctx, 5, "%v %v", ts(p.x), ts(t)).trace()
			}
			if ts(res) != "{}" {
				errostack(ctx, 5, "%v", ts(res)).trace()
			}
		case
			`{=[]Value {=list {=decimal 1} {=strlit test one\n}}}`,
			`{=[]Value {=list {=decimal 2} {=strlit test two\n}}}`,
			`{=[]Value {=list {=decimal 3} {=strlit test thr\n}}}`:
			if ts(res) != "{}" {
				errostack(ctx, 5, "%v", ts(res)).trace()
			}
		default:
			errostack(ctx, 5, "untested: %v, %s", o, ts(res)).trace()
		}
	}

	switch o {
	case defExpand0, defExpand1, 0:
	default:
		errostack(ctx, 5, "untested: %v %s", o, ts(res)).trace()
	}
}

func (p *closure) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && v != nil && p.String() == v.String() {
		erro(pc(ctx,p), "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}
func (p *closure) expand_check(ctx Context, x Value, y bool, _o, _a *[]Value, _res *Value) {
	if j := _project(ctx); j != nil {
		switch j.name {
		case "configure.base":
			p.expand__configure_base(ctx, x, *_res)
		}
		switch j.spec {
		case "testdata/value":
			p.expand__value(ctx, x, *_res)
		case "testdata/value/2":
			p.expand__value_2(ctx, x, *_res)
		case "testdata/value/4":
			p.expand__value_4(ctx, x, *_res)
		case "testdata/value/closure":
			p.expand__value_closure(ctx, x, *_res)
		case "testdata/value/optional":
			p.expand__value_optional(ctx, x, *_res)
		case "testdata/value/bug_01":
			p.expand__value_bug_01(ctx, x, *_res)
		case "testdata/builtins/addprefix":
			p.expand___addprefix(ctx, j, *_o, *_a, x, *_res)
		case "testdata/builtins/foreach":
			p.expand___foreach(ctx, x, *_res)
		case "testdata/rule/shell/for-stdout":
			p.expand_rule_shell_forstdout(ctx, x, *_res)
		}
	}
}
func (p *closure) expand__configure_base(ctx Context, x, res Value) {
}
func (p *closure) expand__value(ctx Context, x, res Value) {
}
func (p *closure) expand__value_2(ctx Context, x, res Value) {
}
func (p *closure) expand__value_4(ctx Context, x, res Value) {
}
func (p *closure) expand__value_closure(ctx Context, x, res Value) {
	switch p.String() {
	case "&(&(foo.tail))":
		if _, y := x.(*closure); y {
		}
	}
}
func (p *closure) expand__value_optional(ctx Context, x, res Value) {
}
func (p *closure) expand__value_bug_01(ctx Context, x, res Value) {
	switch s := p.String() ; s {
	case "&($2.$_)":
		if x, y := do(ctx, evoke_def{}).(*def); y {
			var v2 = auto_get(ctx, "2")
			var s2 = ts(v2)
			switch x.name {
			case ".flag":
				if s, t := res.String(), "&("+v2.String()+".$_)"; s != t {
					erro(pc(ctx,p), "%s → %s != %s : %v", p, s, t, s2).trace()
				}
			case "bug_0.2":
				if x.value.String() != "$(foreach(-unique) $(foreach $1,&($2.$_)),$_)" {
					erro(pc(ctx,p), "%v → %v : %v", s, res, x).trace()
				}
				if v, s := auto_get(ctx, "_"), res.String(); v == nil {
					if t := "&("+v2.String()+".$_)"; s != t {
						erro(pc(ctx,p), "%s → %s != %s : %v", p, s, t, s2).trace()
					}
				} else {
					if t := "&("+v2.String()+"."+v.String()+")"; s != t {
						erro(pc(ctx,p), "%s → %s != %s : %v", p, s, t, s2).trace()
					}
				}
			}
		} else if _, y := do(ctx, evoke_builtin{"foreach"}).(*builtin); y && false {
			erro(pc(ctx,p), "%v → %v : %v : %v", s, res, ts(auto_get(ctx, "_")), do(ctx, evoke_def{})).trace()
		} else if s, t := "&($2.{$1})", res.String(); s != t {
			erro(pc(ctx,p), "%v → %v : %v : %v", s, t, ts(auto_get(ctx, "_")), ts(do(ctx, evoke_x{}))).trace()
		} else if v := auto_get(ctx, "_"); v == nil {
			erro(pc(ctx,p), "%v → %v : %v : %v", s, t, ts(auto_get(ctx, "_")), ts(do(ctx, evoke_x{}))).trace()
		} else if s, t := "{=disjunction {=delegate {=auto 1}}}", ts(v); s != t {
			erro(pc(ctx,p), "%v → %v : %v : %v", s, t, ts(auto_get(ctx, "_")), ts(do(ctx, evoke_x{}))).trace()
		} else if false {
			erro(pc(ctx,p), "%v : %v : %v", s, auto_get(ctx, "_"), ts(do(ctx, evoke_x{}))).trace()
		}
	}
}
func (p *closure) expand___addprefix(ctx Context, j *project, o, a []Value, x, res Value) {
    switch _, parsing := do(ctx, get_parser{}).(*parser); p.String() {
	case "&(none)":
		if s, t := ts(x), "{=word none}"; s != t {
			erro(pc(ctx,x), "%v: %s != %s", x, s, t).trace()
		}
		if truly(ctx, ex_closure{}) {
			if res != nil {
				erro(pc(ctx,x), "%v: %s", res, ts(res)).trace()
			}
		} else {
			if s, t := ts(res), "{=closure {=word none}}"; s != t {
				erro(pc(ctx,x), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "&(.test.$1)":
		if parsing {
			if s, t := ts(x), "{=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}"; s != t {
				erro(pc(ctx,x), "%v: %s != %s", x, s, t).trace()
			}
			if s, t := ts(res), "{=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}"; s != t {
				erro(pc(ctx,x), "%v: %s != %s", res, s, t).trace()
			}
		} else if v := auto_get(ctx, "1"); truly(ctx, ex_closure{}) {
			if v == nil {
				if s, t := ts(x), "{=def .test.}"; s != t {
					erro(pc(ctx,x), "%v → %v : %s != %s", p.x, x, s, t).trace()
				}
				if s, t := ts(res), "{=list {=word test} {=word null}}"; s != t {
					erro(pc(ctx,x), "%v → %v : %s != %s", p, res, s, t).trace()
				}
			} else {
				var t string
				var vs = merge(v)
				if len(vs) > 1 {
					t = "{=list"
					for _, v := range vs {
						t += fmt.Sprintf(" {=compound {=punct .} {=word test} {=punct .} %s}", ts(v))
					}
					t += "}"
				} else {
					t = fmt.Sprintf("{=compound {=punct .} {=word test} {=punct .} %s}", ts(v))
				}
				if s := ts(x); s != t {
					erro(pc(ctx,x), "%v: %s != %s", x, s, t).trace()
				}

				vs = merge(v)
				t = "{=list"
				for _, v := range vs {
					if d := j.def(ctx, fmt.Sprintf(".test.%s", v)); d == nil {
						erro(pc(ctx,p), "%s ; %v %s", ts(v), res, ts(res)).trace()
					} else if v := d.value; v != nil {
						for _, v := range merge(v) {
							if t != "" { t += " " }
							t += ts(v)
						}
					}
				}
				t += "}"
				if s := ts(res); s != t {
					erro(pc(ctx,p), "%v: %v ⇒ %v : %s != %s; %v", p, x, res, s, t, vs).trace()
				}
			}
		} else if truly(ctx, ex_delegate{}) {
			if v == nil {
				if s, t := ts(x), "{=compound {=punct .} {=word test} {=punct .} {=null}}"; s != t {
					erro(pc(ctx,x), "%v: %s != %s", x, s, t).trace()
				}
				if s, t := ts(res), "{=closure {=compound {=punct .} {=word test} {=punct .} {=null}}}"; s != t {
					erro(pc(ctx,x), "%v: %s != %s", res, s, t).trace()
				}
			} else {
				var t string
				switch v.String() {
				case "a", "b", "c":
					if s, t := ts(x), fmt.Sprintf("{=compound {=punct .} {=word test} {=punct .} %s}", ts(v)); s != t {
						erro(pc(ctx,x), "%v: %s != %s", x, s, t).trace()
					}
					if s, t := ts(res), fmt.Sprintf("{=closure {=compound {=punct .} {=word test} {=punct .} %s}}", ts(v)); s != t {
						erro(pc(ctx,x), "%v: %s != %s", res, s, t).trace()
					}
				case "a b":
					t = "{=list"
					for _, v := range merge(v) {
						t += fmt.Sprintf(" {=compound {=punct .} {=word test} {=punct .} %s}", ts(v))
					}
					if s, t := ts(x), t+"}"; s != t {
						erro(pc(ctx,x), "%v: %s != %s", x, s, t).trace()
					}
					t = "{=list"
					for _, v := range merge(v) {
						t += fmt.Sprintf(" {=closure {=compound {=punct .} {=word test} {=punct .} %s}}", ts(v))
					}
					if s, t := ts(res), t+"}"; s != t {
						erro(pc(ctx,x), "%v: %s != %s", res, s, t).trace()
					}
				default:
					erro(pc(ctx,x), "%s: %s", v, res).trace()
				}
			}
		} else {
			if s, t := ts(res), "{=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}"; s != t {
				erro(pc(ctx,x), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "&(.test.a)":
		if truly(ctx, ex_closure{}) {
			if s, t := ts(x), "{=compound {=punct .} {=word test} {=punct .} {=word a}}"; s != t {
				erro(pc(ctx,x), "%v: %s != %s", x, s, t).trace()
			}
			if s, t := ts(res), "{=list {=word ax} {=word ay} {=word az}}"; s != t {
				erro(pc(ctx,x), "%v: %s != %s", res, s, t).trace()
			}
		} else {
			if s, t := ts(res), "{=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}"; s != t {
				erro(pc(ctx,x), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "&(.test.b)":
		if truly(ctx, ex_closure{}) {
			if s, t := ts(x), "{=compound {=punct .} {=word test} {=punct .} {=word b}}"; s != t {
				erro(pc(ctx,x), "%v: %s != %s", x, s, t).trace()
			}
			if s, t := ts(res), "{=list {=word bx} {=word by} {=word bz}}"; s != t {
				erro(pc(ctx,x), "%v: %s != %s", res, s, t).trace()
			}
		} else {
			if s, t := ts(res), "{=closure {=compound {=punct .} {=word test} {=punct .} {=word b}}}"; s != t {
				erro(pc(ctx,x), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "&(.test.$1.x.$2.y.$3.z)":
		var v1 = auto_get(ctx, "1")
		var v2 = auto_get(ctx, "2")
		var v3 = auto_get(ctx, "3")
		if v1 != nil && v2 != nil && v3 != nil {
			var j = _project(ctx)
			var t, n string
			for _, _v1 := range merge(v1) {
				for _, _v2 := range merge(v2) {
					for _, _v3 := range merge(v3) {
						s := fmt.Sprintf("{=compound {=punct .} {=word test} {=punct .} %s {=punct .} {=word x} {=punct .} %s {=punct .} {=word y} {=punct .} %s {=punct .} {=word z}}", ts(_v1), ts(_v2), ts(_v3))
						n += " " + s
						if truly(ctx, ex_closure{}) {
							s := fmt.Sprintf(".test.%s.x.%s.y.%s.z", _v1, _v2, _v3)
							if d := j.def(ctx, s); d != nil {
								for _, v := range merge(d.value) {
									t += " " + ts(v)
								}
							}
						} else if truly(ctx, ex_delegate{}) {
							t += fmt.Sprintf(" {=closure %s}", s)
						}
					}
				}
			}
			if truly(ctx, ex_closure{}, ex_delegate{}) {
				if s, t := ts(x), "{=list"+n+"}"; s != t {
					erro(pc(ctx,x), "%v: %s != %s", x, s, t).trace()
				}
			} else {
				if s, t := ts(x), "{=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}} {=punct .} {=word x} {=punct .} {=delegate {=auto 2}} {=punct .} {=word y} {=punct .} {=delegate {=auto 3}} {=punct .} {=word z}}"; s != t {
					erro(pc(ctx,x), "%v: %s != %s", x, s, t).trace()
				}
			}
			if s, t := ts(res), "{=list"+t+"}"; s != t {
				note(pc(ctx,x), "%s", s)
				note(pc(ctx,x), "%s", t)
				erro(pc(ctx,x), "%v", res).trace()
			}
		} else if s, t := ts(res), "{=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}} {=punct .} {=word x} {=punct .} {=delegate {=auto 2}} {=punct .} {=word y} {=punct .} {=delegate {=auto 3}} {=punct .} {=word z}}}"; s != t {
			erro(pc(ctx,x), "%v: %s != %s", res, s, t).trace()
		}
	case "&(.test.a.x.1.y.0.z)":
		if truly(ctx, ex_closure{}) {
			if s, t := ts(res), "{=list {=word ax} {=word ay} {=word az}}"; s != t {
				erro(pc(ctx,x), "%v: %s != %s", res, s, t).trace()
			}
		} else {
			if s, t := ts(res), "{=closure {=compound {=punct .} {=word test} {=punct .} {=word a} {=punct .} {=word x} {=punct .} {=word 1} {=punct .} {=word y} {=punct .} {=word 0} {=punct .} {=word z}}}"; s != t {
				erro(pc(ctx,x), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "&(.test.a.x.2.y.0.z)":
		if truly(ctx, ex_closure{}) {
			if s, t := ts(res), "{}"; s != t {
				erro(pc(ctx,x), "%v: %s != %s", res, s, t).trace()
			}
		} else {
			if s, t := ts(res), "{=closure {=compound {=punct .} {=word test} {=punct .} {=word a} {=punct .} {=word x} {=punct .} {=word 2} {=punct .} {=word y} {=punct .} {=word 0} {=punct .} {=word z}}}"; s != t {
				erro(pc(ctx,x), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "&(.test.a.x.3.y.0.z)":
		if truly(ctx, ex_closure{}) {
			if s, t := ts(res), "{}"; s != t {
				erro(pc(ctx,x), "%v: %s != %s", res, s, t).trace()
			}
		} else {
			if s, t := ts(res), "{=closure {=compound {=punct .} {=word test} {=punct .} {=word a} {=punct .} {=word x} {=punct .} {=word 3} {=punct .} {=word y} {=punct .} {=word 0} {=punct .} {=word z}}}"; s != t {
				erro(pc(ctx,x), "%v: %s != %s", res, s, t).trace()
			}
		}
	case "&(.test.b.x.1.y.0.z)":
	case "&(.test.b.x.2.y.0.z)":
	case "&(.test.b.x.3.y.0.z)":
	case "&(.test.c.x.1.y.0.z)":
	case "&(.test.c.x.2.y.0.z)":
	case "&(.test.c.x.3.y.0.z)":
	default:
		erro(pc(ctx,p), "%v → %v : %v", p, res, ts(res)).trace()
    }
}
func (p *closure) expand___foreach(ctx Context, x, res Value) {
	switch p.String() {
	case "&(.test.$_)":
		u := auto_get(ctx, "_")
		switch {
		case truly(ctx, ex_closure{}):
			if u == nil {
				erro(pc(ctx,p), "%v %v", p, res).trace()
			}
			if s, t := res.String(), fmt.Sprintf("&(.test.%s)", u); s != t {
				erro(pc(ctx,p), "%s != %s ; %s", s, t, ts(res)).trace()
			}
		case truly(ctx, ex_delegate{}):
			if u == nil {
				erro(pc(ctx,p), "%v %v", p, res).trace()
			}
			if s, t := res.String(), fmt.Sprintf("&(.test.%s)", u); s != t {
				erro(pc(ctx,p), "%s != %s ; %s", s, t, ts(res)).trace()
			}
		default:
			if s, t := res.String(), "&(.test.$_)"; s != t {
				erro(pc(ctx,p), "%s != %s ; %s", s, t, ts(res)).trace()
			}
		}
	case "&(.test.$_.$(or $4,$3))":
		if u := auto_get(ctx, "_"); u == nil {
			erro(pc(ctx,p), "%v %v", p, res).trace()
		}
	case "&(.test.x $_)":
		u := auto_get(ctx, "_")
		if u == nil {
			erro(pc(ctx,p), "%v %v", p, res).trace()
		}
		switch u.String() {
		case "{do.smart}":
			switch {
			case truly(ctx, ex_closure{}):
				if s, t := res.String(), "{=file do.smart}"; s != t {
					erro(pc(ctx,p), "%s != %s ; %s", s, t, ts(res)).trace()
				}
			case truly(ctx, ex_delegate{}):
				if s, t := res.String(), "&(.test.x do.smart)"; s != t {
					erro(pc(ctx,p), "%s != %s ; %s", s, t, ts(res)).trace()
				}
			default:
				erro(pc(ctx,p), "%v %v → %v", p, u, res).trace()
			}
		default:
			erro(pc(ctx,p), "%v %v → %v", p, u, res).trace()
		}
	}
}
func (p *closure) expand_rule_shell_forstdout(ctx Context, x, res Value) {
	var o = try[origin](ctx, get_origin{})
	switch o {
	case defExpand0, defExpand1, 0:
	default:
		errostack(ctx, 5, "untested: %v %s", o, ts(res)).trace()
	}
}

func (p *globpat) cmp_check(ctx Context, v Value, _res *cmpres) {
    if res := *_res; res != cmpEqual && p.String() == v.String() {
        erro(pc(ctx,p), "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
    }
}

func (p *regexpat) cmp_check(ctx Context, v Value, _res *cmpres) {
    if res := *_res; res != cmpEqual && p.String() == v.String() {
        erro(pc(ctx,p), "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
    }
}

func (p *project) unmap_files_check(ctx Context, _k any, res *[]filemap_name) {
	switch p.name {
	case "configure.base":
		if x, y := _k.(Value); y && *res == nil && truly(ctx, is_modify{}) {
			var s = x.string(ctx)
			if strings.HasPrefix(s, ".configure/library/") && strings.HasSuffix(s, ".x") {
				erro(ctx, "%s %v %s", typeof(_k), _k, s).trace()
			}
		}

	case "testllvmconfig":
		var k string
		switch x := _k.(type) {
		case    Value: k = x.String()
		case   string: k = x
		case []string: k = filepath.Join(x...)
		default: erro(ctx, "%v", ts(_k)).trace()
		}
		switch k {
		case "llvm/Config/llvm-config.h.cmake":
			var srcinc string
			if d := p.def(ctx, "srcinc"); d == nil {
				erro(ctx, "%v %v %v", p.name, typeof(k), k).trace()
			} else {
				srcinc = d.string(ctx)
			}
			if n := len(*res); n == 0 {
				erro(ctx, "%v %v %v", p.name, typeof(k), k).trace()
			} else if t := (*res)[0]; t.name != k {
				erro(ctx, "%v %v != %v", typeof(k), k, t.name).trace()
			} else if x, y := t.pattern.(*file); !y {
				erro(ctx, "%v %v != %v", typeof(k), k, t.pattern).trace()
			} else if x.dir != srcinc {
				erro(ctx, "%s != %s", x.dir, srcinc).trace()
			}
		case "llvm/Config/llvm-config.h":
			var outinc string
			if d := p.def(ctx, "outinc"); d == nil {
				erro(ctx, "%v %v %v", p.name, typeof(k), k).trace()
			} else {
				outinc = d.string(ctx)
			}
			if n := len(*res); n == 0 {
				erro(ctx, "%v %v %v", p.name, typeof(k), k).trace()
			} else if t := (*res)[0]; t.name != k {
				erro(ctx, "%v %v != %v", typeof(k), k, t.name).trace()
			} else if x, y := t.pattern.(*file); !y {
				erro(ctx, "%v %v != %v", typeof(k), k, t.pattern).trace()
			} else if false && x.dir != outinc {
				erro(ctx, "%s != %s", x.dir, outinc).trace()
			}
		}
	}
}

func select_file_1_check(ctx Context, m filemap_name, _res **file) {
	if f := *_res; f != nil {
		switch p := _project(ctx); f.name {
		case "llvm/Config/llvm-config.h.cmake":
			s := p.def(ctx, "srcinc").string(ctx)
			if x, y := m.pattern.(*file); !y {
				erro(ctx, "%v", ts(m.pattern)).trace()
			} else if f.name != x.name {
				erro(ctx, "%s != %s", f.name, x.name).trace()
			} else if f.dir != x.dir {
				erro(ctx, "%s != %s", f.dir, x.dir).trace()
			} else if x.dir != s {
				erro(ctx, "%s != %s", x.dir, s).trace()
			} else if f.dir != s {
				erro(ctx, "%s != %s", f.dir, s).trace()
			}
		case "llvm/Config/llvm-config.h":
			s := p.def(ctx, "outinc").string(ctx)
			if x, y := m.pattern.(*file); !y {
				erro(ctx, "%v", ts(m.pattern)).trace()
			} else if f.name != x.name {
				erro(ctx, "%s != %s", f.name, x.name).trace()
			} else if f.dir != s {
				erro(ctx, "%s != %s", f.dir, s).trace()
			} else if false && x.dir != s {
				erro(ctx, "%s != %s", x.dir, s).trace()
			}
		}
	}
}

func select_files_check(ctx Context, m []filemap_name, res *[]*file) {
	if *res == nil { return }

	var p = _project(ctx)

	switch f := (*res)[0]; f.name {
	case "llvm/Config/llvm-config.h.cmake":
		if s := p.def(ctx, "srcinc").string(ctx); f.dir != s {
			erro(ctx, "%s != %s", f.dir, s).trace()
		}
	case "llvm/Config/llvm-config.h":
		if s := p.def(ctx, "outinc").string(ctx); f.dir != s {
			erro(ctx, "%s != %s", f.dir, s).trace()
		}
	}
}

func (a as) file_check(ctx Context, projs []*project, v Value, f **file) {
	if len(projs) == 0 { projs = []*project{ _project(ctx) } }

	var p = projs[0]

	if *f == nil {
		var s = v.string(ctx)
		if s == "" {
			return // note(ctx, "as.file %v", ts(v)).debug()
		}
		if f := findfile(ctx, s, projs...); f != nil {
			for _, m := range unmap_files(ctx, f) {
				erro(ctx, "FIXME: %v (%s) ⇒ %v", v, s, m)
			}
			erro(ctx, "FIXME: %v (%s) ⇒ %v (%s)", v, s, f, f.fullname())
			errostack(ctx, 5).trace()
		}
	} else if (*f).name == "llvm/Config/llvm-config.h.cmake" {
		if s := p.def(ctx, "srcinc").string(ctx); (*f).dir != s {
			erro(ctx, "%s != %s", (*f).dir, s).trace()
		}
	} else if (*f).name == "llvm/Config/llvm-config.h" {
		if s := p.def(ctx, "outinc").string(ctx); (*f).dir != s {
			erro(ctx, "%s != %s", (*f).dir, s).trace()
		}
	}
}

func (a as) fullname_check(ctx Context, projs []*project, t Value, res fullname) {
	if len(projs) == 0 { projs = []*project{ _project(ctx) } }

	var p = projs[0]
	var s = t.string(ctx)

	if res.Value == nil {
		var v = a.Value
		var u = p.unmap_files(ctx, s, nil)
		if 0 < len(u) {
			if t := select_files(ctx, u); t == nil {
				erro(ctx, "%s {=%s %s} %v %v", p.name, typeof(v), v, u, t).trace()
			} else {
				erro(ctx, "%s {=%s %s} %v", p.name, typeof(v), v, u).trace()
			}
		}
	} else {
		switch p.name {
		case "testllvmconfig":
			if x, y := res.Value.(*file); !y {
				erro(ctx, "%v", res.Value).trace()
			} else if x.name == "llvm/Config/llvm-config.h.cmake" {
				if s := p.def(ctx, "srcinc").string(ctx); x.dir != s {
					erro(ctx, "%s != %s", x.dir, s).trace()
				}
			} else if x.name == "llvm/Config/llvm-config.h" {
				if s := p.def(ctx, "outinc").string(ctx); x.dir != s {
					erro(ctx, "%s != %s", x.dir, s).trace()
				}
			}
		}
	}
}

func (f flag) hit_check(ctx Context, c *valcache, _res **valcache, fullmatch *bool) {
	switch p, res := _project(ctx), *_res; p.name {
	case "configure.base":
		if cacheMapping(ctx) && res == nil {
			erro(ctx, "%v %v", res, c.ks(true)).trace()
		}

		var v = f.Value
		var k = v.String()
		var cc, y = p.entries.puncs[MINUS]
		if !y {
			if !cacheMapping(ctx) { break }
			erro(ctx, "%v: %v", p.name, v).trace()
		}

		var ss = cc.keys()
		if len(ss) == 0 {
			erro(ctx, "%v: %v", p.name, v).trace()
		}

		if _, y := ss[k]; y {
			if false && res.String() != k {
				erro(ctx, "%v: %v != %v", p.name, (res), v).trace()
			}
		} else if cacheMapping(ctx) {
			erro(ctx, "%v: %v", p.name, v).trace()
		}
	}
}

func (p *builtin) evoke_check(ctx *evocation, res *Value) {
	var j = _project(ctx)

	switch p.name {
	case "print":
	case "grep":
	}

	if j.name == "configure.base" && p.name == "file" {
		switch (*res).(type) {
		case fullfile:
			if len(ctx.a) == 1 && truly(ctx, is_compound{}) {
				errostack(ctx, 8, "unexpected fullfile: %v → %v", ts(ctx.a[0]), ts(*res)).trace()
			}
		case *file:
			if len(ctx.a) == 1 {
				if truly(ctx, is_compound{}) {
					// note(ctx, "%v → %v", ts(ctx.a[0]), ts(*res)).debug()
				} else if truly(ctx, is_exec{}) {
					errostack(ctx, 8, "expected fullfile: %v → %v", ts(ctx.a[0]), ts(*res)).trace()
				}
			}
		default:
			for _, a := range ctx.a {
				if x, y := a.(*list); !y {
					errostack(pc(ctx,a), 8, "%v ; %v", ts(a), ts(*res)).trace()
				} else if x.len() != 1 {
					errostack(pc(ctx,a), 8, "%v ; %v", ts(x.elems), ts(*res)).trace()
				} else {
					a = x.elems[0]
				}

				var s = a.string(ctx)

				if strings.HasPrefix(ts(a), "{=compound {=fullfile .configure/") {
					if filepath.IsAbs(s) {
						errostack(pc(ctx,a), 8, "%v ; %v", ts(a), ts(*res)).trace()
					}
				}

				if x, y := a.(*compound); y {
					if strings.Contains(ts(a), "{=fullfile .configure/") {
						if strings.Contains(s, "/Volumes/workout/") {
							errostack(pc(ctx,a), 8, "%v ; %v", ts(a), ts(*res)).trace()
						}
					}
				} else if x != nil && x.len() == 0 {
					errostack(pc(ctx,a), 8, "%v ; %v", ts(a), ts(*res)).trace()
				}

				if f := j.file(ctx, s); f == nil {
					if strings.Contains(s, ".configure/") {
						if strings.HasSuffix(s, ".x") || strings.HasSuffix(s, ".o") {
							errostack(pc(ctx,a), 8, "not a file: %v ; %v", ts(a), ts(*res)).trace()
						}
					}
				}
			}
		}
	}

	if truly(ctx, is_exec{}) {
		switch j.name {
		case "configure.base":
			var e = _entry(ctx)
			if e == nil {
				errostack(ctx, 8, "%v %v → %v", ctx.x, ctx.a, *res).trace()
			}
			switch e.destiny().string(ctx) {
			case "-compiles-c", "-library-c", "-symbol-c":
				switch p.name {
				case "file":
					if len(ctx.a) == 1 {
						var s = ctx.a[0].String()
						if strings.HasPrefix(s, "{=file .configure/") {
							if strings.HasSuffix(s, ".c}.x") {
								if _, y := (*res).(fullfile); !y {
									errostack(ctx, 8, "not a fullfile: %v %v → %v", ctx.x, ctx.a[0], *res).trace()
								}
							}
						}
					}
				}
			}
		}
	}
}

func (d *def) set_check(ctx Context, o origin, val Value, app []Value) {
	switch j := _project(ctx); j.spec {
	case "testdata/value/auto":
		switch d.name {
		case "foo1":
			if s, t := ts(d.value), "{=null}"; s != t {
				erro(pc(ctx,d.value), "%v : %s != %s", d, s, t).trace()
			}
		}
	case "testdata/value/optional":
		switch d.name {
		case "v0":
			if s, t := ts(d.value), "{=delegate {=project foo}}"; s != t {
				erro(ctx, "%v: %v: %s != %s", o, val, s, t).trace()
			}
		case "v1":
			if s, t := ts(d.value), "{=delegate {=cond {=word name}}}"; s != t {
				erro(ctx, "%v: %v: %s != %s", o, val, s, t).trace()
			}
		case "v2":
			if s, t := ts(d.value), "{=delegate {=valed {=self foo}}}"; s != t {
				erro(ctx, "%v: %v: %s != %s", o, val, s, t).trace()
			}
		case "v3":
			if s, t := ts(d.value), "{=delegate {=cond {=arrow {=project foo}→{=word baz}}}}"; s != t {
				erro(ctx, "%v: %v: %s != %s", o, val, s, t).trace()
			}
		case "v4":
			if s, t := ts(d.value), "{=delegate {=arrow {=cond {=word fo}}→{=word bar}}}"; s != t {
				erro(ctx, "%v: %v: %s != %s", o, val, s, t).trace()
			}
		case "v5":
			if s, t := ts(d.value), "{=delegate {=cond {=arrow {=cond {=word fo}}→{=word bar}}}}"; s != t {
				erro(ctx, "%v: %v: %s != %s", o, val, s, t).trace()
			}
		case "v6":
			if s, t := ts(d.value), "{=delegate {=builtin foreach} {=list {=delegate {=project foo}}} {=list {=delegate {=arrow {=delegate {=auto _}}→{=word name}}}}}"; s != t {
				erro(ctx, "%v: %v: %s != %s", o, val, s, t).trace()
			}
		case "v7":
			if s, t := ts(d.value), "{=delegate {=builtin foreach} {=list {=delegate {=project foo}}} {=list {=delegate {=cond {=arrow {=delegate {=auto _}}→{=word name}}}}}}"; s != t {
				erro(ctx, "%v: %v: %s != %s", o, val, s, t).trace()
			}
		case "v8":
			if s, t := ts(d.value), "{=delegate {=builtin foreach} {=list {=delegate {=project foo}}} {=list {=delegate {=cond {=arrow {=delegate {=auto _}}→{=word bar}}}}}}"; s != t {
				erro(ctx, "%v: %v: %s != %s", o, val, s, t).trace()
			}
		case "v9":
			if s, t := ts(d.value), "{=delegate {=cond {=arrow {=arrow {=project foo}→{=word name}}→{=word xxxx}}}}"; s != t {
				erro(ctx, "%v: %v: %s != %s", o, val, s, t).trace()
			}
		case "v10", "v11", "v12":
			if s, t := ts(d.value), "{=delegate {=valed {=answer yes}}}"; s != t {
				erro(ctx, "%v %v: %v: %s != %s", d.name, o, val, s, t).trace()
			}
		case "val0":
			if s, t := ts(d.value), "{=project foo}"; s != t {
				erro(ctx, "%v %v: %v: %s != %s", d.name, o, val, s, t).trace()
			}
		case "val2", "val6", "val7":
			if s, t := ts(d.value), "{=self foo}"; s != t {
				erro(ctx, "%v %v: %v: %s != %s", d.name, o, val, s, t).trace()
			}
		case "val1", "val3", "val4", "val5", "val8", "val9":
			if s, t := ts(d.value), "{=null}"; s != t {
				erro(ctx, "%v %v: %v: %s != %s", d.name, o, val, s, t).trace()
			}
		case "val10", "val11", "val12":
			if s, t := ts(d.value), "{=answer yes}"; s != t {
				erro(ctx, "%v %v: %v: %s != %s", d.name, o, val, s, t).trace()
			}
		}
	case "testdata/value":
		switch d.name {
		case "val4":
			if s, t := val.String(), `a\,b\,c,x\,y\,z`; s != t {
				erro(ctx, "%v: %s != %s : %v", o, s, t, ts(val)).trace()
			}
			if s, t := d.value.String(), `a\,b\,c,x\,y\,z`; s != t {
				erro(ctx, "%v: %s != %s : %v", o, s, t, ts(d.value)).trace()
			}
		case "val5":
			if s, t := val.String(), `$(quote a\,b\,c,x\,y\,z)`; s != t {
				erro(ctx, "%v: %s != %s : %v", o, s, t, ts(val)).trace()
			}
			if s, t := d.value.String(), `{=quote a\,b\,c x\,y\,z}`; s != t {
				erro(ctx, "%v: %s != %s : %v", o, s, t, ts(d.value)).trace()
			}
		}
	case "testdata/value/2":
		switch d.name {
		case ".test.ab":
			if s, t := ts(d.value), "{=compound {=word foo_ab} {=flag {=compound {=delegate {=auto 1}} {=flag {=delegate {=auto 2}}}}}}"; s != t {
				erro(pc(ctx,d.value), "%v : %s != %s", d.value, s, t).trace()
			}
		case ".test.ba":
			if s, t := ts(d.value), "{=compound {=word foo_ba} {=flag {=compound {=delegate {=auto 2}} {=flag {=delegate {=auto 1}}}}}}"; s != t {
				erro(pc(ctx,d.value), "%v : %s != %s", d.value, s, t).trace()
			}
		case ".test.0":
			if val != nil && app == nil {
				if s, t := ts(d.value), "{=list {=decimal 1} {=delegate {=builtin closure} {=list {=closure {=compound {=punct .} {=word test} {=punct .} {=word x}}}}} {=decimal 10}}"; s != t {
					erro(pc(ctx,d.value), "%v : %s != %s", d.value, s, t).trace()
				}
			}
			if val == nil && app != nil {
				a0 := app[0].String()
				t := "{=list {=decimal 1} {=delegate {=builtin closure} {=list {=closure {=compound {=punct .} {=word test} {=punct .} {=word x}}}}} {=decimal 10}"
				t += " {=decimal 2} {=delegate {=closure {=compound {=punct .} {=word test} {=punct .} {=word x}}} {=list {=compound {=delegate {=auto 1}} {=delegate {=auto 1}}}} {=list {=compound {=delegate {=auto 2}} {=delegate {=auto 2}}}}} {=decimal 20}"
				if a0 == "2" {
					if s, t := ts(d.value), t+"}"; s != t {
						erro(pc(ctx,app[0]), "%v : %s != %s ; %v", d.value, s, t, app).trace()
					}
				}
				t += " {=decimal 3} {=delegate {=builtin call} {=list {=closure {=def .test.x}}} {=list {=compound {=delegate {=auto 1}} {=delegate {=auto 2}}}} {=list {=compound {=delegate {=auto 2}} {=delegate {=auto 1}}}}} {=delegate {=builtin call} {=[] {=flag {=word closure}}} {=list {=closure {=def .test.x}}} {=list {=compound {=delegate {=auto 1}} {=delegate {=auto 2}}}} {=list {=compound {=delegate {=auto 2}} {=delegate {=auto 1}}}}}"
				if a0 == "3" {
					if s, t := ts(d.value), t+"}"; s != t {
						note(pc(ctx,app[0]), "%s", s)
						note(pc(ctx,app[0]), "%s", t)
						erro(pc(ctx,app[0]), "%v : %s != %s ; %v", d.value, s, t, app).trace()
					}
				}
				t += " {=decimal 4} {=delegate {=auto 3}}"
				if a0 == "4" {
					if s, t := ts(d.value), t+"}"; s != t {
						erro(pc(ctx,app[0]), "%v : %s != %s ; %v", d.value, s, t, app).trace()
					}
				}
			}
		case ".test.1":
		case ".test.2":
		}
	}

	if isNull(d.value) {
		if truly(ctx, ex_def{}) && (app != nil || val != nil) {
			erro(ctx, "%v ; %v %v", d, val, app).trace()
		}
		if o == defExpand0 && (app != nil || !isNull(val)) {
			erro(ctx, "%v ; %v %v", d, val, app).trace()
		}
	}

	if !d.position.valid() && d.name != ".goals" {
		erro(ctx, "%v ; %v %v", d, val, app).trace()
	}
}

func (d *def) evoke_check(ctx *evocation, _res *Value, t time.Time) {
	var j = _project(ctx)
	if u := time.Since(t); u > 2*time.Second {
		notestack(pc(ctx,d.value), 1, "%v: %v; %v", j.name, u, *_res).debug(32)
	}
	switch j.name {
	case "configure.base":
		d.evoke_check_configure_base(ctx, j, *_res)
	}
	switch j.spec {
	case "testdata/value":
		d.evoke_check_testvalue(ctx, j, *_res)
	case "testdata/builtins/foreach":
		d.evoke___foreach(ctx, j, *_res)
	}
}
func (d *def) evoke_check_configure_base(ctx *evocation, j *project, res Value) {
	switch dest := _entry(ctx).destiny().string(ctx); dest {
	case "-cc", "-cxx", "-compiles-c", "-compiles-c++", "-library-c", "-library-c++", "-symbol-c", "-symbol-c++", "-function-c", "-function-c++", "-type-c", "-type-c++", "-variable-c", "-variable-c++", "-struct-member-c", "-struct-member-c++", "-headers-c", "-headers-c++":
		switch d.name {
		case "name":
			if t := auto_find(ctx, "TARGET"); t == nil {
				erro(ctx, "TARGET is nil: %s, %v → %v (%T)", dest, d, ts(res), res).trace()
			}
			if _, y := res.(*path); !y {
				errostack(ctx, 8, "not a path: %s, %v → %v (%T)", dest, d, ts(res), res).trace()
			}
		case "x", "o", "s":
			if !truly(ctx, is_compound{}) && truly(ctx, is_exec{}) {
				if _, y := res.(fullfile); !y {
					errostack(ctx, 8, "not a fullfile: %s, %v → %v (%T)", dest, d, ts(res), res).trace()
				}
			} else {
				if _, y := res.(*file); !y {
					errostack(ctx, 8, "not a file: %s, %v → %v (%T)", dest, d, ts(res), res).trace()
				}
			}
			if s := d.string(ctx); s == "" {
				errostack(ctx, 8, "empty: %s, %v → %v (%T)", dest, d, s, res).trace()
			}
		case "@":
			if _, y := res.(fullfile); !y {
				errostack(ctx, 8, "not a fullfile: %s, %v → %v (%T)", dest, d, ts(res), res).trace()
			}
			if s := d.string(ctx); s == "" {
				errostack(ctx, 8, "empty: %s, %v → %v (%T)", dest, d, s, res).trace()
			}
		}
	case "-feature-c", "-feature-c++", "-sizeof-c", "-sizeof-c++", "-alignof-c", "-alignof-c++":
		// if _, y := res.(fullfile); !y {
		// 	note(ctx, "%v", auto_get(ctx, "@"))
		// 	note(ctx, "%v", auto_get(ctx, "<"))
		// 	note(ctx, "%v", auto_get(ctx, ">"))
		// 	errostack(ctx, 8, "not a fullfile: %s, %v → %v (%T)", dest, d, ts(res), res).trace()
		// }
	case "-program-stdout", "-program-stderr", "-program-status":
		// if _, y := res.(fullfile); !y {
		// 	note(ctx, "%v", auto_get(ctx, "@"))
		// 	note(ctx, "%v", auto_get(ctx, "<"))
		// 	note(ctx, "%v", auto_get(ctx, ">"))
		// 	errostack(ctx, 8, "not a fullfile: %s, %v → %v (%T)", dest, d, ts(res), res).trace()
		// }
	}
}
func (d *def) evoke_check_testvalue(ctx *evocation, j *project, res Value) {
	switch d.name {
	case "disjunction0":
	case "disjunction00":
	case "disjunction01":
		if a := auto_get(ctx, "1"); a == nil {
			erro(pc(ctx,d.value), "$1 is nil : %s", ts(res)).trace()
		} else {
			switch v := a.(type) {
			case *list:
				var t string
				for i, e := range v.elems {
					if 0 < i { t += " " }
					t += "x"+e.String()
				}
				if s := res.String(); s != t {
					erro(pc(ctx,d.value), "%s != %s : %s, %s", s, t, ts(a), ts(res)).trace()
				}
			default:
				erro(pc(ctx,d.value), "%s, %s : %s, %s", a, res, ts(a), ts(res)).trace()
			}
		}
	case "disjunction02":
	case "disjunction1":
	case "disjunction2":
	case "disjunction3":
	}
}
func (d *def) evoke___foreach(ctx *evocation, j *project, res Value) {
	var t string
	switch d.name {
	case "_", "1", "2", "3", "4":
	case ".test.x", ".test.x.a", ".test.x.b", ".test.y.a", ".test.y.b", ".test.z":
	case ".test.h":
		if truly(ctx, ex_closure{}) {
			t = "-"
		} else {
			t = "&(.test.h)"
		}
		if s := res.String(); s != t {
			erro(pc(ctx,d.value), "%s != %s : %v, %s", s, t, ctx.a, ts(res)).trace()
		}
	case ".test.1":
		if s, t := d.value.String(), "x $1 $2 $3 $4 $(foreach $1,&(.test.h)$_)"; s != t {
			erro(pc(ctx,d.value), "%s != %s : %v, %s", s, t, ctx.a, ts(res)).trace()
		}
		switch len(ctx.a) {
		case 4:
			if "a b c,X,Y,Z" == fmt.Sprintf("%v,%v,%v,%v", ctx.a[0], ctx.a[1], ctx.a[2], ctx.a[3]) {
				if truly(ctx, ex_closure{}) {
					t = `x a b c X Y Z -a -b -c`
				} else {
					t = `x a b c X Y Z &(.test.h)a &(.test.h)b &(.test.h)c`
				}
			}
		}
		if s := res.String(); s != t {
			erro(pc(ctx,d.value), "%s != %s : %v, %s", s, t, ctx.a, ts(res)).trace()
		}
	case ".test.2":
		if s, t := d.value.String(), "x $(foreach q p $(foreach $1,&(.test.h)$_),x$_)"; s != t {
			erro(pc(ctx,d.value), "%s != %s : %v, %s", s, t, ctx.a, ts(res)).trace()
		}
		switch len(ctx.a) {
		case 4:
			if "a b c,X,Y,Z" == fmt.Sprintf("%v,%v,%v,%v", ctx.a[0], ctx.a[1], ctx.a[2], ctx.a[3]) {
				if truly(ctx, ex_closure{}) {
					t = `x xq xp x-a x-b x-c`
				} else {
					t = `x xq xp x{&(.test.h)}a x{&(.test.h)}b x{&(.test.h)}c`
				}
			}
		}
		if s := res.String(); s != t {
			erro(pc(ctx,d.value), "%s != %s : %v, %s", s, t, ctx.a, ts(res)).trace()
		}
	case ".test.21":
		if s, t := d.value.String(), "x xq xp"; s != t {
			erro(pc(ctx,d.value), "%s != %s : %v, %s", s, t, ctx.a, ts(res)).trace()
		} else if s := res.String(); s != t {
			erro(pc(ctx,d.value), "%s != %s : %v, %s", s, t, ctx.a, ts(res)).trace()
		}
	case ".test.22":
		if s, t := d.value.String(), "x x-"; s != t {
			erro(pc(ctx,d.value), "%s != %s : %v, %s", s, t, ctx.a, ts(res)).trace()
		} else if s := res.String(); s != t {
			erro(pc(ctx,d.value), "%s != %s : %v, %s", s, t, ctx.a, ts(res)).trace()
		}
	case ".test.23":
		if s, t := d.value.String(), "$(foreach q p $(foreach $1,&(.test.xx)$_),x$_)"; s != t {
			erro(pc(ctx,d.value), "%s != %s : %v, %s", s, t, ctx.a, ts(res)).trace()
		}
		switch len(ctx.a) {
		case 1:
			switch fmt.Sprintf("%v", ctx.a[0]) {
			case "a b c":
				switch {
				case truly(ctx, ex_closure{}):
					if s, t := res.String(), `xq xp xa xb xc`; s != t {
						erro(pc(ctx,d.value), "%s: %s : %s", d.name, s, ts(res)).trace()
					}
				case truly(ctx, ex_delegate{}):
					if s, t := res.String(), `xq xp x{&(.test.xx)}a x{&(.test.xx)}b x{&(.test.xx)}c`; s != t {
						erro(pc(ctx,d.value), "%s: %s : %s", d.name, s, ts(res)).trace()
					}
				default:
					if s, t := res.String(), `xq xp x{&(.test.xx)}a x{&(.test.xx)}b x{&(.test.xx)}c`; s != t {
						erro(pc(ctx,d.value), "%s: %s : %s", d.name, s, ts(res)).trace()
					}
				}
			case "aa bb cc":
				switch {
				case truly(ctx, ex_closure{}):
					if s, t := res.String(), `xq xp xaa xbb xcc`; s != t {
						erro(pc(ctx,d.value), "%s: %s : %s", d.name, s, ts(res)).trace()
					}
				case truly(ctx, ex_delegate{}):
					if s, t := res.String(), `xq xp x{&(.test.xx)}a x{&(.test.xx)}b x{&(.test.xx)}c`; s != t {
						erro(pc(ctx,d.value), "%s: %s : %s", d.name, s, ts(res)).trace()
					}
				default:
					if s, t := res.String(), `xq xp x{&(.test.xx)}a x{&(.test.xx)}b x{&(.test.xx)}c`; s != t {
						erro(pc(ctx,d.value), "%s: %s : %s", d.name, s, ts(res)).trace()
					}
				}
			}
		}
	case ".test.3":
		if s, t := d.value.String(), "x $(foreach $1,&(.test.$_)$1{}88) y $(foreach $1,$(closure .test.$_)$1{}99) z"; s != t {
			erro(pc(ctx,d.value), "%s != %s : %v, %s", s, t, ctx.a, ts(res)).trace()
		}
		switch len(ctx.a) {
		case 1:
			switch fmt.Sprintf("%v", ctx.a[0]) {
			case "foo bar":
				switch {
				case truly(ctx, ex_closure{}):
					if s, t := res.String(), `x y z`; s != t {
						erro(pc(ctx,d.value), "%s: %s : %s", d.name, s, ts(res)).trace()
					}
				case truly(ctx, ex_delegate{}):
					if s, t := res.String(), `x &(.test.foo)foo bar88 y &(.test.bar)foo bar88 z`; s != t {
						erro(pc(ctx,d.value), "%s: %s : %s", d.name, s, ts(res)).trace()
					}
				default:
					if res == nil {
						erro(pc(ctx,d.value), "%v → %s ; %v", d.name, ts(res), auto_get(ctx, "1")).trace()
					}
					if s, t := res.String(), `x $(foreach foo bar,&(.test.$_)foo bar88) y $(foreach foo bar,$(closure .test.$_)foo bar99) z`; s != t {
						note(pc(ctx,d), "%v", d)
						erro(pc(ctx,d.value), "%v → %s : %s", d.name, s, ts(res)).trace()
					}
				}
			default:
				erro(pc(ctx,ctx.a), "%v, %v", ctx.a, res).trace()
			}
		case 4:
			switch fmt.Sprintf("%v,%v,%v,%v", ctx.a[0], ctx.a[1], ctx.a[2], ctx.a[3]) {
			case "a b c,X,Y,Z":
				switch {
				case truly(ctx, ex_closure{}):
					if s, t := res.String(), `x xq xp x-a x-b x-c`; s != t {
						erro(pc(ctx,d.value), "%s: %s : %s", d.name, s, ts(res)).trace()
					}
				case truly(ctx, ex_delegate{}):
					if s, t := res.String(), `x xq xp x{&(.test.h)}a x{&(.test.h)}b x{&(.test.h)}c`; s != t {
						erro(pc(ctx,d.value), "%s: %s : %s", d.name, s, ts(res)).trace()
					}
				default:
					if s, t := res.String(), `x $(foreach foo bar,&(.test.$_)foo bar88) y $(foreach foo bar,$(closure .test.$_)foo bar99) z`; s != t {
						note(pc(ctx,d.value), "%v", d)
						erro(pc(ctx,d.value), "%v → %s : %s", d.name, s, ts(res)).trace()
					}
				}
			default:
				erro(pc(ctx,ctx.a), "%v, %v", ctx.a, res).trace()
			}
		default:
			erro(ctx, "%v %v %v", len(ctx.a), ctx.a, res).trace()
		}
	case ".test.4":
		if s, t := d.value.String(), "$(foreach $1 $2,&(.test.$_.$(or $4,$3)))"; s != t {
			erro(pc(ctx,d.value), "%s != %s : %v, %s", s, t, ctx.a, ts(res)).trace()
		}
		switch len(ctx.a) {
		case 4:
			switch fmt.Sprintf("%v,%v,%v,%v", ctx.a[0], ctx.a[1], ctx.a[2], ctx.a[3]) {
			case "a b c,X,Y,Z":
				if truly(ctx, ex_closure{}) {
					if s, t := res.String(), `x xq xp x-a x-b x-c`; s != t {
						erro(pc(ctx,d.value), "%s != %s : %v, %s", s, t, ctx.a, ts(res)).trace()
					}
				} else {
					if s, t := res.String(), `x xq xp x{&(.test.h)}a x{&(.test.h)}b x{&(.test.h)}c`; s != t {
						erro(pc(ctx,d.value), "%s != %s : %v, %s", s, t, ctx.a, ts(res)).trace()
					}
				}
			default:
				erro(ctx, "%v %v", ctx.a, res).trace()
			}
		default:
			erro(ctx, "%v %v %v", len(ctx.a), ctx.a, res).trace()
		}
	case ".test.5":
		if s, t := d.value.String(), "$(foreach do.smart,&(.test.x $_))"; s != t {
			erro(pc(ctx,d.value), "%s != %s : %v, %s", s, t, ctx.a, ts(res)).trace()
		}
		switch len(ctx.a) {
		case 4:
			switch fmt.Sprintf("%v,%v,%v,%v", ctx.a[0], ctx.a[1], ctx.a[2], ctx.a[3]) {
			case "a b c,X,Y,Z":
				if truly(ctx, ex_closure{}) {
					if s, t := res.String(), `x xq xp x-a x-b x-c`; s != t {
						erro(pc(ctx,d.value), "%s != %s : %v, %s", s, t, ctx.a, ts(res)).trace()
					}
				} else {
					if s, t := res.String(), `x xq xp x{&(.test.h)}a x{&(.test.h)}b x{&(.test.h)}c`; s != t {
						erro(pc(ctx,d.value), "%s != %s : %v, %s", s, t, ctx.a, ts(res)).trace()
					}
				}
			default:
				erro(ctx, "%v %v", ctx.a, res).trace()
			}
		default:
			erro(ctx, "%v %v %v", len(ctx.a), ctx.a, res).trace()
		}
	case ".test.6":
		if s, t := d.value.String(), "$1 $2 $3 $9"; s != t {
			erro(pc(ctx,d.value), "%s != %s : %v, %s", s, t, ctx.a, ts(res)).trace()
		}
		switch len(ctx.a) {
		case 4:
			if "a b c,X,Y,Z" == fmt.Sprintf("%v,%v,%v,%v", ctx.a[0], ctx.a[1], ctx.a[2], ctx.a[3]) {
				if truly(ctx, ex_closure{}) {
					if s, t := res.String(), `x xq xp x-a x-b x-c`; s != t {
						erro(pc(ctx,d.value), "%s != %s : %v, %s", s, t, ctx.a, ts(res)).trace()
					}
				} else {
					if s, t := res.String(), `x xq xp x{&(.test.h)}a x{&(.test.h)}b x{&(.test.h)}c`; s != t {
						erro(pc(ctx,d.value), "%s != %s : %v, %s", s, t, ctx.a, ts(res)).trace()
					}
				}
			}
		default:
			erro(ctx, "%v %v %v", len(ctx.a), ctx.a, res).trace()
		}
	case ".test.61":
		if s, t := d.value.String(), "$(.test.61) - $1 $2 $3 $9"; s != t {
			erro(pc(ctx,d.value), "%s != %s : %v, %s", s, t, ctx.a, ts(res)).trace()
		}
		switch len(ctx.a) {
		case 4:
			if "a b c,X,Y,Z" == fmt.Sprintf("%v,%v,%v,%v", ctx.a[0], ctx.a[1], ctx.a[2], ctx.a[3]) {
				if truly(ctx, ex_closure{}) {
					t = `x xq xp x-a x-b x-c`
				} else {
					t = `x xq xp x{&(.test.h)}a x{&(.test.h)}b x{&(.test.h)}c`
				}
			}
		}
		if s := res.String(); s != t {
			erro(pc(ctx,d.value), "%s != %s : %v, %s", s, t, ctx.a, ts(res)).trace()
		}
	case ".test.7":
		if s, t := d.value.String(), "$(foreach a $(foreach $1,&(.test.z)$_{}zz) b,x$_)"; s != t {
			erro(pc(ctx,d.value), "%s != %s : %v, %s", s, t, ctx.a, ts(res)).trace()
		}
		switch len(ctx.a) {
		case 4:
			if "a b c,X,Y,Z" == fmt.Sprintf("%v,%v,%v,%v", ctx.a[0], ctx.a[1], ctx.a[2], ctx.a[3]) {
				if truly(ctx, ex_closure{}) {
					t = `x xq xp x-a x-b x-c`
				} else {
					t = `x xq xp x{&(.test.h)}a x{&(.test.h)}b x{&(.test.h)}c`
				}
			}
		}
		if s := res.String(); s != t {
			erro(pc(ctx,d.value), "%s != %s : %v, %s", s, t, ctx.a, ts(res)).trace()
		}
	default:
		note(ctx, "%v, %v", ctx.a, d)
		erro(ctx, "%v", res).trace()
	}
}

func auto_find_check(ctx Context, name string, d *def) {
	if p := _project(ctx); p != nil {
		switch p.name {
		case "configure.base":
			if false && d != nil && d.name == "TYPE" && d.value != nil {
				switch d.value.String() {
				case "_Bool", "char", "int", "long", "long long":
				default:
					errostack(pc(ctx,d), 8, "%v %v", d.o, d.value).trace()
				}
			}
		}
		switch p.spec {
		case "testdata/value/auto":
			if t := do(ctx, find_auto{name}); d != nil && t == nil {
				var a = _automatic(ctx)
				var m, _ = a.defs[name]
				note(ctx, "%v", ts(ctx))
				note(ctx, "%v", ts(a))
				errostack(ctx, 8, "%v %v %v", name, d, m).trace()
			}
			if false {
				if ed, _ := do(ctx, evoke_def{"foo"}).(*def); ed != nil {
					note(ctx, "%v %s", d, ts(ctx)).debug(32)
				}
			}
		case "testdata/value/placeholder":
			if false {
				if ed, _ := do(ctx, evoke_def{"val1"}).(*def); ed != nil {
					note(ctx, "%v %s", d, ts(ctx)).debug(32)
				}
				if ed, _ := do(ctx, evoke_def{"val2"}).(*def); ed != nil {
					note(ctx, "%v %s", d, ts(ctx)).debug(32)
				}
				if ed, _ := do(ctx, evoke_def{"val3"}).(*def); ed != nil {
					note(ctx, "%v %s", d, ts(ctx)).debug(32)
				}
				if ed, _ := do(ctx, evoke_def{"val4"}).(*def); ed != nil {
					note(ctx, "%v %s", d, ts(ctx)).debug(32)
				}
				if ed, _ := do(ctx, evoke_def{"val5"}).(*def); ed != nil {
					note(ctx, "%v %s", d, ts(ctx)).debug(32)
				}
			}
		}
	}
	return
}

func (ac *automatic) set_check(ctx Context, o origin, name string, val Value, _out **def, _old *Value) {
	if _project(ctx).name == "configure.base" {
		switch name {
		case "@", "<", ">":
			if val.patterned(ctx) {
				errostack(ctx, 3, "%v %v %v %s", o, name, val, ts(val)).trace()
			} else if s := val.String(); strings.Contains(s, "%") {
				errostack(ctx, 3, "%v %v %v %s", o, name, val, ts(val)).trace()
			}
		case "TYPE":
			v := val.String()

			if strings.Contains(v, "$1") {
				errostack(ctx, 8, "%v %v %v %s %v", o, name, val, ac.defs, *_old).trace()
			}

			a := auto_get(ctx, "@").string(ctx)
			s := strings.ToUpper(val.string(ctx))
			s  = strings.Replace(s, " ", "_",  -1)
			s  = strings.Replace(s, "*", "_P", -1)

			switch {
			case a == "-alignof-c":
				s = "ALIGNOF_" + s
				if x, y := ac.defs["TARGET"]; !y || x.value == nil {
					errostack(ctx, 8, "%v %v %v %s", o, name, val, ac.defs).trace()
				} else if s != x.value.String() {
					errostack(ctx, 8, "%v %v %v : %s != %s", o, name, val, x.value, s).trace()
				} else if strings.Contains(v, "$1") {
					errostack(ctx, 8, "%v %v %v : %s %s", o, name, val, x.value, s).trace()
				}
			case a == "-sizeof-c":
				s = "SIZEOF_" + s
				if x, y := ac.defs["TARGET"]; !y || x.value == nil {
					errostack(ctx, 8, "%v %v %v %s", o, name, val, ac.defs).trace()
				} else if s != x.value.String() {
					errostack(ctx, 8, "%v %v %v : %s != %s", o, name, val, x.value, s).trace()
				} else if strings.Contains(v, "$1") {
					errostack(ctx, 8, "%v %v %v : %s %s", o, name, val, x.value, s).trace()
				}
			case !strings.Contains(a, s):
				// @⇒{=file .configure/type/size/SIZEOF__BOOL.c.x}
				// @⇒{=file .configure/type/size/SIZEOF_CHAR.c.x}
				errostack(ctx, 8, "%v %v %v %s %v", o, name, val, ac.defs, *_old).trace()
			}

			switch {
			case strings.Contains(a, ".configure/type/align/ALIGNOF_"):
				// *⇒'align' 'ALIGNOF_CHAR'
				t := auto_get(ctx, "*").string(ctx)
				if !strings.Contains(t, "ALIGNOF_"+s) {
					errostack(ctx, 8, "%v %v %v %s %v", o, name, val, ac.defs, *_old).trace()
				}
			case strings.Contains(a, ".configure/type/size/SIZEOF_"):
				// *⇒'size' 'SIZEOF_CHAR'
				t := auto_get(ctx, "*").string(ctx)
				if !strings.Contains(t, "SIZEOF_"+s) {
					errostack(ctx, 8, "%v %v %v %s %v", o, name, val, ac.defs, *_old).trace()
				}
			}
		}
	}
}

func (ac *automatic) find_auto_check(ctx Context, d *def, name string) {
	if _project(ctx).name == "configure.base" {
		if name == "TYPE" && d.value != nil {
			if s := d.value.string(ctx); s == "_Bool" {
				if x, y := ac.defs["@"]; y {
					s = strings.ToUpper(s)
					s = strings.Replace(s, " ", "_",  -1)
					s = strings.Replace(s, "*", "_P", -1)
					a := x.String()
					switch {
					case strings.Contains(a, "SIZEOF_"+s),
						strings.Contains(a, "ALIGNOF_"+s),
						strings.Contains(a, "HAVE_TYPE_"+s): // okay
					default:
						erro(ctx, "TYPE is incorrect: %s %v", s, x).trace()
					}
				}
			}
		}
	}
}

func (ac *argumented_ctx) init_args_check(ctx Context, args []Value) {
	if false && 0 < len(args) && args[0].String() == "_Bool" {
		notestack(ctx, 3, "%v %v", args, auto_get(ctx, "1")).debug()
	}
	return
}

func (p *argumented) expand_check(ctx Context, res, val Value, args []Value) {
	if j := _project(ctx); j.name == "llvm.Config" {
		if a := auto_get(ctx, "_"); a != nil && len(p.args) == 1 && len(args) == 1 {
			if x1, y := p.Value.(*delegate); y {
				if a1, y := x1.x.(*auto); y && IsDigits(a1.name) {
					if x2, y := val.(*delegate); y {
						if a2, y := x2.x.(*auto); y && a1.name == a2.name {
							if s, t := a.String(), args[0].String(); s != t {
								errostack(ctx, 5, "%v: %v, %v, %s != %s", a, p, val, s, t).trace()
							} else if false {
								notestack(ctx, 5, "%v: %v, %v, %v, %s", a, p, res, val, t).debug(16)
							}
						}
					}
				}
			}
		}
	}
}

func (p *argumented) traverse_check(ctx Context, str string, args []Value) {
	if x, y := ctx.(*execution); y {
		if _project(ctx).name == "configure.base" {
			if p.Value.String() == "$(name).c.x" && len(p.args) == 3 {
				var a = auto_get(&x.automatic, "@")
				var t = auto_get(&x.automatic, "TYPE")
				if p.args[0].String() != "$(TYPE)" {
					errostack(ctx, 8, "%v %v %v %v", t, a, str, args).trace()
				}
				if p.args[1].String() != "$(INCLUDE)" {
					errostack(ctx, 8, "%v %v %v %v", t, a, str, args).trace()
				}
				if p.args[2].String() != "$(LIB)" {
					errostack(ctx, 8, "%v %v %v %v", t, a, str, args).trace()
				}
				if t != nil && t.string(ctx) != p.args[0].string(ctx) {
					if v, y := x.prerequisite.(*argumented); y && v != nil {
						errostack(ctx, 8, "%v %v %v %v %v %v", t, a, (p==v), v.Value, v.args, args).trace()
					}
				}
			}
		}
	}
}
