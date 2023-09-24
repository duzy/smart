//
//  Copyright (C) 2012-2023, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	"regexp"
	"strings"
	"testing"
    "io/ioutil"
	"path/filepath"
	"fmt"
	"os"
)

var testValidFlag = []*regexp.Regexp{
	regexp.MustCompile(`^--target=[[:alnum:]-]+$`),
	regexp.MustCompile(`^-(?:shared|static|ObjC(?:\+\+)?)$`),
	regexp.MustCompile(`^-(?:std|Werror)=[[:alnum:]_\-]+$`),
	regexp.MustCompile(`^-(?:(?:(?:cxx|stdlib\+\+)-)?isystem(?:-after)?)=?[[:alnum:]_\-/]+$`),
	regexp.MustCompile(`^-[IL]=?[[:alnum:]_\-/]+$`),
	regexp.MustCompile(`^-[DWfl][[:alnum:]_\-]+$`),
	regexp.MustCompile(`^-no[[:alnum:]_\-+]+$`),
	regexp.MustCompile(`^-O[0-6]$`),
	regexp.MustCompile(`^-[vg]$`),
}

func validFlag(s string) (res bool) {
	for _, x := range testValidFlag { if res = x.MatchString(s); res { break }}
	return
}

func validFlags(t testcase, v Value, s string) (res bool) {
	for _, s := range strings.Fields(s) { if res = validFlag(s); !res {
		t.err("%s ; %v", s, v) ; break
	}}
	return
}

func testVariantTarget(t *testing.T) {
	if s := "variant/.target"; !testHasModule(s) { // variant/bootstrap
		t.Logf("skip %s", s)
		return
	}

	var ctx = load_testcase(t, "testdata/modules/target", "testtarget")
	if ctx.Context == nil {
		t.Errorf("fail")
		return
	}

	for k, v := range map[string]string{
		"asm": "c",
		"c": "c",
		"s": "c",
		"S": "c",
		"cpp": "c++",
		"cxx": "c++",
		"c++": "c++",
		"cc": "c++",
		"cu": "cuda",
		"cuda": "cuda",
		"cuh": "cuda",
		"m": "objc",
		"mm": "objc++",
		"swift": "swift",
	} { if d := ctx.def("lang."+k); d == nil {
		ctx.err("lang."+k)
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.value.String() != v {
		ctx.err("%v", d)
	}}

	if d := ctx.def("host.triple"); d == nil {
		ctx.err("host.triple")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.origin != DefExpand1 {
		ctx.err("%v %v", d.origin, d)
	} else if d.value.String() != "&(host.arch)-&(host.arch.ext)-&(host.vendor)-&(host.sys)-&(host.abi)" {
		ctx.err("%v", d)
	}
	if d := ctx.def("target.triple"); d == nil {
		ctx.err("target.triple")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.origin != DefExpand1 {
		ctx.err("%v %v", d.origin, d)
	} else if d.value.String() != "&(target.arch)-&(target.arch.ext)-&(target.vendor)-&(target.sys)-&(target.abi)" {
		ctx.err("%v", d)
	}

	langs  := strings.Fields("c c++ cl cuda cuda++ objc objc++ swift")
	flags1 := strings.Fields("-D -I -L -O -W -Wl -Werror -Wno-error -f -f.ld -m -g -v -no -no.ld"+
		" -isystem -isystem-after -cxx-isystem -stdlib -stdlib++-isystem -diagnostics")
	flags2 := strings.Fields("-l -framework")
	flags3 := strings.Fields("ar asm c cpp cxx oc ocxx cl cuda cudaxx ld")
	flags4 := strings.Fields("ld")
	flags5 := strings.Fields("ld.framework ldlibs loadlibs loadlibes")

	var usev, uses string
	if d := ctx.def("use.*"); d == nil {
		ctx.err("use.*")
	} else if d.origin != DefExpand1 {
		ctx.err("%v %v", d.origin, d)
	}
	if v := ctx.get("use.*"); v == nil {
		ctx.err("use.*")
	} else if usev = v.String(); usev == "" {
		ctx.err("%T %v", v, v)
	} else if uses = v.string(ctx); uses == "" {
		ctx.err("%T %v -> %s", v, v, uses)
	}

	usev = " "+usev
	uses = " "+uses

	for _, flag := range flags1 {
		s1 := fmt.Sprintf("%s(-unique)", flag)
		s2 := fmt.Sprintf("%s~&(target.sys)(-unique)", flag)
		s3 := fmt.Sprintf("%s~foo(-unique)", flag)
		if strings.Count(usev, " "+s1) != 1 { ctx.err("%v ; %v", s1, usev) }
		if strings.Count(usev, " "+s2) != 1 { ctx.err("%v ; %v", s2, usev) }
		if strings.Count(uses, " "+s3) != 1 { ctx.err("%v ; %v", s3, uses) }
		if d := ctx.def(flag); d == nil {
			ctx.err("%s", flag)
		} else if d.origin != DefExpand1 {
			ctx.err("%v %v", d.origin, d)
		}

		for _, lang := range langs {
			s0 := fmt.Sprintf("%s.%s", flag, lang)
			s1 := fmt.Sprintf("%s.%s(-unique)", flag, lang)
			s2 := fmt.Sprintf("%s~&(target.sys).%s(-unique)", flag, lang)
			s3 := fmt.Sprintf("%s~foo.%s(-unique)", flag, lang)
			if strings.Count(usev, " "+s1) != 1 { ctx.err("%v", s1) }
			if strings.Count(usev, " "+s2) != 1 { ctx.err("%v", s2) }
			if strings.Count(uses, " "+s3) != 1 { ctx.err("%v", s3) }
			if d := ctx.def(s0); d == nil {
				ctx.err("%s", s0)
			} else if d.origin != DefExpand1 {
				ctx.err("%v %v", d.origin, d)
			}
		}
	}
	for _, flag := range flags2 {
		s1 := fmt.Sprintf("%s(-unique -reverse)", flag)
		s2 := fmt.Sprintf("%s~&(target.sys)(-unique -reverse)", flag)
		s3 := fmt.Sprintf("%s~foo(-unique -reverse)", flag)
		if strings.Count(usev, " "+s1) != 1 { ctx.err("%v", s1) }
		if strings.Count(usev, " "+s2) != 1 { ctx.err("%v", s2) }
		if strings.Count(uses, " "+s3) != 1 { ctx.err("%v", s3) }
		if d := ctx.def(flag); d == nil {
			ctx.err("%s", flag)
		} else if d.origin != DefExpand1 {
			ctx.err("%v %v", d.origin, d)
		}
	}
	for _, flag := range flags3 {
		s0 := fmt.Sprintf("%sflags", flag)
		s1 := fmt.Sprintf("%sflags(-unique -auto)", flag)
		s2 := fmt.Sprintf("%sflags~&(target.sys)(-unique -auto)", flag)
		s3 := fmt.Sprintf("%sflags~foo(-unique -auto)", flag)
		if strings.Count(usev, " "+s1) != 1 { ctx.err("%v", s1) }
		if strings.Count(usev, " "+s2) != 1 { ctx.err("%v", s2) }
		if strings.Count(uses, " "+s3) != 1 { ctx.err("%v", s3) }
		if d := ctx.def(s0); d == nil {
			ctx.err("%s", s0)
		} else if d.origin != DefExpand1 {
			ctx.err("%v %v", d.origin, d)
		}
	}
	for _, flag := range flags4 { for _, suffix := range strings.Fields("shared program") {
		s0 := fmt.Sprintf("%sflags.%s", flag, suffix)
		s1 := fmt.Sprintf("%sflags.%s(-unique -auto)", flag, suffix)
		s2 := fmt.Sprintf("%sflags~&(target.sys).%s(-unique -auto)", flag, suffix)
		s3 := fmt.Sprintf("%sflags~foo.%s(-unique -auto)", flag, suffix)
		if strings.Count(usev, " "+s1) != 1 { ctx.err("%v", s1) }
		if strings.Count(usev, " "+s2) != 1 { ctx.err("%v", s2) }
		if strings.Count(uses, " "+s3) != 1 { ctx.err("%v", s3) }
		if d := ctx.def(s0); d == nil {
			ctx.err("%s", s0)
		} else if d.origin != DefExpand1 {
			ctx.err("%v %v", d.origin, d)
		}
	}}
	for _, flag := range flags5 {
		s0 := fmt.Sprintf("%s", flag)
		s1 := fmt.Sprintf("%s(-unique -auto -reverse)", flag)
		s2 := fmt.Sprintf("%s~&(target.sys)(-unique -auto -reverse)", flag)
		s3 := fmt.Sprintf("%s~foo(-unique -auto -reverse)", flag)
		if strings.Count(usev, " "+s1) != 1 { ctx.err("%v", s1) }
		if strings.Count(usev, " "+s2) != 1 { ctx.err("%v", s2) }
		if strings.Count(uses, " "+s3) != 1 { ctx.err("%v", s3) }
		if d := ctx.def(s0); d == nil {
			ctx.err("%s", s0)
		} else if d.origin != DefExpand1 {
			ctx.err("%v %v", d.origin, d)
		}
	}

	if v := ctx.get("neg1"); v == nil {
		ctx.err("neg1")
	} else if v.String() != "!foobar" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "!foobar" {
		ctx.err("%T %v -> %s", v, v, s)
	} else if t, y := v.(*negative); !y {
		ctx.err("%T %v", v, v)
	} else if t.true(ctx) {
		ctx.err("%T %v", v, v)
	}
	if v := ctx.get("neg2"); v == nil {
		ctx.err("neg2")
	} else if v.String() != "a!foobar" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "a!foobar" {
		ctx.err("%T %v -> %s", v, v, s)
	}
	if v := ctx.get("neg3"); v == nil {
		ctx.err("neg3")
	} else if v.String() != "&(a!foobar)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "xxx" {
		ctx.err("%T %v -> %s", v, v, s)
	}
	if v := ctx.get("neg4", "xxx"); v == nil {
		ctx.err("neg4")
	} else if v.String() != "&(a!xxx)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "" {
		ctx.err("%T %v -> %s", v, v, s)
	}
	if v := ctx.get("neg4", "foobar"); v == nil {
		ctx.err("neg4")
	} else if v.String() != "&(a!foobar)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "xxx" {
		ctx.err("%T %v -> %s", v, v, s)
	}

	var cflagv, cflags string
	if v := ctx.get("cflags"); v == nil {
		ctx.err("cflags")
	} else if cflagv = v.String(); cflagv == "" {
		ctx.err("%T %v", v, v)
	} else if cflags = v.string(ctx); cflags == "" {
		ctx.err("%T %v -> %s", v, v, cflags)
	}

	if true {
		info(ctx, "%v", cflagv)
		info(ctx, "%v", cflags).debug(1)
	}

	ctx.flush()
}

func testApp(t *testing.T) {
	if s := "app"; !testHasModule(s) {
		t.Logf("skip %s", s)
		return
	}

	var ctx = load_testcase(t, "testdata/modules/app", "testapp")
	if ctx.Context == nil {
		t.Errorf("fail")
		return
	}

	ss := func(s string) string { os := "darwin"
		return strings.Replace(s, "<OS>", os, -1)
	}

	flag1 := func(a ...interface{}) string { // $(.flag $1)
		return fmt.Sprintf("$(foreach(-unique) $(filter-out $(foreach $1,&(%[1]s!$_) &(%[1]s~&(target.sys)!$_)),&(%[1]s) &(%[1]s~&(target.sys))) $(foreach $1,&(%[1]s.$_) &(%[1]s~&(target.sys).$_)),$(or $3,%[1]s)$_$(or $4))", a...)
	}
	flag2 := func(a ...interface{}) string { if len(a) > 1 { a[0], a[1] = a[1], a[0] } // $(.flag $1 yyy,xxx)
		return fmt.Sprintf("$(foreach(-unique) $(filter-out $(foreach $1 %[1]s,&(%[2]s!$_) &(%[2]s~&(target.sys)!$_)),&(%[2]s) &(%[2]s~&(target.sys))) $(foreach $1 %[1]s,&(%[2]s.$_) &(%[2]s~&(target.sys).$_)),%[2]s$_$(or $4))", a...)
	}
	flag3 := func(a ...interface{}) string { // $(.flag $1,xxx,yy)
		return fmt.Sprintf("$(foreach(-unique) $(filter-out &(%[1]s!%[2]s) &(%[1]s~&(target.sys)!%[2]s),&(%[1]s) &(%[1]s~&(target.sys))) &(%[1]s.%[2]s) &(%[1]s~&(target.sys).%[2]s),$(or $3,%[1]s)$_$(or $4))", a...)
	}
	flag4 := func(a ...interface{}) string { // $(.flag $1,xxx,y,y)
		return fmt.Sprintf("$(foreach(-unique) $(filter-out &(%[1]s!%[2]s) &(%[1]s~&(target.sys)!%[2]s) &(%[1]s!c) &(%[1]s~&(target.sys)!c),&(%[1]s) &(%[1]s~&(target.sys))) &(%[1]s.%[2]s) &(%[1]s~&(target.sys).%[2]s) &(%[1]s.%[3]s) &(%[1]s~&(target.sys).%[3]s),%[1]s$_$(or $4))", a...)
	}

	if d := ctx.def(".flag"); d == nil {
		ctx.err(".flag")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if s := d.value.String(); s == "" {
		ctx.err("%v", d)
	} else if strings.Count(s, "$(foreach(-unique) ") != 1 {
		ctx.err("%v", d)
	} else if strings.Count(s, "$(filter-out $(foreach $1,&($2!$_) &($2~&(target.sys)!$_)),&($2) &($2~&(target.sys)))") != 1 {
		ctx.err("%v", d)
	} else if strings.Count(s, "$(foreach $1,&($2.$_) &($2~&(target.sys).$_)),") != 1 {
		ctx.err("%v", d)
	} else if strings.Count(s, ",$(or $3,$2)$_$(or $4))") != 1 {
		ctx.err("%v", d)
	} else if s != flag1("$2") {
		ctx.err("%v ; %v", s, d)
	}
	if v := ctx.get(".flag", []string{"a", "b", "c"}, "-x"); v == nil {
		ctx.err(".flag")
	} else if s := v.String(); s == "" {
		ctx.err("%v", v)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out &") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x!a) &(-x~&(target.sys)!a)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x!b) &(-x~&(target.sys)!b)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x!c) &(-x~&(target.sys)!c),") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, ",&(-x) &(-x~&(target.sys))") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x.a) &(-x~&(target.sys).a)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x.b) &(-x~&(target.sys).b)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x.c) &(-x~&(target.sys).c)") != 1 {
		ctx.err("%v", v)
	} else if s1 := "$(filter-out &(-x!a) &(-x~&(target.sys)!a) &(-x!b) &(-x~&(target.sys)!b) &(-x!c) &(-x~&(target.sys)!c),&(-x) &(-x~&(target.sys)))"; strings.Count(s, s1) != 1 {
		ctx.err("%v", v)
	} else if s2 := "&(-x.a) &(-x~&(target.sys).a) &(-x.b) &(-x~&(target.sys).b) &(-x.c) &(-x~&(target.sys).c)"; strings.Count(s, s2) != 1 {
		ctx.err("%v", v)
	} else if s3 := "$(or $3,-x)$_$(or $4))"; strings.Count(s, s3) != 1 {
		ctx.err("%v", v)
	} else if s != "$(foreach(-unique) "+s1+" "+s2+","+s3 {
		ctx.err("%v", v)
	} else if s := v.string(ctx); s == "" {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-xxx") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-xyy") != 0 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-xzz") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-xsx") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-xsy") != 0 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-xsz") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-xsa") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-xxa") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-xxb") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-xxc") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if s != "-xxx -xzz -xsx -xsz -xxa -xsa -xxb -xxc" {
		ctx.err("%v ; %v", s, v)
	}
	if v := ctx.get(".flag", []string{"a", "b", "c"}, "-z"); v == nil {
		ctx.err(".flag")
	} else if s := v.String(); s == "" {
		ctx.err("%v", v)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out &") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-z!a) &(-z~&(target.sys)!a)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-z!b) &(-z~&(target.sys)!b)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-z!c) &(-z~&(target.sys)!c),") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, ",&(-z) &(-z~&(target.sys))") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-z.a) &(-z~&(target.sys).a)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-z.b) &(-z~&(target.sys).b)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-z.c) &(-z~&(target.sys).c)") != 1 {
		ctx.err("%v", v)
	} else if s1 := "$(filter-out &(-z!a) &(-z~&(target.sys)!a) &(-z!b) &(-z~&(target.sys)!b) &(-z!c) &(-z~&(target.sys)!c),&(-z) &(-z~&(target.sys)))"; strings.Count(s, s1) != 1 {
		ctx.err("%v", v)
	} else if s2 := "&(-z.a) &(-z~&(target.sys).a) &(-z.b) &(-z~&(target.sys).b) &(-z.c) &(-z~&(target.sys).c)"; strings.Count(s, s2) != 1 {
		ctx.err("%v", v)
	} else if s3 := "$(or $3,-z)$_$(or $4))"; strings.Count(s, s3) != 1 {
		ctx.err("%v", v)
	} else if s != "$(foreach(-unique) "+s1+" "+s2+","+s3 {
		ctx.err("%v", v)
	} else if s := v.string(ctx); s != "" {
		ctx.err("%v ; %v", s, v)
	}
	if v := ctx.get(".flag", []string{"a", "b", "c"}, "-x", "-y"); v == nil {
		ctx.err(".flag")
	} else if s := v.String(); s == "" {
		ctx.err("%v", v)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out &") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x!a) &(-x~&(target.sys)!a)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x!b) &(-x~&(target.sys)!b)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x!c) &(-x~&(target.sys)!c),") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, ",&(-x) &(-x~&(target.sys))") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x.a) &(-x~&(target.sys).a)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x.b) &(-x~&(target.sys).b)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x.c) &(-x~&(target.sys).c)") != 1 {
		ctx.err("%v", v)
	} else if s1 := "$(filter-out &(-x!a) &(-x~&(target.sys)!a) &(-x!b) &(-x~&(target.sys)!b) &(-x!c) &(-x~&(target.sys)!c),&(-x) &(-x~&(target.sys)))"; strings.Count(s, s1) != 1 {
		ctx.err("%v", v)
	} else if s2 := "&(-x.a) &(-x~&(target.sys).a) &(-x.b) &(-x~&(target.sys).b) &(-x.c) &(-x~&(target.sys).c)"; strings.Count(s, s2) != 1 {
		ctx.err("%v", v)
	} else if s3 := "-y$_$(or $4))"; strings.Count(s, s3) != 1 {
		ctx.err("%v", v)
	} else if s != "$(foreach(-unique) "+s1+" "+s2+","+s3 {
		ctx.err("%v", v)
	} else if s := v.string(ctx); s == "" {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-yxx") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-yyy") != 0 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-yzz") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-ysx") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-ysy") != 0 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-ysz") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-ysa") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-yxa") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-yxb") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-yxc") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if s != "-yxx -yzz -ysx -ysz -yxa -ysa -yxb -yxc" {
		ctx.err("%v ; %v", s, v)
	}

	if d1, d2 := ctx.def("cppflags"), ctx.def("fooflags"); d1 == nil || d2 == nil {
		ctx.err("cppflags")
		ctx.err("fooflags")
	} else if d1.value == nil || d2.value == nil {
		ctx.err("%v", d1)
		ctx.err("%v", d2)
	} else if s := d1.value.String(); strings.Count(s, "&(patsubst %,--target=%,&(target.triple))") != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, "$(foreach(-unique) $1 $2,&(-v.$_))") != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, "$(foreach(-unique) $1 $2,-std=&(-std.$_) -std=&(std.$_))") != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, flag2("-D", "$2")) != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, flag2("-f", "$2")) != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, flag2("-I", "$2")) != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, flag2("-isystem", "$2")) != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, flag2("-isystem-after", "$2")) != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, flag1("cppflags")) != 1 {
		ctx.err("%v", d1.value)
	} else if s := d2.value.String(); strings.Count(s, "&(patsubst %,--target=%,&(target.triple))") != 1 {
		ctx.err("%v", d2.value)
	} else if strings.Count(s, "$(foreach(-unique) $1 $2,&(-v.$_))") != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, "$(foreach(-unique) $1 $2,-std=&(-std.$_) -std=&(std.$_))") != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, flag2("-D", "$2")) != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, flag2("-f", "$2")) != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, flag2("-I", "$2")) != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, flag2("-isystem", "$2")) != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, flag2("-isystem-after", "$2")) != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, flag1("cppflags")) != 1 {
		ctx.err("%v", d1.value)
	} else if s1, s2 := d1.value.string(ctx), d2.value.string(ctx); s1 != s2 {
		ctx.err("%v", s1)
		ctx.err("%v", s2)
		ctx.err("%v", d1.value)
		ctx.err("%v", d2.value)
	}

	if v1, v2 := ctx.get("cflags"), ctx.get("xflags"); v1 == nil || v2 == nil {
		ctx.err("cflags")
		ctx.err("xflags")
	} else if s := v1.String(); s == "" {
		ctx.err("%T %v", v1, v1)
	} else if strings.Count(s, "&(patsubst %,--target=%,&(target.triple))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(foreach(-unique) $1 c,&(-v.$_))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(foreach(-unique) $1 c,-std=&(-std.$_) -std=&(std.$_))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(if $(or &(-g) &(-g~&(target.sys)) $(foreach $1 c,&(-g.$_) &(-g~&(target.sys).$_))),-g)") != 1 {
		ctx.err("%v", v1)
	} else if t := flag2("-g", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v1), "%v", t) ; ctx.err("%v", v1)
	} else if t := flag2("-O", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v1), "%v", t) ; ctx.err("%v", v1)
	} else if t := flag2("-D", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v1), "%v", t) ; ctx.err("%v", v1)
	} else if t := flag2("-f", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v1), "%v", t) ; ctx.err("%v", v1)
	} else if t := flag2("-m", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v1), "%v", t) ; ctx.err("%v", v1)
	} else if t := flag2("-W", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v1), "%v", t) ; ctx.err("%v", v1)
	} else if t := flag2("-I", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v1), "%v", t) ; ctx.err("%v", v1)
	} else if t := flag2("-no", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v1), "%v", t) ; ctx.err("%v", v1)
	} else if t := flag2("-isystem", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v1), "%v", t) ; ctx.err("%v", v1)
	} else if t := flag2("-isystem-after", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v1), "%v", t) ; ctx.err("%v", v1)
	} else if t := flag1("cflags"); strings.Count(s, t) != 1 {
		noted(of(ctx,v1), "%v", t) ; ctx.err("%v", v1)
	} else if s := v2.String(); s == "" {
		ctx.err("%T %v", v2, v2)
	} else if strings.Count(s, "&(patsubst %,--target=%,&(target.triple))") != 1 {
		ctx.err("%v", v2)
	} else if strings.Count(s, "$(foreach(-unique) $1 c,&(-v.$_))") != 1 {
		ctx.err("%v", v2)
	} else if strings.Count(s, "$(foreach(-unique) $1 c,-std=&(-std.$_) -std=&(std.$_))") != 1 {
		ctx.err("%v", v2)
	} else if strings.Count(s, "$(if $(or &(-g) &(-g~&(target.sys)) $(foreach $1 c,&(-g.$_) &(-g~&(target.sys).$_))),-g)") != 1 {
		ctx.err("%v", v2)
	} else if t := flag2("-g", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v2), "%v", t) ; ctx.err("%v", v2)
	} else if t := flag2("-O", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v2), "%v", t) ; ctx.err("%v", v2)
	} else if t := flag2("-D", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v2), "%v", t) ; ctx.err("%v", v2)
	} else if t := flag2("-f", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v2), "%v", t) ; ctx.err("%v", v2)
	} else if t := flag2("-m", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v2), "%v", t) ; ctx.err("%v", v2)
	} else if t := flag2("-W", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v2), "%v", t) ; ctx.err("%v", v2)
	} else if t := flag2("-I", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v2), "%v", t) ; ctx.err("%v", v2)
	} else if t := flag2("-no", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v2), "%v", t) ; ctx.err("%v", v2)
	} else if t := flag2("-isystem", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v2), "%v", t) ; ctx.err("%v", v2)
	} else if t := flag2("-isystem-after", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v2), "%v", t) ; ctx.err("%v", v2)
	} else if t := flag1("cflags"); strings.Count(s, t) != 1 {
		noted(of(ctx,v2), "%v", t) ; ctx.err("%v", v2)
	} else if s1, s2 := v1.string(ctx), v2.string(ctx); s1 != s2 {
		ctx.err("%T %v -> %s", v2, v2, s)
		ctx.err("%T %v -> %s", v2, v2, s)
	}

	if v := ctx.get("std.fxxbxx"); v == nil {
		ctx.err("std.fxxbxx")
	} else if s := v.string(ctx); s != "stdfxxbxx1" {
		ctx.err("%T %v -> %s", v, v, s)
	}
	if v := ctx.get("-std.fxxbxx"); v == nil {
		ctx.err("-std.fxxbxx")
	} else if s := v.string(ctx); s != "stdfxxbxx2" {
		ctx.err("%T %v -> %s", v, v, s)
	}
	if v1, v2 := ctx.get("cflags", "fxxbxx"), ctx.get("xflags", "fxxbxx"); v1 == nil || v2 == nil {
		ctx.err("cflags")
		ctx.err("xflags")
	} else if s := v1.String(); s == "" {
		ctx.err("%T %v", v1, v1)
	} else if strings.Count(s, "&(patsubst %,--target=%,&(target.triple))") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "&(-v.fxxbxx)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "&(-v.c)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "-std=&(-std.fxxbxx)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "-std=&(std.fxxbxx)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "-std=&(-std.c)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "-std=&(std.c)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "$(if $(or &(-g) &(-g~&(target.sys)) &(-g.fxxbxx) &(-g~&(target.sys).fxxbxx) &(-g.c) &(-g~&(target.sys).c)),-g)") != 1 {
		ctx.err("%v", s)
	} else if t := flag4("-g", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v1), "%v", t) ; ctx.err("%v", v1)
	} else if t := flag4("-O", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v1), "%v", t) ; ctx.err("%v", v1)
	} else if t := flag4("-D", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v1), "%v", t) ; ctx.err("%v", v1)
	} else if t := flag4("-f", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v1), "%v", t) ; ctx.err("%v", v1)
	} else if t := flag4("-m", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v1), "%v", t) ; ctx.err("%v", v1)
	} else if t := flag4("-W", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v1), "%v", t) ; ctx.err("%v", v1)
	} else if t := flag4("-I", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v1), "%v", t) ; ctx.err("%v", v1)
	} else if t := flag4("-no", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v1), "%v", t) ; ctx.err("%v", v1)
	} else if t := flag4("-isystem", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v1), "%v", t) ; ctx.err("%v", v1)
	} else if t := flag4("-isystem-after", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v1), "%v", t) ; ctx.err("%v", v1)
	} else if t := flag3("cflags", "fxxbxx"); strings.Count(s, t) != 1 {
		noted(of(ctx,v1), "%v", t) ; ctx.err("%v", v1)
	} else if s := v2.String(); s == "" {
		ctx.err("%T %v", v2, v2)
	} else if strings.Count(s, "&(patsubst %,--target=%,&(target.triple))") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "&(-v.fxxbxx)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "&(-v.c)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "-std=&(-std.fxxbxx)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "-std=&(std.fxxbxx)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "-std=&(-std.c)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "-std=&(std.c)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "$(if $(or &(-g) &(-g~&(target.sys)) &(-g.fxxbxx) &(-g~&(target.sys).fxxbxx) &(-g.c) &(-g~&(target.sys).c)),-g)") != 1 {
		ctx.err("%v", s)
	} else if t := flag4("-g", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v2), "%v", t) ; ctx.err("%v", v2)
	} else if t := flag4("-O", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v2), "%v", t) ; ctx.err("%v", v2)
	} else if t := flag4("-D", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v2), "%v", t) ; ctx.err("%v", v2)
	} else if t := flag4("-f", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v2), "%v", t) ; ctx.err("%v", v2)
	} else if t := flag4("-m", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v2), "%v", t) ; ctx.err("%v", v2)
	} else if t := flag4("-W", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v2), "%v", t) ; ctx.err("%v", v2)
	} else if t := flag4("-I", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v2), "%v", t) ; ctx.err("%v", v2)
	} else if t := flag4("-no", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v2), "%v", t) ; ctx.err("%v", v2)
	} else if t := flag4("-isystem", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v2), "%v", t) ; ctx.err("%v", v2)
	} else if t := flag4("-isystem-after", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		noted(of(ctx,v2), "%v", t) ; ctx.err("%v", v2)
	} else if t := flag3("cflags", "fxxbxx"); strings.Count(s, t) != 1 {
		noted(of(ctx,v2), "%v", t) ; ctx.err("%v", v2)
	} else if s1 := v1.string(ctx); s1 == "" {
		ctx.err("%T %v -> %s", v2, v2, s1)
	} else if s2 := v2.string(ctx); s2 == "" {
		ctx.err("%T %v -> %s", v2, v2, s2)
	} else if !validFlags(ctx, v1, s1) {
		ctx.err("%s", s1)
	} else if !validFlags(ctx, v2, s2) {
		ctx.err("%s", s2)
	} else if s := s1+s2; s1 != s1 {
		ctx.err("%s", s1)
		ctx.err("%s", s1)
		ctx.err("%v", v1)
		ctx.err("%v", v2)
	} else if strings.Count(s, "std=stdfxxbxx1") != 2 {
		ctx.err("%s", s1)
		ctx.err("%s", s2)
		ctx.err("%v", v1)
		ctx.err("%v", v2)
	} else if strings.Count(s, "-std=stdfxxbxx2") != 2 {
		ctx.err("%s", s1)
		ctx.err("%s", s2)
		ctx.err("%v", v1)
		ctx.err("%v", v2)
	}

	var foo1 = strings.Fields(ss(`cppflags-foo cppflags~foo~<OS>
-std=foostd -ffooF -IfooI -DfooD
-isystemfooisystem -isystem-afterfooisystem-after`))
	var foo2 = strings.Fields(ss(`cxxflags-foo cxxflags~foo~<OS>
-std=foostd -ffooF -IfooI -DfooD -gfooG -OfooO
-isystemfooisystem -isystem-afterfooisystem-after
-cxx-isystemfoocxxisystem -stdlib++-isystemfoostdlib++isystem`))
	var foo3 = strings.Fields(ss(`ldflags-foo ldflags~foo~<OS> -ffooF -OfooO -LfooL
-Wl,fooWl -Wl,-rpath,"foorpath"`))
	var foo4 = strings.Fields(ss(`ldlibs-foo ldlibs~foo~<OS> -lfool`))
	var foo5 = strings.Fields(ss(`loadlibes-foo loadlibes~foo~<OS>`))
	var foo6 = strings.Fields(ss(`loadlibs-foo loadlibs~foo~<OS>`))

	if v := ctx.get(".test.1"); v == nil {
		ctx.err(".test.1")
	} else if s := v.String(); s == "" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s == "" {
		ctx.err("%T %v ⇒ %s", v, v, s)
	} else if false { for _, t := range foo1 { if n := strings.Count(s, t); n != 1 {
		ctx.err("%v : %s ; %T %v", t, s, v, v)
	}}} else if strings.Count(s, "-std=fxxbar") != 1 {
		ctx.err("%T %v ⇒ %s", v, v, s)
	}

	if v := ctx.get(".test.2"); v == nil {
		ctx.err(".test.2")
	} else if s := v.String(); s == "" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s == "" {
		ctx.err("%T %v -> %s", v, v, s)
	} else if false { for _, t := range foo1 { if n := strings.Count(s, t); n != 1 {
		ctx.err("%v : %s ; %T %v", t, s, v, v)
	}}} else if strings.Count(s, "-std=fxxbar") != 1 {
		ctx.err("%v : %s ; %T %v", t, s, v, v)
	}

	if v := ctx.get(".test.3"); v == nil {
		ctx.err(".test.3")
	} else if s := v.String(); s == "" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s == "" {
		ctx.err("%T %v -> %s", v, v, s)
	} else if false { for _, t := range foo1 { if n := strings.Count(s, t); n != 1 {
		ctx.err("%v : %s ; %T %v", t, s, v, v)
	}}} else if strings.Count(s, "-std=fxxbar") != 1 {
		ctx.err("%v : %s ; %T %v", t, s, v, v)
	}

	if v := ctx.get(".test.4"); v == nil {
		ctx.err(".test.4")
	} else if s := v.String(); s == "" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s == "" {
		ctx.err("%T %v -> %s", v, v, s)
	} else { for _, t := range foo1 { if n := strings.Count(s, t); n != 1 {
		ctx.err("%v (%d) : %s ; %T %v", t, n, s, v, v)
	}}}

	if v := ctx.get(".test.5"); v == nil {
		ctx.err(".test.5")
	} else if s := v.String(); s == "" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s == "" {
		ctx.err("%T %v -> %s", v, v, s)
	} else { for _, t := range foo2 { if n := strings.Count(s, t); n != 1 {
		ctx.err("%v (%d) : %s ; %T %v", t, n, s, v, v)
	}}}

	if v := ctx.get(".test.6"); v == nil {
		ctx.err(".test.6")
	} else if s := v.String(); s == "" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s == "" {
		ctx.err("%T %v -> %s", v, v, s)
	} else { for _, t := range foo3 { if n := strings.Count(s, t); n != 1 {
		ctx.err("%v (%d) : %s ; %T %v", t, n, s, v, v)
	}}}

	if v := ctx.get(".test.7"); v == nil {
		ctx.err(".test.7")
	} else if s := v.String(); s == "" {
		ctx.err("%T %v", v, v)
	} else if strings.Count(s, flag3("ldlibs", "foo")) != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, flag3("-l", "foo")) != 1 {
		ctx.err("%v", s)
	} else if s := v.string(ctx); s == "" {
		ctx.err("%T %v -> %s", v, v, s)
	} else { for _, t := range foo4 { if n := strings.Count(s, t); n != 1 {
		ctx.err("%v (%d) : %s ; %T %v", t, n, s, v, v)
	}}}

	if v := ctx.get(".test.8"); v == nil {
		ctx.err(".test.8")
	} else if s := v.String(); s == "" {
		ctx.err("%T %v", v, v)
	} else if strings.Count(s, flag3("loadlibes", "foo")) != 1 {
		ctx.err("%v", s)
	} else if s := v.string(ctx); s == "" {
		ctx.err("%T %v -> %s", v, v, s)
	} else { for _, t := range foo5 { if n := strings.Count(s, t); n != 1 {
		ctx.err("%v (%d) : %s ; %T %v", t, n, s, v, v)
	}}}

	if v := ctx.get(".test.9"); v == nil {
		ctx.err(".test.9")
	} else if s := v.String(); s == "" {
		ctx.err("%T %v", v, v)
	} else if strings.Count(s, flag3("loadlibs", "foo")) != 1 {
		ctx.err("%v", s)
	} else if s := v.string(ctx); s == "" {
		ctx.err("%T %v -> %s", v, v, s)
	} else { for _, t := range foo6 { if n := strings.Count(s, t); n != 1 {
		ctx.err("%v (%d) : %s ; %T %v", t, n, s, v, v)
	}}}

	ctx.flush()
}

func checkLLVMConfig(ctx testcase, s string) {
	if strings.Count(s, "FOO = $(.self)") != 1 {
		ctx.Errorf("%s", s)
	}
}

func testLLVMConfigConfigure(t *testing.T) {
	var cl = init_commandline()
	cl.configure = true

	var ctx = load_testcase(t, "testdata/modules/llvm/config", "", cl)
	if ctx.Context == nil {
		t.Errorf("configure fail")
		return
	}

	defer assured(ctx, true)

	var base, general *Project
	var m = ctx.Project()
	if m == nil {
		ctx.Errorf("configure fail")
		return
	} else if len(m.bases) != 1 {
		ctx.Errorf("bases: %v", m.bases)
		return
	} else if base = m.bases[0]; base.name != "llvm.Config" {
		ctx.Errorf("base: %v", base)
		return
	} else if base.configure == nil {
		ctx.Errorf("configure fail")
		return
	} else if f := file(ctx, ".configure/type/test.c", base.configure); f == nil {
		ctx.Errorf("file .configure/type/test.c")
		return
	} else if f := file(ctx, ".configure/type/xxx/test.c", base.configure); f == nil {
		ctx.Errorf("file .configure/type/xxx/test.c")
		return
	} else if f := file(ctx, ".configure/type/xxx/yyy/test.c", base.configure); f == nil {
		ctx.Errorf("file .configure/type/xxx/yyy/test.c")
		return
	}

	if o := m.resolveObject(ctx, "general"); o == nil {
		ctx.Errorf("general")
	} else if p, y := o.(*projectname); !y {
		ctx.err("%T %v", o, o)
	} else {
		general = p.Project
	}

	if v := ctx.get("/"); v == nil {
		ctx.Errorf("/")
	} else if v.String() != m.absPath {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != m.absPath {
		ctx.err("%T %v %s", v, v, s)
	} else if _, y := v.(*Path); !y {
		ctx.err("%T %v", v, v)
	}

	var chop = fmt.Sprintf("%%%%/.smart/modules/ %s/ %s/ %s/",
		filepath.Dir(general.absPath),
		filepath.Dir(filepath.Dir(general.absPath)),
		filepath.Dir(filepath.Dir(filepath.Dir(general.absPath))))
	if v := ctx.get("rel.chop"); v == nil { // from general
		ctx.Errorf("rel.chop")
	} else if !strings.HasSuffix(v.String(), chop) {
		ctx.err("%T %v", v, v)
	}

	var remnant = ctx.get("rel.remnant")
	if v := remnant; v == nil {
		ctx.Errorf("rel.remnant")
	} else if v.String() != "&(trim-prefix &(rel.chop),&/)" { // from general
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s == "" {
		ctx.err("%T %v", v, v)
	} else if _, y := v.(unexpanded); !y {
		ctx.err("%T %v", v, v)
	}

	var cc1 = closureWith(ctx, base.configure.scope, base.scope)
	var cc2 = closureWith(ctx, base.scope, base.configure.scope)
	if v := remnant; v == nil {
		ctx.Errorf("remnant")
	} else if s := v.string(cc1); s == "" {
		ctx.err("%T %v", v, v)
	}

	if d := base.resolveDef(cc1, "rel.remnant"); d == nil {
		ctx.Errorf("rel.remnant")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "&(trim-prefix &(rel.chop),&/)" { // from general
		ctx.err("%T %v", v, v)
	} else if s := v.string(cc1); s == "" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(cc2); s == "" {
		ctx.err("%T %v", v, v)
	} else if _, y := v.(unexpanded); !y {
		ctx.err("%T %v", v, v)
	}

	p1 := strings.Split(m.absPath,PathSep)
	p2 := strings.Split(base.absPath,PathSep)
	t1 := filepath.Join(append(p1[len(p1)-4:], configuration_sm)...)
	t2 := filepath.Join(append(p2[len(p2)-2:], configuration_sm)...)

	if f := m.tempFile(ctx, configuration_sm); f == nil {
		ctx.err("%v: nil %s", m, configuration_sm)
	} else if s1, s2 := f.fullname(), base.configurationFile.fullname(); s1 == s2 {
		erro(ctx, "%v: %v: %v", m, base, s1)
		erro(ctx, "%v: %v: %v", m, base, s2).debug(1)
		ctx.Errorf("%v: %v", m, base)
	} else if s1 != t1 {
		ctx.Errorf("%v: %v", m, s1)
	} else if s2 != t2 {
		ctx.Errorf("%v: %v", m, s2)
	}
	if f := base.tempFile(ctx, configuration_sm); f == nil {
		ctx.err("%v: nil %s", m, configuration_sm)
	} else if s1, s2 := f.fullname(), base.configurationFile.fullname(); s1 == s2 {
		erro(ctx, "%v: %v: %v", m, base, s1)
		erro(ctx, "%v: %v: %v", m, base, s2).debug(1)
		ctx.Errorf("%v: %v", m, base)
	} else if s1 != t1 {
		ctx.Errorf("%v: %v", m, s1)
	} else if s2 != t2 {
		ctx.Errorf("%v: %v", m, s2)
	}

	if f := base.configuration(cc1); f == nil {
		ctx.err("%v: %v: nil configuration", m, base)
	} else if s1, s2 := f.fullname(), base.configurationFile.fullname(); s1 == s2 {
		erro(ctx, "%v: %v: %v", m, base, s1)
		erro(ctx, "%v: %v: %v", m, base, s2).debug(1)
		ctx.Errorf("%v: %v", m, base)
	} else if s1 != t1 {
		ctx.Errorf("%v: %v", m, s1)
	} else if s2 != t2 {
		ctx.Errorf("%v: %v", m, s2)
	}

	if v := ctx.get("val1"); v == nil {
		ctx.err("val1")
	} else if s := v.String(); s != base.configurationFile.name(ctx) {
		ctx.err("%v , %v", v, base.configurationFile.name(ctx))
	} else if s1, s2 := v.string(ctx), base.configurationFile.fullname(); s1 == s2 {
		erro(ctx, "%v: %v: %v", m, base, s1)
		erro(ctx, "%v: %v: %v", m, base, s2).debug(1)
		ctx.Errorf("%v: %v", m, base)
	} else if s1 != t1 {
		ctx.Errorf("%v: %v", m, s1)
	} else if s2 != t2 {
		ctx.Errorf("%v: %v", m, s2)
	}
	if v := ctx.get("val2"); v == nil {
		ctx.err("val2")
	} else if s := v.String(); s != base.configurationFile.name(ctx) {
		ctx.err("%v , %v", v, base.configurationFile.name(ctx))
	} else if f, y := v.(*File); !y {
		ctx.err("%T %v", v, v)
	} else if s1, s2 := f.fullname(), base.configurationFile.fullname(); s1 == s2 {
		erro(ctx, "%v: %v: %v", m, base, s1)
		erro(ctx, "%v: %v: %v", m, base, s2).debug(1)
		ctx.Errorf("%v: %v", m, base)
	} else if s1 != t1 {
		ctx.Errorf("%v: %v", m, s1)
	} else if s2 != t2 {
		ctx.Errorf("%v: %v", m, s2)
	}

	var rm func(*Project)
	rm = func(p *Project) {
		if f := p.configurationFile; f == nil {
			// skip
		} else if s := f.fullname(); s == "" {
			ctx.err("%v", f)
		} else if e := os.RemoveAll(s); e != nil {
			ctx.err("%v", e)
		} else if e := os.RemoveAll(filepath.Join(filepath.Dir(s),".configure")); e != nil {
			ctx.err("%v", e)
		} else if false {
			noted(ctx, "%v", s).debug(1)
		}
		for _, base := range p.bases { rm(base) }
	}
	rm(ctx.Project())

	ctx.universe().configure(ctx)

	if f := base.configurationFile; f == nil {
		ctx.err("%v: nil configuration", base)
	} else if i, e := os.Stat(f.fullname()); e != nil {
		ctx.err("%s: %v", configuration_sm, e)
	} else if i == nil {
		ctx.err("missing %s", f.fullname())
	}

	if o := base.configure.resolveObject(ctx, "outtmp"); o == nil {
		ctx.err("outtmp: %v", base.configure)
	} else if outtmp, y := o.(*def); !y || outtmp.value == nil {
		ctx.err("outtmp: %T %v", o, o)
	} else if outtmp.value.String() != "&(target.tmp)/&(rel.remnant)" {
		ctx.err("outtmp: %T %v", outtmp.value, outtmp.value)
	} else if s := filepath.Join(outtmp.string(cc1), configuration_sm); s != base.configurationFile.fullname() {
		ctx.err("outtmp: %v != %v", s, base.configurationFile.fullname())
	} else if b, e := ioutil.ReadFile(s); e != nil {
		ctx.Errorf("%v", e)
	} else {
		checkLLVMConfig(ctx, string(b))
	}

	ctx.flush()
}

func testLLVMConfig(t *testing.T) {
	if s := "llvm/Config"; !testHasModule(s) {
		t.Logf("skip %s", s)
		return
	}

	var ctx = load_testcase(t, "testdata/modules/llvm/config", "testllvmconfig")
	if ctx.Context == nil {
		t.Errorf("testllvmconfig fail")
		return
	}

	defer assured(ctx, true)

	if v := ctx.get("enum1", "*AsmPrinter.cpp", "LLVM_ASM_PRINTER"); v == nil {
		ctx.err("enum1")
	} else if s := v.String(); s == "" {
		ctx.err("%v", v)
	} else if strings.Count(s, "AsmPrinter.cpp") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "LLVM_ASM_PRINTER") != 1 {
		ctx.err("%v", v)
	} else if true {
		info(ctx, "%v", v).debug(1)
	}

	ctx.flush()
}

func testToolchainBootingConfigure(t *testing.T) {
	var ctx = load_testcase(t, "testdata/modules/toolchain/booting", "testtoolchainbooting")
	if ctx.Context == nil {
		t.Errorf("testtoolchainbooting fail")
		return
	}

	defer assured(ctx, true)

	ctx.universe().configure(ctx)

	ctx.flush()
}

func testToolchainBooting(t *testing.T) {
	if s := "toolchain/booting"; !testHasModule(s) {
		t.Logf("skip %s", s)
		return
	}

	var ctx = load_testcase(t, "testdata/modules/toolchain/booting", "testtoolchainbooting")
	if ctx.Context == nil {
		t.Errorf("testtoolchainbooting fail")
		return
	}

	defer assured(ctx, true)

	if r := ctx.rule("stamp"); r == nil {
		ctx.err("stamp")
	} else if v, e := ctx.universe().run(); e != nil {
		ctx.err("%v: %v (%v)", r, e, v)
	}

	ctx.flush()
}
