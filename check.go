//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

import (
	"sort"
	"strings"
	"regexp"
)

// NOTE: cannot decalre `checkpoints` as `const` because it's compile-time evaled.
var checkpoints = true // vertag != "final"

var check_prefix_specs = map[string]map[string]map[string]string{
	"p:testdata/value/13": prefix_value_13,
	"p:testdata/builtins/addprefix": prefix__addprefix,
	"p:testdata/builtins/addsuffix": prefix__addsuffix,
	"s:testdata/value/13": suffix_value_13,
	"s:testdata/builtins/addprefix": suffix__addprefix,
	"s:testdata/builtins/addsuffix": suffix__addsuffix,
}
func check_prefix(ctx Context, tag string, x, y Value, res *Value) {
	if j := _project(ctx); j == nil {
		if false {
			var s = try[string](ctx, source{})
			errostack(pc(pc(ctx,y),x), 8, "nil project | %s %v %v %v", s, ts(x), ts(y), ts(*res)).trace()
		}
	} else if spec, ok := check_prefix_specs[tag+":"+j.spec]; ok {
		var (
			src = strings.Split(try[string](ctx,source{}),":")
			xy = func() (s string) {
				if src[0] != "loader.go" { s = src[1] + " " }
				return s+ts(x)+" "+ts(y)
			} ()
			vs = spec[src[0]][xy]
		)
		if rs := (*res).String()+" "+ts(*res); vs == "" {
			errostack(pc(pc(ctx,y),x), 8, "`%v`:`%s`,", xy, rs).trace()
		} else if rs != vs {
			erro(pc(pc(ctx,y),x), "`%v`", xy)
			note(pc(pc(ctx,y),x), `got: %s`, rs)
			notestack(pc(pc(ctx,y),x), 8, `!= : %s`, vs).trace()
		}
	}
}

type sorted_strings []string

var checkspecs = map[string]map[string]map[string]any{
	"testdata/assert":               checkpoints__assert,
	"testdata/locals":               checkpoints__locals,
	"testdata/builtins/addprefix":   checkpoints__addprefix,
	"testdata/builtins/addsuffix":   checkpoints__addsuffix,
	"testdata/builtins/trimprefix":  checkpoints__trimprefix,
	"testdata/builtins/trimsuffix":  checkpoints__trimsuffix,
	"testdata/builtins/if":          checkpoints__if,
	"testdata/builtins/foreach":     checkpoints__foreach,
	"testdata/builtins/foreach/1":   checkpoints__foreach1,
	"testdata/builtins/foreach/2":   checkpoints__foreach2,
	"testdata/builtins/foreach/3":   checkpoints__foreach3,
	"testdata/builtins/foreach/4":   checkpoints__foreach4,
	"testdata/builtins/foreach/5":   checkpoints__foreach5,
	"testdata/builtins/wildcard":    checkpoints__wildcard,
	"testdata/builtins/closure":     checkpoints__closure,
	"testdata/builtins/delegate":    checkpoints__delegate,
	"testdata/builtins/logic":       checkpoints__logic,
	"testdata/builtins/contains":    checkpoints__contains,
	"testdata/builtins/join":        checkpoints__join,
	"testdata/builtins/xor":         checkpoints__xor,
	"testdata/builtins/or":          checkpoints__or,
	"testdata/value":                checkpoints_value,
	"testdata/value/auto":           checkpoints_value_auto,
	"testdata/value/closure":        checkpoints_value_closure,
	"testdata/value/disjunction":    checkpoints_value_disjunction,
	"testdata/value/placeholder":    checkpoints_value_placeholder,
	"testdata/value/optional":       checkpoints_value_optional,
	"testdata/value/optional/foo":   checkpoints_value_optional_foo,
	"testdata/value/glob":           checkpoints_value_glob,
	"testdata/value/1":              checkpoints_value_1,
	"testdata/value/2/0":            checkpoints_value_20,
	"testdata/value/2/1":            checkpoints_value_21,
	"testdata/value/2/2":            checkpoints_value_22,
	"testdata/value/3":              checkpoints_value_3,
	"testdata/value/4":              checkpoints_value_4,
	"testdata/value/5":              checkpoints_value_5,
	"testdata/value/6":              checkpoints_value_6,
	"testdata/value/7":              checkpoints_value_7,
	"testdata/value/8":              checkpoints_value_8,
	"testdata/value/9":              checkpoints_value_9,
	"testdata/value/10":             checkpoints_value_10,
	"testdata/value/11":             checkpoints_value_11,
	"testdata/value/12":             checkpoints_value_12,
	"testdata/value/13":             checkpoints_value_13,
	"testdata/value/bug_01":         checkpoints_value_bug_01,
	"testdata/rule/shell/for-stdout":checkpoints_rule_shell_forstdout,
	"testdata/template":             checkpoints_template,
	"testdata/modifier":             checkpoints_modifiers,
	"testdata/valcache":             checkpoints_valcache,
	"testdata/valcache/1":           checkpoints_valcache1,
	"testdata/valcache/2":           checkpoints_valcache2,
	"testdata/valcache/3":           checkpoints_valcache3,
}
func check(ctx Context, res Value, p Value, x ...Value) {
	var src = strings.Split(try[string](ctx,source{}), ":")
	if src == nil || (len(src) == 1 && src[0] == "") {
		if true {
			switch p.String() {
			case "test.paniconexit0", "test.timeout": // TODO: fix these strange value
				if false { erro(pc(ctx,p), "TODO: check(%v %v %v)", p, res, x).trace() }
			default:
				if false { note(pc(ctx,p), "check(%v %v %v)", p, res, x).debug(10) }
			}
		}
		return
	}

	var (
		spec string
		d, _ = do(ctx, origin_def{}).(*def)
		dn = func() (s string) { if d != nil { s = d.name }; return } ()
		ks = func() (s string) {
			if 2 == len(src) { if src[0] != "loader.go" { s = src[1] + " " }}
			if d != nil { s += line_column(d)+":" }
			if dn != "" { s += dn + " "  }
			s += p.String()+" "+ts(p)
			return
		} ()
		tr = func() (a any) {
			if spec == "" { if j := _project(ctx); j != nil { spec = j.spec } }
			return checkspecs[spec][src[0]][ks]
		} ()
		vs = func() (s string) {
			if n := len(x); n == 1 { s = ts(x[0],ctx)+" " } else if n > 1 { s = ts(x,ctx)+" " }
			switch t := tr.(type) {
			case sorted_strings:
				if l, ok := res.(*list); ok {
					sort.Slice(l.elems, func(i, j int) bool {
						return l.elems[i].String() < l.elems[j].String()
					})
					tr = []string(t)
				}
			}
			s += res.String()+" "+ts(res,ctx)
			return
		} ()
	)
	switch v := tr.(type) {
	case   string:                       if v == vs { /* delete(checkspecs[spec][src[0]],ks); */ return }
	case []string: for _, s := range v { if s == vs { /* delete(checkspecs[spec][src[0]],ks); */ return } }
	case nil:
		if src == nil {
			errostack(pc(ctx,p), 8, "`%v`:`%s`,", ks, vs).trace()
		} else {
			erro(pc(ctx,p), "src=%s p=%v", src, p)
			notestack(pc(ctx,p), 8, "`%v`:`%s`,", ks, vs).trace()
		}
	}
	erro(pc(ctx,p), "`%v`", ks)
	note(pc(ctx,p), `got: %s`, vs)
	notestack(pc(ctx,p), 8, `!= : %v`, tr).trace()
}

var checkstrs = map[string]map[string]map[string]any{
	"testdata/assert":               checkstrs__assert,
	"testdata/locals":               checkstrs__locals,
	"testdata/builtins/addprefix":   checkstrs__addprefix,
	"testdata/builtins/addsuffix":   checkstrs__addsuffix,
	"testdata/builtins/trimprefix":  checkstrs__trimprefix,
	"testdata/builtins/trimsuffix":  checkstrs__trimsuffix,
	"testdata/builtins/if":          checkstrs__if,
	"testdata/builtins/foreach":     checkstrs__foreach,
	"testdata/builtins/foreach/1":   checkstrs__foreach1,
	"testdata/builtins/foreach/2":   checkstrs__foreach2,
	"testdata/builtins/foreach/3":   checkstrs__foreach3,
	"testdata/builtins/foreach/4":   checkstrs__foreach4,
	"testdata/builtins/foreach/5":   checkstrs__foreach5,
	"testdata/builtins/wildcard":    checkstrs__wildcard,
	"testdata/builtins/closure":     checkstrs__closure,
	"testdata/builtins/delegate":    checkstrs__delegate,
	"testdata/builtins/logic":       checkstrs__logic,
	"testdata/builtins/contains":    checkstrs__contains,
	"testdata/builtins/join":        checkstrs__join,
	"testdata/builtins/xor":         checkstrs__xor,
	"testdata/builtins/or":          checkstrs__or,
	"testdata/value":                checkstrs_value,
	"testdata/value/auto":           checkstrs_value_auto,
	"testdata/value/closure":        checkstrs_value_closure,
	"testdata/value/disjunction":    checkstrs_value_disjunction,
	"testdata/value/placeholder":    checkstrs_value_placeholder,
	"testdata/value/optional":       checkstrs_value_optional,
	"testdata/value/optional/foo":   checkstrs_value_optional_foo,
	"testdata/value/glob":           checkstrs_value_glob,
	"testdata/value/1":              checkstrs_value_1,
	"testdata/value/2/0":            checkstrs_value_20,
	"testdata/value/2/1":            checkstrs_value_21,
	"testdata/value/2/2":            checkstrs_value_22,
	"testdata/value/3":              checkstrs_value_3,
	"testdata/value/4":              checkstrs_value_4,
	"testdata/value/5":              checkstrs_value_5,
	"testdata/value/6":              checkstrs_value_6,
	"testdata/value/7":              checkstrs_value_7,
	"testdata/value/8":              checkstrs_value_8,
	"testdata/value/9":              checkstrs_value_9,
	"testdata/value/10":             checkstrs_value_10,
	"testdata/value/11":             checkstrs_value_11,
	"testdata/value/12":             checkstrs_value_12,
	"testdata/value/13":             checkstrs_value_13,
	"testdata/value/bug_01":         checkstrs_value_bug_01,
	"testdata/rule/shell/for-stdout":checkstrs_rule_shell_forstdout,
	"testdata/template":             checkstrs_template,
	"testdata/modifier":             checkstrs_modifiers,
	"testdata/valcache":             checkstrs_valcache,
	"testdata/valcache/1":           checkstrs_valcache1,
	"testdata/valcache/2":           checkstrs_valcache2,
	"testdata/valcache/3":           checkstrs_valcache3,
}
func check_string(ctx Context, p Value, v Value, res string) {
	var (
		src = strings.Split(try[string](ctx,source{}),":")
		d, _ = do(ctx, origin_def{}).(*def)
		dn = func() (s string) { if d != nil { s = d.name }; return } ()
		ks = func() (s string) {
		if src[0] != "loader.go" && 1 < len(src) { s = src[1] + " " }
			s += line_column(d)+":"+dn+" "+p.String()+" "+ts(p)
			return
		} ()
		vs = ts(v,ctx)+" "+res
		ta = checkstrs[_project(ctx).spec][src[0]][ks]
	)
	switch v := ta.(type) {
	case []string: for _, s := range v { if s == vs { return } }
	case   string:                       if v == vs { return }
	case nil: errostack(pc(ctx,p), 8, "`%v`:`%s`,", ks, vs).trace()
	}
	erro(pc(ctx,p), "`%v`", ks)
	note(pc(ctx,p), `got: %s`, vs)
	note(pc(ctx,p), `!= : %s`, ta)
	note(pc(ctx,p), `val: %v`, v)
	notestack(pc(ctx,p), 8, `res: %v`, res).trace()
}

func check_cmp(ctx Context, l, r any, _r *cmpres) {
	var _eq_ = sfmt("%v",l) == sfmt("%v",r)
	switch {
	case !_eq_ && cmpEqual == *_r:
		errostack(pc(pc(ctx,r),l), 3, "%v: %v ⇔ %v | %v ⇔ %v", *_r, l, r, ts(l), ts(r)).trace()
	case _eq_ && cmpEqual != *_r:
		errostack(pc(pc(ctx,r),l), 3, "%v: %v ⇔ %v | %v ⇔ %v", *_r, l, r, ts(l), ts(r)).trace()
	}
}

var checkpoints_com = map[string]map[string]any{
	".": map[string]any{
		"[] [{1:2:word test} {1:6:punct .} {1:7:word paniconexit0}]": "[test.paniconexit0] [{=compound {1:2:word test} {1:6:punct .} {1:7:word paniconexit0}}]",
		"[] [{1:21:word test} {1:25:punct .} {1:26:word timeout}]": "[test.timeout] [{=compound {1:21:word test} {1:25:punct .} {1:26:word timeout}}]",
	},
	"value": map[string]any{
		"[] [{13:11:word x} {13:13:word y}]": "[xy] [{=compound {13:11:word x} {13:13:word y}}]",
	},
}
func check_com(ctx *comctx, a, elems []Value, res *[]Value) {
	var (
		f = dirs(1, bases(_position(ctx).Filename, 2))
		k = sfmt("%v %v", ts(a), ts(elems))
		r = sfmt("%v %v", *res, ts(*res))
		t = checkpoints_com[f][k]
	)
	switch v := t.(type) {
	case   string: if r == v { return }
	case []string: for _, v := range v { if r == v { return } }
	default: if false { note(pc(ctx,elems), "%s: %v, %v => %v", f, k, t, r) }
	}
	if false { erro(pc(ctx,a), `"%v %v":"%v %v",`, a, elems, *res).trace() }
}

var fmt_slot = regexp.MustCompile(`%\[([0-9_a-z]+)\]s`)
var checkpoints_match = map[string]any{
	` builtins`:`false  []`,
	` trimsuffix`:`false  []`,
	`* a.h`:`true a.h [a.h]`,
	`* a`:`true a [a]`,
	`* b.h`:`true b.h [b.h]`,
	`* b`:`true b [b]`,
	`* bar.h`:`true bar.h [bar.h]`,
	`* bar`:`true bar [bar]`,
	`* config`:`true config [config]`,
	`* foo.h`:`true foo.h [foo.h]`,
	`* foo`:`true foo [foo]`,
	`* foobar`:`true foobar [foobar]`,
	`* v1.h`:`true v1.h [v1.h]`,
	`* v2.h`:`true v2.h [v2.h]`,
	`** `:`true  []`,
	`**y abcy`:`true abcy [abc]`,
	`**y dy`:`true dy [d]`,
	`**z xyz`:`true xyz [xy]`,
	`**.auto .test/a/b/c.auto`:`true .test/a/b/c.auto [.test/a/b/c]`,
	`**.c foo.c`:`true foo.c [foo]`,
	`**.c foo/bar.c`:`true foo/bar.c [foo/bar]`,
	`**.c foo/oo/oo/oo/bar.c`:`true foo/oo/oo/oo/bar.c [foo/oo/oo/oo/bar]`,
	`**.def.am foobar/config/*.def.am`:`true foobar/config/*.def.am [foobar/config/*]`,
	`**.h *.h`:`true *.h [*]`,
	`**.h **.h`:`true **.h [**]`,
	`**.h bar.h`:`true bar.h [bar]`,
	`**.h foobar.h`:`true foobar.h [foobar]`,
	`**.h foo.h`:`true foo.h [foo]`,
	`**.h foo/a/b/c/bar.h`:`true foo/a/b/c/bar.h [foo/a/b/c/bar]`,
	`**.h foo/bar/v1.h`:`true foo/bar/v1.h [foo/bar/v1]`,
	`**.h foo/bar/v2.h`:`true foo/bar/v2.h [foo/bar/v2]`,
	`**.h foo/bar/zz/x.h`:`true foo/bar/zz/x.h [foo/bar/zz/x]`,
	`**.h foo/v1.h`:`true foo/v1.h [foo/v1]`,
	`**.h foo/v2.h`:`true foo/v2.h [foo/v2]`,
	`**.h inc/bar.h`:`true inc/bar.h [inc/bar]`,
	`**.h inc/foo.h`:`true inc/foo.h [inc/foo]`,
	`**.h inc/foo/bar/v1.h`:`true inc/foo/bar/v1.h [inc/foo/bar/v1]`,
	`**.h inc/foo/bar/v2.h`:`true inc/foo/bar/v2.h [inc/foo/bar/v2]`,
	`**.h inc/foo/bar/zz/x.h`:`true inc/foo/bar/zz/x.h [inc/foo/bar/zz/x]`,
	`**.h inc/foo/v1.h`:`true inc/foo/v1.h [inc/foo/v1]`,
	`**.h inc/foo/v2.h`:`true inc/foo/v2.h [inc/foo/v2]`,
	`**.o **.o`:`true **.o [**]`,
	`**.def.am *.def.am`:`true *.def.am [*]`,
	`**.def.am **.def.am`:`true **.def.am [**]`,
	`**.def.am foobar.def.am`:`true foobar.def.am [foobar]`,
	`**.def.am foo/1/2/3/bar.def.am`:`true foo/1/2/3/bar.def.am [foo/1/2/3/bar]`,
	`*data testdata`:`true testdata [test]`,
	`*.c foo.c`:`true foo.c [foo]`,
	`*.def.am a.def.am`:`true a.def.am [a]`,
	`*.def.in a.def.in`:`true a.def.in [a]`,
	`*.def.in b.def.in`:`true b.def.in [b]`,
	`*.log *.log`:`true *.log [*]`,
	`*.log foobar.log`:`true foobar.log [foobar]`,
	`*.h **.h`:`true **.h [**]`,
	`*.h a.h`:`true a.h [a]`,
	`*.h b.h`:`true b.h [b]`,
	`*.h bar.h`:`true bar.h [bar]`,
	`*.h c.h`:`true c.h [c]`,
	`*.h foo.h`:`true foo.h [foo]`,
	`*.h v1.h`:`true v1.h [v1]`,
	`*.h v2.h`:`true v2.h [v2]`,
	`*/*.h bar.h`:`false [bar.h] [bar.h]`,
	`*/*.h foo.h`:`false [foo.h] [foo.h]`,
	`*/*.h v1.h`:`false [v1.h] [v1.h]`,
	`*/*.h v2.h`:`false [v2.h] [v2.h]`,
	`*/*/*.h bar.h`:`false [bar.h] [bar.h]`,
	`*/*/*.h foo.h`:`false [foo.h] [foo.h]`,
	`. .test`:`false . []`,
	`.test/**/**z .test/a/b/c/xyz`:`true [.test a b c xyz] [a/b/c xy]`,
	`.test/**y/**y .test/a/b/cy/a/b/c/y`:`false [.test a b cy a b c y] [a/b/cy/a/b/c/]`,
	`.test/**y/**y/z .test/a/b/cy/a/b/c/y/z`:`false [.test a b cy a b c y z] [a/b/cy/a/b/c/ z]`,
	`.test/*.h .test/a.h`:`true [.test a.h] [a]`,
	`.test/*.h .test/a/b.h`:`false [.test] []`,
	`.test/*.h .test/a/b/c.h`:`false [.test] []`,
	`.test/*/*.h .test/a.h`:`false [.test a.h] [a.h]`,
	`.test/*/*.h .test/a/b.h`:`true [.test a b.h] [a b]`,
	`.test/*/*.h .test/a/b/c.h`:`false [.test a] [a]`,
	`.test/*/*/*.h .test/a.h`:`false [.test a.h] [a.h]`,
	`.test/*/*/*.h .test/a/b.h`:`false [.test a b.h] [a b.h]`,
	`.test/*/*/*.h .test/a/b/c.h`:`true [.test a b c.h] [a b c]`,
	`.test/*/*/*.h .test/a/b/c/d.h`:`false [.test a b] [a b]`,
	`.test/*/*/*.h .test/a/b/c/d/e.h`:`false [.test a b] [a b]`,
	`.test/*?y/**y .test/a/b/cy/a/b/c/y`:`true [.test a b cy a b c y] [a/b/c a/b/c/]`,
	`.test/x**/**y .test/xa/b/c/dy`:`true [.test xa b c dy] [a/b/c d]`,
	`.test/x**/**y .test/xabc/abcy`:`true [.test xabc abcy] [abc abc]`,
	`.test/x**y .test/x/xx-yy/y`:`true [.test x xx-yy y] [/xx-yy/]`,
	`.test/x**y .test/xxx-yyx/y`:`true [.test xxx-yyx y] [xx-yyx/]`,
	`.test/x**y .test/xxx-yyx/z`:`false [.test xxx-yyx z] [xx-yyx/z]`,
	`.test/x**y .test/xxx-yyx`:`false [.test xxx-yyx] [xx-yyx]`,
	`.test/x**y .test/xxx-yyy`:`true [.test xxx-yyy] [xx-yy]`,
	`.test/x**y .test/xxx/a/b/c/yyy`:`true [.test xxx a b c yyy] [xx/a/b/c/yy]`,
	`.test/x**y/x** .test/xaaa/bbb/ccc/y/xxx/xx`:`true [.test xaaa bbb ccc y xxx xx] [aaa/bbb/ccc/ xx/xx]`,
	`.test/x**y/x** .test/xaabbccy/xabc`:`true [.test xaabbccy xabc] [aabbcc abc]`,
	`.test/x**y/x**y .test/xaa/bb/ccy/xaa/bb/ccy`:`false [.test xaa bb ccy xaa bb ccy] [aa/bb/ccy/xaa/bb/cc]`,
	`.test/x**y/x**y .test/xaaay/x/aaa/y`:`false [.test xaaay x aaa y] [aaay/x/aaa/]`,
	`.test/x**y/z .test/xxx-yyy/z`:`true [.test xxx-yyy z] [xx-yy]`,
	`.test/x**y/z .test/xxx/a/b/c/yyy/z`:`true [.test xxx a b c yyy z] [xx/a/b/c/yy]`,
	`.test/x*?/**y .test/xa/b/c/dy`:`true [.test xa b c dy] [a b/c/d]`,
	`.test/x*?/**y .test/xabc/abcy`:`true [.test xabc abcy] [abc abc]`,
	`.test/x*?y .test/x/xx-yy/y`:`false [.test x xx-yy] [/xx-yy]`,
	`.test/x*?y/x*?y .test/xaa/bb/ccy/xaa/bb/ccy`:`true [.test xaa bb ccy xaa bb ccy] [aa/bb/cc aa/bb/cc]`,
	`.test/x*?y/x*?y .test/xaaay/x/aaa/y`:`true [.test xaaay x aaa y] [aaa /aaa/]`,
	`/builtins /builtins/trimprefix`:`false [ builtins] []`,
	`builtins builtins/trimprefix`:`false builtins []`,
	`foobar/config/*.def.am **.def.am`:`false [] []`,
	`test* testdata`:`true testdata [data]`,
	`t*a testdata`:`true testdata [estdat]`,
	`^configure\.types\.(<(.+?)>|"(.+?)") configure.types."atomic.h"`:`true configure.types."atomic.h" ["atomic.h"  atomic.h]`,
	`^configure\.types\.(<(.+?)>|"(.+?)") configure.types.<atomic.h>`:`true configure.types.<atomic.h> [<atomic.h> atomic.h ]`,
	`^val([1-9])$ val1`:`true val1 [1]`,
	`^val([1-9])$ val2`:`true val2 [2]`,
	`^val([1-9])$ val3`:`true val3 [3]`,
	`^val([1-9])$ val4`:`true val4 [4]`,
	`^val([1-9])$ val5`:`true val5 [5]`,
	`^var\.([xyz]+) var.xxx`:`true var.xxx [xxx]`,
	`^var\.([xyz]+) var.yyy`:`true var.yyy [yyy]`,
	`^var\.([xyz]+) var.zzz`:`true var.zzz [zzz]`,
	`^var\.([xy]+) var.xxx`:`true var.xxx [xxx]`,
	`^var\.([xy]+) var.yyy`:`true var.yyy [yyy]`,
	`^\.test\.([0-9]+)$ .test.1`:`true .test.1 [1]`,
	`^\.test\.([0-9]+)$ .test.2`:`true .test.2 [2]`,
	`^\.test\.([0-9]+)$ .test.3`:`true .test.3 [3]`,
	`^\.test\.([0-9]+)$ .test.4`:`true .test.4 [4]`,
	`^\.test\.([0-9]+)$ .test.5`:`true .test.5 [5]`,
	`^\.test\.([0-9]+)$ .test.6`:`true .test.6 [6]`,
	`^\.test\.([0-9]+)$ .test.7`:`true .test.7 [7]`,
	`^\.test\.([0-9]+)$ .test.8`:`true .test.8 [8]`,
	`^\.test\.([0-9]+)$ .test.9`:`true .test.9 [9]`,
	`^\.test\.([0-9]+)$ .test.10`:`true .test.10 [10]`,
	`^\.test\.([0-9]+)$ .test.11`:`true .test.11 [11]`,
	`^\.test\.([0-9]+)$ .test.12`:`true .test.12 [12]`,
	`^\.test\.([0-9]+)$ .test.13`:`true .test.13 [13]`,
	`fo{2}(/o{2}){3}/bar\.c foo/oo/oo/oo/bar.c`:`true foo/oo/oo/oo/bar.c [/oo]`,
	`fo{2}/bar\.c foo/bar.c`:`true foo/bar.c []`,
	`fo{2}\.c foo.c`:`true foo.c []`,
	testdata_fmt(`%[1]s builtins/trimprefix`                  ):`false [] []`,
	testdata_fmt(`%%%%/testdata %[1]s/builtins/trimprefix`    ):testdata_fmt(`false [%[2]s] [%[1]s]`,trim_suffix{1,"/testdata"}),
	testdata_fmt(`%%%%/testdata/ %[1]s/builtins/trimprefix`   ):testdata_fmt(`false [%[2]s ] [%[1]s]`,trim_suffix{1,"/testdata"}),
	testdata_fmt(`**/testdata %[1]s/builtins/trimprefix`      ):testdata_fmt(`false [%[2]s] [%[1]s]`,trim_suffix{1,"/testdata"}),
	testdata_fmt(`**/testdata/ %[1]s/builtins/trimprefix`     ):testdata_fmt(`false [%[2]s ] [%[1]s]`,trim_suffix{1,"/testdata"}),
	testdata_fmt(`**/testdata/** %[1]s/builtins/trimprefix`   ):testdata_fmt(`true [%[2]s builtins trimprefix] [%[1]s builtins/trimprefix]`,trim_suffix{1,"/testdata"}),
	testdata_fmt(`**/testdata/** %[1]s/builtins/trimsuffix`   ):testdata_fmt(`true [%[2]s builtins trimsuffix] [%[1]s builtins/trimsuffix]`,trim_suffix{1,"/testdata"}),
	testdata_fmt(`**/testdata/**/ %[1]s/builtins/trimprefix/` ):testdata_fmt(`true [%[2]s builtins trimprefix ] [%[1]s builtins/trimprefix]`,trim_suffix{1,"/testdata"}),
	testdata_fmt(`**/testdata/**/ %[1]s/builtins/trimsuffix/` ):testdata_fmt(`true [%[2]s builtins trimsuffix ] [%[1]s builtins/trimsuffix]`,trim_suffix{1,"/testdata"}),
	testdata_fmt(`*?/testdata %[1]s/builtins/trimprefix`      ):testdata_fmt(`false [%[2]s] [%[1]s]`,trim_suffix{1,"/testdata"}),
	testdata_fmt(`*?/testdata %[1]s`                          ):testdata_fmt(`true [%[2]s] [%[1]s]`,trim_suffix{1,"/testdata"}),
	testdata_fmt(`*?/testdata/ %[1]s/`                        ):testdata_fmt(`true [%[2]s ] [%[1]s]`,trim_suffix{1,"testdata"}),
	testdata_fmt(`*?/testdata/ %[1]s/builtins/trimprefix`     ):testdata_fmt(`false [%[2]s ] [%[1]s]`,trim_suffix{1,"/testdata"}),
	testdata_fmt(`*?/testdata/*? %[1]s/builtins/trimprefix`   ):testdata_fmt(`true [%[2]s builtins trimprefix] [%[1]s builtins/trimprefix]`,trim_suffix{1,"/testdata"}),
	testdata_fmt(`*?/testdata/*?/ %[1]s/builtins/trimprefix/` ):testdata_fmt(`true [%[2]s builtins trimprefix ] [%[1]s builtins/trimprefix]`,trim_suffix{1,"/testdata"}),
	testdata_fmt(`*?/testdata/*? %[1]s/builtins/trimsuffix`   ):testdata_fmt(`true [%[2]s builtins trimsuffix] [%[1]s builtins/trimsuffix]`,trim_suffix{1,"/testdata"}),
	testdata_fmt(`*?/testdata/*?/ %[1]s/builtins/trimsuffix/` ):testdata_fmt(`true [%[2]s builtins trimsuffix ] [%[1]s builtins/trimsuffix]`,trim_suffix{1,"/testdata"}),
	testdata_fmt(`/%%%%/testdata %[1]s/builtins/trimprefix`   ):testdata_fmt(`false [%[2]s] [%[1]s]`,trim_prefix{1,"/"},trim_suffix{1,"/testdata"}),
	testdata_fmt(`/%%%%/testdata/ %[1]s/builtins/trimprefix`  ):testdata_fmt(`false [%[2]s ] [%[1]s]`,trim_prefix{1,"/"},trim_suffix{1,"/testdata"}),
	testdata_fmt(`/**/ %[1]s/builtins/trimprefix/`            ):testdata_fmt(`true [%[2]s builtins trimprefix ] [%[1]s/builtins/trimprefix]`,trim_prefix{1,"/"}),
	testdata_fmt(`/**/ %[1]s/builtins/trimsuffix/`            ):testdata_fmt(`true [%[2]s builtins trimsuffix ] [%[1]s/builtins/trimsuffix]`,trim_prefix{1,"/"}),
	testdata_fmt(`/**/ %[1]s/builtins/trimprefix`             ):testdata_fmt(`false [%[2]s builtins ] [%[1]s/builtins/]`,trim_prefix{1,"/"}),
	testdata_fmt(`/**/ %[1]s/builtins/trimsuffix`             ):testdata_fmt(`false [%[2]s builtins ] [%[1]s/builtins/]`,trim_prefix{1,"/"}),
	testdata_fmt(`/*?/ %[1]s/builtins/trimprefix/`            ):testdata_fmt(`true [%[2]s builtins trimprefix ] [%[1]s/builtins/trimprefix]`,trim_prefix{1,"/"}),
	testdata_fmt(`/*?/ %[1]s/builtins/trimsuffix/`            ):testdata_fmt(`true [%[2]s builtins trimsuffix ] [%[1]s/builtins/trimsuffix]`,trim_prefix{1,"/"}),
	testdata_fmt(`/*?/ %[1]s/builtins/trimprefix`             ):testdata_fmt(`false [%[2]s builtins ] [%[1]s/builtins/]`,trim_prefix{1,"/"}),
	testdata_fmt(`/*?/ %[1]s/builtins/trimsuffix`             ):testdata_fmt(`false [%[2]s builtins ] [%[1]s/builtins/]`,trim_prefix{1,"/"}),
	testdata_fmt(`/**/testdata %[1]s/builtins/trimprefix`     ):testdata_fmt(`false [%[2]s] [%[1]s]`,trim_prefix{1,"/"},trim_suffix{1,"/testdata"}),
	testdata_fmt(`/**/testdata/ %[1]s/builtins/trimprefix`    ):testdata_fmt(`false [%[2]s ] [%[1]s]`,trim_prefix{1,"/"},trim_suffix{1,"/testdata"}),
	testdata_fmt(`/**/testdata/** %[1]s/builtins/trimprefix`  ):testdata_fmt(`true [%[2]s builtins trimprefix] [%[1]s builtins/trimprefix]`,trim_prefix{1,"/"},trim_suffix{1,"/testdata"}),
	testdata_fmt(`/**/testdata/** %[1]s/builtins/trimsuffix`  ):testdata_fmt(`true [%[2]s builtins trimsuffix] [%[1]s builtins/trimsuffix]`,trim_prefix{1,"/"},trim_suffix{1,"/testdata"}),
	testdata_fmt(`/**/testdata/**/ %[1]s/builtins/trimprefix/`):testdata_fmt(`true [%[2]s builtins trimprefix ] [%[1]s builtins/trimprefix]`,trim_prefix{1,"/"},trim_suffix{1,"/testdata"}),
	testdata_fmt(`/**/testdata/**/ %[1]s/builtins/trimsuffix/`):testdata_fmt(`true [%[2]s builtins trimsuffix ] [%[1]s builtins/trimsuffix]`,trim_prefix{1,"/"},trim_suffix{1,"/testdata"}),
	testdata_fmt(`/*?/testdata %[1]s/builtins/trimprefix`     ):testdata_fmt(`false [%[2]s] [%[1]s]`,trim_prefix{1,"/"},trim_suffix{1,"/testdata"}),
	testdata_fmt(`/*?/testdata %[1]s`                         ):testdata_fmt(`true [%[2]s] [%[1]s]`,trim_prefix{1,"/"},trim_suffix{1,"/testdata"}),
	testdata_fmt(`/*?/testdata/ %[1]s/`                       ):testdata_fmt(`true [%[2]s ] [%[1]s]`,trim_prefix{1,"/"},trim_suffix{1,"/testdata"}),
	testdata_fmt(`/*?/testdata/ %[1]s/builtins/trimprefix`    ):testdata_fmt(`false [%[2]s ] [%[1]s]`,trim_prefix{1,"/"},trim_suffix{1,"/testdata"}),
	testdata_fmt(`/*?/testdata/*? %[1]s/builtins/trimprefix`  ):testdata_fmt(`true [%[2]s builtins trimprefix] [%[1]s builtins/trimprefix]`,trim_prefix{1,"/"},trim_suffix{1,"/testdata"}),
	testdata_fmt(`/*?/testdata/*? %[1]s/builtins/trimsuffix`  ):testdata_fmt(`true [%[2]s builtins trimsuffix] [%[1]s builtins/trimsuffix]`,trim_prefix{1,"/"},trim_suffix{1,"/testdata"}),
	testdata_fmt(`/*?/testdata/*?/ %[1]s/builtins/trimprefix/`):testdata_fmt(`true [%[2]s builtins trimprefix ] [%[1]s builtins/trimprefix]`,trim_prefix{1,"/"},trim_suffix{1,"/testdata"}),
	testdata_fmt(`/*?/testdata/*?/ %[1]s/builtins/trimsuffix/`):testdata_fmt(`true [%[2]s builtins trimsuffix ] [%[1]s builtins/trimsuffix]`,trim_prefix{1,"/"},trim_suffix{1,"/testdata"}),
	testdata_fmt(`/*?/test*/*?/ %[1]s/builtins/trimprefix/`   ):testdata_fmt(`true [%[2]s builtins trimprefix ] [%[1]s data builtins/trimprefix]`,trim_prefix{1,"/"},trim_suffix{1,"/testdata"}),
	testdata_fmt(`/*?/test*/**/ %[1]s/builtins/trimprefix/`   ):testdata_fmt(`true [%[2]s builtins trimprefix ] [%[1]s data builtins/trimprefix]`,trim_prefix{1,"/"},trim_suffix{1,"/testdata"}),
	testdata_fmt(`/*?/test*/*?/ %[1]s/builtins/trimsuffix/`   ):testdata_fmt(`true [%[2]s builtins trimsuffix ] [%[1]s data builtins/trimsuffix]`,trim_prefix{1,"/"},trim_suffix{1,"/testdata"}),
	testdata_fmt(`/*?/test*/**/ %[1]s/builtins/trimsuffix/`   ):testdata_fmt(`true [%[2]s builtins trimsuffix ] [%[1]s data builtins/trimsuffix]`,trim_prefix{1,"/"},trim_suffix{1,"/testdata"}),
	testdata_fmt(`/*?/*data/*?/ %[1]s/builtins/trimprefix/`   ):testdata_fmt(`true [%[2]s builtins trimprefix ] [%[1]s test builtins/trimprefix]`,trim_prefix{1,"/"},trim_suffix{1,"/testdata"}),
	testdata_fmt(`/**/*data/*?/ %[1]s/builtins/trimprefix/`   ):testdata_fmt(`true [%[2]s builtins trimprefix ] [%[1]s test builtins/trimprefix]`,trim_prefix{1,"/"},trim_suffix{1,"/testdata"}),
	testdata_fmt(`/*?/*data/*?/ %[1]s/builtins/trimsuffix/`   ):testdata_fmt(`true [%[2]s builtins trimsuffix ] [%[1]s test builtins/trimsuffix]`,trim_prefix{1,"/"},trim_suffix{1,"/testdata"}),
	testdata_fmt(`/**/*data/*?/ %[1]s/builtins/trimsuffix/`   ):testdata_fmt(`true [%[2]s builtins trimsuffix ] [%[1]s test builtins/trimsuffix]`,trim_prefix{1,"/"},trim_suffix{1,"/testdata"}),
	testdata_fmt(`/*?/t*a/*?/ %[1]s/builtins/trimprefix/`     ):testdata_fmt(`true [%[2]s builtins trimprefix ] [%[1]s estdat builtins/trimprefix]`,trim_prefix{1,"/"},trim_suffix{1,"/testdata"}),
	testdata_fmt(`/*?/t*a/*?/ %[1]s/builtins/trimsuffix/`     ):testdata_fmt(`true [%[2]s builtins trimsuffix ] [%[1]s estdat builtins/trimsuffix]`,trim_prefix{1,"/"},trim_suffix{1,"/testdata"}),
	testdata_fmt(`/builtins %[1]s/builtins/trimprefix`        ):`false [] []`,
	testdata_fmt(`/testdata/%%%% %[1]s/builtins/trimsuffix`   ):`false [ testdata builtins trimsuffix] [builtins/trimsuffix]`,
	testdata_fmt(`/testdata/%%%% %[1]s/builtins/trimsuffix/`  ):`false [ testdata builtins trimsuffix ] [builtins/trimsuffix]`,
	testdata_fmt(`/testdata/%%%%/ %[1]s/builtins/trimsuffix/` ):`false [ testdata builtins trimsuffix ] [builtins/trimsuffix]`,
	testdata_fmt(`/testdata/** %[1]s/builtins/trimsuffix`     ):`false [ testdata builtins trimsuffix] [builtins/trimsuffix]`,
	testdata_fmt(`/testdata/** %[1]s/builtins/trimsuffix/`    ):`false [ testdata builtins trimsuffix ] [builtins/trimsuffix]`,
	testdata_fmt(`/testdata/**/ %[1]s/builtins/trimsuffix`    ):`false [] []`,
	testdata_fmt(`/testdata/**/ %[1]s/builtins/trimsuffix/`   ):`false [ testdata builtins trimsuffix ] [builtins/trimsuffix]`,
	testdata_fmt(`testdata/** %[1]s/builtins/trimsuffix`      ):`false [testdata builtins trimsuffix] [builtins/trimsuffix]`,
	testdata_fmt(`testdata/** %[1]s/builtins/trimsuffix/`     ):`false [testdata builtins trimsuffix ] [builtins/trimsuffix]`,
	testdata_fmt(`testdata/**/ %[1]s/builtins/trimsuffix`     ):`false [] []`,
	testdata_fmt(`testdata/**/ %[1]s/builtins/trimsuffix/`    ):`false [testdata builtins trimsuffix ] [builtins/trimsuffix]`,
	testdata_fmt(`testdata/%%%% %[1]s/builtins/trimsuffix`    ):`false [testdata builtins trimsuffix] [builtins/trimsuffix]`,
	testdata_fmt(`testdata/%%%% %[1]s/builtins/trimsuffix/`   ):`false [testdata builtins trimsuffix ] [builtins/trimsuffix]`,
	testdata_fmt(`testdata/%%%%/ %[1]s/builtins/trimsuffix`   ):`false [] []`,
	testdata_fmt(`testdata/%%%%/ %[1]s/builtins/trimsuffix/`  ):`false [testdata builtins trimsuffix ] [builtins/trimsuffix]`,
	testdata_fmt(`testdata/builtins/trimsuffix %[1]s/builtins/trimsuffix`):`false [testdata builtins trimsuffix] []`,
	testdata_fmt(`builtins/trimsuffix %[1]s`,trim_suffix{1,"testdata"}):`false [] []`,
}
func check_match(ctx Context, _p, _v any, full bool, res any, stems []string) {
	var pat, val = joinp(ctx, _p), joinp(ctx, _v)
	var k = sfmt("%v %v", pat, val)
	var t = sfmt("%v %v %v", full, res, stems)
	if fmt_slot.MatchString(k) { k = sfmt(k, testdata_s, strings.Split(testdata_s, pathSep)) }
	if v := checkpoints_match[k]; v == nil {
		switch t {
		case `false <nil> []`:
			return
		default:
			switch s1, s2 := sfmt("%v", pat), sfmt("%v", val); {
			case s1 == "$/" && strings.HasPrefix(s2, "/"): return
			case s2 == "$/" && strings.HasPrefix(s1, "/"): return
			case s1 == s2 && sfmt("true %s []", s1) == t : return
			case strings.HasPrefix(s1, testdata_s):
				if ss1 := strings.Split(s1,pathSep); s1 == s2 {
					if sfmt("true %v []", ss1) == t { return }
				} else {
					if sfmt("false %v []", ss1) == t { return }
				}
			}
		}
		erro(ctx, "`%v`:`%v`,", k, t)
		note(ctx, "pat: %v", ts(pat))
		note(ctx, "val: %v", ts(val)).trace()
	} else if v != t {
		erro(ctx, `%s`, k)
		note(ctx, "got: %s", t)
		note(ctx, "!= : %s", v).trace()
	}
}

func check_filemap(ctx Context, vc *valcache, patt Value, val any, _s_ string) {
	var uc = &uncache{ctx, nil}
	var x, y = hit(uc, vc, val)
	if s := sfmt("%v %v", y, x); s != _s_ {
		erro(pc(ctx,patt), "%v %v | %v %v != %s | %v", patt, val, y, x, _s_, vc).trace()
	}
}
func check_map_files(ctx Context, p *project, patts, paths []Value, _res *[]filemap) {
	if s, t := sfmt("%v", patts), filemap_str(_res); s != t {
		erro(pc(ctx,patts), "%s != %s", s, t).trace()
	}

	var vc = &p.filemap

	for _, patt := range patts {
		var s = patt.String()
		switch check_filemap(ctx, vc, patt, s, sfmt("true {0:%s}", s)); s {
		case "**.h":
			check_filemap(ctx, vc, patt, "foobar.h", "true {0:**.h}")
			check_filemap(ctx, vc, patt, "foo/a/b/c/bar.h", "true {0:**.h}")
			check_filemap(ctx, vc, patt, strings.Split("foo/a/b/c/bar.h",pathSep), "true {0:**.h}")
		case "**.def.am":
			check_filemap(ctx, vc, patt, "foobar.def.am", "true {0:**.def.am}")
			check_filemap(ctx, vc, patt, "foo/1/2/3/bar.def.am", "true {0:**.def.am}")
			check_filemap(ctx, vc, patt, strings.Split("foo/1/2/3/bar.def.am",pathSep), "true {0:**.def.am}")
		case "*.log":
			check_filemap(ctx, vc, patt, "foobar.log", "true {0:*.log}")
			check_filemap(ctx, vc, patt, "foo/bar.log", "false {0:*.log}")
			check_filemap(ctx, vc, patt, []string{"foo","bar.log"}, "false {0:*.log}")
		case "**.o":
			check_filemap(ctx, vc, patt, "foo/123/bar.o", "true {0:**.o}")
		case ".deps/??/??/??????????":
			check_filemap(ctx, vc, patt, ".deps/11/ab/xxxyyyzzz0", "")
		case "&(gen)":
			note(ctx, "%v %v", patt, vc).debug()
		}
	}
}

var checkpoints__string_com = map[string]any{
	`$1$2$3`:`{}{}{}`,
	`&(.test.$_)⌜foo bar⌟{}99`:`{}⌜foo bar⌟{}99`,
	`&(.test.bar)⌜foo bar⌟{}88`:`{}⌜foo bar⌟{}88`,
	`&(.test.bar)⌜foo bar⌟{}99`:`{}⌜foo bar⌟{}99`,
	`&(.test.foo)⌜foo bar⌟{}88`:`{}⌜foo bar⌟{}88`,
	`&(.test.foo)⌜foo bar⌟{}99`:`{}⌜foo bar⌟{}99`,
	`&(.test.h)a`:`-a`, `&(.test.h)b`:`-b`, `&(.test.h)c`:`-c`,
	`&(target.arch)-&(target.vendor)-&(target.os)-&(target.abi)`:`foo-bar-{}-0`,
	`,`:`,`,
	`-a`:`-a`, `-b`:`-b`, `-c`:`-c`,
	`-foobar`:`-foobar`,
	`.`:`.`,
	`.deps`:`.deps`,
	`.test.v`:`.test.v`,
	`.test`:`.test`,
	`.test~&(.test.s)`:`.test~foo`,
	`1x`:`1x`, `2x`:`2x`, `3x`:`3x`,
	`D.c`:`D.c`, `D.c++`:`D.c++`,
	`I.c`:`I.c`, `I.c++`:`I.c++`,
	`V{}{}`:`V{}{}`,
	`Xxa`:`Xxa`, `Xxb`:`Xxb`,
	`X{&(.test.xa)}`:`X~1~`,
	`X{&(.test.xb)}`:`X~2~`,
	`X~1~`:`X~1~`, `X~2~`:`X~2~`,
	`YX{&(.test.xa)}`:`YX~1~`,
	`YX{&(.test.xb)}`:`YX~2~`,
	`YX~1~`:`YX~1~`, `YX~2~`:`YX~2~`,
	`a-b-3-abc`:`a-b-3-abc`,
	`a-b-3`:`a-b-3`,
	`a-b`:`a-b`,
	`a.h`:`a.h`, // {=compound {31:18:word a} {31:19:punct .} {31:20:word h}}
	`a\,b\,c,x\,y\,z`:`a\,b\,c,x\,y\,z`,
	`a\,b\,c`:`a\,b\,c`,
	`aa-bb`:`aa-bb`, `ab-ba`:`ab-ba`,
	`aa`:`aa`, `ab`:`ab`,
	`abc`:`abc`, `acc`:`acc`,
	`aox.o.a`:`aox.o.a`, `aox.o.b`:`aox.o.b`, `aox.o.c`:`aox.o.c`,
	`atomic.h,`:`atomic.h,`, // {=compound {52:29 {51:24:raw atomic.h}} {52:30:punct ,}}
	`atomic.h.`:`atomic.h.`, // {=compound {52:33 {51:24:raw atomic.h}} {52:34:punct .}}
	`ax`:`ax`,
	`axx{}`:`axx{}`, `ayy{}`:`ayy{}`,
	`a{}`:`a{}`,
	`a~`:`a~`,
	`b-3`:`b-3`,
	`b-a`:`b-a`,
	`b.h`:`b.h`,
	`ba-ab`:`ba-ab`,
	`ba`:`ba`,
	`bar,`:`bar,`,
	`bar.c`:`bar.c`,
	`bara`:`bara`, `barb`:`barb`, `barc`:`barc`,
	`baxx{}`:`baxx{}`,
	`bayy{}`:`bayy{}`,
	`bb-aa`:`bb-aa`,
	`bb`:`bb`,
	`bcc`:`bcc`,
	`box.o.a`:`box.o.a`,
	`box.o.b`:`box.o.b`,
	`box.o.c`:`box.o.c`,
	`bx`:`bx`,
	`bx{}`:`bx{}`,
	`by{}`:`by{}`,
	`bz{}`:`bz{}`,
	`b{}`:`b{}`,
	`b~`:`b~`,
	`c.D`:`c.D`, `c++.D`:`c++.D`, `c.I`:`c.I`, `c++.I`:`c++.I`,
	`c.auto`:`c.auto`, `c.test`:`c.test`,
	`c.h`:`c.h`, `d.h`:`d.h`, `e.h`:`e.h`,
	`conf3,`:`conf3,`,
	`do.smart`:`do.smart`,
	`fo-a1`:`fo-a1`, `fo-b2`:`fo-b2`, `fo-c3`:`fo-c3`,
	`fo-ax`:`fo-ax`, `fo-ay`:`fo-ay`, `fo-az`:`fo-az`,
	`fo-bx`:`fo-bx`, `fo-by`:`fo-by`, `fo-bz`:`fo-bz`,
	`fo-cx`:`fo-cx`, `fo-cy`:`fo-cy`, `fo-cz`:`fo-cz`,
	`fo-dx`:`fo-dx`, `fo-dy`:`fo-dy`, `fo-dz`:`fo-dz`,
	`fo-ex`:`fo-ex`, `fo-ey`:`fo-ey`, `fo-ez`:`fo-ez`,
	`fo-fx`:`fo-fx`, `fo-fy`:`fo-fy`, `fo-fz`:`fo-fz`,
	`fo-{&(.test.a.x.1.y.0.z)}`:[]string{`fo-ax`,`fo-ay`,`fo-az`},
	`fo-{&(.test.b.x.1.y.0.z)}`:[]string{`fo-bx`,`fo-by`,`fo-bz`},
	`fo-{&(.test.b.x.2.y.0.z)}`:[]string{`fo-cx`,`fo-cy`,`fo-cz`},
	`fo-{&(.test.c.x.1.y.0.z)}`:[]string{`fo-dx`,`fo-dy`,`fo-dz`},
	`fo-{&(.test.c.x.2.y.0.z)}`:[]string{`fo-ex`,`fo-ey`,`fo-ez`},
	`fo-{&(.test.c.x.3.y.0.z)}`:[]string{`fo-fx`,`fo-fy`,`fo-fz`},
	`fo-{&(.test.⌜a b c⌟.x.⌜1 2 3⌟.y.0.z)}`:[]string{`fo-a1`,`fo-b2`,`fo-c3`},
	`foo-B`:`foo-B`,
	`foo-a`:`foo-a`,
	`foo-b`:`foo-b`,
	`foo-bar-xx-yy-zz`:`foo-bar-xx-yy-zz`,
	`foo.c`:`foo.c`,
	`foo_ab-$1-$2`:`foo_ab-{}-{}`,
	`foo_ab-a-b`:`foo_ab-a-b`,
	`foo_ab-aa-bb`:`foo_ab-aa-bb`,
	`foo_ab-ab-ba`:`foo_ab-ab-ba`,
	`foo_ab-xx-yy`:`foo_ab-xx-yy`,
	`foo_ab-xy-yx`:`foo_ab-xy-yx`,
	`foo_ab-{}{}-{}{}`:`foo_ab-{}{}-{}{}`,
	`foo_ba-$2-$1`:`foo_ba-{}-{}`,
	`foo_ba-b-a`:`foo_ba-b-a`,
	`foo_ba-ba-ab`:`foo_ba-ba-ab`,
	`foo_ba-bb-aa`:`foo_ba-bb-aa`,
	`foo_ba-yy-xx`:`foo_ba-yy-xx`,
	`foo_ba-{}{}-{}{}`:`foo_ba-{}{}-{}{}`,
	`fooa`:`fooa`, `foob`:`foob`, `fooc`:`fooc`,
	`foobar`:`foobar`, `not-foobar`:`not-foobar`,
	`mod-1`:`mod-1`,
	`skip-nil`:`skip-nil`,
	`test-B`:`test-B`,
	`test-a`:`test-a`,
	`test-b`:`test-b`,
	`test-foo-B`:`test-foo-B`,
	`test-foo-a`:`test-foo-a`,
	`test-foo-b`:`test-foo-b`,
	`test-foo`:`test-foo`,
	`test-mod-1`:`test-mod-1`,
	`test.paniconexit0`:`test.paniconexit0`,
	`test.timeout`:`test.timeout`,
	`test.txt`:`test.txt`,
	`v-x-y-z-{}-{}`:`v-x-y-z-{}-{}`,
	`v-x-y-{}-{}-{}`:`v-x-y-{}-{}-{}`,
	`v-x-{}-{}-{}-{}`:`v-x-{}-{}-{}-{}`,
	`v-{}-{}-{}-{}-{}`:`v-{}-{}-{}-{}-{}`, 
	`wy1{}zz`:`wy1{}zz`, `wy2{}zz`:`wy2{}zz`, `wy3{}zz`:`wy3{}zz`,
	`x&(something)`:`x{}`,
	`x-a`:`x-a`, `x-b`:`x-b`, `x-c`:`x-c`, `x-`:`x-`,
	`x-y-3-xyz`:`x-y-3-xyz`, // {=compound {21:36 {19:13 {=compound {19:46 {21:23:word x}} {=flag {=compound {19:52 {21:28:word y}} {=flag {19:58 {19:33:decimal 3}}}}}}}} {=flag {=compound {21:48 {21:23:word x}} {21:53 {21:28:word y}} {21:58 {21:33:word z}}}}}
	`x-y-3-xy{}`:`x-y-3-xy{}`, // {=compound {23:36 {19:13 {=compound {19:46 {1:9:word x}} {=flag {=compound {19:52 {1:9:word y}} {=flag {19:58 {19:33:decimal 3}}}}}}}} {=flag {=compound {23:48 {1:9:word x}} {23:53 {1:9:word y}} {23:58 {19:60:null}}}}}
	`x-y-3`:`x-y-3`, // {=compound {19:46 {1:9:word x}} {=flag {=compound {19:52 {1:9:word y}} {=flag {19:58 {19:33:decimal 3}}}}}}
	`x-y-z-{}-{}`:`x-y-z-{}-{}`,
	`x-y-{}-{}-{}`:`x-y-{}-{}-{}`,
	`x-{}-{}-{}-{}`:`x-{}-{}-{}-{}`,
	`x.o.a`:`x.o.a`,
	`x.o.b`:`x.o.b`,
	`x.o.c`:`x.o.c`,
	`x\,y\,z`:`x\,y\,z`,
	`xa`:`xa`, `xb`:`xb`, `xc`:`xc`,
	`xay1z`:`xay1z`, `xay2z`:`xay2z`, `xay3z`:`xay3z`,
	`xby1z`:`xby1z`, `xby2z`:`xby2z`, `xby3z`:`xby3z`,
	`xcy1z`:`xcy1z`, `xcy2z`:`xcy2z`, `xcy3z`:`xcy3z`,
	`xq`:`xq`, `xp`:`xp`, `xx`:`xx`, `xy`:`xy`, `yx`:`yx`, `yy`:`yy`, `yy-xx`:`yy-xx`,
	`xvw`:`xvw`, `xW{}{}`:`xW{}{}`, `W{}{}`:`W{}{}`,
	`xwy1{}zz`:`xwy1{}zz`, `xwy2{}zz`:`xwy2{}zz`, `xwy3{}zz`:`xwy3{}zz`,
	`xx-yy`:`xx-yy`, // {=compound {10:20:word xx} {=flag {10:23:word yy}}}
	`xxx-yyx`:`xxx-yyx`, // {=compound {6:18:word xxx} {=flag {6:22:word yyx}}}
	`xxx-yyy`:`xxx-yyy`, // {=compound {5:18:word xxx} {=flag {5:22:word yyy}}}
	`xx{}`:`xx{}`,
	`xy-yx`:`xy-yx`,
	`xyz`:`xyz`, // {=compound {21:48 {21:23:word x}} {21:53 {21:28:word y}} {21:58 {21:33:word z}}}
	`xy{}`:`xy{}`, // {=compound {23:48 {1:9:word x}} {23:53 {1:9:word y}} {23:58 {19:60:null}}}
	`x{&(.test.foreach.x)}`:`xvw`,
	`x{&(.test.foreach.x.3)}`:`xV{}{}`,
	`x{&(.test.foreach.x.4)}`:`xW{}{}`,
	`x{&(.test.h)}a`:`x-a`,
	`x{&(.test.h)}b`:`x-b`,
	`x{&(.test.h)}c`:`x-c`,
	`x{&(.test.xx)}a`: ``,
	`x{&(.test.xx)}b`: ``,
	`x{&(.test.xx)}c`: ``,
	`x{&(.test.z)}y1{}zz`:`xwy1{}zz`,
	`x{&(.test.z)}y2{}zz`:`xwy2{}zz`,
	`x{&(.test.z)}y3{}zz`:`xwy3{}zz`,
	`x{a b c}`:[]string{`xa`, `xb`, `xc`},
	`x{}`:`x{}`,
	`x{}aa`:`x{}aa`, `x{}bb`:`x{}bb`, `x{}cc`:`x{}cc`,
	`y-3`:`y-3`, // {=compound {19:52 {1:9:word y}} {=flag {19:58 {19:33:decimal 3}}}}
	`y-x-a`:`y-x-a`,
	`y-z-{}-{}`:`y-z-{}-{}`,
	`y-{}-{}-{}`:`y-{}-{}-{}`,
	`ya`:`ya`, `yb`:`yb`,
	`yxa`:`yxa`, `yxb`:`yxb`, `yxa-yxb`:`yxa-yxb`,
	`yy{}`:`yy{}`,
	`y{}`:`y{}`,
	`z-a`:`z-a`,
	`z-y-x-a`:`z-y-x-a`,
	`z-yxa-yxb`:`z-yxa-yxb`,
	`z-{}`:`z-{}`, `z-{}-{}`:`z-{}-{}`,
	`z{}`:`z{}`,
	`{}-3`:`{}-3`, // {=compound {19:52 {19:54:null}} {=flag {19:58 {19:33:decimal 3}}}}
	`{}-{}-3-{}{}{}`:`{}-{}-3-{}{}{}`, // {=compound {23:36 {19:13 {=compound {19:46 {19:48:null}} {=flag {=compound {19:52 {19:54:null}} {=flag {19:58 {19:33:decimal 3}}}}}}}} {=flag {=compound {23:48 {19:48:null}} {23:53 {19:54:null}} {23:58 {19:60:null}}}}}
	`{}-{}-3`:`{}-{}-3`, // {=compound {19:46 {19:48:null}} {=flag {=compound {19:52 {19:54:null}} {=flag {19:58 {19:33:decimal 3}}}}}}
	`{}-{}-{}`:`{}-{}-{}`, `{}-{}-{}-{}`:`{}-{}-{}-{}`, `{}-{}-{}-{}-{}`:`{}-{}-{}-{}-{}`,
	`{}-{}`:`{}-{}`, // {=compound {3:19 {3:20:null}} {=flag {3:22 {3:23:null}}}}
	`{}aa`:`{}aa`, `{}bb`:`{}bb`, `{}cc`:`{}cc`,
	`{}{}`:`{}{}`, `{}{}{}`:`{}{}{}`, `{}{}-{}{}`:`{}{}-{}{}`,
	`{}⌜foo bar⌟{}88`:`{}⌜foo bar⌟{}88`,
	`{}⌜foo bar⌟{}99`:`{}⌜foo bar⌟{}99`,
	`~1~`:`~1~`,
	`~2~`:`~2~`,
	`~a`:`~a`,
	`~b`:`~b`,
	`~c~`:`~c~`,
}
func check__string_com(ctx Context, _c *compound, _v Value) {
	var (
		k = sfmt("%v", _c)
		t = sfmt("%v", _v)
		v = checkpoints__string_com[k]
	)
	if v == nil {
		note(pc(ctx,_c), "%v", ts(_c))
		note(pc(ctx,_c), "%v", ts(_v))
		erro(pc(ctx,_c), "`%v`:`%v`,", _c, _v).trace()
	} else {
		switch w := v.(type) {
		case []string: for _, v := range w { if t == v { return } }
		case   string:                       if t == w { return }
		}
		note(pc(ctx,_c), "%v", ts(_c))
		note(pc(ctx,_c), "%v", ts(_v))
		erro(pc(ctx,_c), `%v → %v != %v ; %v`, _c, t, v, _c.elems).trace()
	}
	return
}
