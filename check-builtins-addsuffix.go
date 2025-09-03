//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints__addsuffix = map[string]map[string]any{
}

var checkstrs__addsuffix = map[string]map[string]any{
}

var suffix__addprefix = map[string]map[string]string{
	"check-builtins-addprefix_test.go": map[string]string{
		`134 {8:28:null} {8:20:word foo}`: `foo {8:20:word foo}`,
		`136 {8:28:null} {8:20:word foo}`: `foo {8:20:word foo}`,
		`134 {8:28:null} {8:24:word bar}`: `bar {8:24:word bar}`,
		`136 {8:28:null} {8:24:word bar}`: `bar {8:24:word bar}`,
		`151 {9:36:null} {9:20:word foo}`: `foo {9:20:word foo}`,
		`153 {9:36:null} {9:20:word foo}`: `foo {9:20:word foo}`,
		`159 {9:36:null} {9:20:word foo}`: `foo {9:20:word foo}`,
		`165 {9:36:null} {9:20:word foo}`: `foo {9:20:word foo}`,
		`180 {10:36:null} {10:20:word foo}`: `foo {10:20:word foo}`,
		`180 {10:36:null} {13:12:word test}`: `test {13:12:word test}`,
		`180 {10:36:null} {14:12:word null}`: `null {14:12:word null}`,
		`182 {10:36:null} {10:20:word foo}`: `foo {10:20:word foo}`,
		`188 {10:36:null} {10:20:word foo}`: `foo {10:20:word foo}`,
		`194 {10:36:null} {10:20:word foo}`: `foo {10:20:word foo}`,
		`151 {9:36:null} {13:12:word test}`: `test {13:12:word test}`,
		`151 {9:36:null} {14:12:word null}`: `null {14:12:word null}`,
		`165 {9:36:null} {15:12:word ax}`: `ax {15:12:word ax}`,
		`165 {9:36:null} {15:15:word ay}`: `ay {15:15:word ay}`,
		`165 {9:36:null} {15:18:word az}`: `az {15:18:word az}`,
		`165 {9:36:null} {16:12:word bx}`: `bx {16:12:word bx}`,
		`165 {9:36:null} {16:15:word by}`: `by {16:15:word by}`,
		`165 {9:36:null} {16:18:word bz}`: `bz {16:18:word bz}`,
		`194 {10:36:null} {15:12:word ax}`: `ax {15:12:word ax}`,
		`194 {10:36:null} {15:15:word ay}`: `ay {15:15:word ay}`,
		`194 {10:36:null} {15:18:word az}`: `az {15:18:word az}`,
		`194 {10:36:null} {16:12:word bx}`: `bx {16:12:word bx}`,
		`194 {10:36:null} {16:15:word by}`: `by {16:15:word by}`,
		`194 {10:36:null} {16:18:word bz}`: `bz {16:18:word bz}`,
		`153 {9:36:null} {=disjunction {9:24:closure {=compound {9:26:punct .} {9:27:word test} {9:31:punct .} {9:32:null}}}}`: `{&(.test.{})} {=disjunction {9:24:closure {=compound {9:26:punct .} {9:27:word test} {9:31:punct .} {9:32:null}}}}`,
		`159 {9:36:null} {=disjunction {9:24:closure {=compound {9:26:punct .} {9:27:word test} {9:31:punct .} {9:32 {1:9:word a}}}}}`: `{&(.test.a)} {=disjunction {9:24:closure {=compound {9:26:punct .} {9:27:word test} {9:31:punct .} {9:32 {1:9:word a}}}}}`,
		`159 {9:36:null} {=disjunction {9:24:closure {=compound {9:26:punct .} {9:27:word test} {9:31:punct .} {9:32 {1:9:word b}}}}}`: `{&(.test.b)} {=disjunction {9:24:closure {=compound {9:26:punct .} {9:27:word test} {9:31:punct .} {9:32 {1:9:word b}}}}}`,
		`182 {10:36:null} {=disjunction {10:24:closure {=compound {10:26:punct .} {10:27:word test} {10:31:punct .} {10:32:null}}}}`: `{&(.test.{})} {=disjunction {10:24:closure {=compound {10:26:punct .} {10:27:word test} {10:31:punct .} {10:32:null}}}}`,
		`188 {10:36:null} {=disjunction {10:24:closure {=compound {10:26:punct .} {10:27:word test} {10:31:punct .} {10:32 {1:9:word a}}}}}`: `{&(.test.a)} {=disjunction {10:24:closure {=compound {10:26:punct .} {10:27:word test} {10:31:punct .} {10:32 {1:9:word a}}}}}`,
		`188 {10:36:null} {=disjunction {10:24:closure {=compound {10:26:punct .} {10:27:word test} {10:31:punct .} {10:32 {1:9:word b}}}}}`: `{&(.test.b)} {=disjunction {10:24:closure {=compound {10:26:punct .} {10:27:word test} {10:31:punct .} {10:32 {1:9:word b}}}}}`,
	},
}

var suffix__addsuffix = map[string]map[string]string{
}
