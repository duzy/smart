//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

import (
	"strings"
)

// NOTE: cannot decalre `checkpoints` as `const` because it's compile-time evaled.
var checkpoints = true // vertag != "final"

var pre_suf_checkspecs = map[string]map[string]map[string]string{
	"pre:testdata/builtins/addprefix": prefix__addprefix,
	"pre:testdata/builtins/addsuffix": prefix__addsuffix,
	"suf:testdata/builtins/addprefix": suffix__addprefix,
	"suf:testdata/builtins/addsuffix": suffix__addsuffix,
}
func pre_suf_check(ctx Context, tag string, x, y Value, res *Value) {
	if j := _project(ctx); j == nil {
		if false {
			var s = try[string](ctx, source{})
			errostack(pc(pc(ctx,y),x), 8, "nil project | %s %v %v %v", s, ts(x), ts(y), ts(*res)).trace()
		}
	} else if spec, ok := pre_suf_checkspecs[tag+":"+j.spec]; ok {
		var (
			src = strings.Split(try[string](ctx,source{}),":")
			xy = src[1]+" "+ts(x)+" "+ts(y)
			vs = spec[src[0]][xy]
		)
		if rs := (*res).String()+" "+ts(*res); vs == "" {
			if false { note(ctx, "%v", prefix__addprefix[src[0]][xy]) }
			errostack(pc(pc(ctx,y),x), 8, "%s `%v`:`%s`,", src[0], xy, rs).trace()
		} else if rs != vs {
			erro(pc(pc(ctx,y),x), "`%v`", xy)
			note(pc(pc(ctx,y),x), `got: %s`, rs)
			notestack(pc(pc(ctx,y),x), 8, `!= : %s`, vs).trace()
		}
	}
}

var checkspecs = map[string]map[string]map[string]any{
	"testdata/assert":               checkpoints__assert,
	"testdata/locals":               checkpoints__locals,
	"testdata/builtins/addprefix":   checkpoints__addprefix,
	"testdata/builtins/addsuffix":   checkpoints__addsuffix,
	"testdata/builtins/if":          checkpoints__if,
	"testdata/builtins/foreach":     checkpoints__foreach,
	"testdata/builtins/wildcard":    checkpoints__wildcard,
	"testdata/builtins/closure":     checkpoints__closure,
	"testdata/builtins/delegate":    checkpoints__delegate,
	"testdata/value":                checkpoints_value,
	"testdata/value/auto":           checkpoints_value_auto,
	"testdata/value/closure":        checkpoints_value_closure,
	"testdata/value/disjunction":    checkpoints_value_disjunction,
	"testdata/value/placeholder":    checkpoints_value_placeholder,
	"testdata/value/optional":       checkpoints_value_optional,
	"testdata/value/optional/foo":   checkpoints_value_optional_foo,
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
}
func check(ctx Context, p, x Value, res *Value) {
	var (
		d, _ = do(ctx, origin_def{}).(*def)
		dn = func() (s string) { if d != nil { s = d.name }; return } ()
		src = strings.Split(try[string](ctx,source{}),":")
		xy = src[1]+" "+s_line_column(d)+":"+dn+" "+p.String()+" "+ts(p)
		ta = checkspecs[_project(ctx).spec][src[0]][xy]
		rs = ts(x,ctx)+" "+(*res).String()+" "+ts(*res,ctx)
	)
	switch v := ta.(type) {
	case []string: for _, s := range v { if s == rs { return } }
	case string: if rs == v { return }
	case nil:
		errostack(pc(ctx,p), 8, "%s `%v`:`%s`,", src[0], xy, rs).trace()
	}
	erro(pc(ctx,p), "`%v`", xy)
	note(pc(ctx,p), `got: %s`, rs)
	notestack(pc(ctx,p), 8, `!= : %v`, ta).trace()
}

func check_cmp(ctx Context, l, r Value, _r *cmpres) {
	if cmpEqual != *_r && l.String() == r.String() {
		errostack(pc(pc(ctx,r),l), 3, "%v ⇔ %v | %v ⇔ %v → %v", l, r, ts(l), ts(r), *_r).trace()
	}
}
