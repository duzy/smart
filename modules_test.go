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
	"fmt"
)

var testValidFlag = []*regexp.Regexp{
	regexp.MustCompile(`^--target=[[:alnum:]-]+$`),
	regexp.MustCompile(`^-(?:shared|static|ObjC(?:\+\+)?)$`),
	regexp.MustCompile(`^-(?:std|Werror)=[[:alnum:]_\-]+$`),
	regexp.MustCompile(`^-(?:(?:(?:cxx|stdlib\+\+)-)?isystem(?:-after)?)=?[[:alnum:]_\-/]+$`),
	regexp.MustCompile(`^-[IL]=?[[:alnum:]_\-/]+$`),
	regexp.MustCompile(`^-[DWfl][[:alnum:]_\-]+$`),
	regexp.MustCompile(`^-no[[:alnum:]_\-]+$`),
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

func TestVariantTarget(t *testing.T) {
	if s := "variant/.target"; !testHasModule(s) { // variant/bootstrap
		t.Logf("skip %s", s)
		return
	}

	defer func(o commandLineOpts) { options = o } (options)
	options.failOnErrors = false

	var ctx = load_testcase(t, "testdata/modules/target", "testtarget")
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
		" -isystem -isystem-after -cxx-isystem -stdlib++-isystem -stdlib")
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
	} else if uses = v.Strval(ctx); uses == "" {
		ctx.err("%T %v -> %s", v, v, uses)
	}

	for _, flag := range flags1 {
		s1 := fmt.Sprintf("%s(-unique)", flag)
		s2 := fmt.Sprintf("%s~&(target.sys)(-unique)", flag)
		s3 := fmt.Sprintf("%s~foo(-unique)", flag)
		if !strings.Contains(usev, s1) { ctx.err("%v", s1) }
		if !strings.Contains(usev, s2) { ctx.err("%v", s2) }
		if !strings.Contains(uses, s3) { ctx.err("%v", s3) }
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
			if !strings.Contains(usev, s1) { ctx.err("%v", s1) }
			if !strings.Contains(usev, s2) { ctx.err("%v", s2) }
			if !strings.Contains(uses, s3) { ctx.err("%v", s3) }
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
		if !strings.Contains(usev, s1) { ctx.err("%v", s1) }
		if !strings.Contains(usev, s2) { ctx.err("%v", s2) }
		if !strings.Contains(uses, s3) { ctx.err("%v", s3) }
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
		if !strings.Contains(usev, s1) { ctx.err("%v", s1) }
		if !strings.Contains(usev, s2) { ctx.err("%v", s2) }
		if !strings.Contains(uses, s3) { ctx.err("%v", s3) }
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
		if !strings.Contains(usev, s1) { ctx.err("%v", s1) }
		if !strings.Contains(usev, s2) { ctx.err("%v", s2) }
		if !strings.Contains(uses, s3) { ctx.err("%v", s3) }
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
		if !strings.Contains(usev, s1) { ctx.err("%v", s1) }
		if !strings.Contains(usev, s2) { ctx.err("%v", s2) }
		if !strings.Contains(uses, s3) { ctx.err("%v", s3) }
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
	} else if s := v.Strval(ctx); s != "!foobar" {
		ctx.err("%T %v -> %s", v, v, s)
	} else if t, y := v.(*negative); !y {
		ctx.err("%T %v", v, v)
	} else if t.True(ctx) {
		ctx.err("%T %v", v, v)
	}
	if v := ctx.get("neg2"); v == nil {
		ctx.err("neg2")
	} else if v.String() != "a!foobar" {
		ctx.err("%T %v", v, v)
	} else if s := v.Strval(ctx); s != "a!foobar" {
		ctx.err("%T %v -> %s", v, v, s)
	}
	if v := ctx.get("neg3"); v == nil {
		ctx.err("neg3")
	} else if v.String() != "&(a!foobar)" {
		ctx.err("%T %v", v, v)
	} else if s := v.Strval(ctx); s != "xxx" {
		ctx.err("%T %v -> %s", v, v, s)
	}
	if v := ctx.get("neg4", "xxx"); v == nil {
		ctx.err("neg4")
	} else if v.String() != "&(a!xxx)" {
		ctx.err("%T %v", v, v)
	} else if s := v.Strval(ctx); s != "" {
		ctx.err("%T %v -> %s", v, v, s)
	}
	if v := ctx.get("neg4", "foobar"); v == nil {
		ctx.err("neg4")
	} else if v.String() != "&(a!foobar)" {
		ctx.err("%T %v", v, v)
	} else if s := v.Strval(ctx); s != "xxx" {
		ctx.err("%T %v -> %s", v, v, s)
	}

	var cflagv, cflags string
	if v := ctx.get("cflags"); v == nil {
		ctx.err("cflags")
	} else if cflagv = v.String(); cflagv == "" {
		ctx.err("%T %v", v, v)
	} else if cflags = v.Strval(ctx); cflags == "" {
		ctx.err("%T %v -> %s", v, v, cflags)
	}

	if true {
		info(ctx, "%v", cflagv)
		info(ctx, "%v", cflags).debug(1)
	}

	ctx.flush()
}

func TestApp(t *testing.T) {
	if s := "app"; !testHasModule(s) {
		t.Logf("skip %s", s)
		return
	}

	defer func(o commandLineOpts) { options = o } (options)
	options.failOnErrors = false

	var ctx = load_testcase(t, "testdata/modules/app", "testapp")

	ss := func(s string) string { os := "darwin"
		return strings.Replace(s, "<OS>", os, -1)
	}

	if d := ctx.def("_flag"); d == nil {
		ctx.err("_flag")
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
	} else if s != "$(foreach(-unique) $(filter-out $(foreach $1,&($2!$_) &($2~&(target.sys)!$_)),&($2) &($2~&(target.sys))) $(foreach $1,&($2.$_) &($2~&(target.sys).$_)),$(or $3,$2)$_$(or $4))" {
		ctx.err("%v ; %v", s, d)
	}
	if v := ctx.get("_flag", []string{"a", "b", "c"}, "-x"); v == nil {
		ctx.err("_flag")
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
	} else if s := v.Strval(ctx); s == "" {
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
	if v := ctx.get("_flag", []string{"a", "b", "c"}, "-z"); v == nil {
		ctx.err("_flag")
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
	} else if s := v.Strval(ctx); s != "" {
		ctx.err("%v ; %v", s, v)
	}
	if v := ctx.get("_flag", []string{"a", "b", "c"}, "-x", "-y"); v == nil {
		ctx.err("_flag")
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
	} else if s := v.Strval(ctx); s == "" {
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
	} else if strings.Count(s, "$(foreach(-unique) $2 $1,&(-v.$_) -std=&(-std.$_) -std=&(std.$_))") != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 $2,&(-D!$_) &(-D~&(target.sys)!$_)),&(-D) &(-D~&(target.sys))) $(foreach $1 $2,&(-D.$_) &(-D~&(target.sys).$_)),$(or $3,-D)$_$(or $4))") != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 $2,&(-f!$_) &(-f~&(target.sys)!$_)),&(-f) &(-f~&(target.sys))) $(foreach $1 $2,&(-f.$_) &(-f~&(target.sys).$_)),$(or $3,-f)$_$(or $4))") != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 $2,&(-I!$_) &(-I~&(target.sys)!$_)),&(-I) &(-I~&(target.sys))) $(foreach $1 $2,&(-I.$_) &(-I~&(target.sys).$_)),$(or $3,-I)$_$(or $4))") != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 $2,&(-isystem!$_) &(-isystem~&(target.sys)!$_)),&(-isystem) &(-isystem~&(target.sys))) $(foreach $1 $2,&(-isystem.$_) &(-isystem~&(target.sys).$_)),$(or $3,-isystem)$_$(or $4))") != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 $2,&(-isystem-after!$_) &(-isystem-after~&(target.sys)!$_)),&(-isystem-after) &(-isystem-after~&(target.sys))) $(foreach $1 $2,&(-isystem-after.$_) &(-isystem-after~&(target.sys).$_)),$(or $3,-isystem-after)$_$(or $4))") != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, "$(foreach(-unique) $1 $2,&(cppflags.$_) &(cppflags~&(target.sys).$_))") != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, "&(cppflags~&(target.sys))") != 1 {
		ctx.err("%v", d1.value)
	} else if s := d2.value.String(); strings.Count(s, "&(patsubst %,--target=%,&(target.triple))") != 1 {
		ctx.err("%v", d2.value)
	} else if strings.Count(s, "$(foreach(-unique) $2 $1,&(-v.$_) -std=&(-std.$_) -std=&(std.$_))") != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 $2,&(-D!$_) &(-D~&(target.sys)!$_)),&(-D) &(-D~&(target.sys))) $(foreach $1 $2,&(-D.$_) &(-D~&(target.sys).$_)),$(or $3,-D)$_$(or $4))") != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 $2,&(-f!$_) &(-f~&(target.sys)!$_)),&(-f) &(-f~&(target.sys))) $(foreach $1 $2,&(-f.$_) &(-f~&(target.sys).$_)),$(or $3,-f)$_$(or $4))") != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 $2,&(-I!$_) &(-I~&(target.sys)!$_)),&(-I) &(-I~&(target.sys))) $(foreach $1 $2,&(-I.$_) &(-I~&(target.sys).$_)),$(or $3,-I)$_$(or $4))") != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 $2,&(-isystem!$_) &(-isystem~&(target.sys)!$_)),&(-isystem) &(-isystem~&(target.sys))) $(foreach $1 $2,&(-isystem.$_) &(-isystem~&(target.sys).$_)),$(or $3,-isystem)$_$(or $4))") != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 $2,&(-isystem-after!$_) &(-isystem-after~&(target.sys)!$_)),&(-isystem-after) &(-isystem-after~&(target.sys))) $(foreach $1 $2,&(-isystem-after.$_) &(-isystem-after~&(target.sys).$_)),$(or $3,-isystem-after)$_$(or $4))") != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, "$(foreach(-unique) $1 $2,&(cppflags.$_) &(cppflags~&(target.sys).$_))") != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, "&(cppflags~&(target.sys))") != 1 {
		ctx.err("%v", d1.value)
	} else if s1, s2 := d1.value.Strval(ctx), d2.value.Strval(ctx); s1 != s2 {
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
	} else if strings.Count(s, "$(foreach(-unique) c $1,&(-v.$_) -std=&(-std.$_) -std=&(std.$_))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(if $(or &(-g) &(-g~&(target.sys)) $(foreach $1 c,&(-g.$_) &(-g~&(target.sys).$_))),-g)") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 c,&(-g!$_) &(-g~&(target.sys)!$_)),&(-g) &(-g~&(target.sys))) $(foreach $1 c,&(-g.$_) &(-g~&(target.sys).$_)),$(or $3,-g)$_$(or $4))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 c,&(-O!$_) &(-O~&(target.sys)!$_)),&(-O) &(-O~&(target.sys))) $(foreach $1 c,&(-O.$_) &(-O~&(target.sys).$_)),$(or $3,-O)$_$(or $4))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 c,&(-D!$_) &(-D~&(target.sys)!$_)),&(-D) &(-D~&(target.sys))) $(foreach $1 c,&(-D.$_) &(-D~&(target.sys).$_)),$(or $3,-D)$_$(or $4))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 c,&(-f!$_) &(-f~&(target.sys)!$_)),&(-f) &(-f~&(target.sys))) $(foreach $1 c,&(-f.$_) &(-f~&(target.sys).$_)),$(or $3,-f)$_$(or $4))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 c,&(-m!$_) &(-m~&(target.sys)!$_)),&(-m) &(-m~&(target.sys))) $(foreach $1 c,&(-m.$_) &(-m~&(target.sys).$_)),$(or $3,-m)$_$(or $4))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 c,&(-W!$_) &(-W~&(target.sys)!$_)),&(-W) &(-W~&(target.sys))) $(foreach $1 c,&(-W.$_) &(-W~&(target.sys).$_)),$(or $3,-W)$_$(or $4))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 c,&(-I!$_) &(-I~&(target.sys)!$_)),&(-I) &(-I~&(target.sys))) $(foreach $1 c,&(-I.$_) &(-I~&(target.sys).$_)),$(or $3,-I)$_$(or $4))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 c,&(-no!$_) &(-no~&(target.sys)!$_)),&(-no) &(-no~&(target.sys))) $(foreach $1 c,&(-no.$_) &(-no~&(target.sys).$_)),$(or $3,-no)$_$(or $4))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 c,&(-isystem!$_) &(-isystem~&(target.sys)!$_)),&(-isystem) &(-isystem~&(target.sys))) $(foreach $1 c,&(-isystem.$_) &(-isystem~&(target.sys).$_)),$(or $3,-isystem)$_$(or $4))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 c,&(-isystem-after!$_) &(-isystem-after~&(target.sys)!$_)),&(-isystem-after) &(-isystem-after~&(target.sys))) $(foreach $1 c,&(-isystem-after.$_) &(-isystem-after~&(target.sys).$_)),$(or $3,-isystem-after)$_$(or $4))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(foreach(-unique) $1 c,&($(or $3,c)flags.$_) &($(or $3,c)flags~&(target.sys).$_))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "&($(or $3,c)flags~&(target.sys))") != 1 {
		ctx.err("%v", v1)
	} else if s := v2.String(); s == "" {
		ctx.err("%T %v", v2, v2)
	} else if strings.Count(s, "&(patsubst %,--target=%,&(target.triple))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(foreach(-unique) c $1,&(-v.$_) -std=&(-std.$_) -std=&(std.$_))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(if $(or &(-g) &(-g~&(target.sys)) $(foreach $1 c,&(-g.$_) &(-g~&(target.sys).$_))),-g)") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 c,&(-g!$_) &(-g~&(target.sys)!$_)),&(-g) &(-g~&(target.sys))) $(foreach $1 c,&(-g.$_) &(-g~&(target.sys).$_)),$(or $3,-g)$_$(or $4))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 c,&(-O!$_) &(-O~&(target.sys)!$_)),&(-O) &(-O~&(target.sys))) $(foreach $1 c,&(-O.$_) &(-O~&(target.sys).$_)),$(or $3,-O)$_$(or $4))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 c,&(-D!$_) &(-D~&(target.sys)!$_)),&(-D) &(-D~&(target.sys))) $(foreach $1 c,&(-D.$_) &(-D~&(target.sys).$_)),$(or $3,-D)$_$(or $4))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 c,&(-f!$_) &(-f~&(target.sys)!$_)),&(-f) &(-f~&(target.sys))) $(foreach $1 c,&(-f.$_) &(-f~&(target.sys).$_)),$(or $3,-f)$_$(or $4))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 c,&(-m!$_) &(-m~&(target.sys)!$_)),&(-m) &(-m~&(target.sys))) $(foreach $1 c,&(-m.$_) &(-m~&(target.sys).$_)),$(or $3,-m)$_$(or $4))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 c,&(-W!$_) &(-W~&(target.sys)!$_)),&(-W) &(-W~&(target.sys))) $(foreach $1 c,&(-W.$_) &(-W~&(target.sys).$_)),$(or $3,-W)$_$(or $4))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 c,&(-I!$_) &(-I~&(target.sys)!$_)),&(-I) &(-I~&(target.sys))) $(foreach $1 c,&(-I.$_) &(-I~&(target.sys).$_)),$(or $3,-I)$_$(or $4))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 c,&(-no!$_) &(-no~&(target.sys)!$_)),&(-no) &(-no~&(target.sys))) $(foreach $1 c,&(-no.$_) &(-no~&(target.sys).$_)),$(or $3,-no)$_$(or $4))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 c,&(-isystem!$_) &(-isystem~&(target.sys)!$_)),&(-isystem) &(-isystem~&(target.sys))) $(foreach $1 c,&(-isystem.$_) &(-isystem~&(target.sys).$_)),$(or $3,-isystem)$_$(or $4))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out $(foreach $1 c,&(-isystem-after!$_) &(-isystem-after~&(target.sys)!$_)),&(-isystem-after) &(-isystem-after~&(target.sys))) $(foreach $1 c,&(-isystem-after.$_) &(-isystem-after~&(target.sys).$_)),$(or $3,-isystem-after)$_$(or $4))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(foreach(-unique) $1 c,&($(or $3,c)flags.$_) &($(or $3,c)flags~&(target.sys).$_))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "&($(or $3,c)flags~&(target.sys))") != 1 {
		ctx.err("%v", v1)
	} else if s1, s2 := v1.Strval(ctx), v2.Strval(ctx); s1 != s2 {
		ctx.err("%T %v -> %s", v2, v2, s)
		ctx.err("%T %v -> %s", v2, v2, s)
	}

	if v := ctx.get("std.fxxbxx"); v == nil {
		ctx.err("std.fxxbxx")
	} else if s := v.Strval(ctx); s != "stdfxxbxx1" {
		ctx.err("%T %v -> %s", v, v, s)
	}
	if v := ctx.get("-std.fxxbxx"); v == nil {
		ctx.err("-std.fxxbxx")
	} else if s := v.Strval(ctx); s != "stdfxxbxx2" {
		ctx.err("%T %v -> %s", v, v, s)
	}
	if v1, v2 := ctx.get("cflags", "fxxbxx"), ctx.get("xflags", "fxxbxx"); v1 == nil || v2 == nil {
		ctx.err("cflags")
		ctx.err("xflags")
	} else if s := v1.String(); s == "" {
		ctx.err("%T %v", v1, v1)
	} else if strings.Count(s, "&(patsubst %,--target=%,&(target.triple))") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "&(-v.c)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "-std=&(-std.c)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "-std=&(std.c)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "&(-v.fxxbxx)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "-std=&(-std.fxxbxx)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "-std=&(std.fxxbxx)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "$(if $(or &(-g) &(-g~&(target.sys)) &(-g.fxxbxx) &(-g~&(target.sys).fxxbxx) &(-g.c) &(-g~&(target.sys).c)),-g)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out &(-g!fxxbxx) &(-g~&(target.sys)!fxxbxx) &(-g!c) &(-g~&(target.sys)!c),&(-g) &(-g~&(target.sys))) &(-g.fxxbxx) &(-g~&(target.sys).fxxbxx) &(-g.c) &(-g~&(target.sys).c),$(or $3,-g)$_$(or $4))") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out &(-O!fxxbxx) &(-O~&(target.sys)!fxxbxx) &(-O!c) &(-O~&(target.sys)!c),&(-O) &(-O~&(target.sys))) &(-O.fxxbxx) &(-O~&(target.sys).fxxbxx) &(-O.c) &(-O~&(target.sys).c),$(or $3,-O)$_$(or $4))") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out &(-D!fxxbxx) &(-D~&(target.sys)!fxxbxx) &(-D!c) &(-D~&(target.sys)!c),&(-D) &(-D~&(target.sys))) &(-D.fxxbxx) &(-D~&(target.sys).fxxbxx) &(-D.c) &(-D~&(target.sys).c),$(or $3,-D)$_$(or $4))") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out &(-f!fxxbxx) &(-f~&(target.sys)!fxxbxx) &(-f!c) &(-f~&(target.sys)!c),&(-f) &(-f~&(target.sys))) &(-f.fxxbxx) &(-f~&(target.sys).fxxbxx) &(-f.c) &(-f~&(target.sys).c),$(or $3,-f)$_$(or $4))") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out &(-m!fxxbxx) &(-m~&(target.sys)!fxxbxx) &(-m!c) &(-m~&(target.sys)!c),&(-m) &(-m~&(target.sys))) &(-m.fxxbxx) &(-m~&(target.sys).fxxbxx) &(-m.c) &(-m~&(target.sys).c),$(or $3,-m)$_$(or $4))") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out &(-W!fxxbxx) &(-W~&(target.sys)!fxxbxx) &(-W!c) &(-W~&(target.sys)!c),&(-W) &(-W~&(target.sys))) &(-W.fxxbxx) &(-W~&(target.sys).fxxbxx) &(-W.c) &(-W~&(target.sys).c),$(or $3,-W)$_$(or $4))") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out &(-I!fxxbxx) &(-I~&(target.sys)!fxxbxx) &(-I!c) &(-I~&(target.sys)!c),&(-I) &(-I~&(target.sys))) &(-I.fxxbxx) &(-I~&(target.sys).fxxbxx) &(-I.c) &(-I~&(target.sys).c),$(or $3,-I)$_$(or $4))") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out &(-no!fxxbxx) &(-no~&(target.sys)!fxxbxx) &(-no!c) &(-no~&(target.sys)!c),&(-no) &(-no~&(target.sys))) &(-no.fxxbxx) &(-no~&(target.sys).fxxbxx) &(-no.c) &(-no~&(target.sys).c),$(or $3,-no)$_$(or $4))") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out &(-isystem!fxxbxx) &(-isystem~&(target.sys)!fxxbxx) &(-isystem!c) &(-isystem~&(target.sys)!c),&(-isystem) &(-isystem~&(target.sys))) &(-isystem.fxxbxx) &(-isystem~&(target.sys).fxxbxx) &(-isystem.c) &(-isystem~&(target.sys).c),$(or $3,-isystem)$_$(or $4))") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out &(-isystem-after!fxxbxx) &(-isystem-after~&(target.sys)!fxxbxx) &(-isystem-after!c) &(-isystem-after~&(target.sys)!c),&(-isystem-after) &(-isystem-after~&(target.sys))) &(-isystem-after.fxxbxx) &(-isystem-after~&(target.sys).fxxbxx) &(-isystem-after.c) &(-isystem-after~&(target.sys).c),$(or $3,-isystem-after)$_$(or $4))") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "&($(or $3,c)flags.fxxbxx)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "&($(or $3,c)flags~&(target.sys).fxxbxx)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "&($(or $3,c)flags.c)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "&($(or $3,c)flags~&(target.sys).c)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "&($(or $3,c)flags~&(target.sys))") != 1 {
		ctx.err("%v", s)
	} else if s := v2.String(); s == "" {
		ctx.err("%T %v", v2, v2)
	} else if strings.Count(s, "&(patsubst %,--target=%,&(target.triple))") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "&(-v.c)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "-std=&(-std.c)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "-std=&(std.c)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "&(-v.fxxbxx)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "-std=&(-std.fxxbxx)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "-std=&(std.fxxbxx)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "$(if $(or &(-g) &(-g~&(target.sys)) &(-g.fxxbxx) &(-g~&(target.sys).fxxbxx) &(-g.c) &(-g~&(target.sys).c)),-g)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out &(-g!fxxbxx) &(-g~&(target.sys)!fxxbxx) &(-g!c) &(-g~&(target.sys)!c),&(-g) &(-g~&(target.sys))) &(-g.fxxbxx) &(-g~&(target.sys).fxxbxx) &(-g.c) &(-g~&(target.sys).c),$(or $3,-g)$_$(or $4))") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out &(-O!fxxbxx) &(-O~&(target.sys)!fxxbxx) &(-O!c) &(-O~&(target.sys)!c),&(-O) &(-O~&(target.sys))) &(-O.fxxbxx) &(-O~&(target.sys).fxxbxx) &(-O.c) &(-O~&(target.sys).c),$(or $3,-O)$_$(or $4))") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out &(-D!fxxbxx) &(-D~&(target.sys)!fxxbxx) &(-D!c) &(-D~&(target.sys)!c),&(-D) &(-D~&(target.sys))) &(-D.fxxbxx) &(-D~&(target.sys).fxxbxx) &(-D.c) &(-D~&(target.sys).c),$(or $3,-D)$_$(or $4))") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out &(-f!fxxbxx) &(-f~&(target.sys)!fxxbxx) &(-f!c) &(-f~&(target.sys)!c),&(-f) &(-f~&(target.sys))) &(-f.fxxbxx) &(-f~&(target.sys).fxxbxx) &(-f.c) &(-f~&(target.sys).c),$(or $3,-f)$_$(or $4))") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out &(-m!fxxbxx) &(-m~&(target.sys)!fxxbxx) &(-m!c) &(-m~&(target.sys)!c),&(-m) &(-m~&(target.sys))) &(-m.fxxbxx) &(-m~&(target.sys).fxxbxx) &(-m.c) &(-m~&(target.sys).c),$(or $3,-m)$_$(or $4))") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out &(-W!fxxbxx) &(-W~&(target.sys)!fxxbxx) &(-W!c) &(-W~&(target.sys)!c),&(-W) &(-W~&(target.sys))) &(-W.fxxbxx) &(-W~&(target.sys).fxxbxx) &(-W.c) &(-W~&(target.sys).c),$(or $3,-W)$_$(or $4))") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out &(-I!fxxbxx) &(-I~&(target.sys)!fxxbxx) &(-I!c) &(-I~&(target.sys)!c),&(-I) &(-I~&(target.sys))) &(-I.fxxbxx) &(-I~&(target.sys).fxxbxx) &(-I.c) &(-I~&(target.sys).c),$(or $3,-I)$_$(or $4))") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out &(-no!fxxbxx) &(-no~&(target.sys)!fxxbxx) &(-no!c) &(-no~&(target.sys)!c),&(-no) &(-no~&(target.sys))) &(-no.fxxbxx) &(-no~&(target.sys).fxxbxx) &(-no.c) &(-no~&(target.sys).c),$(or $3,-no)$_$(or $4))") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out &(-isystem!fxxbxx) &(-isystem~&(target.sys)!fxxbxx) &(-isystem!c) &(-isystem~&(target.sys)!c),&(-isystem) &(-isystem~&(target.sys))) &(-isystem.fxxbxx) &(-isystem~&(target.sys).fxxbxx) &(-isystem.c) &(-isystem~&(target.sys).c),$(or $3,-isystem)$_$(or $4))") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out &(-isystem-after!fxxbxx) &(-isystem-after~&(target.sys)!fxxbxx) &(-isystem-after!c) &(-isystem-after~&(target.sys)!c),&(-isystem-after) &(-isystem-after~&(target.sys))) &(-isystem-after.fxxbxx) &(-isystem-after~&(target.sys).fxxbxx) &(-isystem-after.c) &(-isystem-after~&(target.sys).c),$(or $3,-isystem-after)$_$(or $4))") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "&($(or $3,c)flags.fxxbxx)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "&($(or $3,c)flags~&(target.sys).fxxbxx)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "&($(or $3,c)flags.c)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "&($(or $3,c)flags~&(target.sys).c)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "&($(or $3,c)flags~&(target.sys))") != 1 {
		ctx.err("%v", s)
	} else if s1 := v1.Strval(ctx); s1 == "" {
		ctx.err("%T %v -> %s", v2, v2, s1)
	} else if s2 := v2.Strval(ctx); s2 == "" {
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

	var str1 = ss(`fooD fooF fooI foostd fooisystem fooisystem-after cppflags-foo cppflags-foo~<OS>`)
	var str2 = ss(`fooF fooG fooD fooO fooI foostd fooisystem foocxxisystem fooisystem-after foostdlib++isystem cxxflags-foo cxxflags-foo~<OS>`)

	if v := ctx.get(".test.1"); v == nil {
		ctx.err(".test.1")
	} else if s := v.String(); s == "" {
		ctx.err("%T %v", v, v)
	} else if s := v.Strval(ctx); s == "" {
		ctx.err("%T %v -> %s", v, v, s)
	} else if false { for _, t := range strings.Fields(str1) { if !strings.Contains(s, t) {
		ctx.err("%v : %s ; %T %v", t, s, v, v)
	}}} else if !strings.Contains(s, "-std=fxxbar") {
		ctx.err("%v : %s ; %T %v", t, s, v, v)
	}

	if v := ctx.get(".test.2"); v == nil {
		ctx.err(".test.2")
	} else if s := v.String(); s == "" {
		ctx.err("%T %v", v, v)
	} else if s := v.Strval(ctx); s == "" {
		ctx.err("%T %v -> %s", v, v, s)
	} else if false { for _, t := range strings.Fields(str1) { if !strings.Contains(s, t) {
		ctx.err("%v : %s ; %T %v", t, s, v, v)
	}}} else if !strings.Contains(s, "-std=fxxbar") {
		ctx.err("%v : %s ; %T %v", t, s, v, v)
	}

	if v := ctx.get(".test.3"); v == nil {
		ctx.err(".test.3")
	} else if s := v.String(); s == "" {
		ctx.err("%T %v", v, v)
	} else if s := v.Strval(ctx); s == "" {
		ctx.err("%T %v -> %s", v, v, s)
	} else if false { for _, t := range strings.Fields(str1) { if !strings.Contains(s, t) {
		ctx.err("%v : %s ; %T %v", t, s, v, v)
	}}} else if !strings.Contains(s, "-std=fxxbar") {
		ctx.err("%v : %s ; %T %v", t, s, v, v)
	}

	if v := ctx.get(".test.4"); v == nil {
		ctx.err(".test.4")
	} else if s := v.String(); s == "" {
		ctx.err("%T %v", v, v)
	} else if s := v.Strval(ctx); s == "" {
		ctx.err("%T %v -> %s", v, v, s)
	} else if true { for _, t := range strings.Fields(str1) { if !strings.Contains(s, t) {
		ctx.err("%v : %s ; %T %v", t, s, v, v)
	}}}

	if v := ctx.get(".test.5"); v == nil {
		ctx.err(".test.5")
	} else if s := v.String(); s == "" {
		ctx.err("%T %v", v, v)
	} else if s := v.Strval(ctx); s == "" {
		ctx.err("%T %v -> %s", v, v, s)
	} else if true { for _, t := range strings.Fields(str2) { if !strings.Contains(s, t) {
		ctx.err("%v : %s ; %T %v", t, s, v, v)
	}}}

	ctx.flush()
}
