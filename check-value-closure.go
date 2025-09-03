//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints_value_closure = map[string]map[string]any{
	"loader.go": map[string]any{
		`4:9:foo_pre &(foo.pre) {4:12:closure {3:9:def foo.pre}}`:`{3:9:def foo.pre} &(foo.pre) {4:12:closure {3:9:def foo.pre}}`,
		`5:9:foo_pos &(foo.pos) {5:12:closure {=compound {5:14:word foo} {5:17:punct .} {5:18:word pos}}}`:`{=compound {5:14:word foo} {5:17:punct .} {5:18:word pos}} &(foo.pos) {5:12:closure {=compound {5:14:word foo} {5:17:punct .} {5:18:word pos}}}`,
		`8:13:foo_nest_z &(foo.tail) {8:18:closure {=compound {8:20:word foo} {8:23:punct .} {8:24:word tail}}}`:`{=compound {8:20:word foo} {8:23:punct .} {8:24:word tail}} &(foo.tail) {8:18:closure {=compound {8:20:word foo} {8:23:punct .} {8:24:word tail}}}`,
		`8:13:foo_nest_z $(&(foo.tail)) {8:16:delegate {8:18:closure {=compound {8:20:word foo} {8:23:punct .} {8:24:word tail}}}}`:`{8:18:closure {=compound {8:20:word foo} {8:23:punct .} {8:24:word tail}}} $(&(foo.tail)) {8:16:delegate {8:18:closure {=compound {8:20:word foo} {8:23:punct .} {8:24:word tail}}}}`,
		`10:12:foo_nest_1 &(foo.tail) {10:18:closure {=compound {10:20:word foo} {10:23:punct .} {10:24:word tail}}}`:`{=compound {10:20:word foo} {10:23:punct .} {10:24:word tail}} {} {10:18:null}`,
		`10:12:foo_nest_1 &(&(foo.tail)) {10:16:closure {10:18:closure {=compound {10:20:word foo} {10:23:punct .} {10:24:word tail}}}}`:`{10:18:null} {} {10:16:null}`,
		`12:12:foo_nest_2 &(foo.tail) {12:18:closure {11:13:def foo.tail}}`:`{11:13:def foo.tail} foo.xxxx {12:18 {=compound {11:20:word foo} {11:23:punct .} {11:24:word xxxx}}}`,
		`12:12:foo_nest_2 &(&(foo.tail)) {12:16:closure {12:18:closure {11:13:def foo.tail}}}`:`{12:18 {=compound {11:20:word foo} {11:23:punct .} {11:24:word xxxx}}} {} {12:16:null}`,
		`14:12:foo_nest_3 &(foo.tail) {14:18:closure {11:13:def foo.tail}}`:`{11:13:def foo.tail} foo.xxxx {14:18 {=compound {11:20:word foo} {11:23:punct .} {11:24:word xxxx}}}`,
		`14:12:foo_nest_3 &(foo.xxxx) {14:16:closure {13:13:def foo.xxxx}}`:`{13:13:def foo.xxxx} foo {14:16 {13:20:word foo}}`,
	},
	"check-value-closure_test.go": map[string]any{
		`18 4:9:foo_pre &(foo.pre) {4:12:closure {3:9:def foo.pre}}`:`{3:9:def foo.pre} foo {4:12 {3:12:word foo}}`,
		`30 5:9:foo_pos &(foo.pos) {5:12:closure {=compound {5:14:word foo} {5:17:punct .} {5:18:word pos}}}`:`{=compound {5:14:word foo} {5:17:punct .} {5:18:word pos}} foo {5:12 {6:12:word foo}}`,
		`42 8:13:foo_nest_z &(foo.tail) {8:18:closure {=compound {8:20:word foo} {8:23:punct .} {8:24:word tail}}}`:`{=compound {8:20:word foo} {8:23:punct .} {8:24:word tail}} foo.xxxx {8:18 {=compound {11:20:word foo} {11:23:punct .} {11:24:word xxxx}}}`,
		`42 8:13:foo_nest_z $(&(foo.tail)) {8:16:delegate {8:18:closure {=compound {8:20:word foo} {8:23:punct .} {8:24:word tail}}}}`:`{8:18 {=compound {11:20:word foo} {11:23:punct .} {11:24:word xxxx}}} foo {8:16 {13:20:word foo}}`,
		`54 9:14:foo_nest_0 &(foo.tail) {9:18:closure {=compound {9:20:word foo} {9:23:punct .} {9:24:word tail}}}`:`{=compound {9:20:word foo} {9:23:punct .} {9:24:word tail}} foo.xxxx {9:18 {=compound {11:20:word foo} {11:23:punct .} {11:24:word xxxx}}}`,
		`54 9:14:foo_nest_0 $(&(foo.tail)) {9:16:delegate {9:18:closure {=compound {9:20:word foo} {9:23:punct .} {9:24:word tail}}}}`:`{9:18 {=compound {11:20:word foo} {11:23:punct .} {11:24:word xxxx}}} foo {9:16 {13:20:word foo}}`,
	},
}

var checkstrs_value_closure = map[string]map[string]any{
	"check-value-closure_test.go": map[string]any{
		`18 4:9:foo_pre &(foo.pre) {4:12:closure {3:9:def foo.pre}}`:`{4:12 {3:12:word foo}} foo`,
		`30 5:9:foo_pos &(foo.pos) {5:12:closure {=compound {5:14:word foo} {5:17:punct .} {5:18:word pos}}}`:`{5:12 {6:12:word foo}} foo`,
		`42 8:13:foo_nest_z $(&(foo.tail)) {8:16:delegate {8:18:closure {=compound {8:20:word foo} {8:23:punct .} {8:24:word tail}}}}`:`{8:16 {13:20:word foo}} foo`,
		`54 9:14:foo_nest_0 $(&(foo.tail)) {9:16:delegate {9:18:closure {=compound {9:20:word foo} {9:23:punct .} {9:24:word tail}}}}`:`{9:16 {13:20:word foo}} foo`,
	},
}
