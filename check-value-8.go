//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints_value_8 = map[string]map[string]any{
	"loader.go": map[string]any{
		`.test {=compound {3:1:punct .} {3:2:word test}}`:`.test {=compound {3:1:punct .} {3:2:word test}}`,
		`.test.u {=compound {4:1:punct .} {4:2:word test} {4:6:punct .} {4:7:word u}}`:`.test.u {=compound {4:1:punct .} {4:2:word test} {4:6:punct .} {4:7:word u}}`,
	},
	"check-value-8_test.go": map[string]any{
		`16 3:10:.test $1 {3:19:delegate {3:20:auto 1}}`:[]string{
			`{3:20:auto 1} .u {3:19 {1:9:word .u}}`,
		},
		`16 3:10:.test $(.test$1) {3:12:delegate {=compound {3:14:punct .} {3:15:word test} {3:19:delegate {3:20:auto 1}}}}`:[]string{
			`{4:9:def .test.u} foobar {3:12 {4:12:word foobar}}`,
		},
		`16 3:10:.test .test$1 {=compound {3:14:punct .} {3:15:word test} {3:19:delegate {3:20:auto 1}}}`:[]string{
			`.test.u {=compound {3:14:punct .} {3:15:word test} {3:19 {1:9:word .u}}}`,
		},
	},
}

var checkstrs_value_8 = map[string]map[string]any{
}
