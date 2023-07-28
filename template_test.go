//
//  Copyright (C) 2012-2023, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	"strings"
	"testing"
)

func TestTemplate(t *testing.T) {
	var ctx = load_testcase(t, "testdata/template", "testtemplate")
	if ctx.Context == nil {
		t.Errorf("fail")
		return
	}

	if v := ctx.get(".test.1"); v == nil {
		t.Errorf(".test.1")
	} else if v.String() == "xxx yyy zzz xxx yyy zzz" {
		ctx.err("%T %v", v, v)
	}

	if v := ctx.get(".test.2"); v == nil {
		t.Errorf(".test.2")
	} else if s := v.String(); s == "" {
		ctx.err("%T %v", v, v)
	} else if strings.Count(s, "test-xxx") != 2 {
		ctx.err("%T %v", v, v)
	} else if strings.Count(s, "test-yyy") != 2 {
		ctx.err("%T %v", v, v)
	} else if strings.Count(s, "test-zzz") != 2 {
		ctx.err("%T %v", v, v)
	}

	if v := ctx.get(".test.3"); v == nil {
		t.Errorf(".test.3")
	} else if s := v.String(); s == "" {
		ctx.err("%T %v", v, v)
	} else if strings.Count(s, "test-xxx") != 2 {
		ctx.err("%T %v", v, v)
	} else if strings.Count(s, "test-yyy") != 2 {
		ctx.err("%T %v", v, v)
	} else if strings.Count(s, "test-zzz") != 2 {
		ctx.err("%T %v", v, v)
	}

	ctx.flush()
}
