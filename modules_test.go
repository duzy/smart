//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
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
	regexp.MustCompile(`^-(?:-target|triple)=[[:alnum:]\.\-_]+$`),
	regexp.MustCompile(`^-(?:shared|static|ObjC(?:\+\+)?)$`),
	regexp.MustCompile(`^-(?:std|Werror)=[[:alnum:]_\-]+$`),
	regexp.MustCompile(`^-(?:(?:(?:cxx|stdlib\+\+)-)?isystem(?:-after)?)=?[[:alnum:]\.\-/_]+$`),
	regexp.MustCompile(`^-Wl,-platform_version,(?:MacOSX|iPhone(?:Simulator)?|AppleTV|Watch(?:Simulator)?|DriverKit),[[:digit:].]+,[[:digit:].]+$`),
	regexp.MustCompile(`^-Wl,(?:-v|-demangle|-rpath,"[^"]+")$`),
	regexp.MustCompile(`^-[IL]=?[[:alnum:]\.\-/_]+$`),
	regexp.MustCompile(`^-[DWfl][[:alnum:]\.\-_]+$`),
	regexp.MustCompile(`^-m(?:arch|linker-version=[[:digit:]]+)$`),
	regexp.MustCompile(`^-no[[:alnum:]\.\-\+_]+$`),
	// regexp.MustCompile(`^-X(?:clang)$`),
	regexp.MustCompile(`^-x(?:c(?:\+\+)?)$`),
	regexp.MustCompile(`^-O[0-6]$`),
	regexp.MustCompile(`^-l[^/]+$`),
	regexp.MustCompile(`^-[cvg]$`),
	regexp.MustCompile(`^-(?:-|i(?:(?:framework)?with)?)sysroot=[[:alnum:]\.\-/_]+$`),
	regexp.MustCompile(`^(?:/(?:[^/]*/)+)?.+\.(?:[coh])$`), // /a/b/c/foo.c
}

var testValidFlagVal = map[*regexp.Regexp][]*regexp.Regexp{
	regexp.MustCompile(`^-(?:std|Werror)$`): []*regexp.Regexp{
		regexp.MustCompile(`^[[:alnum:]\.\-_]+$`),
	},
	regexp.MustCompile(`^-(?:-target|triple)$`): []*regexp.Regexp{
		regexp.MustCompile(`^[[:alnum:]\.\-_]+$`),
	},
	regexp.MustCompile(`^-D$`): []*regexp.Regexp{
		regexp.MustCompile(`^[[:alnum:]\.\-_]+$`),
	},
	regexp.MustCompile(`^-([I]|include)$`): []*regexp.Regexp{
		regexp.MustCompile(`^(?:/(?:[^/]*/)+)?.+\.(?:[ch](?:xx|pp)|inc)$`), // /a/b/c/foo.c
	},
	regexp.MustCompile(`^-o$`): []*regexp.Regexp{
		regexp.MustCompile(`^(?:/(?:[^/]*/)+)?.+\.(?:[ox]|out|exe)$`), // /a/b/c/foo.c
	},
	regexp.MustCompile(`^-framework$`): []*regexp.Regexp{
		regexp.MustCompile(`^[^-].+$`),
	},
	regexp.MustCompile(`^-(L|(?:-|i(?:(?:framework)?with)?)sysroot)$`): []*regexp.Regexp{
		regexp.MustCompile(`^(?:/(?:[^/]*/)+)?.+$`),
	},
	regexp.MustCompile(`^-l$`): []*regexp.Regexp{
		regexp.MustCompile(`^[^/]+$`),
	},
	regexp.MustCompile(`^-X(?:clang)$`): []*regexp.Regexp{
		regexp.MustCompile(`^-triple=[^/]+$`),
	},
}

var testInvalidArg = []*regexp.Regexp{
	regexp.MustCompile(`<[^>]*>`),
}

var testWrongExecOutput = []*regexp.Regexp{
	regexp.MustCompile(`^#error (Unsupported architecture|architecture not supported)`),
	regexp.MustCompile(`^clang: error: unknown argument '[^']+'; did you mean '[^']+'\?`),
	regexp.MustCompile(`^ld: (missing OS version in target triple '([^']+)') in '([^']+)'`),
	regexp.MustCompile(`^ld: (more than three dashes in target triple '([^']+)') in '([^']+)'`),
	regexp.MustCompile(`^ld: (unknown OS in target triple '([^']+)') in '([^']+)'`),
	regexp.MustCompile(`^ld: (unknown file type) in '([^']+)'`),
	regexp.MustCompile(`^ld: (unknown options: (.+))`),
	regexp.MustCompile(`^ld: (Missing -platform_version option)`),
	regexp.MustCompile(`^ld: (-platform_version unknown platform: (.+))`),
	regexp.MustCompile(`^(ld: library '[^=]+=.+?' not found)`), // ld: library 'NAME=bsd' not found

	// Errors caused by wrong #include, example: #include TARGET=HAVE_LIBIEEE
	regexp.MustCompile(`^.+:[[:digit:]]+:[[:digit:]]+: (error: expected "FILENAME" or <FILENAME>)`),
	regexp.MustCompile(`^(#include [^=]+=.+)`),
}
type testSuspicious struct {
	rx *regexp.Regexp
	ignore map[string]struct{}
	i, k int // info, key
}
var testSuspiciousExecOutput = []*testSuspicious{
	&testSuspicious{
		regexp.MustCompile(`^clang:(?: warning:)? (argument unused during compilation: '[^']+' \[[^\]]+\])`),
		map[string]struct{}{}, 1, 0,
	},
	&testSuspicious{
		regexp.MustCompile(`^ld:(?: warning:)? (search path '([^']+)' not found)`),
		map[string]struct{}{}, 1, 2,
	},
	&testSuspicious{
		regexp.MustCompile(`TODO: ^ld:(?: warning:)? (ignoring duplicate libraries: '([^']+)')`),
		map[string]struct{}{}, 1, 2,
	},
	&testSuspicious{
		regexp.MustCompile(`^(ignoring nonexistent directory "([^"]+)")`),
		map[string]struct{}{
			`/Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX.sdk/Library/Frameworks`: struct{}{},
			`/Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX.sdk/usr/local/include`: struct{}{},
			`/usr/local/include`: struct{}{},
			`/usr/include`: struct{}{},
		}, 1, 2,
	},
}

func validFlag(s string) (res bool, vxs []*regexp.Regexp) {
	for _, x := range testValidFlag { if res = x.MatchString(s); res { return }}
	for x, v := range testValidFlagVal { if x.MatchString(s) { return true, v }}
	return
}

func validFlags(ctx *testcase, v Value, s string) (res bool) {
	var rxs []*regexp.Regexp
	var fields = strings.Fields(s)
	for i := 0; i < len(fields); i += 1 {
		flag := fields[i]

		if res, rxs = validFlag(flag); !res {
			ctx.err("invalid flag: %s ; %v{%v}", flag, typeof(v), v) ; break
		} else if len(rxs) == 0 {
			continue
		}

		if i += 1; i == len(fields) {
			ctx.err("wrong flag: %s ; %v{%v}", flag, typeof(v), v)
			return
		}

		val := fields[i]

		for _, rx := range rxs {
			var ( s = val ; n = 0 )
			for ; n < strings.Count(rx.String(), "[ ]"); n += 1 {
				if i+n+1 < len(fields) { s += " " + fields[i+n+1] } else { break }
			}

			if !rx.MatchString(s) {
				ctx.err("wrong flag: %s %s, %v ; %v{%v}", flag, s, rx, typeof(v), v)
			} else {
				if false { note(ctx, "%v: %v ; %v", flag, s, rx).debug() }
				i += n
				break
			}
		}
	}
	return
}

var testValidateClang = regexp.MustCompile(`^@?(?:/(?:[^/]*/)+)?(clang(?:\+{2})?)[[:space:]]+`)
var testValidateOther = regexp.MustCompile(`^@?(?:/(?:[^/]*/)+)?(echo)[[:space:]]+`)
var testValidateOutFilename = regexp.MustCompile(`\.configure/[^/=]+?/[^/=]+$`)

func testValidateExecRecipe(tc *testcase, ctx Context, source string, recipe Value) {
	if source == "" || recipe == nil { return }

	if m := testValidateClang.FindStringSubmatch(source); m != nil {
		if !validFlags(tc, recipe, source[len(m[0]):]) {
			note(ctx, "validate: %v; %v", m, source).debug()
		}
	} else if m := testValidateOther.FindStringSubmatch(source); m != nil {
		// okay
	} else {
		note(ctx, "TODO: validate: %v", source).debug()
	}
}

func testValidateExecOutput(tc *testcase, ctx Context, line string, l int) {
	if s := _position(ctx).Filename; !testValidateOutFilename.MatchString(s) {
		errostack(ctx, 16, "bad out-file: %v", s).trace()
	}
	for _, rx := range testWrongExecOutput {
		if m := rx.FindStringSubmatch(line); len(m) > 0 {
			if len(m) < 2 {
				errostack(ctx, 16, "%v", m[0]).trace()
			} else {
				errostack(ctx, 16, "%v", m[1]).trace()
			}
		}
	}
	for _, t := range testSuspiciousExecOutput {
		if m := t.rx.FindStringSubmatch(line); len(m) > 0 {
			if _, y := t.ignore[m[t.k]]; !y {
				errostack(ctx, 16, "%v", m[t.i]).trace()
			}
		}
	}
}

func testVariantTargetVars(ctx *testcase) {
	testVariantTargetVars1(ctx)
	if v := ctx.val("target.os"); v == nil {
		ctx.err("%v: target.os is nil", _project(ctx))
	}
}
func testVariantTargetVars1(ctx *testcase) {
	var p = _project(ctx)
	var workspace, modules string

	if d := p.resolveDef(ctx, "workspace"); d == nil {
		ctx.err("%v: workspace is nil", p)
	} else if workspace = d.string(ctx); workspace == "" {
		ctx.err("%v: %v", p, d)
	} else if !filepath.IsAbs(workspace) {
		ctx.err("%v: %v", p, workspace)
	} else {
		modules = filepath.Join(workspace, ".smart", "modules")
	}

	for _, s := range []string {
		"variant/.target/.base/do.smart",
		"variant/.target/do.smart",
		"variant/.target/darwin/do.smart",
		"variant/.target/darwin/arm64/do.smart",
		"variant/bootstrap",
		"variant/do.smart",
	}{
		var t = filepath.Join(modules, s)
		if _, y := ctx.chks[t]; !y {
			ctx.err("%v: %v", p, s)
		}
	}

	if v := p.resolve(ctx, "variant"); v == nil {
		ctx.err("%v: variant is nil", p)
	} else if d, y := v.(*def); !y {
		ctx.err("%v: variant is not def: %v (%v)", p, v, typeof(v))
	} else if d.value == nil {
		ctx.err("%v: variant value is nil: %v", p, d)
	} else if s := d.value.string(ctx); s != "darwin/arm64/bootstrap" {
		ctx.err("%v: variant: %v", p, d.value)
	}

	if v := ctx.val("target.arch"); v == nil {
		ctx.err("%v: target.arch is nil", p)
	} else if v.string(ctx) != "arm64" {
		ctx.err("%v: target.arch: %v", p, v)
	}

	if v := ctx.val("target.abi"); v == nil {
		ctx.err("%v: target.abi is nil", p)
	} else if v.string(ctx) != "macho" {
		ctx.err("%v: target.abi: %v", p, v)
	}

	if v := ctx.val("target.vendor"); v == nil {
		ctx.err("%v: target.vendor is nil", p)
	} else if v.string(ctx) != "apple" {
		ctx.err("%v: target.vendor: %v", p, v)
	}

	if v := ctx.val("target.release"); v == nil {
		ctx.err("%v: target.release is nil", p)
	} else if !regexp.MustCompile(`[0-9]+\.[0-9]+\.[0-9]+`).MatchString(v.string(ctx)) {
		ctx.err("%v: target.release: %v", p, v)
	}

	if v := ctx.val("target.sys"); v == nil {
		ctx.err("%v: target.sys is nil", p)
	} else if !regexp.MustCompile(`Darwin[0-9]+\.[0-9]+\.[0-9]+`).MatchString(v.string(ctx)) {
		ctx.err("%v: target.sys: %v", p, v)
	}

	if v := ctx.val("target.triple"); v == nil {
		ctx.err("%v: target.triple is nil", p)
	} else if !regexp.MustCompile(`arm64-apple-Darwin[0-9]+\.[0-9]+\.[0-9]+-macho`).MatchString(v.string(ctx)) {
		ctx.err("%v: target.triple: %v", p, v)
	}
}

func testVariantTarget(ctx *testcase) {
	testVariantTargetVars1(ctx)

	var p = _project(ctx)

	for k, v := range langs_map {
		var s = "lang."+k
		if d := ctx.def(s); d == nil {
			ctx.err("%v : %v", p, s)//, p.names()
		} else if d.value == nil {
			ctx.err("%v", d)
		} else if d.value.String() != v {
			ctx.err("%v", d)
		}
	}

	if d := ctx.def("host.triple"); d == nil {
		ctx.err("host.triple")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, d)
	} else if s, t := d.value.String(), "$(join &(host.arch)&(host.sub) &(host.vendor) &(host.sys) &(host.abi),-)"; s != t {
		ctx.err("%v : %s != %s", tst{d}, s, t)
	} else if s := d.value.string(ctx); strings.Count(s, "-") > 3 {
		ctx.err("more than three dashes: %v: %v", d.value, s)
	}

	if d := ctx.def("target.os"); d == nil {
		ctx.err("%v: target.os is nil", p)
	} else if d.string(ctx) != "foo" {
		ctx.err("%v: target.os: %v", p, d)
	}

	if d := ctx.def("target.triple"); d == nil {
		ctx.err("target.triple")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, d)
	} else if s, t := d.value.String(), "$(join &(target.arch)&(target.sub) &(target.vendor) &(target.sys) &(target.abi),-)"; s != t {
		ctx.err("%v : %s != %s", tst{d}, s, t)
	} else if s := d.value.string(ctx); strings.Count(s, "-") > 3 {
		ctx.err("more than three dashes: %v: %v", d.value, s)
	}

	var usev, uses string
	if d := ctx.def("use.*"); d == nil {
		ctx.err("use.*")
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, d)
	}
	if v := ctx.val("use.*"); v == nil {
		ctx.err("use.*")
	} else if usev = v.String(); usev == "" {
		ctx.err("%v{%v}", typeof(v), v)
	} else if uses = v.string(ctx); uses == "" {
		ctx.err("%v{%v} → %s", typeof(v), v, uses)
	}

	usev = " "+usev
	uses = " "+uses

	for _, flag := range []string{
		"-D", "-I", "-L", "-O", "-W", "-Wl", "-Werror", "-Wno-error",
		"-f", "-f.ld", "-m", "-g", "-v", "-no", "-no.ld",
		"--sysroot", "-isysroot", "-iwithsysroot", "-iframeworkwithsysroot", "-iframework",
		"-isystem", "-isystem-after", "-cxx-isystem", "-stdlib", "-stdlib++-isystem",
		"-diagnostics", "-platform_version",
	}{
		s1 := fmt.Sprintf("%s(-unique)", flag)
		s2 := fmt.Sprintf("&(target.os)~%s(-unique)", flag)
		s3 := fmt.Sprintf("foo~%s(-unique)", flag)
		if strings.Count(usev, " "+s1) != 1 { ctx.err("%v ; %v", s1, usev) }
		if strings.Count(usev, " "+s2) != 1 { ctx.err("%v ; %v", s2, usev) }
		if strings.Count(uses, " "+s3) != 1 { ctx.err("%v ; %v", s3, uses) }
		if d := ctx.def(flag); d == nil {
			ctx.err("%s", flag)
		} else if d.o != defVoid && d.o != defExpand1 && d.o != defExpand2 {
			ctx.err("%v %v", d.o, d)
		}

		for _, lang := range langs_map {
			s0 := fmt.Sprintf("%s.%s", flag, lang)
			s1 := fmt.Sprintf("%s.%s(-unique)", flag, lang)
			s2 := fmt.Sprintf("&(target.os)~%s.%s(-unique)", flag, lang)
			s3 := fmt.Sprintf("foo~%s.%s(-unique)", flag, lang)
			if strings.Count(usev, " "+s1) != 1 { ctx.err("%v", s1) }
			if strings.Count(usev, " "+s2) != 1 { ctx.err("%v", s2) }
			if strings.Count(uses, " "+s3) != 1 { ctx.err("%v", s3) }
			if d := ctx.def(s0); d == nil {
				ctx.err("%s", s0)
			} else if d.o != defVoid && d.o != defExpand1 && d.o != defExpand2 {
				ctx.err("%v %v", d.o, d)
			}
		}
	}
	for _, flag := range []string{"-l","-framework"} {
		s1 := fmt.Sprintf("%s(-unique -reverse)", flag)
		s2 := fmt.Sprintf("&(target.os)~%s(-unique -reverse)", flag)
		s3 := fmt.Sprintf("foo~%s(-unique -reverse)", flag)
		if strings.Count(usev, " "+s1) != 1 { ctx.err("%v", s1) }
		if strings.Count(usev, " "+s2) != 1 { ctx.err("%v", s2) }
		if strings.Count(uses, " "+s3) != 1 { ctx.err("%v", s3) }
		if d := ctx.def(flag); d == nil {
			ctx.err("%s", flag)
		} else if d.o != defVoid && d.o != defExpand1 && d.o != defExpand2 {
			ctx.err("%v %v", d.o, d)
		}
	}
	for _, flag := range []string{"ar","asm","c","cpp","cxx","oc","ocxx","cl","cuda","cudaxx","ld"} {
		s0 := fmt.Sprintf("%sflags", flag)
		s1 := fmt.Sprintf("%sflags(-unique -auto)", flag)
		s2 := fmt.Sprintf("&(target.os)~%sflags(-unique -auto)", flag)
		s3 := fmt.Sprintf("foo~%sflags(-unique -auto)", flag)
		if strings.Count(usev, " "+s1) != 1 { ctx.err("%v", s1) }
		if strings.Count(usev, " "+s2) != 1 { ctx.err("%v", s2) }
		if strings.Count(uses, " "+s3) != 1 { ctx.err("%v", s3) }
		if d := ctx.def(s0); d == nil {
			ctx.err("%s", s0)
		} else if d.o != defVoid && d.o != defExpand1 && d.o != defExpand2 {
			ctx.err("%v %v", d.o, d)
		}
	}
	for _, flag := range []string{"ld"} {
		for _, suffix := range strings.Fields("shared program") {
			s0 := fmt.Sprintf("%sflags.%s", flag, suffix)
			s1 := fmt.Sprintf("%sflags.%s(-unique -auto)", flag, suffix)
			s2 := fmt.Sprintf("&(target.os)~%sflags.%s(-unique -auto)", flag, suffix)
			s3 := fmt.Sprintf("foo~%sflags.%s(-unique -auto)", flag, suffix)
			if strings.Count(usev, " "+s1) != 1 { ctx.err("%v", s1) }
			if strings.Count(usev, " "+s2) != 1 { ctx.err("%v", s2) }
			if strings.Count(uses, " "+s3) != 1 { ctx.err("%v", s3) }
			if d := ctx.def(s0); d == nil {
				ctx.err("%s", s0)
			} else if d.o != defVoid && d.o != defExpand1 && d.o != defExpand2 {
				ctx.err("%v %v", d.o, d)
			}
		}
	}
	for _, flag := range []string{"ld.framework","ldlibs","loadlibs","loadlibes"} {
		s0 := fmt.Sprintf("%s", flag)
		s1 := fmt.Sprintf("%s(-unique -auto -reverse)", flag)
		s2 := fmt.Sprintf("&(target.os)~%s(-unique -auto -reverse)", flag)
		s3 := fmt.Sprintf("foo~%s(-unique -auto -reverse)", flag)
		if strings.Count(usev, " "+s1) != 1 { ctx.err("%v", s1) }
		if strings.Count(usev, " "+s2) != 1 { ctx.err("%v", s2) }
		if strings.Count(uses, " "+s3) != 1 { ctx.err("%v", s3) }
		if d := ctx.def(s0); d == nil {
			ctx.err("%s", s0)
		} else if d.o != defVoid && d.o != defExpand1 && d.o != defExpand2 {
			ctx.err("%v %v", d.o, d)
		}
	}

	s := "neg1"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if ts(d.value) != "{=negative {=word foobar}}" {
		ctx.err("%v", tst{d.value})
	}
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if v.String() != "!foobar" {
		ctx.err("%v", tst{v})
	} else if s := v.string(ctx); s != "!foobar" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	} else if x, y := v.(negative); !y {
		ctx.err("%v", tst{v})
	} else if t := x.Value.true(ctx); x.true(ctx) != !t {
		ctx.err("%v : !%v != %v", tst{v}, t, x.true(ctx))
	}

	s = "neg2"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if ts(d.value) != "{=compound {=word a} {=negative {=word foobar}}}" {
		ctx.err("%v", tst{d.value})
	}
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if v.String() != "a!foobar" {
		ctx.err("%v", tst{v})
	} else if s := v.string(ctx); s != "a!foobar" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	}

	s = "neg3"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if ts(d.value) != "{=closure {=compound {=word a} {=negative {=word foobar}}}}" {
		ctx.err("%v", tst{d.value})
	}
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if v.String() != "&(a!foobar)" {
		ctx.err("%v", tst{v})
	} else if s := v.string(ctx); s != "xxx" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	}

	s = "neg4"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if ts(d.value) != "{=delegate {=builtin foreach} {=list {=delegate {=auto 1}}} {=list {=closure {=compound {=word a} {=negative {=delegate {=auto _}}}}}}}" {
		ctx.err("%v", tst{d.value})
	}
	if v := ctx.val(s, "xxx"); v == nil {
		ctx.err(s)
	} else if ts(v) != "{=cond {=closure {=compound {=word a} {=negative {=word xxx}}}}}" {
		ctx.err("%v", tst{v})
	} else if s := v.String(); s != "&(a!xxx)?" {
		ctx.err("%v : %s", tst{v}, s)
	} else if s := v.string(ctx); s != "" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	}
	if v := ctx.val(s, "foobar"); v == nil {
		ctx.err(s)
	} else if ts(v) != "{=cond {=closure {=compound {=word a} {=negative {=word foobar}}}}}" {
		ctx.err("%v", tst{v})
	} else if s := v.String(); s != "&(a!foobar)?" {
		ctx.err("%v : %s", tst{v}, s)
	} else if s := v.string(ctx); s != "xxx" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	}

	s = "cflags"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if ts(v) != "{=delegate {=def .flags} {=list {=delegate {=auto 1}}} {=list {=word c}}}" {
		ctx.err("%v", tst{v})
	} else {
		var t = v.expand(_final(ctx))

		var str1 = t.String()
		for _, s := range []string{
			"&(-v.{$1})? "+ctx.vs("-v.c"),
			"-std=&(-std.{$1})? -std=&(std.{$1})? -std=&(-std.c)? -std="+ctx.vs("std.c"),
		}{
			if strings.Count(str1, s) != 1 { ctx.err("%s : %s", s, str1) }
		}

		var str2 = ts(t)
		for _, s := range []string{
			"{=cond {=closure {=flag {=compound {=word v} {=punct .} {=disjunction {=delegate {=auto 1}}}}}}}",
			"{=cond {=pair {=flag {=word std}}={=closure {=flag {=compound {=word std} {=punct .} {=disjunction {=delegate {=auto 1}}}}}}}}",
		}{
			if strings.Count(str2, s) != 1 { ctx.err("%s : %s", s, str2) }
		}
	}
	if v := ctx.val(s, "z"); v == nil {
		ctx.err(s)
	} else {
		var str1 = v.String()
		for _, s := range []string{
			"&(-v.z)? &(-v.c)?",
			"-std=&(-std.z)? -std=&(std.z)? -std=&(-std.c)? -std=&(std.c)?",
		}{
			if strings.Count(str1, s) != 1 { ctx.err("%s : %s", s, str1) }
		}

		var str2 = ts(v)
		for _, s := range []string{
			"{=cond {=closure {=flag {=compound {=word v} {=punct .} {=word z}}}}}",
			"{=cond {=closure {=flag {=compound {=word v} {=punct .} {=word c}}}}}",
		}{
			if strings.Count(str2, s) != 1 { ctx.err("%s : %s", s, str2) }
		}

		var str3 = v.expand(_final(ctx)).String()
		for _, s := range []string{
			"&(-v.z)? "+ctx.vs("-v.c"),
			"-std=&(-std.z)? -std=&(std.z)? -std=&(-std.c)? -std="+ctx.vs("std.c"),
		}{
			if strings.Count(str3, s) != 1 { ctx.err("%s : %s", s, str3) }
		}

		var str4 = ts(v)
		for _, s := range []string{
			"{=cond {=closure {=flag {=compound {=word v} {=punct .} {=word z}}}}}",
			"{=cond {=closure {=flag {=compound {=word v} {=punct .} {=word c}}}}}",
		}{
			if strings.Count(str4, s) != 1 { ctx.err("%s : %s", s, str4) }
		}
	}

	s = "cxxflags"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if ts(v) != "{=delegate {=def .flags+} {=list {=delegate {=auto 1}}} {=list {=word c++}} {=list {=word cxx}}}" {
		ctx.err("%v", tst{v})
	} else {
		var t = v.expand(_final(ctx))

		var str1 = t.String()
		for _, s := range []string{
			"&(-v.{$1})? "+ctx.vs("-v.c++"),
			"-std=&(-std.{$1})? -std=&(std.{$1})? -std=&(-std.c++)? -std="+ctx.vs("std.c++"),
		}{
			if strings.Count(str1, s) != 1 { ctx.err("%s : %s", s, str1) }
		}

		var str2 = ts(t)
		for _, s := range []string{
			"{=cond {=closure {=flag {=compound {=word v} {=punct .} {=disjunction {=delegate {=auto 1}}}}}}}",
			"{=cond {=pair {=flag {=word std}}={=closure {=flag {=compound {=word std} {=punct .} {=disjunction {=delegate {=auto 1}}}}}}}}",
		}{
			if strings.Count(str2, s) != 1 { ctx.err("%s : %s", s, str2) }
		}
	}
	if v := ctx.val(s, "z++"); v == nil {
		ctx.err(s)
	} else {
		var str1 = v.String()
		for _, s := range []string{
			"&(-v.z++)? &(-v.c++)?",
			"-std=&(-std.z++)? -std=&(std.z++)? -std=&(-std.c++)? -std=&(std.c++)?",
		}{
			if strings.Count(str1, s) != 1 { ctx.err("%s : %s", s, str1) }
		}

		var str2 = ts(v)
		for _, s := range []string{
			"{=cond {=closure {=flag {=compound {=word v} {=punct .} {=word z++}}}}}",
			"{=cond {=closure {=flag {=compound {=word v} {=punct .} {=word c++}}}}}",
		}{
			if strings.Count(str2, s) != 1 { ctx.err("%s : %s", s, str2) }
		}

		var str3 = v.expand(_final(ctx)).String()
		for _, s := range []string{
			"&(-v.z++)? "+ctx.vs("-v.c++"),
			"-std=&(-std.z++)? -std=&(std.z++)? -std=&(-std.c++)? -std="+ctx.vs("std.c++"),
		}{
			if strings.Count(str3, s) != 1 { ctx.err("%s : %s", s, str3) }
		}

		var str4 = ts(v)
		for _, s := range []string{
			"{=cond {=closure {=flag {=compound {=word v} {=punct .} {=word z++}}}}}",
			"{=cond {=closure {=flag {=compound {=word v} {=punct .} {=word c++}}}}}",
		}{
			if strings.Count(str4, s) != 1 { ctx.err("%s : %s", s, str4) }
		}
	}
}

func testApp(ctx *testcase) {
	testVariantTargetVars(ctx)

	var p = _project(ctx)

	if p.configure != nil {
		ctx.err("%v: nil configure", p)
	}

	// cc, cancel := context.WithTimeout(context.Background(), 2400*time.Second)
	// defer cancel()

	s := ".flag"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value ; v == nil {
		ctx.err("%v", d)
	} else {
		var t = v.expand(_final(ctx))

		var str1 = t.String()
		for _, s := range []string {
			"{$(filter-out $(foreach $1,&($2!$_) &(darwin~$2!$_)),&($2) &(darwin~$2))}?",
			"{$(foreach $1,&($2.$_) &(darwin~$2.$_))}?",
		}{
			if strings.Count(str1, s) != 1 { ctx.err("%s : %s", s, str1) }
		}

		var str2 = ts(t)
		for _, s := range []string {
		}{
			if strings.Count(str2, s) != 1 { ctx.err("%s : %s", s, str2) }
		}
	}
	if v := ctx.val(s, []string{"a", "b", "c"}, "-x"); v == nil {
		ctx.err(s)
	} else {
		var str1 = v.String()
		for _, s := range []string{
			"$(or $3,-x){$(filter-out &(-x!a)? &(&(target.os)~-x!a)? &(-x!b)? &(&(target.os)~-x!b)? &(-x!c)? &(&(target.os)~-x!c)?,&(-x) &(&(target.os)~-x))}$(or $4)?",
			"$(or $3,-x){&(-x.a)}$(or $4)?",
			"$(or $3,-x){&(&(target.os)~-x.a)}$(or $4)?",
			"$(or $3,-x){&(-x.b)}$(or $4)?",
			"$(or $3,-x){&(&(target.os)~-x.b)}$(or $4)?",
			"$(or $3,-x){&(-x.c)}$(or $4)?",
			"$(or $3,-x){&(&(target.os)~-x.c)}$(or $4)?",
		}{
			if strings.Count(str1, s) != 1 { ctx.err("%s : %s", s, str1) }
		}

		var str2 = ts(v)
		for _, s := range []string{
		}{
			if strings.Count(str2, s) != 1 { ctx.err("%s : %s", s, str2) }
		}

		var str3 = v.expand(_final(ctx)).String()
		for _, s := range []string{
		}{
			if strings.Count(str3, s) != 1 { ctx.err("%s : %s", s, str3) }
		}

		var str4 = ts(v)
		for _, s := range []string{
		}{
			if strings.Count(str4, s) != 1 { ctx.err("%s : %s", s, str4) }
		}
	}

	// select {
	// case <-cc.Done():
	// }
}

func _testApp(ctx *testcase) {
	if _project(ctx).configure != nil {
		ctx.err("%v: nil configure", _project(ctx))
	}

	ss := func(s string) string { os := "darwin"
		return strings.Replace(s, "<OS>", os, -1)
	}

	flag1 := func(a ...any) string { // $(.flag $1)
		return fmt.Sprintf("$(foreach(-unique) $(filter-out $(foreach $1,&(%[1]s!$_) &(&(target.os)~%[1]s!$_)),&(%[1]s) &(&(target.os)~%[1]s)) $(foreach $1,&(%[1]s.$_) &(&(target.os)~%[1]s.$_)),$(or $3,%[1]s)$_$(or $4))", a...)
	}
	flag2 := func(a ...any) string { if len(a) > 1 { a[0], a[1] = a[1], a[0] } // $(.flag $1 yyy,xxx)
		return fmt.Sprintf("$(foreach(-unique) $(filter-out $(foreach $1 %[1]s,&(%[2]s!$_) &(&(target.os)~%[2]s!$_)),&(%[2]s) &(&(target.os)~%[2]s)) $(foreach $1 %[1]s,&(%[2]s.$_) &(&(target.os)~%[2]s.$_)),%[2]s$_$(or $4))", a...)
	}
	flag3 := func(a ...any) string { // $(.flag $1,xxx,yy)
		return fmt.Sprintf("$(foreach(-unique) $(filter-out &(%[1]s!%[2]s) &(&(target.os)~%[1]s!%[2]s),&(%[1]s) &(&(target.os)~%[1]s)) &(%[1]s.%[2]s) &(&(target.os)~%[1]s.%[2]s),$(or $3,%[1]s)$_$(or $4))", a...)
	}
	flag4 := func(a ...any) string { // $(.flag $1,xxx,y,y)
		return fmt.Sprintf("$(foreach(-unique) $(filter-out &(%[1]s!%[2]s) &(&(target.os)~%[1]s!%[2]s) &(%[1]s!c) &(&(target.os)~%[1]s!c),&(%[1]s) &(&(target.os)~%[1]s)) &(%[1]s.%[2]s) &(&(target.os)~%[1]s.%[2]s) &(%[1]s.%[3]s) &(&(target.os)~%[1]s.%[3]s),%[1]s$_$(or $4))", a...)
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
	} else if strings.Count(s, "$(filter-out $(foreach $1,&($2!$_) &(&(target.os)~$2!$_)),&($2) &(&(target.os)~$2))") != 1 {
		ctx.err("%v", d)
	} else if strings.Count(s, "$(foreach $1,&($2.$_) &(&(target.os)~$2.$_)),") != 1 {
		ctx.err("%v", d)
	} else if strings.Count(s, ",$(or $3,$2)$_$(or $4))") != 1 {
		ctx.err("%v", d)
	} else if s != flag1("$2") {
		ctx.err("%v ; %v", s, d)
	}
	if v := ctx.val(".flag", []string{"a", "b", "c"}, "-x"); v == nil {
		ctx.err(".flag")
	} else if s := v.String(); s == "" {
		ctx.err("%v", v)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out &") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x!a) &(&(target.os)~-x!a)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x!b) &(&(target.os)~-x!b)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x!c) &(&(target.os)~-x!c),") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, ",&(-x) &(&(target.os)~-x)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x.a) &(&(target.os)~-x.a)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x.b) &(&(target.os)~-x.b)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x.c) &(&(target.os)~-x.c)") != 1 {
		ctx.err("%v", v)
	} else if s1 := "$(filter-out &(-x!a) &(&(target.os)~-x!a) &(-x!b) &(&(target.os)~-x!b) &(-x!c) &(&(target.os)~-x!c),&(-x) &(&(target.os)~-x))"; strings.Count(s, s1) != 1 {
		ctx.err("%v", v)
	} else if s2 := "&(-x.a) &(&(target.os)~-x.a) &(-x.b) &(&(target.os)~-x.b) &(-x.c) &(&(target.os)~-x.c)"; strings.Count(s, s2) != 1 {
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
	if v := ctx.val(".flag", []string{"a", "b", "c"}, "-z"); v == nil {
		ctx.err(".flag")
	} else if s := v.String(); s == "" {
		ctx.err("%v", v)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out &") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-z!a) &(&(target.os)~-z!a)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-z!b) &(&(target.os)~-z!b)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-z!c) &(&(target.os)~-z!c),") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, ",&(-z) &(&(target.os)~-z)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-z.a) &(&(target.os)~-z.a)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-z.b) &(&(target.os)~-z.b)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-z.c) &(&(target.os)~-z.c)") != 1 {
		ctx.err("%v", v)
	} else if s1 := "$(filter-out &(-z!a) &(&(target.os)~-z!a) &(-z!b) &(&(target.os)~-z!b) &(-z!c) &(&(target.os)~-z!c),&(-z) &(&(target.os)~-z))"; strings.Count(s, s1) != 1 {
		ctx.err("%v", v)
	} else if s2 := "&(-z.a) &(&(target.os)~-z.a) &(-z.b) &(&(target.os)~-z.b) &(-z.c) &(&(target.os)~-z.c)"; strings.Count(s, s2) != 1 {
		ctx.err("%v", v)
	} else if s3 := "$(or $3,-z)$_$(or $4))"; strings.Count(s, s3) != 1 {
		ctx.err("%v", v)
	} else if s != "$(foreach(-unique) "+s1+" "+s2+","+s3 {
		ctx.err("%v", v)
	} else if s := v.string(ctx); s != "" {
		ctx.err("%v ; %v", s, v)
	}
	if v := ctx.val(".flag", []string{"a", "b", "c"}, "-x", "-y"); v == nil {
		ctx.err(".flag")
	} else if s := v.String(); s == "" {
		ctx.err("%v", v)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out &") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x!a) &(&(target.os)~-x!a)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x!b) &(&(target.os)~-x!b)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x!c) &(&(target.os)~-x!c),") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, ",&(-x) &(&(target.os)~-x)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x.a) &(&(target.os)~-x.a)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x.b) &(&(target.os)~-x.b)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x.c) &(&(target.os)~-x.c)") != 1 {
		ctx.err("%v", v)
	} else if s1 := "$(filter-out &(-x!a) &(&(target.os)~-x!a) &(-x!b) &(&(target.os)~-x!b) &(-x!c) &(&(target.os)~-x!c),&(-x) &(&(target.os)~-x))"; strings.Count(s, s1) != 1 {
		ctx.err("%v", v)
	} else if s2 := "&(-x.a) &(&(target.os)~-x.a) &(-x.b) &(&(target.os)~-x.b) &(-x.c) &(&(target.os)~-x.c)"; strings.Count(s, s2) != 1 {
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
	} else if d1.value == nil {
		ctx.err("%v", d1)
	} else if d2.value == nil {
		ctx.err("%v", d2)
	} else if s := d1.value.String(); true && strings.Count(s, "&(cross.target)") != 1 {
		ctx.err("%v", d1.value)
	} else if false && strings.Count(s, "$(foreach(-unique) &(target.triple),--target=$_ -Xclang -triple=$_)") != 1 {
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
	} else if s := d2.value.String(); true && strings.Count(s, "&(cross.target)") != 1 {
		ctx.err("%v", d2.value)
	} else if false && strings.Count(s, "$(foreach(-unique) &(target.triple),--target=$_ -Xclang -triple=$_)") != 1 {
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

	if v1, v2 := ctx.val("cflags"), ctx.val("xflags"); v1 == nil || v2 == nil {
		ctx.err("cflags")
		ctx.err("xflags")
	} else if s := v1.String(); s == "" {
		ctx.err("%T %v", v1, v1)
	} else if true && strings.Count(s, "&(cross.target)") != 1 {
		ctx.err("%v", v1)
	} else if false && strings.Count(s, "$(foreach(-unique) &(target.triple),--target=$_ -Xclang -triple=$_)") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(foreach(-unique) $1 c,&(-v.$_))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(foreach(-unique) $1 c,-std=&(-std.$_) -std=&(std.$_))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(if $(or &(-g) &(&(target.os)~-g) $(foreach $1 c,&(-g.$_) &(&(target.os)~-g.$_))),-g)") != 1 {
		ctx.err("%v", v1)
	} else if t := flag2("-g", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag2("-O", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag2("-D", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag2("-f", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag2("-m", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag2("-W", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag2("-I", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag2("-no", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag2("-isystem", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag2("-isystem-after", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag1("cflags"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if s := v2.String(); s == "" {
		ctx.err("%T %v", v2, v2)
	} else if true && strings.Count(s, "&(cross.target)") != 1 {
		ctx.err("%v", v2)
	} else if false && strings.Count(s, "$(foreach(-unique) &(target.triple),--target=$_ -Xclang -triple=$_)") != 1 {
		ctx.err("%v", v2)
	} else if strings.Count(s, "$(foreach(-unique) $1 c,&(-v.$_))") != 1 {
		ctx.err("%v", v2)
	} else if strings.Count(s, "$(foreach(-unique) $1 c,-std=&(-std.$_) -std=&(std.$_))") != 1 {
		ctx.err("%v", v2)
	} else if strings.Count(s, "$(if $(or &(-g) &(&(target.os)~-g) $(foreach $1 c,&(-g.$_) &(&(target.os)~-g.$_))),-g)") != 1 {
		ctx.err("%v", v2)
	} else if t := flag2("-g", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag2("-O", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag2("-D", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag2("-f", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag2("-m", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag2("-W", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag2("-I", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag2("-no", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag2("-isystem", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag2("-isystem-after", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag1("cflags"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if s1, s2 := v1.string(ctx), v2.string(ctx); s1 != s2 {
		ctx.err("%v → %s ; %v → %s", v1, s1, v2, s2)
	}

	var crossbuild bool
	if v := ctx.val("cross.build"); v == nil {
		ctx.err("cross.build")
	} else if ct := ctx.val("cross.target"); ct == nil {
		ctx.err("cross.target")
	} else if crossbuild = v.true(ctx); crossbuild {
		if strings.Count(ct.string(ctx), "-") <= 0 {
			ctx.err("cross.target: %v → %v", ct, ct.string(ctx))
		}
	}

	if v := ctx.val("std.fxxbxx"); v == nil {
		ctx.err("std.fxxbxx")
	} else if s := v.string(ctx); s != "stdfxxbxx1" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	}
	if v := ctx.val("-std.fxxbxx"); v == nil {
		ctx.err("-std.fxxbxx")
	} else if s := v.string(ctx); s != "stdfxxbxx2" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	}
	if v1, v2 := ctx.val("cflags", "fxxbxx"), ctx.val("xflags", "fxxbxx"); v1 == nil || v2 == nil {
		ctx.err("cflags")
		ctx.err("xflags")
	} else if s := v1.String(); s == "" {
		ctx.err("%v", v1)
	} else if true && strings.Count(s, "&(cross.target)") != 1 {
		ctx.err("%v", v1)
	} else if false && strings.Count(s, "$(foreach(-unique) &(target.triple),--target=$_ -Xclang -triple=$_)") != 1 {
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
	} else if strings.Count(s, "$(if $(or &(-g) &(&(target.os)~-g) &(-g.fxxbxx) &(&(target.os)~-g.fxxbxx) &(-g.c) &(&(target.os)~-g.c)),-g)") != 1 {
		ctx.err("%v", s)
	} else if t := flag4("-g", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag4("-O", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag4("-D", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag4("-f", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag4("-m", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag4("-W", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag4("-I", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag4("-no", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag4("-isystem", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag4("-isystem-after", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag3("cflags", "fxxbxx"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if s := v2.String(); s == "" {
		ctx.err("%T %v", v2, v2)
	} else if true && strings.Count(s, "&(cross.target)") != 1 {
		ctx.err("%v", v2)
	} else if false && strings.Count(s, "$(foreach(-unique) &(target.triple),--target=$_ -Xclang -triple=$_)") != 1 {
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
	} else if strings.Count(s, "$(if $(or &(-g) &(&(target.os)~-g) &(-g.fxxbxx) &(&(target.os)~-g.fxxbxx) &(-g.c) &(&(target.os)~-g.c)),-g)") != 1 {
		ctx.err("%v", s)
	} else if t := flag4("-g", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag4("-O", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag4("-D", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag4("-f", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag4("-m", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag4("-W", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag4("-I", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag4("-no", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag4("-isystem", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag4("-isystem-after", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag3("cflags", "fxxbxx"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if s1 := v1.string(ctx); s1 == "" {
		ctx.err("%v → %s", ts(v2), s1)
	} else if s2 := v2.string(ctx); s2 == "" {
		ctx.err("%v → %s", ts(v2), s2)
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

	if v := ctx.val(".test.1"); v == nil {
		ctx.err(".test.1")
	} else if s := v.String(); s == "" {
		ctx.err("%v", tst{v})
	} else if s := v.string(ctx); s == "" {
		ctx.err("%v ⇒ %s", ts(v), s)
	} else if false {
		for _, t := range foo1 {
			if n := strings.Count(s, t); n != 1 {
				ctx.err("%v : %s ; %v", t, s, ts(v))
			}
		}
	} else if strings.Count(s, "-std=fxxbar") != 1 {
		ctx.err("%v ⇒ %s", ts(v), s)
	}

	if v := ctx.val(".test.2"); v == nil {
		ctx.err(".test.2")
	} else if s := v.String(); s == "" {
		ctx.err("%v", tst{v})
	} else if s := v.string(ctx); s == "" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	} else if false {
		for _, t := range foo1 {
			if n := strings.Count(s, t); n != 1 {
				ctx.err("%v : %s ; %v", t, s, ts(v))
			}
		}
	} else if strings.Count(s, "-std=fxxbar") != 1 {
		ctx.err("%v", v)
	}

	if v := ctx.val(".test.3"); v == nil {
		ctx.err(".test.3")
	} else if s := v.String(); s == "" {
		ctx.err("%v", tst{v})
	} else if s := v.string(ctx); s == "" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	} else if false {
		for _, t := range foo1 {
			if n := strings.Count(s, t); n != 1 {
				ctx.err("%v : %s ; %v", t, s, ts(v))
			}
		}
	} else if strings.Count(s, "-std=fxxbar") != 1 {
		ctx.err("%v", v)
	}

	if v := ctx.val(".test.4"); v == nil {
		ctx.err(".test.4")
	} else if s := v.String(); s == "" {
		ctx.err("%v", tst{v})
	} else if s := v.string(ctx); s == "" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	} else {
		for _, t := range foo1 {
			if n := strings.Count(s, t); n != 1 {
				ctx.err("%v (%d) : %s ; %v", t, n, s, ts(v))
			}
		}
	}

	if v := ctx.val(".test.5"); v == nil {
		ctx.err(".test.5")
	} else if s := v.String(); s == "" {
		ctx.err("%v", tst{v})
	} else if s := v.string(ctx); s == "" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	} else {
		for _, t := range foo2 {
			if n := strings.Count(s, t); n != 1 {
				ctx.err("%v (%d) : %s ; %v", t, n, s, v)
			}
		}
	}

	if v := ctx.val(".test.6"); v == nil {
		ctx.err(".test.6")
	} else if s := v.String(); s == "" {
		ctx.err("%v", tst{v})
	} else if s := v.string(ctx); s == "" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	} else {
		for _, t := range foo3 {
			if n := strings.Count(s, t); n != 1 {
				ctx.err("%v (%d) : %s ; %v", t, n, s, ts(v))
			}
		}
	}

	if v := ctx.val(".test.7"); v == nil {
		ctx.err(".test.7")
	} else if s := v.String(); s == "" {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, flag3("ldlibs", "foo")) != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, flag3("-l", "foo")) != 1 {
		ctx.err("%v", s)
	} else if s := v.string(ctx); s == "" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	} else {
		for _, t := range foo4 {
			if n := strings.Count(s, t); n != 1 {
				ctx.err("%v (%d) : %s ; %v", t, n, s, ts(v))
			}
		}
	}

	if v := ctx.val(".test.8"); v == nil {
		ctx.err(".test.8")
	} else if s := v.String(); s == "" {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, flag3("loadlibes", "foo")) != 1 {
		ctx.err("%v", s)
	} else if s := v.string(ctx); s == "" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	} else {
		for _, t := range foo5 {
			if n := strings.Count(s, t); n != 1 {
				ctx.err("%v (%d) : %s ; %v", t, n, s, ts(v))
			}
		}
	}

	if v := ctx.val(".test.9"); v == nil {
		ctx.err(".test.9")
	} else if s := v.String(); s == "" {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, flag3("loadlibs", "foo")) != 1 {
		ctx.err("%v", s)
	} else if s := v.string(ctx); s == "" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	} else {
		for _, t := range foo6 {
			if n := strings.Count(s, t); n != 1 {
				ctx.err("%v (%d) : %s ; %v", t, n, s, ts(v))
			}
		}
	}
}

const testCxxincConfigLines = `
_LIBCPP_ABI_VERSION = '2'
_LIBCPP_ABI_NAMESPACE = '_extbit'
_LIBCPP_ABI_DEFINES = ((plain c) '//TODO: #define ...')
_LIBCPP_EXTRA_SITE_DEFINES = ((plain c) '//TODO: #define ...')
_LIBCPP_HAS_MUSL_LIBC = {=no}
_LIBCPP_HAS_PARALLEL_ALGORITHMS = {=no}
_LIBCPP_PSTL_CPU_BACKEND_SERIAL = {=no}
_LIBCPP_PSTL_CPU_BACKEND_THREAD = {=yes}
_LIBCPP_TYPEINFO_COMPARISON_IMPLEMENTATION = 1
LIBCXX_ENABLE_FILESYSTEM = {=yes}
LIBCXX_ENABLE_FSTREAM = {=yes}
LIBCXX_ENABLE_LOCALIZATION = {=yes}
LIBCXX_ENABLE_THREADS = {=yes}
LIBCXX_ENABLE_WIDE_CHARACTERS = {=yes}
requires_LIBCXX_ENABLE_WIDE_CHARACTERS =
requires_LIBCXX_ENABLE_FILESYSTEM =
requires_LIBCXX_ENABLE_THREADS =
requires_LIBCXX_ENABLE_LOCALIZATION =
requires_LIBCXX_ENABLE_FSTREAM =
`

const testCxxabiConfigLines = `
LIBCXXABI_ENABLE_NEW_DELETE_DEFINITIONS = {=yes}
LIBCXXABI_ENABLE_EXCEPTIONS = {=yes}
LIBCXXABI_ENABLE_THREADS = {=yes}
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
	testVariantTargetVars(ctx)

	var names = []string{
		".configure", "configuration.sm", "stamp", "foo.log",
		".deps/11/22/333333333333333333333333333333333333333333333333333333333333",
		".grep/11/22/333333333333333333333333333333333333333333333333333333333333",
		".cache/11/22/333333333333333333333333333333333333333333333333333333333333",
		".configure/type/align/test.c",
		".configure/type/size/test.c",
		".configure/type/test.c",
		".configure/type/xxx/test.c",
		".configure/type/xxx/yyy/test.c",
		".configure/xxx/yyy/zzz/test.c",
		".configure/xxx/yyy/zzz/test.c++",
		".configure/xxx/yyy/zzz/test.o",
		".configure/xxx/yyy/zzz/test.log",
		".configure/test_xxx.c",
		".configure/test_xxx.c++",
		".configure/test_xxx.log",
		".configure/xxx.x",
		".configure/xxx.o",
		".configure/xxx.c",
		".configure/xxx.c++",
		".configure/xxx.log",
		".configure/std/x.stdc.headers.o",
		".configure/std/x.stdc.headers.c",
		".configure/std/x.stdc.headers.c++",
		".configure/std/x.words.bigendian",
		".configure/std/x.words.bigendian.o",
		".configure/std/x.words.bigendian.c",
		".configure/std/x.words.bigendian.c++",
		".configure/std/x.float.words.bigendian.o",
		".configure/std/x.float.words.bigendian.c",
		".configure/std/x.float.words.bigendian.c++",
	}

	var p = _project(ctx)
	for _, name := range names {
		if f := p.unmap_files(ctx, name, nil); f == nil {
			ctx.err("unmap %s", name)
		}
	}

	var tail = "/arm64-darwin"
	var name, ver1, ver2, ver3 string
	var ver1_val, ver2_val, ver3_val Value
	if !strings.HasSuffix(p.absPath, tail) {
		ctx.err("%v: %v", p, p.absPath)
	}

	name = "LLVM_VERSION_MAJOR"
	if v := ctx.val(name); v == nil {
		ctx.err("%s", name)
	} else if v.String() != "$(grep {=regex ^ *set\\(LLVM_VERSION_MAJOR +([0-9]+) *\\)},$1,{=file LLVMVersion.cmake})" { // '18'
		ctx.err("%s: %v : %s", name, v, tst{v})
	} else if t := v.expand(_final(ctx)); ts(t) != "{=strlit 20}" {
		ctx.err("%s: %v : %s", name, v, tst{t})
	} else if ver1_val, ver1 = v, v.string(ctx); ver1 != "20" {
		ctx.err("%s: %v : %v", name, ver1, v)
	}

	name = "LLVM_VERSION_MINOR"
	if v := ctx.val(name); v == nil {
		ctx.err("%s", name)
	} else if v.String() != "$(grep {=regex ^ *set\\(LLVM_VERSION_MINOR +([0-9]+) *\\)},$1,{=file LLVMVersion.cmake})" { // '0'
		ctx.err("%s: %v : %s", name, v, tst{v})
	} else if t := v.expand(_final(ctx)); ts(t) != "{=strlit 0}" {
		ctx.err("%s: %v : %s", name, v, tst{t})
	} else if ver2_val, ver2 = v, v.string(ctx); ver2 != "0" {
		ctx.err("%s: %v : %v", name, ver2, v)
	}

	name = "LLVM_VERSION_PATCH"
	if v := ctx.val(name); v == nil {
		ctx.err("%s", name)
	} else if v.String() != "$(grep {=regex ^ *set\\(LLVM_VERSION_PATCH +([0-9]+) *\\)},$1,{=file LLVMVersion.cmake})" { // '0'
		ctx.err("%s: %v : %s", name, v, tst{v})
	} else if t := v.expand(_final(ctx)); ts(t) != "{=strlit 0}" {
		ctx.err("%s: %v : %s", name, v, tst{t})
	} else if ver3_val, ver3 = v, v.string(ctx); ver3 != "0" {
		ctx.err("%s: %v : %v", name, ver3, v)
	}

	var general *project
	if o := ctx.obj("general"); o == nil {
		ctx.err("general")
	} else if p, y := o.(*project); !y {
		ctx.err("%v", tst{o})
	} else {
		general = p
	}

	var proj, base *project
	if proj = _project(ctx); proj == nil {
		ctx.err("configure fail")
	} else if !strings.HasSuffix(proj.absPath, tail) {
		ctx.err("%v: %v %v", proj, proj.absPath, tail)
	} else if proj.configure != nil {
		ctx.err("%v: configure", proj)
	} else if proj.resolve(ctx, "general") != ctx.obj("general") {
		ctx.err("%v: %v != %v", proj, proj.resolve(ctx, "general"), ctx.obj("general"))
	} else if len(proj.bases) != 1 {
		ctx.err("bases: %v", proj.bases)
	} else if base = proj.bases[0]; base.name != "testllvmconfig" {
		ctx.err("base: %v", base)
	} else if proj = base; len(proj.bases) != 1 {
		ctx.err("bases: %v", base.bases)
	} else if proj.configure != nil {
		ctx.err("proj.configure")
	} else if proj.resolve(ctx, "general") != ctx.obj("general") {
		ctx.err("%v: %v != %v", proj, proj.resolve(ctx, "general"), ctx.obj("general"))
	} else if base = proj.bases[0]; base.name != "llvm.Config" {
		ctx.err("base: %v", base)
	} else if base.configure == nil {
		ctx.err("base.configure")
	} else {
		for _, name := range names {
			if f := findfile(ctx, name, base.configure); f == nil {
				ctx.err("file %s", name)
			}
		}
	}

	var ver Value
	s := "configure.version"
	if o := proj.resolve(ctx, s); o == nil {
		ctx.err("%v: %s", proj, s)
	} else if false {
		if o2 := proj.configure.resolve(ctx, s); o != o2 {
			ctx.err("%v: %v != %v", proj, o, o2)
		}
	} else if d, y := o.(*def); !y {
		ctx.err("%v", o)
	} else if ver = d.value; ver == nil {
		ctx.err("%v", o)
	} else if ver.string(ctx) != fmt.Sprintf("%v.%v.%v", ver1, ver2, ver3) {
		ctx.err("%v: %v (%v.%v.%v)", typeof(ver), ver.string(ctx), ver1, ver2, ver3)
	} else if d, y = ver.(*def); !y {
		ctx.err("%v", o)
	} else if ver = d.value; ver == nil {
		ctx.err("%v", o)
	} else if ver.String() != fmt.Sprintf("%v.%v.%v", ver1_val, ver2_val, ver3_val) {
		ctx.err("%v", ts(ver))
	}

	s = "configure.package"
	if o := proj.resolve(ctx, s); o == nil {
		ctx.err(s)
	} else if false {
		if o2 := proj.configure.resolve(ctx, s); o != o2 {
			ctx.err("%v: %v != %v", proj, o, o2)
		}
	} else if d, y := o.(*def); !y {
		ctx.err("%v", o)
	} else if d.value == nil {
		ctx.err("%v", o)
	} else if pkg := "extbit.llvm"; d.value.String() != pkg { // "&(name)"
		ctx.err("%v", ts(d.value))
	} else if r := proj.entry(ctx, _strlit(_position(ctx), "VERSION"), false); r == nil {
		ctx.err("VERSION")
	} else if len(r.programs()) != 1 {
		ctx.err("VERSION: %v", r)
	} else if recipes := r.programs()[0].recipes; len(recipes) != 1 {
		ctx.err("VERSION: %v", r)
	} else {
		recipe := recipes[0]//.expand(_final(ctx))
		verval := ver.expand(_final(ctx))
		if x, y := recipe.(*list); !y {
			ctx.err("%v", tst{recipe})
		} else if x.len() == 0 { // != 1
			ctx.err("%v %v", x.len(), ts(x.elems))
		} else if elem := x.elems[0]; elem == nil {
			ctx.err("%v", tst{elem})
		} else if x, y := elem.(*def); !y {
			ctx.err("%v", tst{elem})
		} else if v := x.value.expand(_final(ctx)); v == nil {
			ctx.err("%v", tst{x.value})
		} else if false && v.String() != verval.String() {
			ctx.err("%v: %v != %v", typeof(v), v, verval)
		}

		s = "PACKAGE"
		if r = proj.entry(ctx, _strlit(_position(ctx), s), false); r == nil {
			ctx.err(s)
		} else if len(r.programs()) != 1 {
			ctx.err("%s: %v", s, r)
		}

		s = "PACKAGE_NAME"
		if r = proj.entry(ctx, _strlit(_position(ctx), s), false); r == nil {
			ctx.err(s)
		} else if len(r.programs()) != 1 {
			ctx.err("%s: %v", s, r)
		}

		s = "PACKAGE_VERSION"
		if r = proj.entry(ctx, _strlit(_position(ctx), s), false); r == nil {
			ctx.err(s)
		} else if len(r.programs()) != 1 {
			ctx.err("%s: %v", s, r)
		}

		s = "PACKAGE_VENDOR"
		if r = proj.entry(ctx, _strlit(_position(ctx), s), false); r == nil {
			ctx.err(s)
		} else if len(r.programs()) != 1 {
			ctx.err("%s: %v", s, r)
		}

		s = "PACKAGE_TARNAME"
		if r = proj.entry(ctx, _strlit(_position(ctx), s), false); r == nil {
			ctx.err(s)
		} else if len(r.programs()) != 1 {
			ctx.err("%s: %v", s, r)
		}

		s = "PACKAGE_STRING"
		if r = proj.entry(ctx, _strlit(_position(ctx), s), false); r == nil {
			ctx.err(s)
		} else if len(r.programs()) != 1 {
			ctx.err("s: %v", s, r)
		}

		s = "PACKAGE_URL"
		if r = proj.entry(ctx, _strlit(_position(ctx), s), false); r == nil {
			ctx.err(s)
		} else if len(r.programs()) != 1 {
			ctx.err("%s: %v", s, r)
		}

		s = "PACKAGE_BUGREPORT"
		if r = proj.entry(ctx, _strlit(_position(ctx), s), false); r == nil {
			ctx.err(s)
		} else if len(r.programs()) != 1 {
			ctx.err("%s: %v", s, r)
		}
	}

	if v := ctx.val("/", proj); v == nil {
		ctx.err("/")
	} else if _, y := v.(*path); !y {
		ctx.err("%v", ts(v))
	} else if v.String() != proj.absPath {
		ctx.err("%v != %v", ts(v), proj.absPath)
	} else if v.string(ctx) != proj.absPath {
		ctx.err("%v != %v", ts(v), proj.absPath)
	}

	var outtmp string
	var outtmp_val = ctx.val("outtmp", proj)
	if v := outtmp_val; v == nil {
		ctx.err("%v", ts(v))
	} else if outtmp = v.string(/*closure_with(ctx, proj)*/ctx); outtmp == "" {
		ctx.err("%v", ts(v))
	} else if strings.HasSuffix(outtmp, tail) {
		outtmp = strings.TrimSuffix(outtmp, tail)
	}

	if !filepath.IsAbs(outtmp) {
		ctx.err("%v: %v → %v", proj, outtmp_val, outtmp)
	} else if i := strings.Index(outtmp, "/testdata/"); i <= 0 {
		ctx.err("%v: %v → %v", proj, outtmp_val, outtmp)
	} else if !strings.HasSuffix(proj.absPath, outtmp[i:]) {
		note(ctx, "%v: %v (%v, %v)", proj, proj.absPath, proj.spec, proj.rel)
		ctx.err("%v: %v → %v", proj, outtmp_val, outtmp)
	}

	s = "root1"
	if v1 := ctx.val(s); v1 == nil {
		ctx.err(s)
	} else if _, y := v1.(*path); !y {
		ctx.err("%v", ts(v1))
	} else if v1.String() != proj.absPath {
		ctx.err("%v", ts(v1))
	} else if v1.string(ctx) != proj.absPath {
		ctx.err("%v: %v", _project(ctx), tv(v1))
	} else if v2 := ctx.val("root2"); v2 == nil {
		ctx.err("root2")
	} else if _, y := v2.(*path); !y {
		ctx.err("%v", ts(v2))
	} else if v2.String() != proj.absPath {
		ctx.err("%v", ts(v2))
	} else if v2.string(ctx) != proj.absPath {
		ctx.err("%v", ts(v2))
	} else if v2.string(ctx) != v1.string(ctx) {
		ctx.err("%v: %v != %v", _project(ctx), v2, v1)
	} else if v3 := ctx.val("root3"); v3 == nil {
		ctx.err("root3")
	} else if c, y := v3.(*closure); !y {
		ctx.err("%v", tst{v3})
	} else if t := c.expand(_final(ctx)); t == nil {
		ctx.err("%v", c)
	} else if p, y := t.(*path); !y || len(p.elems) == 0 {
		ctx.err("%v: %v: %v", c, typeof(t), t)
	} else if p.elems[len(p.elems)-1].String() != filepath.Base(tail) {
		ctx.err("%v: %v → %v ; %v", proj, c, p, tail)
	} else if cc := closure_with(ctx.Context, proj); _project(cc) == nil {
		ctx.err("%v: %v != %v", c, _project(cc), proj)
	} else if t := c.expand(_final(cc)); t == nil {
		ctx.err("%v", c)
	} else if p, y := t.(*path); !y {
		ctx.err("%v: %v", c, tv(t))
	} else if len(p.elems) == 0 {
		ctx.err("%v: %v", c, p.elems)
	} else if p.elems[len(p.elems)-1].String() == filepath.Base(tail) {
		ctx.err("%v: %v → %v ; %v", proj, c, p, tail)
	} else if v3.String() != "&/" {
		ctx.err("%v", tv(v3))
	} else if v3.string(cc) != v1.string(ctx) {
		note(ctx, "%v: %v", _project(ctx), v1.string(ctx))
		note(ctx, "%v: %v", _project(ctx), v3.string(ctx))
		ctx.err("%v: %v ; %v{%v}",  _project(ctx), v3, typeof(ctx.Context), typeof(inner(ctx.Context)))
	} else if v3.string(cc) != v2.string(ctx) {
		note(ctx, "%v: %v", _project(ctx), v2.string(ctx))
		note(ctx, "%v: %v", _project(ctx), v3.string(ctx))
		ctx.err("%v: %v ; %v{%v}",  _project(ctx), v3, typeof(ctx.Context), typeof(inner(ctx.Context)))
	} else if v3.string(ctx) == v1.string(ctx) {
		note(ctx, "%v: %v", _project(ctx), v1.string(ctx))
		note(ctx, "%v: %v", _project(ctx), v3.string(ctx))
		ctx.err("%v: %v ; %v{%v}",  _project(ctx), v3, typeof(ctx.Context), typeof(inner(ctx.Context)))
	} else if v3.string(ctx) == v2.string(ctx) {
		note(ctx, "%v: %v", _project(ctx), v2.string(ctx))
		note(ctx, "%v: %v", _project(ctx), v3.string(ctx))
		ctx.err("%v: %v ; %v{%v}",  _project(ctx), v3, typeof(ctx.Context), typeof(inner(ctx.Context)))
	} else if !strings.HasSuffix(v3.string(ctx), tail) {
		ctx.err("%v: %v", tv(v3), tail)
	} else if strings.HasSuffix(v3.string(cc), tail) {
		ctx.err("%v: %v", tv(v3), tail)
	}

	var chop0, chop1 string
	var chop3 = fmt.Sprintf("%%%%/.smart/modules/ %s/ %s/ %s/",
		filepath.Dir(general.absPath),
		filepath.Dir(filepath.Dir(general.absPath)),
		filepath.Dir(filepath.Dir(filepath.Dir(general.absPath))))

	if v := ctx.val("chop0"); v == nil {
		ctx.err("chop0")
	} else if chop0 = v.string(ctx); chop0 == "" {
		ctx.err("%v", tv(v))
	}

	if v := ctx.val("chop1"); v == nil {
		ctx.err("chop1")
	} else if chop1 = v.string(ctx); chop1 == "" {
		ctx.err("%v", tv(v))
	} else if strings.HasSuffix(chop1, tail) {
		ctx.err("%v %s", tv(v), chop1)
	}

	if v := ctx.val("rel.chop"); v == nil { // from general
		ctx.err("rel.chop")
	} else if !strings.HasPrefix(v.String(), chop1) {
		ctx.err("%v", tv(v))
	} else if !strings.HasPrefix(v.string(ctx), chop1) {
		ctx.err("%v", tv(v))
	} else if !strings.HasSuffix(v.String(), chop0) {
		ctx.err("%v", tv(v))
	} else if !strings.HasSuffix(v.string(ctx), chop0) {
		ctx.err("%v", tv(v))
	} else if !strings.HasSuffix(v.String(), chop3) {
		ctx.err("%v", tv(v))
	} else if !strings.HasSuffix(v.string(ctx), chop3) {
		ctx.err("%v", tv(v))
	}

	var cc1 = closure_with(ctx.Context, base.configure, base)
	var cc2 = closure_with(ctx.Context, base, base.configure)
	if s, t := ts(cc1), "{=term llvm.Config {=term configure {=term arm64-darwin {=universe …/llvm/config/arm64-darwin}}}}"; s != t {
		ctx.err("%s != %s", s, t)
	}
	if s, t := ts(cc2), "{=term configure {=term llvm.Config {=term arm64-darwin {=universe …/llvm/config/arm64-darwin}}}}"; s != t {
		ctx.err("%s != %s", s, t)
	}

	var remnant string
	var remnant_val = ctx.val("rel.remnant")
	if v := remnant_val; v == nil {
		ctx.err("%v: rel.remnant", proj)
	} else if _, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if v.String() != "$(trim-prefix &(rel.chop),&/)" { // from general
		ctx.err("%v", tst{v})
	} else if s0 := v.string(ctx); s0 == "" {
		ctx.err("%v", tst{v})
	} else if s1 := v.string(cc1); s1 == "" {
		ctx.err("%v", tst{v})
	} else if s2 := v.string(cc2); s2 == "" {
		ctx.err("%v", tst{v})
	} else if v0 := ctx.val("remnant0"); v0 == nil {
		ctx.err("%v: remnant0", proj)
	} else if v0.string(ctx) == v.string(ctx) {
		ctx.err("%v: %v", proj,	tv(v))
	} else if v0.string(ctx) != v.string(closure_with(ctx, proj)) {
		note(ctx, "%v → %v", v, v0.string(ctx))
		note(ctx, "%v → %v", v,  v.string(closure_with(ctx, proj)))
		note(ctx, "%v → %v", v,  v.string(closure_with(ctx.Context, base)))
		note(ctx, "%v → %v", v,  v.string(closure_with(ctx.Context, base.configure)))
		ctx.err("%v: %v", proj, tv(v))
	} else if v1 := ctx.val("remnant1"); v0 == nil {
		ctx.err("%v: remnant1", proj)
	} else if v1.string(ctx) == v.string(ctx) {
		ctx.err("%v: %v", proj, tv(v))
	} else if v1.string(ctx) != v.string(closure_with(ctx, proj)) {
		note(ctx, "%v → %v", v, v1.string(ctx))
		note(ctx, "%v → %v", v,  v.string(closure_with(ctx, proj)))
		note(ctx, "%v → %v", v,  v.string(closure_with(ctx.Context, base)))
		note(ctx, "%v → %v", v,  v.string(closure_with(ctx.Context, base.configure)))
		ctx.err("%v: %v", proj, tv(v))
	} else if strings.HasSuffix(s1, tail) {
		note(ctx, "%v → %v", v, s0)
		note(ctx, "%v → %v", v, s1)
		note(ctx, "%v → %v", v, s2)
		ctx.err("%v: %v", proj, tv(v))
	} else {
		remnant = s1
	}

	if remnant == "" {
		ctx.err("%v: %v", proj, tv(remnant_val))
	}

	s = "rel.remnant"
	if d := proj.resolveDef(cc1, s); d == nil {
		ctx.err("%v: %s", proj, s)
	} else if c := base.resolveDef(cc1, s); c != d {
		ctx.err("%v : %v", tst{d}, tst{c})
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if v.String() != "$(trim-prefix &(rel.chop),&/)" { // from general
		ctx.err("%v", tst{v})
	} else if s1 := v.string(cc1); s1 == "" {
		ctx.err("%v", tst{v})
	} else if s2 := v.string(cc2); s2 == "" {
		ctx.err("%v", tst{v})
	}

	s = "val1"
	if v := ctx.val(s, proj); v == nil {
		ctx.err("%s", s)
	} else if c := base.configuration_sm(ctx); c == nil {
		ctx.err("%v", base)
	} else if s, t := ident(ctx, v), ident(ctx, c); s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if v.string(ctx) == c.fullname() {
		ctx.err("%v: %v", proj, base)
	} else if v.string(ctx) != joinpath(outtmp, configuration_sm) {
		note(ctx, "%v: %v", v, v.string(ctx))
		note(ctx, "%v: %v/%v", v, outtmp, configuration_sm)
		ctx.err("%v: different (%v)", v, proj)
	}

	s = "val2"
	if v := ctx.val(s, proj); v == nil {
		ctx.err("%s", s)
	} else if c := base.configuration_sm(ctx); c == nil {
		ctx.err("%v", base)
	} else if s, t := ident(ctx, v), ident(ctx, c); s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if f, y := v.(*file); !y {
		ctx.err("%v", tst{v})
	} else if f.fullname() == c.fullname() {
		ctx.err("%v: %v", proj, base)
	} else if f.fullname() != joinpath(outtmp, configuration_sm) {
		note(ctx, "%v: %v", v, f.fullname())
		note(ctx, "%v: %v/%v", v, outtmp, configuration_sm)
		ctx.err("%v: different (%v)", v, proj)
	}

	var srcinc string
	if v := ctx.val("srcinc"); v == nil {
		ctx.err("srcinc")
	} else if srcinc = v.string(ctx); srcinc == "" {
		ctx.err("%v", tst{v})
	}

	var outinc string
	if v := ctx.val("outinc"); v == nil {
		ctx.err("outinc")
	} else if outinc = v.string(ctx); outinc == "" {
		ctx.err("%v", tst{v})
	}

	s = "val3"
	if d := ctx.def(s+".a"); d == nil {
		ctx.err(s+".a")
	} else if v1 := d.value; v1 == nil {
		ctx.err("%v", d)
	} else if x, y := v1.(fullname); !y {
		ctx.err("%v: %v", proj.name, tst{v1})
	} else if z, y := x.Value.(*file); !y {
		ctx.err("%v: %v", proj.name, tst{x.Value})
	} else if z.dir != srcinc {
		ctx.err("%s: %s != %s", z.name, z.dir, srcinc)
	} else if d := ctx.def(s+".b"); d == nil {
		ctx.err(s+".b")
	} else if v2 := d.value; v2 == nil {
		ctx.err("%v", d)
	} else if _, y := v2.(*path); !y {
		ctx.err("%v: %v", proj.name, tst{v2})
	} else if s1, s2 := v1.string(ctx), v2.string(ctx); s1 != s2 {
		note(ctx, "%v: %s", proj.name, s1)
		note(ctx, "%v: %s", proj.name, s2)
		ctx.err("%v: %v != %v", proj.name, tst{v1}, tst{v2})
	}

	s = "val4"
	if d := ctx.def(s+".a"); d == nil {
		ctx.err(s+".a")
	} else if v1 := d.value; v1 == nil {
		ctx.err("%v", d)
	} else if x, y := v1.(fullname); !y {
		ctx.err("%v: %v", proj.name, tst{v1})
	} else if z, y := x.Value.(*file); !y {
		ctx.err("%v: %v", proj.name, tst{x.Value})
	} else if z.dir != outinc {
		ctx.err("%s: %s != %s", z.name, z.dir, srcinc)
	} else if d := ctx.def(s+".b"); d == nil {
		ctx.err(s+".b")
	} else if v2 := d.value; v2 == nil {
		ctx.err("%v", d)
	} else if _, y := v2.(*path); false && !y {
		ctx.err("%v: %v", proj.name, tst{v2})
	} else if s1, s2 := v1.string(ctx), v2.string(ctx); s1 != s2 {
		note(ctx, "%v: %s", proj.name, s1)
		note(ctx, "%v: %s", proj.name, s2)
		ctx.err("%v: %v != %v", proj.name, tst{v1}, tst{v2})
	}

	if f := proj.file(closure_with(ctx.Context, proj), configuration_sm); f == nil {
		ctx.err("%v: nil %s", proj, configuration_sm)
	} else if ident(ctx, f) != configuration_sm {
		ctx.err("%v: %v", f, base)
	} else if !filepath.IsAbs(f.fullname()) {
		ctx.err("%v: %v", f, base)
	} else if c := base.configuration_sm(ctx); c == nil {
		ctx.err("%v: %v", f, base)
	} else if !filepath.IsAbs(c.fullname()) {
		ctx.err("%v: %v", f, base)
	} else if f.fullname() == c.fullname() {
		ctx.err("%v: %v %v", f, f.fullname(), c.fullname())
	} else if x := proj.file(ctx, configuration_sm); x == nil {
		ctx.err("%v: nil %s", proj, configuration_sm)
	} else if f.fullname() == x.fullname() {
		ctx.err("%v: %v %v", f, f.fullname(), x.fullname())
	} else if f.fullname() != joinpath(outtmp, configuration_sm) {
		note(ctx, "%v: %v", f, f.fullname())
		note(ctx, "%v: %v", x, x.fullname())
		note(ctx, "%v: %v/%v", f, outtmp, configuration_sm)
		ctx.err("%v: different (%s)", f, proj.absPath)
	}

	if f := base.file(closure_with(ctx.Context, proj), configuration_sm); f == nil {
		ctx.err("%v: nil %s", proj, configuration_sm)
	} else if ident(ctx, f) != configuration_sm {
		ctx.err("%v: %v", f, base)
	} else if !filepath.IsAbs(f.fullname()) {
		ctx.err("%v: %v", f, base)
	} else if c := base.configuration_sm(ctx); c == nil {
		ctx.err("%v: %v", f, base)
	} else if !filepath.IsAbs(c.fullname()) {
		ctx.err("%v: %v", f, base)
	} else if f.fullname() == c.fullname() {
		ctx.err("%v: %v %v", f, f.fullname(), c.fullname())
	} else if x := base.file(ctx, configuration_sm); x == nil {
		ctx.err("%v: nil %s", base, configuration_sm)
	} else if f.fullname() == x.fullname() {
		ctx.err("%v: %v %v", f, f.fullname(), x.fullname())
	} else if f.fullname() != joinpath(outtmp, configuration_sm) {
		note(ctx, "%v: %v", f, f.fullname())
		note(ctx, "%v: %v/%v", f, outtmp, configuration_sm)
		ctx.err("%v: different (%s)", f, proj.absPath)
	}

	if f := proj.tempfile(closure_with(ctx.Context, proj), configuration_sm); f == nil {
		ctx.err("%v: nil %s", proj, configuration_sm)
	} else if ident(ctx, f) != configuration_sm {
		ctx.err("%v: %v", f, base)
	} else if !filepath.IsAbs(f.fullname()) {
		ctx.err("%v: %v", f, base)
	} else if c := base.configuration_sm(ctx); c == nil {
		ctx.err("%v: %v", f, base)
	} else if !filepath.IsAbs(c.fullname()) {
		ctx.err("%v: %v", f, base)
	} else if f.fullname() == c.fullname() {
		ctx.err("%v: %v %v", f, f.fullname(), c.fullname())
	} else if x := proj.tempfile(ctx, configuration_sm); x == nil {
		ctx.err("%v: nil %s", proj, configuration_sm)
	} else if f.fullname() == x.fullname() {
		ctx.err("%v: %v %v", f, f.fullname(), x.fullname())
	} else if f.fullname() != joinpath(outtmp, configuration_sm) {
		note(ctx, "%v: %v", f, f.fullname())
		note(ctx, "%v: %v/%v", f, outtmp, configuration_sm)
		ctx.err("%v: different (%s)", f, proj.absPath)
	}

	if f := base.tempfile(closure_with(ctx.Context, proj), configuration_sm); f == nil {
		ctx.err("%v: nil %s", proj, configuration_sm)
	} else if ident(ctx, f) != configuration_sm {
		ctx.err("%v: %v", f, base)
	} else if !filepath.IsAbs(f.fullname()) {
		ctx.err("%v: %v", f, base)
	} else if c := base.configuration_sm(ctx); c == nil {
		ctx.err("%v: %v", f, base)
	} else if !filepath.IsAbs(c.fullname()) {
		ctx.err("%v: %v", f, base)
	} else if f.fullname() == c.fullname() {
		ctx.err("%v: %v %v", f, f.fullname(), c.fullname())
	} else if x := base.tempfile(ctx, configuration_sm); x == nil {
		ctx.err("%v: nil %s", base, configuration_sm)
	} else if f.fullname() == x.fullname() {
		ctx.err("%v: %v %v", f, f.fullname(), x.fullname())
	} else if f.fullname() != joinpath(outtmp, configuration_sm) {
		note(ctx, "%v: %v", f, f.fullname())
		note(ctx, "%v: %v/%v", f, outtmp, configuration_sm)
		ctx.err("%v: different (%s)", f, proj.absPath)
	}

	if f := base.configuration_sm(closure_with(ctx.Context, base.configure)); f == nil {
		ctx.err("%v: %v: nil configuration", proj, base)
	} else if ident(ctx, f) != configuration_sm {
		ctx.err("%v: %v", f.name, base)
	} else if !filepath.IsAbs(f.fullname()) {
		ctx.err("%v: %v", f.name, base)
	} else if c := base.configuration_sm(ctx); c == nil {
		ctx.err("%v: %v", f.name, base)
	} else if !filepath.IsAbs(c.fullname()) {
		ctx.err("%v: %v", f.name, base)
	} else if false && f.fullname() != c.fullname() {
		note(ctx, "%v: %v", f.name, f.fullname())
		note(ctx, "%v: %v", c.name, c.fullname())
		ctx.err("%v: %v", f.name, proj)
	} else if false && f.fullname() == joinpath(outtmp, configuration_sm) {
		note(ctx, "%v: %v", f.name, f.fullname())
		note(ctx, "%v: %v/%v", f.name, outtmp, configuration_sm)
		ctx.err("%v: different", f.name)
	}

	// configure(&exec_check{unmap_uncheck_ctx{ctx},
	// 	func(_ctx Context, source string, recipe Value) {
	// 		testValidateExecRecipe(ctx, _ctx, source, recipe)
	// 	},
	// 	func(_ctx Context, line string, l int) {
	// 		testValidateExecOutput(ctx, _ctx, line, l)
	// 	},
	// })

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

	if f := base.configuration_sm(ctx); f == nil {
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
	} else if !strings.Contains(s, "FOO1 = {=yes}")  {
		ctx.err("%s", b)
	} else if true {
		note(ctx, "%v\n%s", f.fullname(), b).debug()
	}

	if o := base.configure.resolve(ctx, "outtmp"); o == nil {
		ctx.err("%v", base.configure)
	} else if outtmp, y := o.(*def); !y || outtmp.value == nil {
		ctx.err("%v", ts(o))
	} else if outtmp.value.String() != "&(target.tmp)/&(rel.remnant)" {
		ctx.err("%v", ts(outtmp.value))
	} else if f := base.configuration_sm(ctx); f == nil {
		ctx.err("configuration.sm")
	} else if s := filepath.Join(outtmp.string(cc1), configuration_sm); s != f.fullname() {
		ctx.err("%v != %v", s, f.fullname())
	} else if b, e := ioutil.ReadFile(s); e != nil {
		ctx.err("%v", e)
	} else if s := string(b); s == "" {
		ctx.err("%v", outtmp.value)
	} else if strings.Count(s, "FOO = $(.self)") != 1 {
		ctx.err("%v %s", outtmp.value, s)
	}

	if false {
		if v := ctx.val("enum1", "*AsmPrinter.cpp", "LLVM_ASM_PRINTER"); v == nil {
			ctx.err("enum1")
		} else if s := v.String(); s == "" {
			ctx.err("%v", v)
		} else if strings.Count(s, "AsmPrinter.cpp") != 1 {
			ctx.err("%v", v)
		} else if strings.Count(s, "LLVM_ASM_PRINTER") != 1 {
			ctx.err("%v", v)
		} else if true {
			note(pc(ctx,v), "%v", v).debug()
		}
	}
}

func testLLVMConfig2(ctx *testcase) {
	testVariantTargetVars(ctx)

	if false {
		if v := ctx.val("enum1", "*AsmPrinter.cpp", "LLVM_ASM_PRINTER"); v == nil {
			ctx.err("enum1")
		} else if s := v.String(); s == "" {
			ctx.err("%v", v)
		} else if strings.Count(s, "AsmPrinter.cpp") != 1 {
			ctx.err("%v", v)
		} else if strings.Count(s, "LLVM_ASM_PRINTER") != 1 {
			ctx.err("%v", v)
		} else if true {
			note(pc(ctx,v), "%v", v).debug()
		}
	}
}

func testToolchainBooting(ctx *testcase) {
	testVariantTargetVars(ctx)

	// configure(&exec_check{ctx,
	// 	func(_ctx Context, source string, recipe Value) {
	// 		testValidateExecRecipe(ctx, _ctx, source, recipe)
	// 	},
	// 	func(_ctx Context, line string, l int) {
	// 		testValidateExecOutput(ctx, _ctx, line, l)
	// 	},
	// })

	if r := ctx.rule("stamp"); r == nil {
		ctx.err("stamp")
	} else if v := _universe(ctx).run(); v != nil {
		ctx.err("%v: %v", r, v)
	}
}
