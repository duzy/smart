//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints_value_11 = map[string]map[string]any{
	"loader.go": map[string]any{
		`.test.0 {=compound {3:1:punct .} {3:2:word test} {3:6:punct .} {3:7:decimal 0}}`:`.test.0 {=compound {3:1:punct .} {3:2:word test} {3:6:punct .} {3:7:decimal 0}}`,
		`.test.x {=compound {4:1:punct .} {4:2:word test} {4:6:punct .} {4:7:word x}}`:`.test.x {=compound {4:1:punct .} {4:2:word test} {4:6:punct .} {4:7:word x}}`,
		`.test {=compound {5:1:punct .} {5:2:word test}}`:`.test {=compound {5:1:punct .} {5:2:word test}}`,
		`.test.s1 {=compound {6:1:punct .} {6:2:word test} {6:6:punct .} {6:7:word s1}}`:`.test.s1 {=compound {6:1:punct .} {6:2:word test} {6:6:punct .} {6:7:word s1}}`,
		`.test.s2 {=compound {7:1:punct .} {7:2:word test} {7:6:punct .} {7:7:word s2}}`:`.test.s2 {=compound {7:1:punct .} {7:2:word test} {7:6:punct .} {7:7:word s2}}`,
		`.test.1 {=compound {9:1:punct .} {9:2:word test} {9:6:punct .} {9:7:decimal 1}}`:`.test.1 {=compound {9:1:punct .} {9:2:word test} {9:6:punct .} {9:7:decimal 1}}`,
		`.test⌜.s1 .s2⌟ {=compound {10:1:punct .} {10:2:word test} {=list {=compound {10:7:punct .} {10:8:word s1}} {=compound {10:11:punct .} {10:12:word s2}}}}`:`.test⌜.s1 .s2⌟ {=compound {10:1:punct .} {10:2:word test} {=list {=compound {10:7:punct .} {10:8:word s1}} {=compound {10:11:punct .} {10:12:word s2}}}}`,

		`.s1 {=compound {10:7:punct .} {10:8:word s1}}`:`.s1 {=compound {10:7:punct .} {10:8:word s1}}`,
		`.s2 {=compound {10:11:punct .} {10:12:word s2}}`:`.s2 {=compound {10:11:punct .} {10:12:word s2}}`,

		`4:10:.test.x $1 {4:20:delegate {3:21:auto 1}}`:`{} {4:20 {3:21:null}}`,
		`4:10:.test.x .test$1 {=compound {4:15:punct .} {4:16:word test} {4:20:delegate {3:21:auto 1}}}`:`.test{} {=compound {4:15:punct .} {4:16:word test} {4:20 {3:21:null}}}`,
		`4:10:.test.x &(.test$1) {4:13:closure {=compound {4:15:punct .} {4:16:word test} {4:20:delegate {3:21:auto 1}}}}`:`&(.test{}) {4:13:closure {=compound {4:15:punct .} {4:16:word test} {4:20 {3:21:null}}}}`,

		`5:10:.test $1 {4:20:delegate {3:21:auto 1}}`:`{} {} {4:20:null}`,
		`5:10:.test .test{} {=compound {4:15:punct .} {4:16:word test} {4:20 {3:21:null}}}`:`.test{} {=compound {4:15:punct .} {4:16:word test} {4:20 {3:21:null}}}`,
		`5:10:.test &(.test{}) {4:13:closure {=compound {4:15:punct .} {4:16:word test} {4:20 {3:21:null}}}}`:`&(.test) {4:13:closure {5:10:def .test}}`,
		`5:10:.test $(.test.x) {5:13:delegate {4:10:def .test.x}}`:`&(.test) {5:13 {4:13:closure {5:10:def .test}}}`,
	},
	"check-value-11_test.go": map[string]any{
		`16 3:11:.test.0 $1 {3:20:delegate {3:21:auto 1}}`:[]string{`{} {3:20 {3:21:null}}`,`{} {4:20 {3:21:null}}`},
		`16 3:11:.test.0 .test$1 {=compound {3:15:punct .} {3:16:word test} {3:20:delegate {3:21:auto 1}}}`:`.test{} {=compound {3:15:punct .} {3:16:word test} {3:20 {3:21:null}}}`,
		`16 3:11:.test.0 &(.test$1) {3:13:closure {=compound {3:15:punct .} {3:16:word test} {3:20:delegate {3:21:auto 1}}}}`:`{} {3:13 {5:13 {4:13 {5:10:null}}}}`,
		`16 3:11:.test.0 &(.test) {4:13:closure {5:10:def .test}}`:`{} {4:13 {5:10:null}}`,

		`18 3:11:.test.0 $1 {3:20:delegate {3:21:auto 1}}`:`.s1 {3:20 {1:9:word .s1}}`,
		`18 3:11:.test.0 .test$1 {=compound {3:15:punct .} {3:16:word test} {3:20:delegate {3:21:auto 1}}}`:`.test.s1 {=compound {3:15:punct .} {3:16:word test} {3:20 {1:9:word .s1}}}`,
		`18 3:11:.test.0 &(.test$1) {3:13:closure {=compound {3:15:punct .} {3:16:word test} {3:20:delegate {3:21:auto 1}}}}`:`&(.test.s1) {3:13:closure {6:10:def .test.s1}}`,

		`24 3:11:.test.0 $1 {3:20:delegate {3:21:auto 1}}`:`.s2 {3:20 {1:9:word .s2}}`,
		`24 3:11:.test.0 .test$1 {=compound {3:15:punct .} {3:16:word test} {3:20:delegate {3:21:auto 1}}}`:`.test.s2 {=compound {3:15:punct .} {3:16:word test} {3:20 {1:9:word .s2}}}`,
		`24 3:11:.test.0 &(.test$1) {3:13:closure {=compound {3:15:punct .} {3:16:word test} {3:20:delegate {3:21:auto 1}}}}`:`&(.test.s2) {3:13:closure {7:10:def .test.s2}}`,

		`22 3:11:.test.0 &(.test.s1) {3:13:closure {6:10:def .test.s1}}`:`foo {3:13 {6:13:word foo}}`,
		`28 3:11:.test.0 &(.test.s2) {3:13:closure {7:10:def .test.s2}}`:`bar {3:13 {7:13:word bar}}`,

		`30 3:11:.test.0 $1 {3:20:delegate {3:21:auto 1}}`:`.s1 .s2 {=list {3:20 {1:9:word .s1}} {3:20 {1:9:word .s2}}}`,
		`30 3:11:.test.0 .test$1 {=compound {3:15:punct .} {3:16:word test} {3:20:delegate {3:21:auto 1}}}`:`.test⌜.s1 .s2⌟ {=compound {3:15:punct .} {3:16:word test} {=list {3:20 {1:9:word .s1}} {3:20 {1:9:word .s2}}}}`,
		`30 3:11:.test.0 &(.test$1) {3:13:closure {=compound {3:15:punct .} {3:16:word test} {3:20:delegate {3:21:auto 1}}}}`:`&(.test⌜.s1 .s2⌟) {3:13:closure {10:16:def .test⌜.s1 .s2⌟}}`,

		`34 3:11:.test.0 .test⌜.s1 .s2⌟ {=compound {3:15:punct .} {3:16:word test} {=list {3:20 {1:9:word .s1}} {3:20 {1:9:word .s2}}}}`:`.test⌜.s1 .s2⌟ {=compound {3:15:punct .} {3:16:word test} {=list {3:20 {1:9:word .s1}} {3:20 {1:9:word .s2}}}}`,
		`34 3:11:.test.0 &(.test⌜.s1 .s2⌟) {3:13:closure {10:16:def .test⌜.s1 .s2⌟}}`:`foobar {3:13 {10:19:word foobar}}`,

		`43 9:9:.test.1 $1 {9:19:delegate {3:21:auto 1}}`:`{} {9:19 {3:21:null}}`,
		`43 9:9:.test.1 .test{$1} {=compound {9:13:punct .} {9:14:word test} {9:18:disjunction {9:19:delegate {3:21:auto 1}}}}`:`{} {9:13:null}`,
		`43 9:9:.test.1 &(.test{$1}) {9:11:closure {=compound {9:13:punct .} {9:14:word test} {9:18:disjunction {9:19:delegate {3:21:auto 1}}}}}`:`{} {9:11:null}`,

		`44 9:9:.test.1 $1 {9:19:delegate {3:21:auto 1}}`:`{} {9:19 {3:21:null}}`,

		`45 9:9:.test.1 $1 {9:19:delegate {3:21:auto 1}}`:`.s1 .s2 {=list {9:19 {1:9:word .s1}} {9:19 {1:9:word .s2}}}`,
		`45 9:9:.test.1 .test{$1} {=compound {9:13:punct .} {9:14:word test} {9:18:disjunction {9:19:delegate {3:21:auto 1}}}}`:`.test.s1 .test.s2 {=list {=compound {9:13:punct .} {9:14:word test} {9:19 {1:9:word .s1}}} {=compound {9:13:punct .} {9:14:word test} {9:19 {1:9:word .s2}}}}`,
		`45 9:9:.test.1 &(.test{$1}) {9:11:closure {=compound {9:13:punct .} {9:14:word test} {9:18:disjunction {9:19:delegate {3:21:auto 1}}}}}`:`&(.test.s1) &(.test.s2) {=list {9:11:closure {6:10:def .test.s1}} {9:11:closure {7:10:def .test.s2}}}`,

		`49 9:9:.test.1 &(.test.s1) {9:11:closure {6:10:def .test.s1}}`:`foo {9:11 {6:13:word foo}}`,
		`49 9:9:.test.1 &(.test.s2) {9:11:closure {7:10:def .test.s2}}`:`bar {9:11 {7:13:word bar}}`,

		`58 5:10:.test &(.test) {4:13:closure {5:10:def .test}}`:[]string{`{} {4:13 {5:10:null}}`,`{} {4:13 {5:13 {4:13 {5:10:null}}}}`},
	},
}

var checkstrs_value_11 = map[string]map[string]any{
	"check-value-11_test.go": map[string]any{
		`16 3:11:.test.0 &(.test$1) {3:13:closure {=compound {3:15:punct .} {3:16:word test} {3:20:delegate {3:21:auto 1}}}}`:`{3:13 {5:13 {4:13 {5:10:null}}}} `,
		`22 3:11:.test.0 &(.test.s1) {3:13:closure {6:10:def .test.s1}}`:`{3:13 {6:13:word foo}} foo`,
		`28 3:11:.test.0 &(.test.s2) {3:13:closure {7:10:def .test.s2}}`:`{3:13 {7:13:word bar}} bar`,
		`34 3:11:.test.0 &(.test⌜.s1 .s2⌟) {3:13:closure {10:16:def .test⌜.s1 .s2⌟}}`:`{3:13 {10:19:word foobar}} foobar`,
		`43 9:9:.test.1 &(.test{$1}) {9:11:closure {=compound {9:13:punct .} {9:14:word test} {9:18:disjunction {9:19:delegate {3:21:auto 1}}}}}`:`{9:11:null} `,
		`49 9:9:.test.1 &(.test.s1) {9:11:closure {6:10:def .test.s1}}`:`{9:11 {6:13:word foo}} foo`,
		`49 9:9:.test.1 &(.test.s2) {9:11:closure {7:10:def .test.s2}}`:`{9:11 {7:13:word bar}} bar`,
		`58 5:10:.test &(.test) {4:13:closure {5:10:def .test}}`:`{4:13 {5:13 {4:13 {5:10:null}}}} `,
	},
}
