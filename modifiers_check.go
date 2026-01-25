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
			debug(ctx, "%s %s", s, fn)
		}
		if strings.HasPrefix(s, "{=file .configure/") {
			if _, e := os.Stat(fn); e != nil {
				debug(pc(ctx,fn), "%s", s, trace{})
			} else if false {
				debug(pc(ctx,fn), "%s", s)
			}
			if strings.HasPrefix(s, ".x}") {
				rx := regexp.MustCompile(`\.configure/.*?/%\..*$`)
				sm := rx.FindStringSubmatch(content)
				if sm != nil {
					debug(pc(ctx,fn), "%v", sm, trace{})
				}
			}
		}
	}
}
