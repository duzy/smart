//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints_value_12 = map[string]map[string]any{
	"loader.go": map[string]any{
		`.test.y {=compound {3:1:punct .} {3:2:word test} {3:6:punct .} {3:7:word y}}`:`.test.y {=compound {3:1:punct .} {3:2:word test} {3:6:punct .} {3:7:word y}}`,
		`.test.x {=compound {4:1:punct .} {4:2:word test} {4:6:punct .} {4:7:word x}}`:`.test.x {=compound {4:1:punct .} {4:2:word test} {4:6:punct .} {4:7:word x}}`,
		`.test.0 {=compound {6:1:punct .} {6:2:word test} {6:6:punct .} {6:7:decimal 0}}`:`.test.0 {=compound {6:1:punct .} {6:2:word test} {6:6:punct .} {6:7:decimal 0}}`,
		`.test.1 {=compound {7:1:punct .} {7:2:word test} {7:6:punct .} {7:7:decimal 1}}`:`.test.1 {=compound {7:1:punct .} {7:2:word test} {7:6:punct .} {7:7:decimal 1}}`,
		`.test.w {=compound {9:1:punct .} {9:2:word test} {9:6:punct .} {9:7:word w}}`:`.test.w {=compound {9:1:punct .} {9:2:word test} {9:6:punct .} {9:7:word w}}`,
		`.test. {=compound {10:1:punct .} {10:2:word test} {10:6:punct .}}`:`.test. {=compound {10:1:punct .} {10:2:word test} {10:6:punct .}}`,

		`7:9:.test.1 $1 {3:18:delegate {3:19:auto 1}}`:[]string{`.{} {3:18 {=compound {4:21:punct .} {4:22 {7:22 {3:19:null}}}}}`},
		`7:9:.test.1 $1 {4:22:delegate {3:19:auto 1}}`:[]string{`{} {4:22 {7:22 {3:19:null}}}`},
		`7:9:.test.1 $1 {7:22:delegate {3:19:auto 1}}`:[]string{`{} {7:22 {3:19:null}}`},
		`7:9:.test.1 .$1 {=compound {4:21:punct .} {4:22:delegate {3:19:auto 1}}}`:`.{} {=compound {4:21:punct .} {4:22 {7:22 {3:19:null}}}}`,
		`7:9:.test.1 .{} {=compound {4:21:punct .} {4:22 {7:22 {3:19:null}}}}`:`.{} {=compound {4:21:punct .} {4:22 {7:22 {3:19:null}}}}`,
		`7:9:.test.1 .test$1 {=compound {3:13:punct .} {3:14:word test} {3:18:delegate {3:19:auto 1}}}`:`.test.{} {=compound {3:13:punct .} {3:14:word test} {3:18 {=compound {4:21:punct .} {4:22 {7:22 {3:19:null}}}}}}`,
		`7:9:.test.1 &(.test$1) {3:11:closure {=compound {3:13:punct .} {3:14:word test} {3:18:delegate {3:19:auto 1}}}}`:[]string{
			`&(.test.{}) {3:11:closure {=compound {3:13:punct .} {3:14:word test} {3:18 {=compound {4:21:punct .} {4:22 {7:22 {3:19:null}}}}}}}`,
		},
		`7:9:.test.1 $(.test.x $1) {7:12:delegate {4:9:def .test.x} {=list {7:22:delegate {3:19:auto 1}}}}`:[]string{
			`&(.test.{}) {7:12 {4:11 {3:11:closure {=compound {3:13:punct .} {3:14:word test} {3:18 {=compound {4:21:punct .} {4:22 {7:22 {3:19:null}}}}}}}}}`,
		},
		`7:9:.test.1 $(.test.y .$1) {4:11:delegate {3:9:def .test.y} {=list {=compound {4:21:punct .} {4:22:delegate {3:19:auto 1}}}}}`:[]string{
			`&(.test.{}) {4:11 {3:11:closure {=compound {3:13:punct .} {3:14:word test} {3:18 {=compound {4:21:punct .} {4:22 {7:22 {3:19:null}}}}}}}}`,
		},
	},
	"check-value-12_test.go": map[string]any{
		`16 6:10:.test.0 $1 {6:22:delegate {3:19:auto 1}}`:`w {6:22 {1:9:word w}}`,
		`16 6:10:.test.0 $1 {4:22:delegate {3:19:auto 1}}`:`w {4:22 {6:22 {1:9:word w}}}`,
		`16 6:10:.test.0 $1 {3:18:delegate {3:19:auto 1}}`:`.w {3:18 {=compound {4:21:punct .} {4:22 {6:22 {1:9:word w}}}}}`,
		`16 6:10:.test.0 .$1 {=compound {4:21:punct .} {4:22:delegate {3:19:auto 1}}}`:`.w {=compound {4:21:punct .} {4:22 {6:22 {1:9:word w}}}}`,
		`16 6:10:.test.0 .w {=compound {4:21:punct .} {4:22 {6:22 {1:9:word w}}}}`:`.w {=compound {4:21:punct .} {4:22 {6:22 {1:9:word w}}}}`,
		`16 6:10:.test.0 .test$1 {=compound {3:13:punct .} {3:14:word test} {3:18:delegate {3:19:auto 1}}}`:`.test.w {=compound {3:13:punct .} {3:14:word test} {3:18 {=compound {4:21:punct .} {4:22 {6:22 {1:9:word w}}}}}}`,
		`16 6:10:.test.0 &(.test$1) {3:11:closure {=compound {3:13:punct .} {3:14:word test} {3:18:delegate {3:19:auto 1}}}}`:`&(.test.w) {3:11:closure {9:9:def .test.w}}`,
		`16 6:10:.test.0 $(.test.x $1) {6:12:delegate {4:9:def .test.x} {=list {6:22:delegate {3:19:auto 1}}}}`:`&(.test.w) {6:12 {4:11 {3:11:closure {9:9:def .test.w}}}}`,
		`16 6:10:.test.0 $(.test.y .$1) {4:11:delegate {3:9:def .test.y} {=list {=compound {4:21:punct .} {4:22:delegate {3:19:auto 1}}}}}`:`&(.test.w) {4:11 {3:11:closure {9:9:def .test.w}}}`,

		`20 6:10:.test.0 &(.test.w) {3:11:closure {9:9:def .test.w}}`:`foobaz {3:11 {9:12:word foobaz}}`,

		`22 6:10:.test.0 $1 {6:22:delegate {3:19:auto 1}}`:`w {6:22 {1:9:word w}}`,
		`22 6:10:.test.0 $1 {4:22:delegate {3:19:auto 1}}`:`w {4:22 {6:22 {1:9:word w}}}`,
		`22 6:10:.test.0 $1 {3:18:delegate {3:19:auto 1}}`:`.w {3:18 {=compound {4:21:punct .} {4:22 {6:22 {1:9:word w}}}}}`,
		`22 6:10:.test.0 .$1 {=compound {4:21:punct .} {4:22:delegate {3:19:auto 1}}}`:`.w {=compound {4:21:punct .} {4:22 {6:22 {1:9:word w}}}}`,
		`22 6:10:.test.0 .w {=compound {4:21:punct .} {4:22 {6:22 {1:9:word w}}}}`:`.w {=compound {4:21:punct .} {4:22 {6:22 {1:9:word w}}}}`,
		`22 6:10:.test.0 .test$1 {=compound {3:13:punct .} {3:14:word test} {3:18:delegate {3:19:auto 1}}}`:`.test.w {=compound {3:13:punct .} {3:14:word test} {3:18 {=compound {4:21:punct .} {4:22 {6:22 {1:9:word w}}}}}}`,
		`22 6:10:.test.0 &(.test$1) {3:11:closure {=compound {3:13:punct .} {3:14:word test} {3:18:delegate {3:19:auto 1}}}}`:`foobaz {3:11 {9:12:word foobaz}}`,
		`22 6:10:.test.0 $(.test.x $1) {6:12:delegate {4:9:def .test.x} {=list {6:22:delegate {3:19:auto 1}}}}`:`foobaz {6:12 {4:11 {3:11 {9:12:word foobaz}}}}`,
		`22 6:10:.test.0 $(.test.y .$1) {4:11:delegate {3:9:def .test.y} {=list {=compound {4:21:punct .} {4:22:delegate {3:19:auto 1}}}}}`:`foobaz {4:11 {3:11 {9:12:word foobaz}}}`,

		`35 7:9:.test.1 .{} {=compound {4:21:punct .} {4:22 {7:22 {3:19:null}}}}`:`.{} {=compound {4:21:punct .} {4:22 {7:22 {3:19:null}}}}`,
		`35 7:9:.test.1 .test.{} {=compound {3:13:punct .} {3:14:word test} {3:18 {=compound {4:21:punct .} {4:22 {7:22 {3:19:null}}}}}}`:`.test.{} {=compound {3:13:punct .} {3:14:word test} {3:18 {=compound {4:21:punct .} {4:22 {7:22 {3:19:null}}}}}}`,
		`35 7:9:.test.1 &(.test.{}) {3:11:closure {=compound {3:13:punct .} {3:14:word test} {3:18 {=compound {4:21:punct .} {4:22 {7:22 {3:19:null}}}}}}}`:`www {3:11 {10:12:word www}}`,
	},
}

var checkstrs_value_12 = map[string]map[string]any{
	"check-value-12_test.go": map[string]any{
		`20 6:10:.test.0 &(.test.w) {3:11:closure {9:9:def .test.w}}`:`{3:11 {9:12:word foobaz}} foobaz`,
		`35 7:9:.test.1 &(.test.{}) {3:11:closure {=compound {3:13:punct .} {3:14:word test} {3:18 {=compound {4:21:punct .} {4:22 {7:22 {3:19:null}}}}}}}`:`{3:11 {10:12:word www}} www`,
	},
}
