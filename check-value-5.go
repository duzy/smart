//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints_value_5 = map[string]map[string]any{
	"loader.go": map[string]any{
		`8:10:.test.x1 $1 {8:24:delegate {3:16:auto 1}}`:`{} {} {8:24:null}`,
		`6:10:.test.y1 $1 {6:24:delegate {3:16:auto 1}}`:`{} {} {6:24:null}`,
		`4:10:.test.z1 $1 {4:15:delegate {3:16:auto 1}}`:`{} {} {4:15:null}`,
		`8:10:.test.x1 $(.test.y1 $1) {8:13:delegate {6:10:def .test.y1} {=list {8:24:delegate {3:16:auto 1}}}}`:`{6:10:def .test.y1} z-{} {8:13 {6:13 {=compound {4:13:word z} {=flag {4:15:null}}}}}`,
		`6:10:.test.y1 $(.test.z1 $1) {6:13:delegate {4:10:def .test.z1} {=list {6:24:delegate {3:16:auto 1}}}}`:`{4:10:def .test.z1} z-{} {6:13 {=compound {4:13:word z} {=flag {4:15:null}}}}`,
		`10:10:.test.1 $1 {10:24:delegate {3:16:auto 1}}`:`{} {} {10:24:null}`,
		`10:10:.test.1 $(.test.x1 $1) {10:13:delegate {8:10:def .test.x1} {=list {10:24:delegate {3:16:auto 1}}}}`:`{8:10:def .test.x1} z-{} {10:13 {8:13 {6:13 {=compound {4:13:word z} {=flag {4:15:null}}}}}}`,
	},
	"check-value-5_test.go": map[string]any{
		`18 9:11:.test.0 $1 {3:15:delegate {3:16:auto 1}}`:`{3:11:def 1} {} {3:15 {5:24 {7:24 {9:24:null}}}}`,
		`18 9:11:.test.0 $1 {5:24:delegate {3:16:auto 1}}`:`{5:11:def 1} {} {5:24 {7:24 {9:24:null}}}`,
		`18 9:11:.test.0 $1 {7:24:delegate {3:16:auto 1}}`:`{7:11:def 1} {} {7:24 {9:24:null}}`,
		`18 9:11:.test.0 $1 {9:24:delegate {3:16:auto 1}}`:`{} {} {9:24:null}`,
		`18 9:11:.test.0 $(.test.x0 $1) {9:13:delegate {7:11:def .test.x0} {=list {9:24:delegate {3:16:auto 1}}}}`:`{7:11:def .test.x0} z-{} {9:13 {7:13 {5:13 {=compound {3:13:word z} {=flag {3:15 {5:24 {7:24 {9:24:null}}}}}}}}}`,
		`18 9:11:.test.0 $(.test.y0 $1) {7:13:delegate {5:11:def .test.y0} {=list {7:24:delegate {3:16:auto 1}}}}`:`{5:11:def .test.y0} z-{} {7:13 {5:13 {=compound {3:13:word z} {=flag {3:15 {5:24 {7:24 {9:24:null}}}}}}}}`,
		`18 9:11:.test.0 $(.test.z0 $1) {5:13:delegate {3:11:def .test.z0} {=list {5:24:delegate {3:16:auto 1}}}}`:`{3:11:def .test.z0} z-{} {5:13 {=compound {3:13:word z} {=flag {3:15 {5:24 {7:24 {9:24:null}}}}}}}`,
		`20 9:11:.test.0 $(.test.x0 $1) {9:13:delegate {7:11:def .test.x0} {=list {9:24:delegate {3:16:auto 1}}}}`:`{7:11:def .test.x0} z-a {9:13 {7:13 {5:13 {=compound {3:13:word z} {=flag {3:15 {5:24 {7:24 {9:24 {1:9:word a}}}}}}}}}}`,
		`20 9:11:.test.0 $(.test.y0 $1) {7:13:delegate {5:11:def .test.y0} {=list {7:24:delegate {3:16:auto 1}}}}`:`{5:11:def .test.y0} z-a {7:13 {5:13 {=compound {3:13:word z} {=flag {3:15 {5:24 {7:24 {9:24 {1:9:word a}}}}}}}}}`,
		`20 9:11:.test.0 $(.test.z0 $1) {5:13:delegate {3:11:def .test.z0} {=list {5:24:delegate {3:16:auto 1}}}}`:`{3:11:def .test.z0} z-a {5:13 {=compound {3:13:word z} {=flag {3:15 {5:24 {7:24 {9:24 {1:9:word a}}}}}}}}`,
		`20 9:11:.test.0 $1 {3:15:delegate {3:16:auto 1}}`:`{3:11:def 1} a {3:15 {5:24 {7:24 {9:24 {1:9:word a}}}}}`,
		`20 9:11:.test.0 $1 {5:24:delegate {3:16:auto 1}}`:`{5:11:def 1} a {5:24 {7:24 {9:24 {1:9:word a}}}}`,
		`20 9:11:.test.0 $1 {7:24:delegate {3:16:auto 1}}`:`{7:11:def 1} a {7:24 {9:24 {1:9:word a}}}`,
		`20 9:11:.test.0 $1 {9:24:delegate {3:16:auto 1}}`:`{9:11:def 1} a {9:24 {1:9:word a}}`,
	},
}

var checkstrs_value_5 = map[string]map[string]any{
	"check-value-5_test.go": map[string]any{
		`18 9:11:.test.0 $(.test.x0 $1) {9:13:delegate {7:11:def .test.x0} {=list {9:24:delegate {3:16:auto 1}}}}`:`{9:13 {7:13 {5:13 {=compound {3:13:word z} {=flag {3:15 {5:24 {7:24 {9:24:null}}}}}}}}} `,
	},
}
