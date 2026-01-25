//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints_value_9 = map[string]map[string]any{
	"loader.go": map[string]any{
		`.test.x {=compound {3:1:punct .} {3:2:word test} {3:6:punct .} {3:7:word x}}`:`.test.x {=compound {3:1:punct .} {3:2:word test} {3:6:punct .} {3:7:word x}}`,
		`.test {=compound {4:1:punct .} {4:2:word test}}`:`.test {=compound {4:1:punct .} {4:2:word test}}`,
		`.test.w {=compound {5:1:punct .} {5:2:word test} {5:6:punct .} {5:7:word w}}`:`.test.w {=compound {5:1:punct .} {5:2:word test} {5:6:punct .} {5:7:word w}}`,
	},
	"check-value-9_test.go": map[string]any{
		`16 4:9:.test $1 {3:18:delegate {3:19:auto 1}}`:`.w {3:18 {=compound {4:21:punct .} {4:22 {1:9:word w}}}}`,
		`16 4:9:.test $1 {4:22:delegate {3:19:auto 1}}`:`w {4:22 {1:9:word w}}`,
		`16 4:9:.test .$1 {=compound {4:21:punct .} {4:22:delegate {3:19:auto 1}}}`:`.w {=compound {4:21:punct .} {4:22 {1:9:word w}}}`,
		`16 4:9:.test .w {=compound {4:21:punct .} {4:22 {1:9:word w}}}`:`.w {=compound {4:21:punct .} {4:22 {1:9:word w}}}`,
		`16 4:9:.test .test$1 {=compound {3:13:punct .} {3:14:word test} {3:18:delegate {3:19:auto 1}}}`:`.test.w {=compound {3:13:punct .} {3:14:word test} {3:18 {=compound {4:21:punct .} {4:22 {1:9:word w}}}}}`,
		`16 4:9:.test $(.test$1) {3:11:delegate {=compound {3:13:punct .} {3:14:word test} {3:18:delegate {3:19:auto 1}}}}`:`foobar {3:11 {5:12:word foobar}}`,
		`16 4:9:.test $(.test.x .$1) {4:11:delegate {3:9:def .test.x} {=list {=compound {4:21:punct .} {4:22:delegate {3:19:auto 1}}}}}`:`foobar {4:11 {3:11 {5:12:word foobar}}}`,
	},
}

var checkstrs_value_9 = map[string]map[string]any{
}
