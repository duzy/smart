//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints_value_6 = map[string]map[string]any{
	"loader.go": map[string]any{
		`.test.z {=compound {3:1:punct .} {3:2:word test} {3:6:punct .} {3:7:word z}}`:`.test.z {=compound {3:1:punct .} {3:2:word test} {3:6:punct .} {3:7:word z}}`,
		`.test.y {=compound {4:1:punct .} {4:2:word test} {4:6:punct .} {4:7:word y}}`:`.test.y {=compound {4:1:punct .} {4:2:word test} {4:6:punct .} {4:7:word y}}`,
		`.test.x {=compound {5:1:punct .} {5:2:word test} {5:6:punct .} {5:7:word x}}`:`.test.x {=compound {5:1:punct .} {5:2:word test} {5:6:punct .} {5:7:word x}}`,
		`.test {=compound {6:1:punct .} {6:2:word test}}`:`.test {=compound {6:1:punct .} {6:2:word test}}`,

		`6:8:.test $1 {3:13:delegate {3:14:auto 1}}`:`{3:14:auto 1} y-x-a {3:13 {=compound {4:21:word y} {=flag {4:23 {=compound {5:21:word x} {=flag {5:23 {6:21:word a}}}}}}}}`,
		`6:8:.test $1 {4:23:delegate {3:14:auto 1}}`:`{3:14:auto 1} x-a {4:23 {=compound {5:21:word x} {=flag {5:23 {6:21:word a}}}}}`,
		`6:8:.test $1 {5:23:delegate {3:14:auto 1}}`:`{3:14:auto 1} a {5:23 {6:21:word a}}`,
		`6:8:.test $(.test.x a) {6:11:delegate {5:9:def .test.x} {=list {6:21:word a}}}`:`{5:9:def .test.x} z-y-x-a {6:11 {5:11 {4:11 {=compound {3:11:word z} {=flag {3:13 {=compound {4:21:word y} {=flag {4:23 {=compound {5:21:word x} {=flag {5:23 {6:21:word a}}}}}}}}}}}}}`,
		`6:8:.test $(.test.y x-$1) {5:11:delegate {4:9:def .test.y} {=list {=compound {5:21:word x} {=flag {5:23:delegate {3:14:auto 1}}}}}}`:`{4:9:def .test.y} z-y-x-a {5:11 {4:11 {=compound {3:11:word z} {=flag {3:13 {=compound {4:21:word y} {=flag {4:23 {=compound {5:21:word x} {=flag {5:23 {6:21:word a}}}}}}}}}}}}`,
		`6:8:.test $(.test.z y-$1) {4:11:delegate {3:9:def .test.z} {=list {=compound {4:21:word y} {=flag {4:23:delegate {3:14:auto 1}}}}}}`:`{3:9:def .test.z} z-y-x-a {4:11 {=compound {3:11:word z} {=flag {3:13 {=compound {4:21:word y} {=flag {4:23 {=compound {5:21:word x} {=flag {5:23 {6:21:word a}}}}}}}}}}}`,
		`6:8:.test x-$1 {=compound {5:21:word x} {=flag {5:23:delegate {3:14:auto 1}}}}`:`x-a {=compound {5:21:word x} {=flag {5:23 {6:21:word a}}}}`,
		`6:8:.test x-a {=compound {5:21:word x} {=flag {5:23 {6:21:word a}}}}`:`x-a {=compound {5:21:word x} {=flag {5:23 {6:21:word a}}}}`,
		`6:8:.test y-$1 {=compound {4:21:word y} {=flag {4:23:delegate {3:14:auto 1}}}}`:`y-x-a {=compound {4:21:word y} {=flag {4:23 {=compound {5:21:word x} {=flag {5:23 {6:21:word a}}}}}}}`,
		`6:8:.test y-x-a {=compound {4:21:word y} {=flag {4:23 {=compound {5:21:word x} {=flag {5:23 {6:21:word a}}}}}}}`:`y-x-a {=compound {4:21:word y} {=flag {4:23 {=compound {5:21:word x} {=flag {5:23 {6:21:word a}}}}}}}`,
		`6:8:.test z-$1 {=compound {3:11:word z} {=flag {3:13:delegate {3:14:auto 1}}}}`:`z-y-x-a {=compound {3:11:word z} {=flag {3:13 {=compound {4:21:word y} {=flag {4:23 {=compound {5:21:word x} {=flag {5:23 {6:21:word a}}}}}}}}}}`,
	},
	"check-value-6_test.go": map[string]any{
		`18 6:8:.test x-a {=compound {5:21:word x} {=flag {5:23 {6:21:word a}}}}`:`x-a {=compound {5:21:word x} {=flag {5:23 {6:21:word a}}}}`,
		`18 6:8:.test y-x-a {=compound {4:21:word y} {=flag {4:23 {=compound {5:21:word x} {=flag {5:23 {6:21:word a}}}}}}}`:`y-x-a {=compound {4:21:word y} {=flag {4:23 {=compound {5:21:word x} {=flag {5:23 {6:21:word a}}}}}}}`,
		`18 6:8:.test z-y-x-a {=compound {3:11:word z} {=flag {3:13 {=compound {4:21:word y} {=flag {4:23 {=compound {5:21:word x} {=flag {5:23 {6:21:word a}}}}}}}}}}`:`z-y-x-a {=compound {3:11:word z} {=flag {3:13 {=compound {4:21:word y} {=flag {4:23 {=compound {5:21:word x} {=flag {5:23 {6:21:word a}}}}}}}}}}`,
		`22 6:8:.test x-a {=compound {5:21:word x} {=flag {5:23 {6:21:word a}}}}`:`x-a {=compound {5:21:word x} {=flag {5:23 {6:21:word a}}}}`,
		`22 6:8:.test y-x-a {=compound {4:21:word y} {=flag {4:23 {=compound {5:21:word x} {=flag {5:23 {6:21:word a}}}}}}}`:`y-x-a {=compound {4:21:word y} {=flag {4:23 {=compound {5:21:word x} {=flag {5:23 {6:21:word a}}}}}}}`,
		`22 6:8:.test z-y-x-a {=compound {3:11:word z} {=flag {3:13 {=compound {4:21:word y} {=flag {4:23 {=compound {5:21:word x} {=flag {5:23 {6:21:word a}}}}}}}}}}`:`z-y-x-a {=compound {3:11:word z} {=flag {3:13 {=compound {4:21:word y} {=flag {4:23 {=compound {5:21:word x} {=flag {5:23 {6:21:word a}}}}}}}}}}`,
	},
}

var checkstrs_value_6 = map[string]map[string]any{
}
