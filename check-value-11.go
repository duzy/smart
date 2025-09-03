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
		`5:10:.test &(.test{}) {4:13:closure {=compound {4:15:punct .} {4:16:word test} {4:20:null}}}`:`{=compound {4:15:punct .} {4:16:word test} {4:20:null}} &(.test{}) {4:13:closure {=compound {4:15:punct .} {4:16:word test} {4:20:null}}}`,
		`5:10:.test $(.test.x) {5:13:delegate {4:10:def .test.x}}`:`{4:10:def .test.x} &(.test{}) {5:13 {4:13:closure {=compound {4:15:punct .} {4:16:word test} {4:20:null}}}}`,
	},
	"check-value-11_test.go": map[string]any{
		`16 3:11:.test.0 $1 {3:20:delegate {3:21:auto 1}}`:sfmt(`{} {} {%[1]s/value/11/do.smart:3:20:null}`,testdata_s),
		`16 3:11:.test.0 &(.test$1) {3:13:closure {=compound {3:15:punct .} {3:16:word test} {3:20:delegate {3:21:auto 1}}}}`:sfmt(`{=compound {%[1]s/value/11/do.smart:3:15:punct .} {%[1]s/value/11/do.smart:3:16:word test} {%[1]s/value/11/do.smart:3:20:null}} {} {%[1]s/value/11/do.smart:3:13:null}`,testdata_s),
		`18 3:11:.test.0 $1 {3:20:delegate {3:21:auto 1}}`:`{3:11:def 1} .v1 .v2 {=list {3:20 {1:9:word .v1}} {3:20 {1:9:word .v2}}}`,
		`18 3:11:.test.0 &(.test$1) {3:13:closure {=compound {3:15:punct .} {3:16:word test} {3:20:delegate {3:21:auto 1}}}}`:`{=list {=compound {3:15:punct .} {3:16:word test} {3:20 {1:9:word .v1}}} {=compound {3:15:punct .} {3:16:word test} {3:20 {1:9:word .v2}}}} &(.test.v1) &(.test.v2) {=list {3:13:closure {=compound {3:15:punct .} {3:16:word test} {3:20 {1:9:word .v1}}}} {3:13:closure {=compound {3:15:punct .} {3:16:word test} {3:20 {1:9:word .v2}}}}}`,
		`22 3:11:.test.0 &(.test.v1) {3:13:closure {=compound {3:15:punct .} {3:16:word test} {3:20 {1:9:word .v1}}}}`:sfmt(`{=compound {%[1]s/value/11/do.smart:3:15:punct .} {%[1]s/value/11/do.smart:3:16:word test} {%[1]s/value/11/do.smart:3:20 {%[1]s/value/11/do.smart:1:9:word .v1}}} foo {%[1]s/value/11/do.smart:3:13 {%[1]s/value/11/do.smart:6:13:word foo}}`,testdata_s),
		`22 3:11:.test.0 &(.test.v2) {3:13:closure {=compound {3:15:punct .} {3:16:word test} {3:20 {1:9:word .v2}}}}`:sfmt(`{=compound {%[1]s/value/11/do.smart:3:15:punct .} {%[1]s/value/11/do.smart:3:16:word test} {%[1]s/value/11/do.smart:3:20 {%[1]s/value/11/do.smart:1:9:word .v2}}} bar {%[1]s/value/11/do.smart:3:13 {%[1]s/value/11/do.smart:7:13:word bar}}`,testdata_s),
		`31 5:10:.test &(.test{}) {4:13:closure {=compound {4:15:punct .} {4:16:word test} {4:20:null}}}`:[]string{
			sfmt(`{=compound {%[1]s/value/11/do.smart:4:15:punct .} {%[1]s/value/11/do.smart:4:16:word test} {%[1]s/value/11/do.smart:4:20:null}} {} {%[1]s/value/11/do.smart:4:13 {%[1]s/value/11/do.smart:5:13 {%[1]s/value/11/do.smart:4:13 {%[1]s/value/11/do.smart:5:10:null}}}}`,testdata_s),
			`{=compound {4:15:punct .} {4:16:word test} {4:20:null}} {} {4:13 {5:10:null}}`,
		},
	},
}

var checkstrs_value_11 = map[string]map[string]any{
}
