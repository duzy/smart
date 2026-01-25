//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints_value_13 = map[string]map[string]any{
	"loader.go": map[string]any{
		`g!foobar {=compound {5:2:word g} {=negative {5:4:word foobar}}}`:`g!foobar {=compound {5:2:word g} {=negative {5:4:word foobar}}}`,

		`3:5:foo g!foobar {=compound {3:11:word g} {=negative {3:13:word foobar}}}`:`g!foobar {=compound {3:11:word g} {=negative {3:13:word foobar}}}`,
		`3:5:foo &(-g!foobar) {3:8:closure {=flag {=compound {3:11:word g} {=negative {3:13:word foobar}}}}}`:`&(-g!foobar) {3:8:closure {=flag {=compound {3:11:word g} {=negative {3:13:word foobar}}}}}`,

		`5:11:-g!foobar not-foobar {=compound {5:14:word not} {=flag {5:18:word foobar}}}`:`not-foobar {=compound {5:14:word not} {=flag {5:18:word foobar}}}`,
	},
	"check-value-13_test.go": map[string]any{
		`20 3:5:foo g!foobar {=compound {3:11:word g} {=negative {3:13:word foobar}}}`:`g!foobar {=compound {3:11:word g} {=negative {3:13:word foobar}}}`,
		`20 3:5:foo &(-g!foobar) {3:8:closure {=flag {=compound {3:11:word g} {=negative {3:13:word foobar}}}}}`:`not-foobar {3:8 {=compound {5:14:word not} {=flag {5:18:word foobar}}}}`,
		`20 3:5:foo not-foobar {=compound {5:14:word not} {=flag {5:18:word foobar}}}`:`not-foobar {=compound {5:14:word not} {=flag {5:18:word foobar}}}`,
	},
}

var checkstrs_value_13 = map[string]map[string]any{
	"check-value-13_test.go": map[string]any{
		`20 3:5:foo &(-g!foobar) {3:8:closure {=flag {=compound {3:11:word g} {=negative {3:13:word foobar}}}}}`:`{3:8 {=compound {5:14:word not} {=flag {5:18:word foobar}}}} not-foobar`,
	},
}

var prefix_value_13 = map[string]map[string]string{
	"loader.go": map[string]string{
		`{3:11:word g} {=negative {3:13:word foobar}}`:`g!foobar {=compound {3:11:word g} {=negative {3:13:word foobar}}}`,
		`{=flag {3:11:word g}} {=negative {3:13:word foobar}}`:`-g!foobar {=flag {=compound {3:11:word g} {=negative {3:13:word foobar}}}}`,
		`{=flag {5:2:word g}} {=negative {5:4:word foobar}}`:`-g!foobar {=flag {=compound {5:2:word g} {=negative {5:4:word foobar}}}}`,
		`{5:2:word g} {=negative {5:4:word foobar}}`:`g!foobar {=compound {5:2:word g} {=negative {5:4:word foobar}}}`,
		`{5:14:word not} {=flag {5:18:word foobar}}`:`not-foobar {=compound {5:14:word not} {=flag {5:18:word foobar}}}`,
	},
}

var suffix_value_13 = map[string]map[string]string{
	"loader.go": map[string]string{
	},
}
