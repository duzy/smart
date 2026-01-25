//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints__locals = map[string]map[string]any{
	"loader.go": map[string]any{
		`7:6:foo1 $(foo) {7:9:delegate {3:5:def foo}}`:`foobar {7:9 {3:8:word foobar}}`,
		`9:6:foo2 $(foo) {9:9:delegate {3:5:def foo}}`:`x {9:9 {8:9:word x}}`,
		`13:6:foo3 $(foo) {13:9:delegate {3:5:def foo}}`:`foobar {13:9 {3:8:word foobar}}`,
	},
}

var checkstrs__locals = map[string]map[string]any{
}
