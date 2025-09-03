//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints_value_8 = map[string]map[string]any{
	"check-value-8_test.go": map[string]any{
		`16 3:10:.test $1 {3:19:delegate {3:20:auto 1}}`:`{3:10:def 1} .u {3:19 {1:9:word .u}}`,
		`16 3:10:.test $(.test$1) {3:12:delegate {=compound {3:14:punct .} {3:15:word test} {3:19:delegate {3:20:auto 1}}}}`:`{=compound {3:14:punct .} {3:15:word test} {3:19 {1:9:word .u}}} foobar {3:12 {4:12:word foobar}}`,
	},
}

var checkstrs_value_8 = map[string]map[string]any{
}
