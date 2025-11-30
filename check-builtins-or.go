//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints__or = map[string]map[string]any{
	"loader.go": map[string]any{
		`val11.0 {=compound {3:1:word val11} {3:6:punct .} {3:7:decimal 0}}`:`val11.0 {=compound {3:1:word val11} {3:6:punct .} {3:7:decimal 0}}`,
		`4:7:val11 $(or -no,-yes,xx) {4:10:delegate {4:12:builtin or} {=list {=flag {4:16:word no}}} {=list {=flag {4:20:word yes}}} {=list {4:24:word xx}}}`:`{4:12:builtin or} -no {4:10 {=flag {4:16:word no}}}`,
		`5:7:val12 $(or -yes,-no,xx) {5:10:delegate {5:12:builtin or} {=list {=flag {5:16:word yes}}} {=list {=flag {5:21:word no}}} {=list {5:24:word xx}}}`:`{5:12:builtin or} -yes {5:10 {=flag {5:16:word yes}}}`,
		`6:7:val13 $(or -false,-no,xx) {6:10:delegate {6:12:builtin or} {=list {=flag {6:16:word false}}} {=list {=flag {6:23:word no}}} {=list {6:26:word xx}}}`:`{6:12:builtin or} -false {6:10 {=flag {6:16:word false}}}`,
	},
	"check-builtins-or_test.go": map[string]any{
	},
}

var checkstrs__or = map[string]map[string]any{
	"check-builtins-or_test.go": map[string]any{
	},
}
