//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints_value_13 = map[string]map[string]any{
	"loader.go": map[string]any{
		`3:5:foo &(-g!foobar) {3:8:closure {=flag {=compound {3:11:word g} {=negative {3:13:word foobar}}}}}`:`{=flag {=compound {3:11:word g} {=negative {3:13:word foobar}}}} &(-g!foobar) {3:8:closure {=flag {=compound {3:11:word g} {=negative {3:13:word foobar}}}}}`,
	},
	"check-value-13_test.go": map[string]any{
		`20 3:5:foo &(-g!foobar) {3:8:closure {=flag {=compound {3:11:word g} {=negative {3:13:word foobar}}}}}`:`{=flag {=compound {/Volumes/workspace/go/src/extbit.io/smart/testdata/value/13/do.smart:3:11:word g} {=negative {/Volumes/workspace/go/src/extbit.io/smart/testdata/value/13/do.smart:3:13:word foobar}}}} not-foobar {/Volumes/workspace/go/src/extbit.io/smart/testdata/value/13/do.smart:3:8 {=compound {/Volumes/workspace/go/src/extbit.io/smart/testdata/value/13/do.smart:5:14:word not} {=flag {/Volumes/workspace/go/src/extbit.io/smart/testdata/value/13/do.smart:5:18:word foobar}}}}`,
	},
}

var checkstrs_value_13 = map[string]map[string]any{
}
