//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints_modifiers = map[string]map[string]any{
	"check-modifiers_test.go": map[string]any{
		`43 4:7:val1 $(val) {4:22:delegate {3:6:def val}}`:`{3:6:def val} foobar {4:22 {3:9:word foobar}}`,
		`54 5:6:val2 $(val) {5:22:delegate {3:6:def val}}`:`{3:6:def val} foobar {5:22 {3:9:word foobar}}`,
	},
}

var checkstrs_modifiers = map[string]map[string]any{
	"check-modifiers_test.go": map[string]any{
		`43 4:7:val1 $(val) {4:22:delegate {3:6:def val}}`:`{4:22 {3:9:word foobar}} foobar`,
		`54 5:6:val2 $(val) {5:22:delegate {3:6:def val}}`:`{5:22 {3:9:word foobar}} foobar`,
	},
}
