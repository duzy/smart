//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	"regexp"
	"strings"
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
	switch {
	case truly(ctx, is_autoload{ "/app/.configure" }):
		return map[string]string{
			"header <stdlib.h>": "yes",
			"function exit": "yes",
		}
	case truly(ctx, is_autoload{ "/app/stdarg/.configure" }):
		return map[string]string{
			"header <stdarg.h>": "yes",
			"type va_list": "yes",
			"size of va_list": "8",
			"align of va_list": "8",
			"function va_arg": "no",
			"function va_copy": "no",
			"function va_start": "no",
			"function va_end": "no",
			"symbol va_arg": "yes",
			"symbol va_copy": "yes",
			"symbol va_start": "yes",
			"symbol va_end": "yes",
		}
	case truly(ctx, is_autoload{ "/app/basic/.configure" }):
		errostack(ctx, 5, "%s %s %s", sm[1], sm[2], m[1]).trace()
		return map[string]string{
			"header <stdarg.h>": "yes",
			"type va_list": "yes",
			"size of va_list": "8",
			"align of va_list": "8",
			"function va_arg": "no",
			"function va_copy": "no",
			"function va_start": "no",
			"function va_end": "no",
			"symbol va_arg": "yes",
			"symbol va_copy": "yes",
			"symbol va_start": "yes",
			"symbol va_end": "yes",
		}
	case truly(ctx, is_autoload{ "/app/simple/.configure" }):
		errostack(ctx, 5, "%s %s %s", sm[1], sm[2], m[1]).trace()
		return map[string]string{
			"header <stdarg.h>": "yes",
			"type va_list": "yes",
			"size of va_list": "8",
			"align of va_list": "8",
			"function va_arg": "no",
			"function va_copy": "no",
			"function va_start": "no",
			"function va_end": "no",
			"symbol va_arg": "yes",
			"symbol va_copy": "yes",
			"symbol va_start": "yes",
			"symbol va_end": "yes",
		}
	case truly(ctx, is_autoload{ "/app/complex/.configure" }):
		errostack(ctx, 5, "%s %s %s", sm[1], sm[2], m[1]).trace()
		return map[string]string{
			"header <stdarg.h>": "yes",
			"type va_list": "yes",
			"size of va_list": "8",
			"align of va_list": "8",
			"function va_arg": "no",
			"function va_copy": "no",
			"function va_start": "no",
			"function va_end": "no",
			"symbol va_arg": "yes",
			"symbol va_copy": "yes",
			"symbol va_start": "yes",
			"symbol va_end": "yes",
		}
	case truly(ctx, is_autoload{ "/llvm/Config/.configure" }):
		return map[string]string{
			"header <sysexits.h>": "yes",
			"header <stdlib.h>": "yes",
			"function abort": "yes",
			"function atexit": "yes",
			"function exit": "yes",
			"function calloc": "yes",
			"function malloc": "yes",
			"function valloc": "yes",
			"function realloc": "yes",
			"function free": "yes",
			"function getenv": "yes",
			"function putenv": "yes",
			"function setenv": "yes",
			"function unsetenv": "yes",
			"function clearenv": "no",
			"function secure_getenv": "no",
			"symbol EX_OK": "yes",
			"symbol EX_USAGE": "yes",
			"symbol EX_DATAERR": "yes",
			"symbol EX_NOINPUT": "yes",
			"symbol EX_NOUSER": "yes",
			"symbol EX_NOHOST": "yes",
			"symbol EX_UNAVAILABLE": "yes",
			"symbol EX_SOFTWARE": "yes",
			"symbol EX_OSERR": "yes",
			"symbol EX_OSFILE": "yes",
			"symbol EX_CANTCREAT": "yes",
			"symbol EX_IOERR": "yes",
			"symbol EX_TEMPFAIL": "yes",
			"symbol EX_PROTOCOL": "yes",
			"symbol EX_NOPERM": "yes",
			"symbol EX_CONFIG": "yes",
		}
	default:
		errostack(ctx, 5, "%s %s %s", sm[1], sm[2], m[1]).trace()
		return nil
	}
}

var rx_checking_for = regexp.MustCompile(`^checking for (.+?) …$`)
var rx_checking_std = regexp.MustCompile(`^checking for (header|type|function|symbol|(?:align|size) of) (.+?) …$`)
var rx_checking_res = regexp.MustCompile(`^… (.+?)\n$`)
func (l ul) configure2_check(ctx *execution, op, val, res Value, a, b *diagpoint) {
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
			errostack(ctx, 6, "%s %s : %v %v %v : %s", sm[1], sm[2], tv(op), tv(val), tv(res), b.message).trace()
		}
		switch sm[1] {
		case "llvm revision":
			if len(m[1]) != 40 {
				errostack(ctx, 6, "%s: %s", sm[1], m[1]).trace()
			}
		}
	} else {
		errostack(ctx, 5, "%s | %s", a.message, strings.TrimSpace(b.message)).trace()
	}
}
