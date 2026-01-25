//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints_value_7 = map[string]map[string]any{
	"loader.go": map[string]any{
		`.test.z {=compound {3:1:punct .} {3:2:word test} {3:6:punct .} {3:7:word z}}`:`.test.z {=compound {3:1:punct .} {3:2:word test} {3:6:punct .} {3:7:word z}}`,
		`.test.y {=compound {4:1:punct .} {4:2:word test} {4:6:punct .} {4:7:word y}}`:`.test.y {=compound {4:1:punct .} {4:2:word test} {4:6:punct .} {4:7:word y}}`,
		`.test.x {=compound {5:1:punct .} {5:2:word test} {5:6:punct .} {5:7:word x}}`:`.test.x {=compound {5:1:punct .} {5:2:word test} {5:6:punct .} {5:7:word x}}`,
		`.test {=compound {6:1:punct .} {6:2:word test}}`:`.test {=compound {6:1:punct .} {6:2:word test}}`,
	},
	"check-value-7_test.go": map[string]any{
		`37 6:9:.test $1 {3:13:delegate {3:14:auto 1}}`:[]string{
			`yxa {3:13 {=compound {4:21:word y} {4:22 {=compound {5:21:word x} {5:22 {6:22 {1:9:word a}}}}}}}`,
		},
		`37 6:9:.test $2 {3:16:delegate {3:17:auto 2}}`:[]string{
			`yxb {3:16 {=compound {4:25:word y} {4:26 {=compound {5:25:word x} {5:26 {6:26 {1:9:word b}}}}}}}`,
		},
		`37 6:9:.test $1 {4:22:delegate {3:14:auto 1}}`:[]string{
			`xa {4:22 {=compound {5:21:word x} {5:22 {6:22 {1:9:word a}}}}}`,
		},
		`37 6:9:.test $2 {4:26:delegate {3:17:auto 2}}`:[]string{
			`xb {4:26 {=compound {5:25:word x} {5:26 {6:26 {1:9:word b}}}}}`,
		},
		`37 6:9:.test $1 {5:22:delegate {3:14:auto 1}}`:[]string{
			`a {5:22 {6:22 {1:9:word a}}}`,
		},
		`37 6:9:.test $2 {5:26:delegate {3:17:auto 2}}`:[]string{
			`b {5:26 {6:26 {1:9:word b}}}`,
			`b {6:26 {1:9:word b}}`,
		},
		`37 6:9:.test $1 {6:22:delegate {3:14:auto 1}}`:[]string{
			`a {6:22 {1:9:word a}}`,
		},
		`37 6:9:.test $2 {6:26:delegate {3:17:auto 2}}`:[]string{
			`b {6:26 {1:9:word b}}`,
		},
		`37 6:9:.test $1-$2 {=compound {3:13:delegate {3:14:auto 1}} {=flag {3:16:delegate {3:17:auto 2}}}}`:[]string{
			`yxa-yxb {=compound {3:13 {=compound {4:21:word y} {4:22 {=compound {5:21:word x} {5:22 {6:22 {1:9:word a}}}}}}} {=flag {3:16 {=compound {4:25:word y} {4:26 {=compound {5:25:word x} {5:26 {6:26 {1:9:word b}}}}}}}}}`,
		},
		`37 6:9:.test x$1 {=compound {5:21:word x} {5:22:delegate {3:14:auto 1}}}`:[]string{
			`xa {=compound {5:21:word x} {5:22 {6:22 {1:9:word a}}}}`,
		},
		`37 6:9:.test x$2 {=compound {5:25:word x} {5:26:delegate {3:17:auto 2}}}`:[]string{
			`xb {=compound {5:25:word x} {5:26 {6:26 {1:9:word b}}}}`,
		},
		`37 6:9:.test xa {=compound {5:21:word x} {5:22 {6:22 {1:9:word a}}}}`:`xa {=compound {5:21:word x} {5:22 {6:22 {1:9:word a}}}}`,
		`37 6:9:.test xb {=compound {5:25:word x} {5:26 {6:26 {1:9:word b}}}}`:`xb {=compound {5:25:word x} {5:26 {6:26 {1:9:word b}}}}`,
		`37 6:9:.test y$1 {=compound {4:21:word y} {4:22:delegate {3:14:auto 1}}}`:[]string{
			`yxa {=compound {4:21:word y} {4:22 {=compound {5:21:word x} {5:22 {6:22 {1:9:word a}}}}}}`,
		},
		`37 6:9:.test y$2 {=compound {4:25:word y} {4:26:delegate {3:17:auto 2}}}`:[]string{
			`yxb {=compound {4:25:word y} {4:26 {=compound {5:25:word x} {5:26 {6:26 {1:9:word b}}}}}}`,
		},
		`37 6:9:.test yxa {=compound {4:21:word y} {4:22 {=compound {5:21:word x} {5:22 {6:22 {1:9:word a}}}}}}`:`yxa {=compound {4:21:word y} {4:22 {=compound {5:21:word x} {5:22 {6:22 {1:9:word a}}}}}}`,
		`37 6:9:.test yxb {=compound {4:25:word y} {4:26 {=compound {5:25:word x} {5:26 {6:26 {1:9:word b}}}}}}`:`yxb {=compound {4:25:word y} {4:26 {=compound {5:25:word x} {5:26 {6:26 {1:9:word b}}}}}}`,
		`37 6:9:.test z-$1-$2 {=compound {3:11:word z} {=flag {=compound {3:13:delegate {3:14:auto 1}} {=flag {3:16:delegate {3:17:auto 2}}}}}}`:[]string{
			`z-yxa-yxb {=compound {3:11:word z} {=flag {=compound {3:13 {=compound {4:21:word y} {4:22 {=compound {5:21:word x} {5:22 {6:22 {1:9:word a}}}}}}} {=flag {3:16 {=compound {4:25:word y} {4:26 {=compound {5:25:word x} {5:26 {6:26 {1:9:word b}}}}}}}}}}}`,
		},
		`37 6:9:.test $(.test.x $1,$2) {6:11:delegate {5:9:def .test.x} {=list {6:22:delegate {3:14:auto 1}}} {=list {6:26:delegate {3:17:auto 2}}}}`:[]string{
			`z-yxa-yxb {6:11 {5:11 {4:11 {=compound {3:11:word z} {=flag {=compound {3:13 {=compound {4:21:word y} {4:22 {=compound {5:21:word x} {5:22 {6:22 {1:9:word a}}}}}}} {=flag {3:16 {=compound {4:25:word y} {4:26 {=compound {5:25:word x} {5:26 {6:26 {1:9:word b}}}}}}}}}}}}}}`,
		},
		`37 6:9:.test $(.test.y x$1,x$2) {5:11:delegate {4:9:def .test.y} {=list {=compound {5:21:word x} {5:22:delegate {3:14:auto 1}}}} {=list {=compound {5:25:word x} {5:26:delegate {3:17:auto 2}}}}}`:[]string{
			`z-yxa-yxb {5:11 {4:11 {=compound {3:11:word z} {=flag {=compound {3:13 {=compound {4:21:word y} {4:22 {=compound {5:21:word x} {5:22 {6:22 {1:9:word a}}}}}}} {=flag {3:16 {=compound {4:25:word y} {4:26 {=compound {5:25:word x} {5:26 {6:26 {1:9:word b}}}}}}}}}}}}}`,
		},
		`37 6:9:.test $(.test.z y$1,y$2) {4:11:delegate {3:9:def .test.z} {=list {=compound {4:21:word y} {4:22:delegate {3:14:auto 1}}}} {=list {=compound {4:25:word y} {4:26:delegate {3:17:auto 2}}}}}`:[]string{
			`z-yxa-yxb {4:11 {=compound {3:11:word z} {=flag {=compound {3:13 {=compound {4:21:word y} {4:22 {=compound {5:21:word x} {5:22 {6:22 {1:9:word a}}}}}}} {=flag {3:16 {=compound {4:25:word y} {4:26 {=compound {5:25:word x} {5:26 {6:26 {1:9:word b}}}}}}}}}}}}`,
		},

		`41 6:9:.test xa {=compound {5:21:word x} {5:22 {6:22 {1:9:word a}}}}`:`xa {=compound {5:21:word x} {5:22 {6:22 {1:9:word a}}}}`,
		`41 6:9:.test xb {=compound {5:25:word x} {5:26 {6:26 {1:9:word b}}}}`:`xb {=compound {5:25:word x} {5:26 {6:26 {1:9:word b}}}}`,
		`41 6:9:.test yxa {=compound {4:21:word y} {4:22 {=compound {5:21:word x} {5:22 {6:22 {1:9:word a}}}}}}`:`yxa {=compound {4:21:word y} {4:22 {=compound {5:21:word x} {5:22 {6:22 {1:9:word a}}}}}}`,
		`41 6:9:.test yxb {=compound {4:25:word y} {4:26 {=compound {5:25:word x} {5:26 {6:26 {1:9:word b}}}}}}`:`yxb {=compound {4:25:word y} {4:26 {=compound {5:25:word x} {5:26 {6:26 {1:9:word b}}}}}}`,
		`41 6:9:.test yxa-yxb {=compound {3:13 {=compound {4:21:word y} {4:22 {=compound {5:21:word x} {5:22 {6:22 {1:9:word a}}}}}}} {=flag {3:16 {=compound {4:25:word y} {4:26 {=compound {5:25:word x} {5:26 {6:26 {1:9:word b}}}}}}}}}`:`yxa-yxb {=compound {3:13 {=compound {4:21:word y} {4:22 {=compound {5:21:word x} {5:22 {6:22 {1:9:word a}}}}}}} {=flag {3:16 {=compound {4:25:word y} {4:26 {=compound {5:25:word x} {5:26 {6:26 {1:9:word b}}}}}}}}}`,
		`41 6:9:.test z-yxa-yxb {=compound {3:11:word z} {=flag {=compound {3:13 {=compound {4:21:word y} {4:22 {=compound {5:21:word x} {5:22 {6:22 {1:9:word a}}}}}}} {=flag {3:16 {=compound {4:25:word y} {4:26 {=compound {5:25:word x} {5:26 {6:26 {1:9:word b}}}}}}}}}}}`:`z-yxa-yxb {=compound {3:11:word z} {=flag {=compound {3:13 {=compound {4:21:word y} {4:22 {=compound {5:21:word x} {5:22 {6:22 {1:9:word a}}}}}}} {=flag {3:16 {=compound {4:25:word y} {4:26 {=compound {5:25:word x} {5:26 {6:26 {1:9:word b}}}}}}}}}}}`,
	},
}

var checkstrs_value_7 = map[string]map[string]any{
}
