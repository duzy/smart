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

func configure_conv_chk_darwin(ctx Context, sm, m []string) map[string]string {
	switch {
	case truly(ctx, is_autoload{ "/app/.configure" }):
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
	default:
		errostack(ctx, 5, "%s %s %s", sm[1], sm[2], m[1]).trace()
		return nil
	}
}

var rx_checking_for = regexp.MustCompile(`^checking for (header|type|function|symbol|(?:align|size) of) (.+?) …$`)
var rx_checking_res = regexp.MustCompile(`^… (.+?)\n$`)
func (l ul) configure_conv_check(ctx *execution, vt, val, res Value, a, b *diagpoint) {
	if sm := rx_checking_for.FindStringSubmatch(a.message); sm != nil {
		m := rx_checking_res.FindStringSubmatch(b.message)
		if m == nil {
			errostack(ctx, 5, "%s %s %s", sm[1], sm[2], b.message).trace()
		}

		var chk map[string]string

		chk = configure_conv_chk_darwin(ctx, sm, m)

		if x, y := chk[sm[1]+" "+sm[2]]; y {
			if m[1] != x {
				a := auto_get(ctx, "@")
				errostack(ctx, 5, "%s: %s, %s, %s != %s", a, sm[1], sm[2], m[1], x).trace()
			}
		} else {
			errostack(ctx, 5, "%s %s %s", sm[1], sm[2], m[1]).trace()
		}
	} else {
		erro(ctx, "%s %s", a.message, strings.TrimSpace(b.message)).debug()
	}
}
