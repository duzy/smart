//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	"regexp"
)

func (cc *configurecontext) execute_check(ctx *execution, e entry, p *project, s *string, _d **def) {
	switch p.name {
	case "testdefaultconfigure":
		if d := *_d; d == nil {
			erro(ctx, "%v", e).trace()
		} else {
			switch d.name {
			case "FOO":
				if d.value.String() != "{=self testdefaultconfigure}" {
					erro(ctx, "%v", d.value).trace()
				}
			}
		}
	}
}

func configure2_chk_darwin(ctx Context, sm, m []string) map[string]string {
	return map[string]string{
		"header <atomic.h>": "no",
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
		"symbol EX_CANTCREAT": "yes",
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
	}
}

var rx_sha256 = regexp.MustCompile(`[0-9a-f]{40}`) // 24be3fc4dbc8099b28a7afa44fd7711d62a4580b
var rx_checking_for = regexp.MustCompile(`^checking for (.+?) …$`)
var rx_checking_std = regexp.MustCompile(`^checking for (header|type|function|symbol|(?:align|size) of) (.+?) …$`)
var rx_checking_res = regexp.MustCompile(`^… (.+?)\n$`)
func (l ul) configure2_check(ctx *execution, ops []entry, op, val, res Value, a, b *diagpoint) {
	if sm := rx_checking_std.FindStringSubmatch(a.message); sm != nil {
		m := rx_checking_res.FindStringSubmatch(b.message)
		if m == nil {
			errostack(ctx, 6, "%s %s : %v %v %v : %s", sm[1], sm[2], tv(op), tv(val), tv(res), b.message).trace()
		}

		var chk map[string]string

		chk = configure2_chk_darwin(ctx, sm, m)

		if x, y := chk[sm[1]+" "+sm[2]]; y {
			if m[1] != x {
				a := auto_get(ctx, "@")
				errostack(ctx, 5, "%s: %s, %s, %s != %s", a, sm[1], sm[2], m[1], x).trace()
			}
		} else {
			errostack(ctx, 5, "%s %s %s", sm[1], sm[2], m[1]).trace()
		}
	} else if sm := rx_checking_for.FindStringSubmatch(a.message); sm != nil {
		m := rx_checking_res.FindStringSubmatch(b.message)
		if m == nil {
			errostack(ctx, 6, "%s: %v, %v %v %v", sm[1], ops, ts(op), ts(val), ts(res)).trace()
		}
		switch sm[1] {
		case "llvm revision":
			if len(m[1]) != 40 || !rx_sha256.MatchString(m[1]) {
				errostack(ctx, 6, "%s: %s", sm[1], m[1]).trace()
			}
		}
	} else {
		errostack(ctx, 5, "%s, %s %s %s", ops, ts(op), ts(val), ts(res)).trace()
	}
}

func _configure2_chk(ctx Context) {
	switch {
	case truly(ctx, is_autoload{ "/app/.configure" }):
	case truly(ctx, is_autoload{ "/app/stdarg/.configure" }):
	case truly(ctx, is_autoload{ "/app/basic/.configure" }):
	case truly(ctx, is_autoload{ "/app/simple/.configure" }):
	case truly(ctx, is_autoload{ "/app/complex/.configure" }):
	case truly(ctx, is_autoload{ "/llvm/Config/.configure" }):
	}
}
