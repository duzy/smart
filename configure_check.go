//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func (cc *configurecontext) execute_check(ctx Context, e entry, p *project, s *string, _d **def) {
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
