//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints_value_1 = map[string]map[string]any{
	"loader.go": map[string]any{
		`1120 0:0: $(.test.foo) {4:16:delegate {3:11:def .test.foo}}`:`{3:11:def .test.foo} -{} {4:16 {=flag {3:15:null}}}`,
		`1120 0:0: $(.test.foo) {5:24:delegate {3:11:def .test.foo}}`:`{3:11:def .test.foo} -{} {5:24 {=flag {3:15:null}}}`,
		`1120 0:0: $(.test.foo) {8:16:delegate {3:11:def .test.foo}}`:`{3:11:def .test.foo} -foo {8:16 {=flag {7:15:word foo}}}`,
		`1120 0:0: $(.test.foo) {9:24:delegate {3:11:def .test.foo}}`:`{3:11:def .test.foo} -foo {9:24 {=flag {7:15:word foo}}}`,
		`1120 0:0: $(equal $(.test.foo)foobar,-foobar) {4:8:delegate {4:10:builtin equal} {=list {=compound {4:16:delegate {3:11:def .test.foo}} {4:28:word foobar}}} {=list {=flag {4:36:word foobar}}}}`:`{4:10:builtin equal} {=true} {4:8 {4:10:true}}`,
		`1120 0:0: $(equal -foobar,$(.test.foo)foobar) {5:8:delegate {5:10:builtin equal} {=list {=flag {5:17:word foobar}}} {=list {=compound {5:24:delegate {3:11:def .test.foo}} {5:36:word foobar}}}}`:`{5:10:builtin equal} {=true} {5:8 {5:10:true}}`,
		`1120 0:0: $(equal $(.test.foo)bar,-foobar) {8:8:delegate {8:10:builtin equal} {=list {=compound {8:16:delegate {3:11:def .test.foo}} {8:28:word bar}}} {=list {=flag {8:33:word foobar}}}}`:`{8:10:builtin equal} {=true} {8:8 {8:10:true}}`,
		`1120 0:0: $(equal -foobar,$(.test.foo)bar) {9:8:delegate {9:10:builtin equal} {=list {=flag {9:17:word foobar}}} {=list {=compound {9:24:delegate {3:11:def .test.foo}} {9:36:word bar}}}}`:`{9:10:builtin equal} {=true} {9:8 {9:10:true}}`,
	},
}
