//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints_value_12 = map[string]map[string]any{
	"loader.go": map[string]any{
		`1120 7:9:.test.1 $1 {3:18:delegate {3:19:auto 1}}`:`{3:9:def 1} .{} {3:18 {=compound {4:21:punct .} {4:22 {7:22:null}}}}`,
		`1120 7:9:.test.1 $1 {4:22:delegate {3:19:auto 1}}`:`{4:9:def 1} {} {4:22 {7:22:null}}`,
		`1120 7:9:.test.1 $1 {7:22:delegate {3:19:auto 1}}`:`{} {} {7:22:null}`,
		`1120 7:9:.test.1 &(.test$1) {3:11:closure {=compound {3:13:punct .} {3:14:word test} {3:18:delegate {3:19:auto 1}}}}`:`{=compound {3:13:punct .} {3:14:word test} {3:18 {4:21:punct .}} {3:18 {4:22 {7:22:null}}}} &(.test.{}) {3:11:closure {=compound {3:13:punct .} {3:14:word test} {3:18 {4:21:punct .}} {3:18 {4:22 {7:22:null}}}}}`,
		`1120 7:9:.test.1 $(.test.x $1) {7:12:delegate {4:9:def .test.x} {=list {7:22:delegate {3:19:auto 1}}}}`:`{4:9:def .test.x} &(.test.{}) {7:12 {4:11 {3:11:closure {=compound {3:13:punct .} {3:14:word test} {3:18 {4:21:punct .}} {3:18 {4:22 {7:22:null}}}}}}}`,
		`1120 7:9:.test.1 $(.test.y .$1) {4:11:delegate {3:9:def .test.y} {=list {=compound {4:21:punct .} {4:22:delegate {3:19:auto 1}}}}}`:`{3:9:def .test.y} &(.test.{}) {4:11 {3:11:closure {=compound {3:13:punct .} {3:14:word test} {3:18 {4:21:punct .}} {3:18 {4:22 {7:22:null}}}}}}`,
	},
	"check-value-12_test.go": map[string]any{
		`16 6:10:.test.0 $1 {6:22:delegate {3:19:auto 1}}`:`{6:10:def 1} w {6:22 {1:9:word w}}`,
		`16 6:10:.test.0 $1 {4:22:delegate {3:19:auto 1}}`:`{4:9:def 1} w {4:22 {6:22 {1:9:word w}}}`,
		`16 6:10:.test.0 $1 {3:18:delegate {3:19:auto 1}}`:`{3:9:def 1} .w {3:18 {=compound {4:21:punct .} {4:22 {6:22 {1:9:word w}}}}}`,
		`16 6:10:.test.0 &(.test$1) {3:11:closure {=compound {3:13:punct .} {3:14:word test} {3:18:delegate {3:19:auto 1}}}}`:`{=compound {3:13:punct .} {3:14:word test} {3:18 {4:21:punct .}} {3:18 {4:22 {6:22 {1:9:word w}}}}} &(.test.w) {3:11:closure {=compound {3:13:punct .} {3:14:word test} {3:18 {4:21:punct .}} {3:18 {4:22 {6:22 {1:9:word w}}}}}}`,
		`16 6:10:.test.0 $(.test.x $1) {6:12:delegate {4:9:def .test.x} {=list {6:22:delegate {3:19:auto 1}}}}`:`{4:9:def .test.x} &(.test.w) {6:12 {4:11 {3:11:closure {=compound {3:13:punct .} {3:14:word test} {3:18 {4:21:punct .}} {3:18 {4:22 {6:22 {1:9:word w}}}}}}}}`,
		`16 6:10:.test.0 $(.test.y .$1) {4:11:delegate {3:9:def .test.y} {=list {=compound {4:21:punct .} {4:22:delegate {3:19:auto 1}}}}}`:`{3:9:def .test.y} &(.test.w) {4:11 {3:11:closure {=compound {3:13:punct .} {3:14:word test} {3:18 {4:21:punct .}} {3:18 {4:22 {6:22 {1:9:word w}}}}}}}`,
		`20 6:10:.test.0 &(.test.w) {3:11:closure {=compound {3:13:punct .} {3:14:word test} {3:18 {4:21:punct .}} {3:18 {4:22 {6:22 {1:9:word w}}}}}}`:sfmt(`{=compound {%[1]s/value/12/do.smart:3:13:punct .} {%[1]s/value/12/do.smart:3:14:word test} {%[1]s/value/12/do.smart:3:18 {%[1]s/value/12/do.smart:4:21:punct .}} {%[1]s/value/12/do.smart:3:18 {%[1]s/value/12/do.smart:4:22 {%[1]s/value/12/do.smart:6:22 {%[1]s/value/12/do.smart:1:9:word w}}}}} foobaz {%[1]s/value/12/do.smart:3:11 {%[1]s/value/12/do.smart:9:12:word foobaz}}`,testdata_dir),
		`22 6:10:.test.0 $1 {6:22:delegate {3:19:auto 1}}`:`{6:10:def 1} w {6:22 {1:9:word w}}`,
		`22 6:10:.test.0 $1 {4:22:delegate {3:19:auto 1}}`:`{4:9:def 1} w {4:22 {6:22 {1:9:word w}}}`,
		`22 6:10:.test.0 $1 {3:18:delegate {3:19:auto 1}}`:`{3:9:def 1} .w {3:18 {=compound {4:21:punct .} {4:22 {6:22 {1:9:word w}}}}}`,
		`22 6:10:.test.0 &(.test$1) {3:11:closure {=compound {3:13:punct .} {3:14:word test} {3:18:delegate {3:19:auto 1}}}}`:`{=compound {3:13:punct .} {3:14:word test} {3:18 {4:21:punct .}} {3:18 {4:22 {6:22 {1:9:word w}}}}} foobaz {3:11 {9:12:word foobaz}}`,
		`22 6:10:.test.0 $(.test.x $1) {6:12:delegate {4:9:def .test.x} {=list {6:22:delegate {3:19:auto 1}}}}`:`{4:9:def .test.x} foobaz {6:12 {4:11 {3:11 {9:12:word foobaz}}}}`,
		`22 6:10:.test.0 $(.test.y .$1) {4:11:delegate {3:9:def .test.y} {=list {=compound {4:21:punct .} {4:22:delegate {3:19:auto 1}}}}}`:`{3:9:def .test.y} foobaz {4:11 {3:11 {9:12:word foobaz}}}`,
		`35 7:9:.test.1 &(.test.{}) {3:11:closure {=compound {3:13:punct .} {3:14:word test} {3:18 {4:21:punct .}} {3:18 {4:22 {7:22:null}}}}}`:sfmt(`{=compound {%[1]s/value/12/do.smart:3:13:punct .} {%[1]s/value/12/do.smart:3:14:word test} {%[1]s/value/12/do.smart:3:18 {%[1]s/value/12/do.smart:4:21:punct .}} {%[1]s/value/12/do.smart:3:18 {%[1]s/value/12/do.smart:4:22 {%[1]s/value/12/do.smart:7:22:null}}}} www {%[1]s/value/12/do.smart:3:11 {%[1]s/value/12/do.smart:10:12:word www}}`,testdata_dir),
	},
}
