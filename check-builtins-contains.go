//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints__contains = map[string]map[string]any{
	"loader.go": map[string]any{
		`.test.1 {=compound {5:1:punct .} {5:2:word test} {5:6:punct .} {5:7:decimal 1}}`:`.test.1 {=compound {5:1:punct .} {5:2:word test} {5:6:punct .} {5:7:decimal 1}}`,
		`.test.2 {=compound {6:1:punct .} {6:2:word test} {6:6:punct .} {6:7:decimal 2}}`:`.test.2 {=compound {6:1:punct .} {6:2:word test} {6:6:punct .} {6:7:decimal 2}}`,
		`.test.3 {=compound {7:1:punct .} {7:2:word test} {7:6:punct .} {7:7:decimal 3}}`:`.test.3 {=compound {7:1:punct .} {7:2:word test} {7:6:punct .} {7:7:decimal 3}}`,
		`3:5:val $1 {3:14:delegate {3:15:auto 1}}`:`{3:15:auto 1} {} {3:14 {3:15:null}}`,
		`5:9:.test.1 $(contains a,$(val)) {5:12:delegate {5:14:builtin contains} {=list {5:23:word a}} {=list {5:25:delegate {3:5:def val}}}}`:`{5:14:builtin contains} {} {5:12:null}`,
		`6:9:.test.2 $(contains x b c,$(val)) {6:12:delegate {6:14:builtin contains} {=list {6:23:word x} {6:25:word b} {6:27:word c}} {=list {6:29:delegate {3:5:def val}}}}`:`{6:14:builtin contains} {} {6:12:null}`,
		`7:9:.test.3 $(contains x,$(val)) {7:12:delegate {7:14:builtin contains} {=list {7:23:word x}} {=list {7:25:delegate {3:5:def val}}}}`:`{7:14:builtin contains} {} {7:12:null}`,
	},
}

var checkstrs__contains = map[string]map[string]any{
	"check-builtins-contains_test.go": map[string]any{
	},
}
