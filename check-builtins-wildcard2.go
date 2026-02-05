//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints__wildcard2 = map[string]map[string]any{
	"loader.go": map[string]any{
		`x.h {=compound {4:32:word x} {4:33:punct .} {4:34:word h}}`:`x.h {=compound {4:32:word x} {4:33:punct .} {4:34:word h}}`,
		`$/ {4:45:delegate {1:1:def /}}`:testdata_f(`%[1]s/builtins/wildcard/2 {4:45 {=path %[3]s {1:1:raw builtins} {1:1:raw wildcard} {1:1:raw 2}}}`),
		`$(dir $/) {4:39:delegate {4:41:builtin dir} {=list {4:45:delegate {1:1:def /}}}}`:testdata_f(`%[1]s/builtins/wildcard {4:39 {=path %[3]s {4:41:raw builtins} {4:41:raw wildcard}}}`,line_column_s("4:41")),
		`f*? {=compound {4:28:word f} {=glob {4:29:meta *?}}}`:`f*? {=compound {4:28:word f} {=glob {4:29:meta *?}}}`,

		`7:6:val1 $(wildcard(-sort) foo/xv1y.h foo/bar/v2.h foo/bar/v1.h) {7:10:delegate {7:12:builtin wildcard} [{=flag {7:22:word sort}}] {=list {=path {7:28:word foo} {=compound {7:32:word xv1y} {7:36:punct .} {7:37:word h}}} {=path {7:39:word foo} {7:43:word bar} {=compound {7:47:word v2} {7:49:punct .} {7:50:word h}}} {=path {7:52:word foo} {7:56:word bar} {=compound {7:60:word v1} {7:62:punct .} {7:63:word h}}}}}`:`{=file foo/bar/v1.h} {=file foo/bar/v2.h} {=file foo/xv1y.h} {=list {7:10 {=file foo/bar/v1.h}} {7:10 {=file foo/bar/v2.h}} {7:10 {=file foo/xv1y.h}}}`,

		`8:6:val2 x.h {=compound {4:32:word x} {4:33:punct .} {4:34:word h}}`:`x.h {=compound {4:32:word x} {4:33:punct .} {4:34:word h}}`,
		`8:6:val2 $(wildcard(-sort) foo/b?r/v?.h) {8:10:delegate {8:12:builtin wildcard} [{=flag {8:22:word sort}}] {=list {=path {8:28:word foo} {=glob {8:32:word b} {8:33:meta ?} {8:34:word r}} {=glob {8:36:word v} {8:37:meta ?} {8:38:punct .} {8:39:word h}}}}}`:`{=file foo/bar/v1.h} {=file foo/bar/v2.h} {=list {8:10 {=file foo/bar/v1.h}} {8:10 {=file foo/bar/v2.h}}}`,

		`9:6:val3 $(wildcard(-sort) foo/xv*y.h) {9:10:delegate {9:12:builtin wildcard} [{=flag {9:22:word sort}}] {=list {=path {9:28:word foo} {=glob {9:32:word xv} {9:34:meta *} {9:35:word y} {9:36:punct .} {9:37:word h}}}}}`:`{=file foo/xv1y.h} {=file foo/xv22y.h} {=file foo/xv333y.h} {=list {9:10 {=file foo/xv1y.h}} {9:10 {=file foo/xv22y.h}} {9:10 {=file foo/xv333y.h}}}`,

		`10:6:val4 x.h {=compound {4:32:word x} {4:33:punct .} {4:34:word h}}`:`x.h {=compound {4:32:word x} {4:33:punct .} {4:34:word h}}`,
		`10:6:val4 $(wildcard(-sort) foo/**/x.h) {10:10:delegate {10:12:builtin wildcard} [{=flag {10:22:word sort}}] {=list {=path {10:28:word foo} {=glob {10:32:meta **}} {=compound {10:35:word x} {10:36:punct .} {10:37:word h}}}}}`:`{} {10:10 {10:12:null}}`,

		`11:6:val5 x.h {=compound {4:32:word x} {4:33:punct .} {4:34:word h}}`:`x.h {=compound {4:32:word x} {4:33:punct .} {4:34:word h}}`,
		`11:6:val5 $(wildcard(-sort) fo?/**/x.h) {11:10:delegate {11:12:builtin wildcard} [{=flag {11:22:word sort}}] {=list {=path {=glob {11:28:word fo} {11:30:meta ?}} {=glob {11:32:meta **}} {=compound {11:35:word x} {11:36:punct .} {11:37:word h}}}}}`:`{} {11:10 {11:12:null}}`,
	},
}

var checkstrs__wildcard2 = map[string]map[string]any{
	"loader.go": map[string]any{
		`0:0: $/ {4:45:delegate {1:1:def /}}`:testdata_f(`{4:45 {=path %[3]s {1:1:raw builtins} {1:1:raw wildcard} {1:1:raw 2}}} %[1]s/builtins/wildcard/2`),
	},
}
