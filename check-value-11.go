//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints_value_11 = map[string]map[string]any{
	"loader.go": map[string]any{
		`4:10:.test.x $1 {4:20:delegate {3:21:auto 1}}`:`{} {} {4:20:null}`,
		`4:10:.test.x &(.test$1) {4:13:closure {=compound {4:15:punct .} {4:16:word test} {4:20:delegate {3:21:auto 1}}}}`:`{=compound {4:15:punct .} {4:16:word test} {4:20:null}} &(.test{}) {4:13:closure {=compound {4:15:punct .} {4:16:word test} {4:20:null}}}`,
		`5:10:.test $1 {4:20:delegate {3:21:auto 1}}`:`{} {} {4:20:null}`,
		`5:10:.test &(.test{}) {4:13:closure {=compound {4:15:punct .} {4:16:word test} {4:20:null}}}`:`{=compound {4:15:punct .} {4:16:word test} {4:20:null}} &(.test) {4:13:closure {5:10:def .test}}`,
		`5:10:.test $(.test.x) {5:13:delegate {4:10:def .test.x}}`:`{4:10:def .test.x} &(.test) {5:13 {4:13:closure {5:10:def .test}}}`,
	},
	"check-value-11_test.go": map[string]any{
		`16 3:11:.test.0 $1 {3:20:delegate {3:21:auto 1}}`:`{} {} {3:20:null}`,
		`16 3:11:.test.0 &(.test$1) {3:13:closure {=compound {3:15:punct .} {3:16:word test} {3:20:delegate {3:21:auto 1}}}}`:`{=compound {3:15:punct .} {3:16:word test} {3:20:null}} {} {3:13 {5:13 {4:13 {5:10:null}}}}`,
		`16 3:11:.test.0 &(.test) {4:13:closure {5:10:def .test}}`:`{5:10:def .test} {} {4:13 {5:10:null}}`,
		`18 3:11:.test.0 $1 {3:20:delegate {3:21:auto 1}}`:`{3:11:def 1} .v1 .v2 {=list {3:20 {1:9:word .v1}} {3:20 {1:9:word .v2}}}`,
		`18 3:11:.test.0 &(.test$1) {3:13:closure {=compound {3:15:punct .} {3:16:word test} {3:20:delegate {3:21:auto 1}}}}`:`{=list {=compound {3:15:punct .} {3:16:word test} {3:20 {1:9:word .v1}}} {=compound {3:15:punct .} {3:16:word test} {3:20 {1:9:word .v2}}}} &(.test.v1) &(.test.v2) {=list {3:13:closure {6:10:def .test.v1}} {3:13:closure {7:10:def .test.v2}}}`,
		`22 3:11:.test.0 &(.test.v1) {3:13:closure {6:10:def .test.v1}}`:`{6:10:def .test.v1} foo {3:13 {6:13:word foo}}`,
		`22 3:11:.test.0 &(.test.v2) {3:13:closure {7:10:def .test.v2}}`:`{7:10:def .test.v2} bar {3:13 {7:13:word bar}}`,
		`31 5:10:.test &(.test) {4:13:closure {5:10:def .test}}`:[]string{
			`{5:10:def .test} {} {4:13 {5:13 {4:13 {5:10:null}}}}`,
			`{5:10:def .test} {} {4:13 {5:10:null}}`,
		},
	},
}

var checkstrs_value_11 = map[string]map[string]any{
	"check-value-11_test.go": map[string]any{
		`16 3:11:.test.0 &(.test$1) {3:13:closure {=compound {3:15:punct .} {3:16:word test} {3:20:delegate {3:21:auto 1}}}}`:`{3:13 {5:13 {4:13 {5:10:null}}}} `,
		`22 3:11:.test.0 &(.test.v1) {3:13:closure {6:10:def .test.v1}}`:`{3:13 {6:13:word foo}} foo`,
		`22 3:11:.test.0 &(.test.v2) {3:13:closure {7:10:def .test.v2}}`:`{3:13 {7:13:word bar}} bar`,
		`31 5:10:.test &(.test) {4:13:closure {5:10:def .test}}`:`{4:13 {5:13 {4:13 {5:10:null}}}} `,
	},
}
