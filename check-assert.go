//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints__assert = map[string]map[string]any{
	"loader.go": map[string]any{
		`0:0: $(foo) {16:16:delegate {3:5:def foo}}`:`{3:5:def foo} foo {16:16 {3:9:word foo}}`,
		`0:0: $(equal $(foo),foo) {16:8:delegate {16:10:builtin equal} {=list {16:16:delegate {3:5:def foo}}} {=list {16:23:word foo}}}`:`{16:10:builtin equal} {=true} {16:8 {16:10:true}}`,
	},
}

var checkstrs__assert = map[string]map[string]any{
}
