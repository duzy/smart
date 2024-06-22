///
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func (p *parser) define_check(ctx Context, tok token, ident, value Value, d **def) {
	if *d == nil {
		erro(ctx, "%v %v %v", ident, tok, ts(value)).trace()
	} else if (*d).value == nil && value != nil {
		erro(ctx, "%v %v %v", ident, tok, ts(value)).trace()
	}
}
