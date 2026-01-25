//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints__assert = map[string]map[string]any{
	"loader.go": map[string]any{
		`$(foo) {16:16:delegate {3:5:def foo}}`:`foo {16:16 {3:9:word foo}}`,
		`$(equal $(foo),foo) {16:8:delegate {16:10:builtin equal} {=list {16:16:delegate {3:5:def foo}}} {=list {16:23:word foo}}}`:`{=true} {16:8 {16:10:true}}`,
		`foobar{} {=compound {13:8:word foobar} {13:15:null}}`:`foobar{} {=compound {13:8:word foobar} {13:15:null}}`,
	},
}

var checkstrs__assert = map[string]map[string]any{
}
