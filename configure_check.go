//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

import (
	"regexp"
)

func (cc *configurecontext) execute_check(ctx *execution, e entry, p *project, s *string, _d **def) {
	switch p.name {
	case "testdefaultconfigure":
		if d := *_d; d == nil {
			debug(ctx, "%v", e, trace{})
		} else {
			switch d.name {
			case "FOO":
				if d.value.String() != "{=self testdefaultconfigure}" {
					debug(ctx, "%v", d.value, trace{})
				}
			}
		}
	}
}

var rx_sha256 = regexp.MustCompile(`[0-9a-f]{40}`) // 24be3fc4dbc8099b28a7afa44fd7711d62a4580b
var rx_checking_for = regexp.MustCompile(`^checking for (.+?) …$`)
var rx_checking_res = regexp.MustCompile(`^… (.*?)\n$`)
func (l ul) configure_val_check(ctx *execution, name string, op Value, vals []Value, a, b *diagpoint) {
	var c Context = ctx
	var ss, _ = do(ctx, l_filename("?")).([]string)
	for _, s := range ss { c = pc(c, s) }

	if sm := rx_checking_for.FindStringSubmatch(a.message); sm != nil {
		m := rx_checking_res.FindStringSubmatch(b.message)
		if m == nil {
			errostack(ctx, 6, "%s: %v %v, %v, %v", sm[1], op, vals, l.project.elems[name], ctx.recipes, trace{})
		}

		var chk map[string]string

		chk = configure_chk_darwin(ctx, sm, m)

		if x, y := chk[sm[1]]; y {
			switch {
			case x == "?!":
				if false {
					debug(c, "%s: %s: %s", auto_get(ctx, "@"), sm[1], m[1])
				}
			case x == "?SHA256!":
				if !rx_sha256.MatchString(m[1]) {
					debug(c, "%s: %s: %s", auto_get(ctx, "@"), sm[1], m[1], trace{})
				}
			case x == "?OUTBIN!":
				if d := l.project.resolveDef(ctx, "outbin"); d == nil {
					debug(c, "%s: %s: %s", auto_get(ctx, "@"), sm[1], m[1], trace{})
				} else if s := __string(ctx, d); s != m[1] {
					debug(c, "%s: %s: %s != %s", auto_get(ctx, "@"), sm[1], m[1], s, trace{})
				}
			case x != m[1]:
				debug(c, "%s: %s: %s != %s", auto_get(ctx, "@"), sm[1], m[1], x, trace{})
			}
		} else {
			debug(c, "unknown configure check: %s %s", sm[1], m[1], trace{})
		}
	} else {
		debug(c, "%v → %v", ts(op), ts(vals), trace{})
	}
}

func configure_chk_darwin(ctx Context, sm, m []string) map[string]string {
	return map[string]string{
		"version": "?!",
		"package": "?!",
		"package name": "?!",
		"package version": "?!",
		"package vendor": "?!",
		"package tarname": "?!",
		"package string": "?!",
		"package url": "?!",
		"package bug report": "?!",

		"c++abi new/delete definitions": "yes",
		"c++abi exceptions": "yes",
		"c++abi threads": "yes",

		"libc++ ABI namespace": "_extbit",
		"libc++ ABI version": "2",
		"libc++ ABI defines": "// TODO: #define ...\\n",
		"libc++ extra site defines": "// TODO: #define ...\\n",
		"libc++ filesystem": "yes",
		"libc++ fstream": "yes",
		"libc++ localization support": "yes",
		"libc++ threads support": "yes",
		"libc++ wide characters": "yes",
		"typeinfo comparison implementation": "1",
		"parallel algorithms": "no",
		"pstl cpu backend serial": "no",
		"pstl cpu backend thread": "yes",
		"musl libc": "no",

		"llvm revision": "?SHA256!",
		"llvm version": "20.0.0",
		"llvm version suffix": "extbit",
		"llvm version string": "ExtBit_20.0.0",
		"llvm version information": "ExtBit LLVM",
		"llvm native arch": "AArch64",
		"llvm with polly": "yes",
		"llvm libdir suffix": "",

		"unix platform (darwin)": "yes",
		"host arch": "arm64",
		"host triple": "arm64-apple-Darwin24.3.0-extbit",
		"default target triple": "arm64-apple-Darwin24.3.0-extbit",
		"default target environment variable name": "",
		"all build targets": "AArch64 AMDGPU ARM BPF Hexagon Lanai Mips MSP430 NVPTX PowerPC Sparc SystemZ WebAssembly X86 XCore",
		"targets with jit support": "X86 PowerPC AArch64 ARM Mips SystemZ",
		"targets to build": "AArch64",
		"target to use for llvm jit": "host",
		"experimental targets to build": "",
		"external polly source directory": "",
		"statically link polly into tools": "yes",
		"link polly into tools": "yes",
		"tools/polly directory": "no",
		"build with polly": "yes",
		"exceptions": "yes",
		"DAGiSel COV": "no",
		"curl enabled": "yes",
		"collection of GlobalISel rule coverage": "no",
		"embedding backtraces": "yes",
		"embedding backtraces on crash": "yes",
		"memory dumps on crashes": "yes",
		"interpreter external function call with libffi": "yes",
		"httplib enabled": "yes",
		"DIA SDK enabled": "no",
		"dump functions even when assertions are disabled": "yes",
		"stats enabled": "yes",
		"terminfo database enabled if available": "yes",
		"threads enabled if available": "yes",
		"libxar enabled if available": "yes",
		"libxml2 enabled if available": "yes",
		"libedit enabled if available": "yes",
		"libpfm enabled for performance counters if available": "yes",
		"zlib enabled for compression/decompression": "yes",
		"z3 constraint solver is supported in LLVM": "no",
		"z3 resolver install directory": "",
		"z3 enabled": "no",
		"forced using stats": "no",
		"forced using old toolchain": "no",
		"GISEL_COV enabled": "no",
		"GISEL_COV prefix": "",
		"c headers": "yes",
		"c99 headers": "yes",
		"c11 headers": "no",
		"c99 variadic macros": "yes",
		"gcc variadic macros": "yes",
		"va_list": "yes",
		"va_copy": "yes",
		"std::atomic": "yes",
		"std::atomic and <cstdint>": "yes",
		"atomic primitives in <stdatomic.h>": "yes",
		"intel atomic primitives": "yes",
		"solaris atomic operations": "",
		"builtin atomic": "yes",
		"pthread in libc": "yes",
		"perf_branch_entry.cycles in libpfm": "no",
		"show target and host info when tools are invoked with --version": "yes",

		"using libxml2": "",
		"using JITEvents (Intel)": "",
		"using oprofile": "",
		"using perf": "",

		"ffi library directory": "",
		"ffi include directory": "",
		"backtraces enabled": "yes",
		"crash overrides enabled": "yes",

		"host linker version": "",
		"return sig type": "void",
		"tools install directory": "?OUTBIN!",
		"utils install directory": "?OUTBIN!",
		"llvm plugin extension": ".extbit",
		"shared library extension": ".dylib",
		"abi breaking checks enabled": "no",
		"reverse iteration enabled": "no",

		"tensorflow api support": "no",
		"tensorflow aot support": "no",

		"strdup": "strdup",
		"stricmp": "stricmp",

		"header <atomic.h>": "no",
		"header <mbarrier.h>": "no",
		"header <dirent.h>": "yes",
		"header <stdarg.h>": "yes",
		"header <stdatomic.h>": "yes",
		"header <stdlib.h>": "yes",
		"header <sysexits.h>": "yes",
		"header <xar/xar.h>": "yes",
		"library xar": "yes",
		"type atomic_bool": "yes",
		"type atomic_char": "yes",
		"type atomic_flag": "yes",
		"type atomic_int": "yes",
		"type atomic_intmax_t": "yes",
		"type atomic_intptr_t": "yes",
		"type atomic_llong": "yes",
		"type atomic_long": "yes",
		"type atomic_ptrdiff_t": "yes",
		"type atomic_schar": "yes",
		"type atomic_short": "yes",
		"type atomic_size_t": "yes",
		"type atomic_uchar": "yes",
		"type atomic_uint": "yes",
		"type atomic_uintmax_t": "yes",
		"type atomic_uintptr_t": "yes",
		"type atomic_ullong": "yes",
		"type atomic_ulong": "yes",
		"type atomic_ushort": "yes",
		"type struct dirent": "yes",
		"type va_list": "yes",
		"align of va_list": "8",
		"size of va_list": "8",
		"function abort": "yes",
		"function atexit": "yes",
		"function calloc": "yes",
		"function clearenv": "no",
		"function exit": "yes",
		"function free": "yes",
		"function getenv": "yes",
		"function malloc": "yes",
		"function putenv": "yes",
		"function realloc": "yes",
		"function secure_getenv": "no",
		"function setenv": "yes",
		"function unsetenv": "yes",
		"function va_arg": "no",
		"function va_copy": "no",
		"function va_end": "no",
		"function va_start": "no",
		"function valloc": "yes",
		"function xar_add": "yes",
		"function xar_attr_get": "yes",
		"function xar_attr_set": "yes",
		"function xar_close": "yes",
		"function xar_create": "no",
		"function xar_delete": "no",
		"function xar_extract": "yes",
		"function xar_list": "no",
		"function xar_open": "yes",
		"symbol EX_CANNOTCREAT": "yes",
		"symbol EX_CONFIG": "yes",
		"symbol EX_DATAERR": "yes",
		"symbol EX_IOERR": "yes",
		"symbol EX_NOHOST": "yes",
		"symbol EX_NOINPUT": "yes",
		"symbol EX_NOPERM": "yes",
		"symbol EX_NOUSER": "yes",
		"symbol EX_OK": "yes",
		"symbol EX_OSERR": "yes",
		"symbol EX_OSFILE": "yes",
		"symbol EX_PROTOCOL": "yes",
		"symbol EX_SOFTWARE": "yes",
		"symbol EX_TEMPFAIL": "yes",
		"symbol EX_UNAVAILABLE": "yes",
		"symbol EX_USAGE": "yes",
		"symbol va_arg": "yes",
		"symbol va_copy": "yes",
		"symbol va_end": "yes",
		"symbol va_start": "yes",
		"(struct dirent)": "yes",
		"(struct dirent).d_type": "yes",

		"FOO1": "yes",
		"FOO2": "true",
		"FOO3": "true",
		"FOO4": "true",
	}
}

func _configure_chk(ctx Context) {
	switch {
	case truly(ctx, is_autoload{ "/app/.configure" }):
	case truly(ctx, is_autoload{ "/app/stdarg/.configure" }):
	case truly(ctx, is_autoload{ "/app/basic/.configure" }):
	case truly(ctx, is_autoload{ "/app/simple/.configure" }):
	case truly(ctx, is_autoload{ "/app/complex/.configure" }):
	case truly(ctx, is_autoload{ "/llvm/Config/.configure" }):
	}
}
