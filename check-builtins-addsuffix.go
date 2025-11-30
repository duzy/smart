//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints__addsuffix = map[string]map[string]any{
	"loader.go": map[string]any{
		`.test.a {=compound {7:1:punct .} {7:2:word test} {7:6:punct .} {7:7:word a}}`:`.test.a {=compound {7:1:punct .} {7:2:word test} {7:6:punct .} {7:7:word a}}`,
		`.test.b {=compound {8:1:punct .} {8:2:word test} {8:6:punct .} {8:7:word b}}`:`.test.b {=compound {8:1:punct .} {8:2:word test} {8:6:punct .} {8:7:word b}}`,
		`.test.c {=compound {9:1:punct .} {9:2:word test} {9:6:punct .} {9:7:word c}}`:`.test.c {=compound {9:1:punct .} {9:2:word test} {9:6:punct .} {9:7:word c}}`,
	},
	"check-builtins-addsuffix_test.go": map[string]any{
		`19 3:6:val1 $(addsuffix =xxx,foo) {3:8:delegate {3:10:builtin addsuffix} {=list {=pair {3:20} {3:21:word xxx}}} {=list {3:25:word foo}}}`:`{3:10:builtin addsuffix} foo=xxx {3:8 {=pair {3:25:word foo} {3:21:word xxx}}}`,
		`21 3:6:val1 $(addsuffix =xxx,foo) {3:8:delegate {3:10:builtin addsuffix} {=list {=pair {3:20} {3:21:word xxx}}} {=list {3:25:word foo}}}`:`{3:10:builtin addsuffix} foo=xxx {3:8 {=pair {3:25:word foo} {3:21:word xxx}}}`,
		`36 4:6:val2 $(addsuffix =xxx,foo bar) {4:8:delegate {4:10:builtin addsuffix} {=list {=pair {4:20} {4:21:word xxx}}} {=list {4:25:word foo} {4:29:word bar}}}`:`{4:10:builtin addsuffix} foo=xxx bar=xxx {=list {4:8 {=pair {4:25:word foo} {4:21:word xxx}}} {4:8 {=pair {4:29:word bar} {4:21:word xxx}}}}`,
		`38 4:6:val2 $(addsuffix =xxx,foo bar) {4:8:delegate {4:10:builtin addsuffix} {=list {=pair {4:20} {4:21:word xxx}}} {=list {4:25:word foo} {4:29:word bar}}}`:`{4:10:builtin addsuffix} foo=xxx bar=xxx {=list {4:8 {=pair {4:25:word foo} {4:21:word xxx}}} {4:8 {=pair {4:29:word bar} {4:21:word xxx}}}}`,

		`53 5:6:val3 $1 {5:33:delegate {5:34:auto 1}}`:`{5:34:auto 1} {} {5:33 {5:34:null}}`,
		`53 5:6:val3 .test.$1 {=compound {5:27:punct .} {5:28:word test} {5:32:punct .} {5:33:delegate {5:34:auto 1}}}`:`.test.{} {=compound {5:27:punct .} {5:28:word test} {5:32:punct .} {5:33 {5:34:null}}}`,
		`53 5:6:val3 &(.test.$1) {5:25:closure {=compound {5:27:punct .} {5:28:word test} {5:32:punct .} {5:33:delegate {5:34:auto 1}}}}`:`{=compound {5:27:punct .} {5:28:word test} {5:32:punct .} {5:33 {5:34:null}}} {} {5:25:null}`,
		`53 5:6:val3 $(addsuffix =xxx,&(.test.$1)) {5:8:delegate {5:10:builtin addsuffix} {=list {=pair {5:20} {5:21:word xxx}}} {=list {5:25:closure {=compound {5:27:punct .} {5:28:word test} {5:32:punct .} {5:33:delegate {5:34:auto 1}}}}}}`:`{5:10:builtin addsuffix} {} {5:8 {5:10:null}}`,

		`55 5:6:val3 $1 {5:33:delegate {5:34:auto 1}}`:`{5:34:auto 1} {} {5:33 {5:34:null}}`,
		`55 5:6:val3 .test.$1 {=compound {5:27:punct .} {5:28:word test} {5:32:punct .} {5:33:delegate {5:34:auto 1}}}`:`.test.{} {=compound {5:27:punct .} {5:28:word test} {5:32:punct .} {5:33 {5:34:null}}}`,
		`55 5:6:val3 &(.test.$1) {5:25:closure {=compound {5:27:punct .} {5:28:word test} {5:32:punct .} {5:33:delegate {5:34:auto 1}}}}`:`{=compound {5:27:punct .} {5:28:word test} {5:32:punct .} {5:33 {5:34:null}}} &(.test.{}) {5:25:closure {=compound {5:27:punct .} {5:28:word test} {5:32:punct .} {5:33 {5:34:null}}}}`,
		`55 5:6:val3 $(addsuffix =xxx,&(.test.$1)) {5:8:delegate {5:10:builtin addsuffix} {=list {=pair {5:20} {5:21:word xxx}}} {=list {5:25:closure {=compound {5:27:punct .} {5:28:word test} {5:32:punct .} {5:33:delegate {5:34:auto 1}}}}}}`:`{5:10:builtin addsuffix} {&(.test.{})}=xxx {5:8 {=pair {5:25:disjunction {5:25:closure {=compound {5:27:punct .} {5:28:word test} {5:32:punct .} {5:33 {5:34:null}}}}} {5:21:word xxx}}}`,

		`59 5:6:val3 .test.{} {=compound {5:27:punct .} {5:28:word test} {5:32:punct .} {5:33 {5:34:null}}}`:`.test.{} {=compound {5:27:punct .} {5:28:word test} {5:32:punct .} {5:33 {5:34:null}}}`,
		`59 5:6:val3 &(.test.{}) {5:25:closure {=compound {5:27:punct .} {5:28:word test} {5:32:punct .} {5:33 {5:34:null}}}}`:`{=compound {5:27:punct .} {5:28:word test} {5:32:punct .} {5:33 {5:34:null}}} {} {5:25:null}`,
	},
}

var checkstrs__addsuffix = map[string]map[string]any{
	"check-builtins-addsuffix_test.go": map[string]any{
		`19 3:6:val1 $(addsuffix =xxx,foo) {3:8:delegate {3:10:builtin addsuffix} {=list {=pair {3:20} {3:21:word xxx}}} {=list {3:25:word foo}}}`:`{3:8 {=pair {3:25:word foo} {3:21:word xxx}}} foo=xxx`,
		`36 4:6:val2 $(addsuffix =xxx,foo bar) {4:8:delegate {4:10:builtin addsuffix} {=list {=pair {4:20} {4:21:word xxx}}} {=list {4:25:word foo} {4:29:word bar}}}`:`{=list {4:8 {=pair {4:25:word foo} {4:21:word xxx}}} {4:8 {=pair {4:29:word bar} {4:21:word xxx}}}} foo=xxx bar=xxx`,
		`53 5:6:val3 $(addsuffix =xxx,&(.test.$1)) {5:8:delegate {5:10:builtin addsuffix} {=list {=pair {5:20} {5:21:word xxx}}} {=list {5:25:closure {=compound {5:27:punct .} {5:28:word test} {5:32:punct .} {5:33:delegate {5:34:auto 1}}}}}}`:`{5:8 {5:10:null}} `,
	},
}

var prefix__addsuffix = map[string]map[string]string{
	"loader.go": map[string]string{
		`{5:27:punct .} {5:28:word test}`:`.test {=compound {5:27:punct .} {5:28:word test}}`,
		`{=compound {5:27:punct .} {5:28:word test}} {5:32:punct .}`:`.test. {=compound {5:27:punct .} {5:28:word test} {5:32:punct .}}`,
		`{=compound {5:27:punct .} {5:28:word test} {5:32:punct .}} {5:33:delegate {5:34:auto 1}}`:`.test.$1 {=compound {5:27:punct .} {5:28:word test} {5:32:punct .} {5:33:delegate {5:34:auto 1}}}`,

		`{7:1:punct .} {7:2:word test}`:`.test {=compound {7:1:punct .} {7:2:word test}}`,
		`{=compound {7:1:punct .} {7:2:word test}} {7:6:punct .}`:`.test. {=compound {7:1:punct .} {7:2:word test} {7:6:punct .}}`,
		`{=compound {7:1:punct .} {7:2:word test} {7:6:punct .}} {7:7:word a}`:`.test.a {=compound {7:1:punct .} {7:2:word test} {7:6:punct .} {7:7:word a}}`,

		`{8:1:punct .} {8:2:word test}`:`.test {=compound {8:1:punct .} {8:2:word test}}`,
		`{=compound {8:1:punct .} {8:2:word test}} {8:6:punct .}`:`.test. {=compound {8:1:punct .} {8:2:word test} {8:6:punct .}}`,
		`{=compound {8:1:punct .} {8:2:word test} {8:6:punct .}} {8:7:word b}`:`.test.b {=compound {8:1:punct .} {8:2:word test} {8:6:punct .} {8:7:word b}}`,

		`{9:1:punct .} {9:2:word test}`:`.test {=compound {9:1:punct .} {9:2:word test}}`,
		`{=compound {9:1:punct .} {9:2:word test}} {9:6:punct .}`:`.test. {=compound {9:1:punct .} {9:2:word test} {9:6:punct .}}`,
		`{=compound {9:1:punct .} {9:2:word test} {9:6:punct .}} {9:7:word c}`:`.test.c {=compound {9:1:punct .} {9:2:word test} {9:6:punct .} {9:7:word c}}`,
	},
	"check-builtins-addsuffix_test.go": map[string]string{
		`134 {8:28:null} {8:20:word foo}`:`foo {8:20:word foo}`,
		`19 {3:25:word foo} {=pair {3:20} {3:21:word xxx}}`:`foo=xxx {=pair {3:25:word foo} {3:21:word xxx}}`,
		`21 {3:25:word foo} {=pair {3:20} {3:21:word xxx}}`:`foo=xxx {=pair {3:25:word foo} {3:21:word xxx}}`,
		`36 {4:25:word foo} {=pair {4:20} {4:21:word xxx}}`:`foo=xxx {=pair {4:25:word foo} {4:21:word xxx}}`,
		`36 {4:29:word bar} {=pair {4:20} {4:21:word xxx}}`:`bar=xxx {=pair {4:29:word bar} {4:21:word xxx}}`,
		`38 {4:25:word foo} {=pair {4:20} {4:21:word xxx}}`:`foo=xxx {=pair {4:25:word foo} {4:21:word xxx}}`,
		`38 {4:29:word bar} {=pair {4:20} {4:21:word xxx}}`:`bar=xxx {=pair {4:29:word bar} {4:21:word xxx}}`,
		`55 {5:25:disjunction {5:25:closure {=compound {5:27:punct .} {5:28:word test} {5:32:punct .} {5:33 {5:34:null}}}}} {=pair {5:20} {5:21:word xxx}}`:`{&(.test.{})}=xxx {=pair {5:25:disjunction {5:25:closure {=compound {5:27:punct .} {5:28:word test} {5:32:punct .} {5:33 {5:34:null}}}}} {5:21:word xxx}}`,
	},
}

var suffix__addsuffix = map[string]map[string]string{
}
