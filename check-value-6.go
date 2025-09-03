//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints_value_6 = map[string]map[string]any{
	"loader.go": map[string]any{
		`6:8:.test $1 {3:13:delegate {3:14:auto 1}}`:`{3:9:def 1} y-x-a {3:13 {=compound {4:21:word y} {=flag {4:23 {=compound {5:21:word x} {=flag {5:23 {6:21:word a}}}}}}}}`,
		`6:8:.test $1 {4:23:delegate {3:14:auto 1}}`:`{4:9:def 1} x-a {4:23 {=compound {5:21:word x} {=flag {5:23 {6:21:word a}}}}}`,
		`6:8:.test $1 {5:23:delegate {3:14:auto 1}}`:`{5:9:def 1} a {5:23 {6:21:word a}}`,
		`6:8:.test $(.test.x a) {6:11:delegate {5:9:def .test.x} {=list {6:21:word a}}}`:`{5:9:def .test.x} z-y-x-a {6:11 {5:11 {4:11 {=compound {3:11:word z} {=flag {3:13 {=compound {4:21:word y} {=flag {4:23 {=compound {5:21:word x} {=flag {5:23 {6:21:word a}}}}}}}}}}}}}`,
		`6:8:.test $(.test.y x-$1) {5:11:delegate {4:9:def .test.y} {=list {=compound {5:21:word x} {=flag {5:23:delegate {3:14:auto 1}}}}}}`:`{4:9:def .test.y} z-y-x-a {5:11 {4:11 {=compound {3:11:word z} {=flag {3:13 {=compound {4:21:word y} {=flag {4:23 {=compound {5:21:word x} {=flag {5:23 {6:21:word a}}}}}}}}}}}}`,
		`6:8:.test $(.test.z y-$1) {4:11:delegate {3:9:def .test.z} {=list {=compound {4:21:word y} {=flag {4:23:delegate {3:14:auto 1}}}}}}`:`{3:9:def .test.z} z-y-x-a {4:11 {=compound {3:11:word z} {=flag {3:13 {=compound {4:21:word y} {=flag {4:23 {=compound {5:21:word x} {=flag {5:23 {6:21:word a}}}}}}}}}}}`,
	},
	"check-value-6_test.go": map[string]any{
	},
}

var checkstrs_value_6 = map[string]map[string]any{
}
