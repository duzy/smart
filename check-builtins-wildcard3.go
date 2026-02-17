//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints__wildcard3 = map[string]map[string]any{
	"loader.go": map[string]any{
		`$/ {4:45:delegate {1:1:def /}}`:testdata_f(`%[1]s/builtins/wildcard/3 {4:45 {=path %[3]s {1:1:raw builtins} {1:1:raw wildcard} {1:1:raw 3}}}`),
		`$(dir $/) {4:39:delegate {4:41:builtin dir} {=list {4:45:delegate {1:1:def /}}}}`:testdata_f(`%[1]s/builtins/wildcard {4:39 {=path %[3]s {4:41:raw builtins} {4:41:raw wildcard}}}`,line_column_s("4:41")),

		`7:6:val1 $(wildcard(-sort) foo/bar/v2.h foo/bar/v1.h) {7:10:delegate {7:12:builtin wildcard} [{=flag {7:22:word sort}}] {=list {=path {7:28:word foo} {7:32:word bar} {=compound {7:36:word v2} {7:38:punct .} {7:39:word h}}} {=path {7:41:word foo} {7:45:word bar} {=compound {7:49:word v1} {7:51:punct .} {7:52:word h}}}}}`:`{=file foo/bar/v1.h} {=file foo/bar/v2.h} {=list {7:10 {=file foo/bar/v1.h}} {7:10 {=file foo/bar/v2.h}}}`,

		`8:6:val2 $(wildcard(-sort) foo/bar/xyz???.h) {8:10:delegate {8:12:builtin wildcard} [{=flag {8:22:word sort}}] {=list {=path {8:28:word foo} {8:32:word bar} {=glob {8:36:word xyz} {8:39:meta ?} {8:40:meta ?} {8:41:meta ?} {8:42:punct .} {8:43:word h}}}}}`:`{} {8:10 {8:12:null}}`,

		`9:6:val3 $(wildcard(-sort) foo/bar/xyz???.txt) {9:10:delegate {9:12:builtin wildcard} [{=flag {9:22:word sort}}] {=list {=path {9:28:word foo} {9:32:word bar} {=glob {9:36:word xyz} {9:39:meta ?} {9:40:meta ?} {9:41:meta ?} {9:42:punct .} {9:43:word txt}}}}}`:`{=file foo/bar/xyz000.txt} {9:10 {=file foo/bar/xyz000.txt}}`,
	},
}

var checkstrs__wildcard3 = map[string]map[string]any{
	"loader.go": map[string]any{
		`0:0: $/ {4:45:delegate {1:1:def /}}`:testdata_f(`{4:45 {=path %[3]s {1:1:raw builtins} {1:1:raw wildcard} {1:1:raw 3}}} %[1]s/builtins/wildcard/3`),
	},
}
