//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//
//go:build checkpoints

package smart

import (
	"regexp"
	"strings"
	"os"
)

func (ctx *modifier_updatefile) x_check(target Value, filename, content string, args []Value, result any) {
	var p = _project(ctx)
	if p.name == "configure.base" {
		var s, fn = target.String(), filename
		if strings.HasPrefix(s, ".c}") {
			note(ctx, "%s %s", s, fn).debug()
		}
		if strings.HasPrefix(s, "{=file .configure/") {
			if _, e := os.Stat(fn); e != nil {
				errostack(pc(ctx,fn), 2, "%s", s).trace()
			} else if false {
				notestack(pc(ctx,fn), 2, "%s", s).debug(3)
			}
			if strings.HasPrefix(s, ".x}") {
				rx := regexp.MustCompile(`\.configure/.*?/%\..*$`)
				sm := rx.FindStringSubmatch(content)
				if sm != nil {
					errostack(pc(ctx,fn), 2, "%v", sm).trace()
				}
			}
		}
	}
}
