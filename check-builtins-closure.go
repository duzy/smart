//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints__closure = map[string]map[string]any{
	"loader.go": map[string]any{
		`.test {=compound {7:1:punct .} {7:2:word test}}`:`.test {=compound {7:1:punct .} {7:2:word test}}`,

		`4:7:val1 .test {=compound {4:14:punct .} {4:15:word test}}`:`.test {=compound {4:14:punct .} {4:15:word test}}`,
		`4:7:val1 &(.test) {4:12:closure {=compound {4:14:punct .} {4:15:word test}}}`:`{=compound {4:14:punct .} {4:15:word test}} &(.test) {4:12:closure {=compound {4:14:punct .} {4:15:word test}}}`,
		`4:7:val1 &(&(.test)) {4:10:closure {4:12:closure {=compound {4:14:punct .} {4:15:word test}}}}`:`{4:12:closure {=compound {4:14:punct .} {4:15:word test}}} &(&(.test)) {4:10:closure {4:12:closure {=compound {4:14:punct .} {4:15:word test}}}}`,

		`5:6:val2 .test {=compound {5:14:punct .} {5:15:word test}}`:`.test {=compound {5:14:punct .} {5:15:word test}}`,
		`5:6:val2 &(.test) {5:12:closure {=compound {5:14:punct .} {5:15:word test}}}`:`{=compound {5:14:punct .} {5:15:word test}} {} {5:12:null}`,
		`5:6:val2 &(&(.test)) {5:10:closure {5:12:closure {=compound {5:14:punct .} {5:15:word test}}}}`:`{5:12:null} {} {5:10:null}`,
	},
}

var checkstrs__closure = map[string]map[string]any{
}
