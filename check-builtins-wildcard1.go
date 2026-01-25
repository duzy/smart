//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints__wildcard1 = map[string]map[string]any{
	"loader.go": map[string]any{
		`x.h {=compound {4:17:word x} {4:18:punct .} {4:19:word h}}`:`x.h {=compound {4:17:word x} {4:18:punct .} {4:19:word h}}`,
		`x.h {=compound {4:59:word x} {4:60:punct .} {4:61:word h}}`:`x.h {=compound {4:59:word x} {4:60:punct .} {4:61:word h}}`,

		`$/ {4:72:delegate {1:1:def /}}`:testdata_f(`%[1]s/builtins/wildcard/1 {4:72 {=path %[3]s {1:1:word builtins} {1:1:word wildcard} {1:1:word 1}}}`),

		`$(dir $/) {4:66:delegate {4:68:builtin dir} {=list {4:72:delegate {1:1:def /}}}}`:testdata_f(`%[1]s/builtins/wildcard {4:66 {=path %[3]s {4:68:word builtins} {4:68:word wildcard}}}`,line_column_s("4:68")),

		`7:6:val1 x.h {=compound {4:17:word x} {4:18:punct .} {4:19:word h}}`:`x.h {=compound {4:17:word x} {4:18:punct .} {4:19:word h}}`,
		`8:6:val2 x.h {=compound {4:17:word x} {4:18:punct .} {4:19:word h}}`:`x.h {=compound {4:17:word x} {4:18:punct .} {4:19:word h}}`,
		`9:6:val3 x.h {=compound {4:17:word x} {4:18:punct .} {4:19:word h}}`:`x.h {=compound {4:17:word x} {4:18:punct .} {4:19:word h}}`,
		`10:6:val4 x.h {=compound {4:17:word x} {4:18:punct .} {4:19:word h}}`:`x.h {=compound {4:17:word x} {4:18:punct .} {4:19:word h}}`,
		`11:6:val5 x.h {=compound {4:17:word x} {4:18:punct .} {4:19:word h}}`:`x.h {=compound {4:17:word x} {4:18:punct .} {4:19:word h}}`,
		`12:6:val6 x.h {=compound {4:17:word x} {4:18:punct .} {4:19:word h}}`:`x.h {=compound {4:17:word x} {4:18:punct .} {4:19:word h}}`,
		`13:6:val7 x.h {=compound {4:17:word x} {4:18:punct .} {4:19:word h}}`:`x.h {=compound {4:17:word x} {4:18:punct .} {4:19:word h}}`,
		`14:6:val8 x.h {=compound {4:17:word x} {4:18:punct .} {4:19:word h}}`:`x.h {=compound {4:17:word x} {4:18:punct .} {4:19:word h}}`,

		`7:6:val1 x.h {=compound {4:59:word x} {4:60:punct .} {4:61:word h}}`:`x.h {=compound {4:59:word x} {4:60:punct .} {4:61:word h}}`,
		`8:6:val2 x.h {=compound {4:59:word x} {4:60:punct .} {4:61:word h}}`:`x.h {=compound {4:59:word x} {4:60:punct .} {4:61:word h}}`,
		`9:6:val3 x.h {=compound {4:59:word x} {4:60:punct .} {4:61:word h}}`:`x.h {=compound {4:59:word x} {4:60:punct .} {4:61:word h}}`,
		`10:6:val4 x.h {=compound {4:59:word x} {4:60:punct .} {4:61:word h}}`:`x.h {=compound {4:59:word x} {4:60:punct .} {4:61:word h}}`,

		`7:6:val1 $(wildcard(-sort) *.h) {7:10:delegate {7:12:builtin wildcard} [{=flag {7:22:word sort}}] {=list {=glob {7:29:meta *} {7:30:punct .} {7:31:word h}}}}`:`{} {7:10 {7:12:null}}`,
		`8:6:val2 $(wildcard(-sort) **.h) {8:10:delegate {8:12:builtin wildcard} [{=flag {8:22:word sort}}] {=list {=glob {8:28:meta **} {8:30:punct .} {8:31:word h}}}}`:`{=file foo/bar/v1.h} {=file foo/bar/v2.h} {=file foo/bar/zz/x.h} {=file foo/v1.h} {=file foo/v2.h} {=file foo/xv1y.h} {=file foo/xv22y.h} {=file foo/xv333y.h} {=file foobar/x.h} {=list {8:10 {=file foo/bar/v1.h}} {8:10 {=file foo/bar/v2.h}} {8:10 {=file foo/bar/zz/x.h}} {8:10 {=file foo/v1.h}} {8:10 {=file foo/v2.h}} {8:10 {=file foo/xv1y.h}} {8:10 {=file foo/xv22y.h}} {8:10 {=file foo/xv333y.h}} {8:10 {=file foobar/x.h}}}`,
		`9:6:val3 $(wildcard(-sort) foo/*.h) {9:10:delegate {9:12:builtin wildcard} [{=flag {9:22:word sort}}] {=list {=path {9:28:word foo} {=glob {9:32:meta *} {9:33:punct .} {9:34:word h}}}}}`:`{=file foo/v1.h} {=file foo/v2.h} {=file foo/xv1y.h} {=file foo/xv22y.h} {=file foo/xv333y.h} {=list {9:10 {=file foo/v1.h}} {9:10 {=file foo/v2.h}} {9:10 {=file foo/xv1y.h}} {9:10 {=file foo/xv22y.h}} {9:10 {=file foo/xv333y.h}}}`,
		`10:6:val4 $(wildcard(-sort) foo/bar/*.h) {10:10:delegate {10:12:builtin wildcard} [{=flag {10:22:word sort}}] {=list {=path {10:28:word foo} {10:32:word bar} {=glob {10:36:meta *} {10:37:punct .} {10:38:word h}}}}}`:`{=file foo/bar/v1.h} {=file foo/bar/v2.h} {=list {10:10 {=file foo/bar/v1.h}} {10:10 {=file foo/bar/v2.h}}}`,
		`11:6:val5 $(wildcard(-sort) foo/bar/*/*.h) {11:10:delegate {11:12:builtin wildcard} [{=flag {11:22:word sort}}] {=list {=path {11:28:word foo} {11:32:word bar} {=glob {11:36:meta *}} {=glob {11:38:meta *} {11:39:punct .} {11:40:word h}}}}}`:`{=file foo/bar/zz/x.h} {11:10 {=file foo/bar/zz/x.h}}`,
		`12:6:val6 $(wildcard(-sort) foo/bar/z?/?.h) {12:10:delegate {12:12:builtin wildcard} [{=flag {12:22:word sort}}] {=list {=path {12:28:word foo} {12:32:word bar} {=glob {12:36:word z} {12:37:meta ?}} {=glob {12:39:meta ?} {12:40:punct .} {12:41:word h}}}}}`:`{=file foo/bar/zz/x.h} {12:10 {=file foo/bar/zz/x.h}}`,
		`13:6:val7 $(wildcard(-sort) foo/bar/zz/?.h) {13:10:delegate {13:12:builtin wildcard} [{=flag {13:22:word sort}}] {=list {=path {13:28:word foo} {13:32:word bar} {13:36:word zz} {=glob {13:39:meta ?} {13:40:punct .} {13:41:word h}}}}}`:`{=file foo/bar/zz/x.h} {13:10 {=file foo/bar/zz/x.h}}`,
		`14:6:val9 $(wildcard(-sort) foo/bar/v1.h foo/bar/v2.h) {14:10:delegate {14:12:builtin wildcard} [{=flag {14:22:word sort}}] {=list {=path {14:28:word foo} {14:32:word bar} {=compound {14:36:word v1} {14:38:punct .} {14:39:word h}}} {=path {14:41:word foo} {14:45:word bar} {=compound {14:49:word v2} {14:51:punct .} {14:52:word h}}}}}`:`{=file foo/bar/v1.h} {=file foo/bar/v2.h} {=list {14:10 {=file foo/bar/v1.h}} {14:10 {=file foo/bar/v2.h}}}`,
		`15:6:va10 $(wildcard(-sort) foo/bar/*.hh) {15:10:delegate {15:12:builtin wildcard} [{=flag {15:22:word sort}}] {=list {=path {15:28:word foo} {15:32:word bar} {=glob {15:36:meta *} {15:37:punct .} {15:38:word hh}}}}}`:`{=file foo/bar/v3.hh} {15:10 {=file foo/bar/v3.hh}}`,
	},
}

var checkstrs__wildcard1 = map[string]map[string]any{
	"loader.go": map[string]any{
		`0:0: $/ {4:72:delegate {1:1:def /}}`:testdata_f(`{4:72 {=path %[3]s {1:1:word builtins} {1:1:word wildcard} {1:1:word 1}}} %[1]s/builtins/wildcard/1`),
	},
}
