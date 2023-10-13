//
//  Copyright (C) 2012-2023, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	"regexp"
	"strings"
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

func validFlags(ctx *testcase, v Value, s string) (res bool) {
	for _, s := range strings.Fields(s) { if res = validFlag(s); !res {
		ctx.err("%s ; %v", s, v) ; break
	}}
	return
}

func testVariantTarget(ctx *testcase) {
	langs  := strings.Fields("c c++ cl cuda cuda++ objc objc++ swift")
	flags1 := strings.Fields("-D -I -L -O -W -Wl -Werror -Wno-error -f -f.ld -m -g -v -no -no.ld"+
		" -isystem -isystem-after -cxx-isystem -stdlib -stdlib++-isystem -diagnostics")
	flags2 := strings.Fields("-l -framework")
	flags3 := strings.Fields("ar asm c cpp cxx oc ocxx cl cuda cudaxx ld")
	flags4 := strings.Fields("ld")
	flags5 := strings.Fields("ld.framework ldlibs loadlibs loadlibes")

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
}

func testApp(ctx *testcase) {
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
		ctx.err("%v", v)
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
		ctx.err("%v", v)
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
}

const testCxxincConfigLines = `
_LIBCPP_ABI_VERSION = '2'
_LIBCPP_ABI_NAMESPACE = '_extbit'
_LIBCPP_ABI_DEFINES = ((plain c) '//TODO: #define ...')
_LIBCPP_EXTRA_SITE_DEFINES = ((plain c) '//TODO: #define ...')
_LIBCPP_HAS_MUSL_LIBC = no{}
_LIBCPP_HAS_PARALLEL_ALGORITHMS = no{}
_LIBCPP_PSTL_CPU_BACKEND_SERIAL = no{}
_LIBCPP_PSTL_CPU_BACKEND_THREAD = yes{}
_LIBCPP_TYPEINFO_COMPARISON_IMPLEMENTATION = 1
LIBCXX_ENABLE_FILESYSTEM = yes{}
LIBCXX_ENABLE_FSTREAM = yes{}
LIBCXX_ENABLE_LOCALIZATION = yes{}
LIBCXX_ENABLE_THREADS = yes{}
LIBCXX_ENABLE_WIDE_CHARACTERS = yes{}
requires_LIBCXX_ENABLE_WIDE_CHARACTERS =
requires_LIBCXX_ENABLE_FILESYSTEM =
requires_LIBCXX_ENABLE_THREADS =
requires_LIBCXX_ENABLE_LOCALIZATION =
requires_LIBCXX_ENABLE_FSTREAM =
`

const testCxxabiConfigLines = `
LIBCXXABI_ENABLE_NEW_DELETE_DEFINITIONS = yes{}
LIBCXXABI_ENABLE_EXCEPTIONS = yes{}
LIBCXXABI_ENABLE_THREADS = yes{}
`

const testAppConfigLines = `
VERSION = 0.0.1
PACKAGE = extbit.app
PACKAGE_NAME = 'app'
PACKAGE_VERSION = 0.0.1
PACKAGE_VENDOR = 'ExtBit LLC'
PACKAGE_TARNAME = 'app'-0.0.1
PACKAGE_STRING = "app-0.0.1"
PACKAGE_URL = "https://extbit.dev/package/extbit.app/0.0.1"
PACKAGE_BUGREPORT = "https://extbit.dev/package/extbit.app/0.0.1/bugs"
HAVE_ALLOCA_H =
HAVE_ARPA_INET_H =
HAVE_ARPA_NAMESER_H =
HAVE_ARPA_TFTP_H =
HAVE_ASM_TYPES_H =
HAVE_ASSERT_H =
HAVE_ATOMIC_H =
HAVE_BLUETOOTH_BLUETOOTH_H =
HAVE_BLUETOOTH_H =
HAVE_BSD_STDLIB_H =
HAVE_BSD_STRING_H =
HAVE_BSD_UNISTD_H =
HAVE_COMPLEX_H =
HAVE_CONIO_H =
HAVE_CRASHREPORTERCLIENT_H =
HAVE_CRYPT_H =
HAVE_CTYPE_H =
HAVE_CURSES_H =
HAVE_DB_H =
HAVE_DIRECT_H =
HAVE_DIRENT_H =
HAVE_DLFCN_H =
HAVE_DL_H =
HAVE_EDITLINE_READLINE_H =
HAVE_ENDIAN_H =
HAVE_ERRNO_H =
HAVE_EXECINFO_H =
HAVE_FCNTL_H =
HAVE_FENV_H =
HAVE_FFI_FFI_H =
HAVE_FFI_H =
HAVE_FLOAT_H =
HAVE_FP_CLASS_H =
HAVE_GDBM-NDBM_H =
HAVE_GDBMERRNO_H =
HAVE_GDBM_H =
HAVE_GDBM_NDBM_H =
HAVE_GRP_H =
HAVE_IEEEFP_H =
HAVE_IFADDRS_H =
HAVE_INTRIN_H =
HAVE_INTTYPES_H =
HAVE_IO_H =
HAVE_JEMALLOC_JEMALLOC_H =
HAVE_LANGINFO_H =
HAVE_LIBBSD =
HAVE_LIBCRYPT =
HAVE_LIBCURSES =
HAVE_LIBDBM =
HAVE_LIBDL =
HAVE_LIBDLD =
HAVE_LIBEDIT =
HAVE_LIBGEN =
HAVE_LIBGEN_H =
HAVE_LIBHISTORY =
HAVE_LIBIEEE =
HAVE_LIBINTL =
HAVE_LIBINTL_H =
HAVE_LIBJEMALLOC =
HAVE_LIBLZMA =
HAVE_LIBNCURSES =
HAVE_LIBNCURSESW =
HAVE_LIBPFM =
HAVE_LIBPSAPI =
HAVE_LIBPTHREAD =
HAVE_LIBREADLINE =
HAVE_LIBRESOLV =
HAVE_LIBRT =
HAVE_LIBSENDFILE =
HAVE_LIBTERMINFO =
HAVE_LIBTINFO =
HAVE_LIBUNWIND =
HAVE_LIBUTIL =
HAVE_LIBUUID =
HAVE_LIBXAR =
HAVE_LIBZ =
HAVE_LIMITS_H =
HAVE_LINK_H =
HAVE_LINUX_CAN_BCM_H =
HAVE_LINUX_CAN_H =
HAVE_LINUX_CAN_J1939_H =
HAVE_LINUX_CAN_RAW_H =
HAVE_LINUX_CLOSE_RANGE_H =
HAVE_LINUX_IF_ALG_H =
HAVE_LINUX_MEMFD_H =
HAVE_LINUX_NETLINK_H =
HAVE_LINUX_QRTR_H =
HAVE_LINUX_RANDOM_H =
HAVE_LINUX_SOUNDCARD_H =
HAVE_LINUX_TCP_H =
HAVE_LINUX_TIPC_H =
HAVE_LINUX_VM_SOCKETS_H =
HAVE_LINUX_WAIT_H =
HAVE_LOCALE_H =
HAVE_LZMA_H =
HAVE_MACH-O_DYLD_H =
HAVE_MACH_MACH_H =
HAVE_MACH_MACH_TIME_H =
HAVE_MALLOC_H =
HAVE_MALLOC_MALLOC_H =
HAVE_MALLOC_NP_H =
HAVE_MATH_H =
HAVE_MBARRIER_H =
HAVE_MEMORY_H =
HAVE_MKDEV_H =
HAVE_NCURSES_H =
HAVE_NDBM_H =
HAVE_NDIR_H =
HAVE_NETDB_H =
HAVE_NETINET_IN_H =
HAVE_NETINET_TCP_H =
HAVE_NETPACKET_PACKET_H =
HAVE_NET_IF_H =
HAVE_PERFMON_PERF_EVENT_H =
HAVE_PERFMON_PFMLIB_H =
HAVE_PERFMON_PFMLIB_PERF_EVENT_H =
HAVE_POLL_H =
HAVE_PROCESS_H =
HAVE_PTHREAD_H =
HAVE_PTY_H =
HAVE_PWD_H =
HAVE_READLINE_HISTORY_H =
HAVE_READLINE_READLINE_H =
HAVE_RESOLV_H =
HAVE_SCHED_H =
HAVE_SEMAPHORE_H =
HAVE_SETJMP_H =
HAVE_SHADOW_H =
HAVE_SIGNAL_H =
HAVE_SPAWN_H =
HAVE_STDARG_H =
HAVE_STDATOMIC_H =
HAVE_STDBOOL_H =
HAVE_STDDEF_H =
HAVE_STDINT_H =
HAVE_STDIO_H =
HAVE_STDLIB_H =
HAVE_STRINGS_H =
HAVE_STRING_H =
HAVE_STROPTS_H =
HAVE_STRUCT_ADDRINFO =
HAVE_STRUCT_PASSWD =
HAVE_STRUCT_PASSWD_PW_GECOS =
HAVE_STRUCT_PASSWD_PW_PASSWD =
HAVE_STRUCT_SOCKADDR =
HAVE_STRUCT_SOCKADDR_SA_LEN =
HAVE_STRUCT_SOCKADDR_STORAGE =
HAVE_STRUCT_SOCKADDR_STORAGE_SS_FAMILY =
HAVE_STRUCT_SOCKADDR_STORAGE___SS_FAMILY =
HAVE_STRUCT_STAT =
HAVE_STRUCT_STATFS =
HAVE_STRUCT_STATFS_F_FLAGS =
HAVE_STRUCT_STATFS_F_FSTYPENAME =
HAVE_STRUCT_STATVFS =
HAVE_STRUCT_STATVFS_F_FLAGS =
HAVE_STRUCT_STATVFS_F_FSTYPENAME =
HAVE_STRUCT_STAT_ST_ATIMESPEC_TV_NSEC =
HAVE_STRUCT_STAT_ST_ATIM_NSEC =
HAVE_STRUCT_STAT_ST_ATIM_TV_NSEC =
HAVE_STRUCT_STAT_ST_BIRTHTIME =
HAVE_STRUCT_STAT_ST_BLKSIZE =
HAVE_STRUCT_STAT_ST_BLOCKS =
HAVE_STRUCT_STAT_ST_CTIMESPEC_TV_NSEC =
HAVE_STRUCT_STAT_ST_CTIM_NSEC =
HAVE_STRUCT_STAT_ST_CTIM_TV_NSEC =
HAVE_STRUCT_STAT_ST_FLAGS =
HAVE_STRUCT_STAT_ST_GEN =
HAVE_STRUCT_STAT_ST_MTIMESPEC_TV_NSEC =
HAVE_STRUCT_STAT_ST_MTIM_NSEC =
HAVE_STRUCT_STAT_ST_MTIM_TV_NSEC =
HAVE_STRUCT_STAT_ST_RDEV =
HAVE_STRUCT_TIMEVAL =
HAVE_STRUCT_TIMEVAL_TV_SEC =
HAVE_STRUCT_TIMEVAL_TV_USEC =
HAVE_STRUCT_TM =
HAVE_STRUCT_TM_TM_ZONE =
HAVE_SYSEXITS_H =
HAVE_SYSMACROS_H =
HAVE_SYS_AUDIOIO_H =
HAVE_SYS_AUDIO_H =
HAVE_SYS_BSDTTY_H =
HAVE_SYS_DEVPOLL_H =
HAVE_SYS_DIR_H =
HAVE_SYS_ENDIAN_H =
HAVE_SYS_EPOLL_H =
HAVE_SYS_EVENTFD_H =
HAVE_SYS_EVENT_H =
HAVE_SYS_FILE_H =
HAVE_SYS_FILIO_H =
HAVE_SYS_IOCTL_H =
HAVE_SYS_KERN_CONTROL_H =
HAVE_SYS_LOADAVG_H =
HAVE_SYS_LOCK_H =
HAVE_SYS_MEMFD_H =
HAVE_SYS_MKDEV_H =
HAVE_SYS_MMAN_H =
HAVE_SYS_MODEM_H =
HAVE_SYS_MOUNT_H =
HAVE_SYS_NDIR_H =
HAVE_SYS_PARAM_H =
HAVE_SYS_POLLSET_H =
HAVE_SYS_POLL_H =
HAVE_SYS_RANDOM_H =
HAVE_SYS_RESOURCE_H =
HAVE_SYS_SELECT_H =
HAVE_SYS_SENDFILE_H =
HAVE_SYS_SOCKET_H =
HAVE_SYS_SOCKIO_H =
HAVE_SYS_SOUNDCARD_H =
HAVE_SYS_STATFS_H =
HAVE_SYS_STATVFS_H =
HAVE_SYS_STAT_H =
HAVE_SYS_SYSCALL_H =
HAVE_SYS_SYSMACROS_H =
HAVE_SYS_SYS_DOMAIN_H =
HAVE_SYS_TERMIO_H =
HAVE_SYS_TIMEB_H =
HAVE_SYS_TIMES_H =
HAVE_SYS_TIME_H =
HAVE_SYS_TYPES_H =
HAVE_SYS_UIO_H =
HAVE_SYS_UN_H =
HAVE_SYS_UTIME_H =
HAVE_SYS_UTSNAME_H =
HAVE_SYS_VFS_H =
HAVE_SYS_WAIT_H =
HAVE_SYS_XATTR_H =
HAVE_TERMIOS_H =
HAVE_TERMIO_H =
HAVE_TERM_H =
HAVE_TIME_H =
HAVE_TYPE_ATOMIC_INT =
HAVE_TYPE_ATOMIC_UINTPTR_T =
HAVE_TYPE_BLKCNT_T =
HAVE_TYPE_BLKSIZE_T =
HAVE_TYPE_BOOL =
HAVE_TYPE_CHAR =
HAVE_TYPE_CLOCKID_T =
HAVE_TYPE_CLOCK_T =
HAVE_TYPE_CONST_CHAR =
HAVE_TYPE_DEV_T =
HAVE_TYPE_DOUBLE =
HAVE_TYPE_FLOAT =
HAVE_TYPE_FPOS_T =
HAVE_TYPE_FSBLKCNT_T =
HAVE_TYPE_FSFILCNT_T =
HAVE_TYPE_GID_T =
HAVE_TYPE_ID_T =
HAVE_TYPE_INO_T =
HAVE_TYPE_INT =
HAVE_TYPE_KEY_T =
HAVE_TYPE_LONG =
HAVE_TYPE_LONG_DOUBLE =
HAVE_TYPE_LONG_LONG =
HAVE_TYPE_MODE_T =
HAVE_TYPE_NLINK_T =
HAVE_TYPE_OFF_T =
HAVE_TYPE_PID_T =
HAVE_TYPE_PTHREAD_ATTR_T =
HAVE_TYPE_PTHREAD_CONDATTR_T =
HAVE_TYPE_PTHREAD_COND_T =
HAVE_TYPE_PTHREAD_KEY_T =
HAVE_TYPE_PTHREAD_MUTEXATTR_T =
HAVE_TYPE_PTHREAD_MUTEX_T =
HAVE_TYPE_PTHREAD_ONCE_T =
HAVE_TYPE_PTHREAD_RWLOCKATTR_T =
HAVE_TYPE_PTHREAD_RWLOCK_T =
HAVE_TYPE_PTHREAD_T =
HAVE_TYPE_PTRDIFF_T =
HAVE_TYPE_SA_FAMILY_T =
HAVE_TYPE_SHORT =
HAVE_TYPE_SIGINFO_T =
HAVE_TYPE_SIGNED_CHAR =
HAVE_TYPE_SIZE_T =
HAVE_TYPE_SOCKLEN_T =
HAVE_TYPE_SSIZE_T =
HAVE_TYPE_SUSECONDS_T =
HAVE_TYPE_TIMER_T =
HAVE_TYPE_TIME_T =
HAVE_TYPE_UID_T =
HAVE_TYPE_UINT32_T =
HAVE_TYPE_UINT64_T =
HAVE_TYPE_UINTPTR_T =
HAVE_TYPE_USECONDS_T =
HAVE_TYPE_VOID_P =
HAVE_TYPE_WCHAR_T =
HAVE_TYPE__BOOL =
HAVE_TYPE___INT64 =
HAVE_TYPE___INT64_T =
HAVE_UNISTD_H =
HAVE_UNWIND_H =
HAVE_UTIL_H =
HAVE_UTIME_H =
HAVE_UTMP_H =
HAVE_UUID_H =
HAVE_UUID_UUID_H =
HAVE_VALGRIND_VALGRIND_H =
HAVE_WCHAR_H =
HAVE_ZLIB_H =
ALIGNOF_ATOMIC_INT =
ALIGNOF_ATOMIC_INT_CODE =
ALIGNOF_ATOMIC_UINTPTR_T =
ALIGNOF_ATOMIC_UINTPTR_T_CODE =
ALIGNOF_BLKCNT_T =
ALIGNOF_BLKCNT_T_CODE =
ALIGNOF_BLKSIZE_T =
ALIGNOF_BLKSIZE_T_CODE =
ALIGNOF_BOOL =
ALIGNOF_BOOL_CODE =
ALIGNOF_CHAR =
ALIGNOF_CHAR_CODE =
ALIGNOF_CLOCKID_T =
ALIGNOF_CLOCKID_T_CODE =
ALIGNOF_CLOCK_T =
ALIGNOF_CLOCK_T_CODE =
ALIGNOF_CONST_CHAR =
ALIGNOF_CONST_CHAR_CODE =
ALIGNOF_DEV_T =
ALIGNOF_DEV_T_CODE =
ALIGNOF_DOUBLE =
ALIGNOF_DOUBLE_CODE =
ALIGNOF_FLOAT =
ALIGNOF_FLOAT_CODE =
ALIGNOF_FPOS_T =
ALIGNOF_FPOS_T_CODE =
ALIGNOF_FSBLKCNT_T =
ALIGNOF_FSBLKCNT_T_CODE =
ALIGNOF_FSFILCNT_T =
ALIGNOF_FSFILCNT_T_CODE =
ALIGNOF_GID_T =
ALIGNOF_GID_T_CODE =
ALIGNOF_ID_T =
ALIGNOF_ID_T_CODE =
ALIGNOF_INO_T =
ALIGNOF_INO_T_CODE =
ALIGNOF_INT =
ALIGNOF_INT_CODE =
ALIGNOF_KEY_T =
ALIGNOF_KEY_T_CODE =
ALIGNOF_LONG =
ALIGNOF_LONG_CODE =
ALIGNOF_LONG_DOUBLE =
ALIGNOF_LONG_DOUBLE_CODE =
ALIGNOF_LONG_LONG =
ALIGNOF_LONG_LONG_CODE =
ALIGNOF_MODE_T =
ALIGNOF_MODE_T_CODE =
ALIGNOF_NLINK_T =
ALIGNOF_NLINK_T_CODE =
ALIGNOF_OFF_T =
ALIGNOF_OFF_T_CODE =
ALIGNOF_PID_T =
ALIGNOF_PID_T_CODE =
ALIGNOF_PTHREAD_ATTR_T =
ALIGNOF_PTHREAD_ATTR_T_CODE =
ALIGNOF_PTHREAD_CONDATTR_T =
ALIGNOF_PTHREAD_CONDATTR_T_CODE =
ALIGNOF_PTHREAD_COND_T =
ALIGNOF_PTHREAD_COND_T_CODE =
ALIGNOF_PTHREAD_KEY_T =
ALIGNOF_PTHREAD_KEY_T_CODE =
ALIGNOF_PTHREAD_MUTEXATTR_T =
ALIGNOF_PTHREAD_MUTEXATTR_T_CODE =
ALIGNOF_PTHREAD_MUTEX_T =
ALIGNOF_PTHREAD_MUTEX_T_CODE =
ALIGNOF_PTHREAD_ONCE_T =
ALIGNOF_PTHREAD_ONCE_T_CODE =
ALIGNOF_PTHREAD_RWLOCKATTR_T =
ALIGNOF_PTHREAD_RWLOCKATTR_T_CODE =
ALIGNOF_PTHREAD_RWLOCK_T =
ALIGNOF_PTHREAD_RWLOCK_T_CODE =
ALIGNOF_PTHREAD_T =
ALIGNOF_PTHREAD_T_CODE =
ALIGNOF_PTRDIFF_T =
ALIGNOF_PTRDIFF_T_CODE =
ALIGNOF_SA_FAMILY_T =
ALIGNOF_SA_FAMILY_T_CODE =
ALIGNOF_SHORT =
ALIGNOF_SHORT_CODE =
ALIGNOF_SIGINFO_T =
ALIGNOF_SIGINFO_T_CODE =
ALIGNOF_SIGNED_CHAR =
ALIGNOF_SIGNED_CHAR_CODE =
ALIGNOF_SIZE_T =
ALIGNOF_SIZE_T_CODE =
ALIGNOF_SOCKLEN_T =
ALIGNOF_SOCKLEN_T_CODE =
ALIGNOF_SSIZE_T =
ALIGNOF_SSIZE_T_CODE =
ALIGNOF_SUSECONDS_T =
ALIGNOF_SUSECONDS_T_CODE =
ALIGNOF_TIMER_T =
ALIGNOF_TIMER_T_CODE =
ALIGNOF_TIME_T =
ALIGNOF_TIME_T_CODE =
ALIGNOF_UID_T =
ALIGNOF_UID_T_CODE =
ALIGNOF_UINT32_T =
ALIGNOF_UINT32_T_CODE =
ALIGNOF_UINT64_T =
ALIGNOF_UINT64_T_CODE =
ALIGNOF_UINTPTR_T =
ALIGNOF_UINTPTR_T_CODE =
ALIGNOF_USECONDS_T =
ALIGNOF_USECONDS_T_CODE =
ALIGNOF_VOID_P =
ALIGNOF_VOID_P_CODE =
ALIGNOF_WCHAR_T =
ALIGNOF_WCHAR_T_CODE =
ALIGNOF__BOOL =
ALIGNOF__BOOL_CODE =
ALIGNOF___INT64 =
ALIGNOF___INT64_CODE =
ALIGNOF___INT64_T =
ALIGNOF___INT64_T_CODE =
SIZEOF_ATOMIC_INT =
SIZEOF_ATOMIC_INT_CODE =
SIZEOF_ATOMIC_UINTPTR_T =
SIZEOF_ATOMIC_UINTPTR_T_CODE =
SIZEOF_BLKCNT_T =
SIZEOF_BLKCNT_T_CODE =
SIZEOF_BLKSIZE_T =
SIZEOF_BLKSIZE_T_CODE =
SIZEOF_BOOL =
SIZEOF_BOOL_CODE =
SIZEOF_CHAR =
SIZEOF_CHAR_CODE =
SIZEOF_CLOCKID_T =
SIZEOF_CLOCKID_T_CODE =
SIZEOF_CLOCK_T =
SIZEOF_CLOCK_T_CODE =
SIZEOF_CONST_CHAR =
SIZEOF_CONST_CHAR_CODE =
SIZEOF_DEV_T =
SIZEOF_DEV_T_CODE =
SIZEOF_DOUBLE =
SIZEOF_DOUBLE_CODE =
SIZEOF_FLOAT =
SIZEOF_FLOAT_CODE =
SIZEOF_FPOS_T =
SIZEOF_FPOS_T_CODE =
SIZEOF_FSBLKCNT_T =
SIZEOF_FSBLKCNT_T_CODE =
SIZEOF_FSFILCNT_T =
SIZEOF_FSFILCNT_T_CODE =
SIZEOF_GID_T =
SIZEOF_GID_T_CODE =
SIZEOF_ID_T =
SIZEOF_ID_T_CODE =
SIZEOF_INO_T =
SIZEOF_INO_T_CODE =
SIZEOF_INT =
SIZEOF_INT_CODE =
SIZEOF_KEY_T =
SIZEOF_KEY_T_CODE =
SIZEOF_LONG =
SIZEOF_LONG_CODE =
SIZEOF_LONG_DOUBLE =
SIZEOF_LONG_DOUBLE_CODE =
SIZEOF_LONG_LONG =
SIZEOF_LONG_LONG_CODE =
SIZEOF_MODE_T =
SIZEOF_MODE_T_CODE =
SIZEOF_NLINK_T =
SIZEOF_NLINK_T_CODE =
SIZEOF_OFF_T =
SIZEOF_OFF_T_CODE =
SIZEOF_PID_T =
SIZEOF_PID_T_CODE =
SIZEOF_PTHREAD_ATTR_T =
SIZEOF_PTHREAD_ATTR_T_CODE =
SIZEOF_PTHREAD_CONDATTR_T =
SIZEOF_PTHREAD_CONDATTR_T_CODE =
SIZEOF_PTHREAD_COND_T =
SIZEOF_PTHREAD_COND_T_CODE =
SIZEOF_PTHREAD_KEY_T =
SIZEOF_PTHREAD_KEY_T_CODE =
SIZEOF_PTHREAD_MUTEXATTR_T =
SIZEOF_PTHREAD_MUTEXATTR_T_CODE =
SIZEOF_PTHREAD_MUTEX_T =
SIZEOF_PTHREAD_MUTEX_T_CODE =
SIZEOF_PTHREAD_ONCE_T =
SIZEOF_PTHREAD_ONCE_T_CODE =
SIZEOF_PTHREAD_RWLOCKATTR_T =
SIZEOF_PTHREAD_RWLOCKATTR_T_CODE =
SIZEOF_PTHREAD_RWLOCK_T =
SIZEOF_PTHREAD_RWLOCK_T_CODE =
SIZEOF_PTHREAD_T =
SIZEOF_PTHREAD_T_CODE =
SIZEOF_PTRDIFF_T =
SIZEOF_PTRDIFF_T_CODE =
SIZEOF_SA_FAMILY_T =
SIZEOF_SA_FAMILY_T_CODE =
SIZEOF_SHORT =
SIZEOF_SHORT_CODE =
SIZEOF_SIGINFO_T =
SIZEOF_SIGINFO_T_CODE =
SIZEOF_SIGNED_CHAR =
SIZEOF_SIGNED_CHAR_CODE =
SIZEOF_SIZE_T =
SIZEOF_SIZE_T_CODE =
SIZEOF_SOCKLEN_T =
SIZEOF_SOCKLEN_T_CODE =
SIZEOF_SSIZE_T =
SIZEOF_SSIZE_T_CODE =
SIZEOF_SUSECONDS_T =
SIZEOF_SUSECONDS_T_CODE =
SIZEOF_TIMER_T =
SIZEOF_TIMER_T_CODE =
SIZEOF_TIME_T =
SIZEOF_TIME_T_CODE =
SIZEOF_UID_T =
SIZEOF_UID_T_CODE =
SIZEOF_UINT32_T =
SIZEOF_UINT32_T_CODE =
SIZEOF_UINT64_T =
SIZEOF_UINT64_T_CODE =
SIZEOF_UINTPTR_T =
SIZEOF_UINTPTR_T_CODE =
SIZEOF_USECONDS_T =
SIZEOF_USECONDS_T_CODE =
SIZEOF_VOID_P =
SIZEOF_VOID_P_CODE =
SIZEOF_WCHAR_T =
SIZEOF_WCHAR_T_CODE =
SIZEOF__BOOL =
SIZEOF__BOOL_CODE =
SIZEOF___INT64 =
SIZEOF___INT64_CODE =
SIZEOF___INT64_T =
SIZEOF___INT64_T_CODE =
`
func testLLVMConfig1(ctx *testcase) {
	var proj, base, general *Project
	if proj = ctx.Project(); proj == nil {
		ctx.err("configure fail")
	} else if len(proj.bases) != 1 {
		ctx.err("bases: %v", proj.bases)
	} else if base = proj.bases[0]; base.name != "llvm.Config" {
		ctx.err("base: %v", base)
	} else if base.configure == nil {
		ctx.err("configure fail")
	} else if f := file(ctx, ".configure/type/test.c", base.configure); f == nil {
		ctx.err("file .configure/type/test.c")
	} else if f := file(ctx, ".configure/type/align/test.c", base.configure); f == nil {
		ctx.err("file .configure/type/test.c")
	} else if f := file(ctx, ".configure/type/size/test.c", base.configure); f == nil {
		ctx.err("file .configure/type/test.c")
	} else if f := file(ctx, ".configure/type/xxx/test.c", base.configure); f != nil {
		ctx.err("file .configure/type/xxx/test.c")
	} else if f := file(ctx, ".configure/type/xxx/yyy/test.c", base.configure); f != nil {
		ctx.err("file .configure/type/xxx/yyy/test.c")
	} else if o := proj.resolve(ctx, "general"); o == nil {
		ctx.err("general")
	} else if p, y := o.(*projectname); !y {
		ctx.err("%T %v", o, o)
	} else {
		general = p.Project
	}

	if v := ctx.get("/"); v == nil {
		ctx.err("/")
	} else if s := v.String(); s != proj.absPath {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != proj.absPath {
		ctx.err("%T %v %s", v, v, s)
	} else if _, y := v.(*Path); !y {
		ctx.err("%T %v", v, v)
	}

	var chop = fmt.Sprintf("%%%%/.smart/modules/ %s/ %s/ %s/",
		filepath.Dir(general.absPath),
		filepath.Dir(filepath.Dir(general.absPath)),
		filepath.Dir(filepath.Dir(filepath.Dir(general.absPath))))

	if v := ctx.get("rel.chop"); v == nil { // from general
		ctx.err("rel.chop")
	} else if !strings.HasSuffix(v.String(), chop) {
		ctx.err("%T %v", v, v)
	}

	var remnant = ctx.get("rel.remnant")
	if v := remnant; v == nil {
		ctx.err("rel.remnant")
	} else if s := v.String(); s != "$(trim-prefix &(rel.chop),&/)" { // from general
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s == "" {
		ctx.err("%T %v", v, v)
	} else if _, y := v.(*delegate); !y {
		ctx.err("%T %v", v, v)
	}

	var cc1 = closureWith(ctx, base.configure.scope, base.scope)
	var cc2 = closureWith(ctx, base.scope, base.configure.scope)
	if v := remnant; v == nil {
		ctx.err("remnant")
	} else if s := v.string(cc1); s == "" {
		ctx.err("%T %v", v, v)
	}

	if d := base.resolveDef(cc1, "rel.remnant"); d == nil {
		ctx.err("rel.remnant")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "$(trim-prefix &(rel.chop),&/)" { // from general
		ctx.err("%T %v", v, v)
	} else if s := v.string(cc1); s == "" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(cc2); s == "" {
		ctx.err("%T %v", v, v)
	} else if _, y := v.(*delegate); !y {
		ctx.err("%T %v", v, v)
	}

	p1 := strings.Split(proj.absPath,PathSep)
	// p2 := strings.Split(base.absPath,PathSep)
	t1 := strings.Join(append(p1[len(p1)-4:], configuration_sm),PathSep)
	// t2 := strings.Join(append(p2[len(p2)-2:], configuration_sm),PathSep)

	var outtmp string
	if v := ctx.get("outtmp"); v == nil {
		ctx.err("%T %v", v, v)
	} else if outtmp = v.string(ctx); outtmp == "" {
		ctx.err("%T %v", v, v)
	}

	if f := proj.tempFile(ctx, configuration_sm); f == nil {
		ctx.err("%v: nil %s", proj, configuration_sm)
	} else if s1 := f.fullname(); !filepath.IsAbs(s1) {
		ctx.err("%v: %v", f, base)
	} else if s2 := base.configurationFile.fullname(); !filepath.IsAbs(s2) {
		ctx.err("%v: %v", f, base)
	} else if s1 == s2 {
		ctx.err("%v: %v %v", f, s1, s2)
	} else if s1 != strings.Join([]string{outtmp, configuration_sm}, PathSep) {
		ctx.err("%v: %v %v %v", f, s1, outtmp, t1)
		// } else if s2 != strings.Join([]string{r2, configuration_sm}, PathSep) {
		// 	ctx.err("%v: %v %v %v", f, s2, r2, t2)
	}

	if f := base.tempFile(ctx, configuration_sm); f == nil {
		ctx.err("%v: nil %s", proj, configuration_sm)
	} else if s1 := f.fullname(); !filepath.IsAbs(s1) {
		ctx.err("%v: %v", f, base)
	} else if s2 := base.configurationFile.fullname(); !filepath.IsAbs(s2) {
		ctx.err("%v: %v", f, base)
	} else if s1 == s2 {
		ctx.err("%v: %v %v", f, s1, s2)
	} else if s1 != strings.Join([]string{outtmp, configuration_sm}, PathSep) {
		ctx.err("%v: %v %v %v", f, s1, outtmp, t1)
		// } else if s2 != strings.Join([]string{r2, configuration_sm}, PathSep) {
		// 	ctx.err("%v: %v %v %v", f, s2, r2, t2)
	}

	if f := base.configuration(cc1); f == nil {
		ctx.err("%v: %v: nil configuration", proj, base)
	} else if s1 := f.fullname(); !filepath.IsAbs(s1) {
		ctx.err("%v: %v", f, base)
	} else if s2 := base.configurationFile.fullname(); !filepath.IsAbs(s2) {
		ctx.err("%v: %v", f, base)
	} else if s1 == s2 {
		ctx.err("%v: %v %v", f, s1, s2)
	} else if s1 != strings.Join([]string{outtmp, configuration_sm}, PathSep) {
		ctx.err("%v: %v %v %v", f, s1, outtmp, t1)
		// } else if s2 != strings.Join([]string{r2, configuration_sm}, PathSep) {
		// 	ctx.err("%v: %v %v %v", f, s2, r2, t2)
	}

	if v := ctx.get("val1"); v == nil {
		ctx.err("val1")
	} else if s := v.String(); s != base.configurationFile.name(ctx) {
		ctx.err("%v , %v", v, base.configurationFile.name(ctx))
	} else if s1, s2 := v.string(ctx), base.configurationFile.fullname(); s1 == s2 {
		erro(ctx, "%v: %v: %v", proj, base, s1)
		erro(ctx, "%v: %v: %v", proj, base, s2).debug(1)
		ctx.err("%v: %v", proj, base)
	} else if s1 != strings.Join([]string{outtmp, configuration_sm}, PathSep) {
		ctx.err("%v: %v %v %v", v, s1, outtmp, t1)
		// } else if s2 != strings.Join([]string{r2, configuration_sm}, PathSep) {
		// 	ctx.err("%v: %v %v %v", f, s2, r2, t2)
	}

	if v := ctx.get("val2"); v == nil {
		ctx.err("val2")
	} else if s := v.String(); s != base.configurationFile.name(ctx) {
		ctx.err("%v , %v", v, base.configurationFile.name(ctx))
	} else if f, y := v.(*File); !y {
		ctx.err("%T %v", v, v)
	} else if s1, s2 := f.fullname(), base.configurationFile.fullname(); s1 == s2 {
		erro(ctx, "%v: %v: %v", proj, base, s1)
		erro(ctx, "%v: %v: %v", proj, base, s2).debug(1)
		ctx.err("%v: %v", proj, base)
	} else if s1 != strings.Join([]string{outtmp, configuration_sm}, PathSep) {
		ctx.err("%v: %v %v %v", v, s1, outtmp, t1)
		// } else if s2 != strings.Join([]string{r2, configuration_sm}, PathSep) {
		// 	ctx.err("%v: %v %v %v", f, s2, r2, t2)
	}

	ctx.universe().configure(ctx)

	if s := filepath.Join(outtmp, "lib", "c++inc"); s == "" {
		ctx.err("%s", s)
	} else if i, e := os.Stat(s); e != nil {
		ctx.err("%v", e)
	} else if i == nil {
		ctx.err("missing %s", s)
	} else if b, e := ioutil.ReadFile(s); e != nil {
		ctx.err("%v", e)
	} else if s := string(b); s == "" {
		ctx.err("%s", s)
	} else {
		lines := testCxxincConfigLines // TODO: specific lines for different OS
		for i, l := range strings.Split(lines, "\n") {
			if l != "" && strings.Count(s, l) != 1 { ctx.err("%d. %s", i, l) }
			if n := strings.Index(l, " "); n <= 0 { ctx.err("%d. %s", i, l)	} else {
				if name := strings.TrimSpace(l[:n]); name == "" {
					ctx.err("%d. %s", i, l)
				} else if strings.HasPrefix(name, "ALIGNOF_") && !strings.HasSuffix(name, "_CODE") {
					ctx.err("%d. %s", i, l)
				} else if strings.HasPrefix(name, "ALIGNOF_") &&  strings.HasSuffix(name, "_CODE") {
					ctx.err("%d. %s", i, l)
				} else if strings.HasPrefix(name, "SIZEOF_") && !strings.HasSuffix(name, "_CODE") {
					ctx.err("%d. %s", i, l)
				} else if strings.HasPrefix(name, "SIZEOF_") &&  strings.HasSuffix(name, "_CODE") {
					ctx.err("%d. %s", i, l)
				} else if strings.HasPrefix(name, "HAVE_LIB") {
					ctx.err("%d. %s", i, l)
				} else if strings.HasPrefix(name, "HAVE_STRUCT_") {
					ctx.err("%d. %s", i, l)
				} else if strings.HasPrefix(name, "HAVE_TYPE_") {
					ctx.err("%d. %s", i, l)
				} else if strings.HasPrefix(name, "HAVE_") && strings.HasSuffix(name, "_H") {
					ctx.err("%d. %s", i, l)
				} else {
					ctx.err("%d. %s", i, l)
				}
			}
		}
	}

	if s := filepath.Join(outtmp, "lib", "c++abi"); s == "" {
		ctx.err("%s", s)
	} else if i, e := os.Stat(s); e != nil {
		ctx.err("%v", e)
	} else if i == nil {
		ctx.err("missing %s", s)
	} else if b, e := ioutil.ReadFile(s); e != nil {
		ctx.err("%v", e)
	} else if s := string(b); s == "" {
		ctx.err("%s", s)
	} else {
		lines := testCxxabiConfigLines // TODO: specific lines for different OS
		for i, l := range strings.Split(lines, "\n") {
			if l != "" && strings.Count(s, l) != 1 { ctx.err("%d. %s", i, l) }
		}
	}

	if s := filepath.Join(outtmp, "app"); s == "" {
		ctx.err("%s", s)
	} else if i, e := os.Stat(s); e != nil {
		ctx.err("%v", e)
	} else if i == nil {
		ctx.err("missing %s", s)
	} else if b, e := ioutil.ReadFile(s); e != nil {
		ctx.err("%v", e)
	} else if s := string(b); s == "" {
		ctx.err("%s", s)
	} else {
		lines := testAppConfigLines // TODO: app configs for different OS
		for i, l := range strings.Split(lines, "\n") {
			if l != "" && strings.Count(s, l) != 1 { ctx.err("%d. %s", i, l) }
		}
	}

	if f := base.configurationFile; f == nil {
		ctx.err("%v: nil configuration", base)
	} else if s := f.fullname(); s == "" {
		ctx.err("%v", f)
	} else if i, e := os.Stat(s); e != nil {
		ctx.err("%s: %v", configuration_sm, e)
	} else if i == nil {
		ctx.err("missing %s", s)
	} else if b, e := ioutil.ReadFile(s); e != nil {
		ctx.err("%v", e)
	} else if s := string(b); s == "" {
		ctx.err("%s", s)
	} else if !strings.Contains(s, "FOO1 = yes{}")  {
		ctx.err("%s", b)
	} else if true {
		noted(ctx, "%v\n%s", f.fullname(), b).debug(1)
	}

	if o := base.configure.resolve(ctx, "outtmp"); o == nil {
		ctx.err("outtmp: %v", base.configure)
	} else if outtmp, y := o.(*def); !y || outtmp.value == nil {
		ctx.err("outtmp: %T %v", o, o)
	} else if outtmp.value.String() != "&(target.tmp)/&(rel.remnant)" {
		ctx.err("outtmp: %T %v", outtmp.value, outtmp.value)
	} else if s := filepath.Join(outtmp.string(cc1), configuration_sm); s != base.configurationFile.fullname() {
		ctx.err("outtmp: %v != %v", s, base.configurationFile.fullname())
	} else if b, e := ioutil.ReadFile(s); e != nil {
		ctx.err("%v", e)
	} else if s := string(b); s == "" {
		ctx.err("%v", outtmp.value)
	} else if strings.Count(s, "FOO = $(.self)") != 1 {
		ctx.err("%v %s", outtmp.value, s)
	}

	if v := ctx.get("enum1", "*AsmPrinter.cpp", "LLVM_ASM_PRINTER"); v == nil {
		ctx.err("enum1")
	} else if s := v.String(); s == "" {
		ctx.err("%v", v)
	} else if strings.Count(s, "AsmPrinter.cpp") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "LLVM_ASM_PRINTER") != 1 {
		ctx.err("%v", v)
	} else if true {
		noted(ctx, "%v", v).debug(1)
	}
}

func testLLVMConfig2(ctx *testcase) {
	if v := ctx.get("enum1", "*AsmPrinter.cpp", "LLVM_ASM_PRINTER"); v == nil {
		ctx.err("enum1")
	} else if s := v.String(); s == "" {
		ctx.err("%v", v)
	} else if strings.Count(s, "AsmPrinter.cpp") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "LLVM_ASM_PRINTER") != 1 {
		ctx.err("%v", v)
	} else if true {
		noted(ctx, "%v", v).debug(1)
	}
}

func testToolchainBooting(ctx *testcase) {
	ctx.universe().configure(ctx)

	if r := ctx.rule("stamp"); r == nil {
		ctx.err("stamp")
	} else if v, e := ctx.universe().run(); e != nil {
		ctx.err("%v: %v (%v)", r, e, v)
	}
}
