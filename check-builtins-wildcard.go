//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints__wildcard = map[string]map[string]any{
	"loader.go": map[string]any{
		`x.h {=compound {4:17:word x} {4:18:punct .} {4:19:word h}}`:`x.h {=compound {4:17:word x} {4:18:punct .} {4:19:word h}}`,

		`$/ {4:39:delegate {1:1:def /}}`:testdata_f(`%[1]s/builtins/wildcard {4:39 {=path %[3]s {1:1:raw builtins} {1:1:raw wildcard}}}`),

		`7:5:top $/ {7:8:delegate {1:1:def /}}`:testdata_f(`%[1]s/builtins/wildcard {7:8 {=path %[3]s {1:1:raw builtins} {1:1:raw wildcard}}}`),
		`8:5:inc $/ {8:8:delegate {1:1:def /}}`:testdata_f(`%[1]s/builtins/wildcard {8:8 {=path %[3]s {1:1:raw builtins} {1:1:raw wildcard}}}`),

		`17:6:val1 $/ {17:32:delegate {1:1:def /}}`:testdata_f(`%[1]s/builtins/wildcard {17:32 {=path %[3]s {1:1:raw builtins} {1:1:raw wildcard}}}`),
		`17:6:val1 $(wildcard(-sort -dir=$/) inc/*.h inc/*/*.h inc/*/*/*.h) {17:10:delegate {17:12:builtin wildcard} [{=flag {17:22:word sort}} {=pair {=flag {17:28:word dir}} {17:32:delegate {1:1:def /}}}] {=list {=path {17:36:word inc} {=glob {17:40:meta *} {17:41:punct .} {17:42:word h}}} {=path {17:44:word inc} {=glob {17:48:meta *}} {=glob {17:50:meta *} {17:51:punct .} {17:52:word h}}} {=path {17:54:word inc} {=glob {17:58:meta *}} {=glob {17:60:meta *}} {=glob {17:62:meta *} {17:63:punct .} {17:64:word h}}}}}`:`{=file inc/bar.h} {=file inc/foo.h} {=file inc/foo/bar/v1.h} {=file inc/foo/bar/v2.h} {=file inc/foo/v1.h} {=file inc/foo/v2.h} {=file inc/foo/xv1y.h} {=file inc/foo/xv22y.h} {=file inc/foo/xv333y.h} {=file inc/foobar/config/x.h} {=file inc/foobar/x.h} {=list {17:10 {=file inc/bar.h}} {17:10 {=file inc/foo.h}} {17:10 {=file inc/foo/bar/v1.h}} {17:10 {=file inc/foo/bar/v2.h}} {17:10 {=file inc/foo/v1.h}} {17:10 {=file inc/foo/v2.h}} {17:10 {=file inc/foo/xv1y.h}} {17:10 {=file inc/foo/xv22y.h}} {17:10 {=file inc/foo/xv333y.h}} {17:10 {=file inc/foobar/config/x.h}} {17:10 {=file inc/foobar/x.h}}}`,

		`18:6:val2 $/ {18:32:delegate {1:1:def /}}`:testdata_f(`%[1]s/builtins/wildcard {18:32 {=path %[3]s {1:1:raw builtins} {1:1:raw wildcard}}}`),
		`18:6:val2 $(wildcard(-sort -dir=$/) inc/**.h) {18:10:delegate {18:12:builtin wildcard} [{=flag {18:22:word sort}} {=pair {=flag {18:28:word dir}} {18:32:delegate {1:1:def /}}}] {=list {=path {18:36:word inc} {=glob {18:40:meta **} {18:42:punct .} {18:43:word h}}}}}`:`{=file inc/bar.h} {=file inc/foo.h} {=file inc/foo/bar/v1.h} {=file inc/foo/bar/v2.h} {=file inc/foo/bar/zz/x.h} {=file inc/foo/v1.h} {=file inc/foo/v2.h} {=file inc/foo/xv1y.h} {=file inc/foo/xv22y.h} {=file inc/foo/xv333y.h} {=file inc/foobar/config/x.h} {=file inc/foobar/x.h} {=list {18:10 {=file inc/bar.h}} {18:10 {=file inc/foo.h}} {18:10 {=file inc/foo/bar/v1.h}} {18:10 {=file inc/foo/bar/v2.h}} {18:10 {=file inc/foo/bar/zz/x.h}} {18:10 {=file inc/foo/v1.h}} {18:10 {=file inc/foo/v2.h}} {18:10 {=file inc/foo/xv1y.h}} {18:10 {=file inc/foo/xv22y.h}} {18:10 {=file inc/foo/xv333y.h}} {18:10 {=file inc/foobar/config/x.h}} {18:10 {=file inc/foobar/x.h}}}`,

		`19:6:val3 $/ {19:32:delegate {1:1:def /}}`:testdata_f(`%[1]s/builtins/wildcard {19:32 {=path %[3]s {1:1:raw builtins} {1:1:raw wildcard}}}`),
		`19:6:val3 $(wildcard(-sort -dir=$//inc) **.h) {19:10:delegate {19:12:builtin wildcard} [{=flag {19:22:word sort}} {=pair {=flag {19:28:word dir}} {=path {19:32:delegate {1:1:def /}} {19:35:word inc}}}] {=list {=glob {19:40:meta **} {19:42:punct .} {19:43:word h}}}}`:`{=file bar.h} {=file foo.h} {=file foo/bar/v1.h} {=file foo/bar/v2.h} {=file foo/bar/zz/x.h} {=file foo/v1.h} {=file foo/v2.h} {=file foo/xv1y.h} {=file foo/xv22y.h} {=file foo/xv333y.h} {=file foobar/config/x.h} {=file foobar/x.h} {=list {19:10 {=file bar.h}} {19:10 {=file foo.h}} {19:10 {=file foo/bar/v1.h}} {19:10 {=file foo/bar/v2.h}} {19:10 {=file foo/bar/zz/x.h}} {19:10 {=file foo/v1.h}} {19:10 {=file foo/v2.h}} {19:10 {=file foo/xv1y.h}} {19:10 {=file foo/xv22y.h}} {19:10 {=file foo/xv333y.h}} {19:10 {=file foobar/config/x.h}} {19:10 {=file foobar/x.h}}}`,

		`20:6:val4 x.h {=compound {4:17:word x} {4:18:punct .} {4:19:word h}}`:`x.h {=compound {4:17:word x} {4:18:punct .} {4:19:word h}}`,
		`20:6:val4 $(wildcard(-sort) **.h) {20:10:delegate {20:12:builtin wildcard} [{=flag {20:22:word sort}}] {=list {=glob {20:28:meta **} {20:30:punct .} {20:31:word h}}}}`:`{=file bar.h} {=file foo.h} {=file foo/bar/v1.h} {=file foo/bar/v2.h} {=file foo/bar/zz/x.h} {=file foo/v1.h} {=file foo/v2.h} {=file foo/xv1y.h} {=file foo/xv22y.h} {=file foo/xv333y.h} {=file foobar/config/x.h} {=file foobar/x.h} {=list {20:10 {=file bar.h}} {20:10 {=file foo.h}} {20:10 {=file foo/bar/v1.h}} {20:10 {=file foo/bar/v2.h}} {20:10 {=file foo/bar/zz/x.h}} {20:10 {=file foo/v1.h}} {20:10 {=file foo/v2.h}} {20:10 {=file foo/xv1y.h}} {20:10 {=file foo/xv22y.h}} {20:10 {=file foo/xv333y.h}} {20:10 {=file foobar/config/x.h}} {20:10 {=file foobar/x.h}}}`,

		`21:6:val5 x.h {=compound {4:17:word x} {4:18:punct .} {4:19:word h}}`:`x.h {=compound {4:17:word x} {4:18:punct .} {4:19:word h}}`,
		`21:6:val5 $(wildcard(-sort) *.h) {21:10:delegate {21:12:builtin wildcard} [{=flag {21:22:word sort}}] {=list {=glob {21:29:meta *} {21:30:punct .} {21:31:word h}}}}`:`{=file bar.h} {=file foo.h} {=list {21:10 {=file bar.h}} {21:10 {=file foo.h}}}`,

		`24:6:fix1 x.h {=compound {4:17:word x} {4:18:punct .} {4:19:word h}}`:`x.h {=compound {4:17:word x} {4:18:punct .} {4:19:word h}}`,
		`24:6:fix1 $(wildcard(-sort) foobar/config/*.def.am) {24:10:delegate {24:12:builtin wildcard} [{=flag {24:22:word sort}}] {=list {=path {24:40:word foobar} {24:47:word config} {=glob {24:54:meta *} {24:55:punct .} {24:56:word def} {24:59:punct .} {24:60:word am}}}}}`:`{=file foobar/config/a.def.am} {24:10 {=file foobar/config/a.def.am}}`,

		`25:6:fix2 x.h {=compound {4:17:word x} {4:18:punct .} {4:19:word h}}`:`x.h {=compound {4:17:word x} {4:18:punct .} {4:19:word h}}`,
		`25:6:fix2 $(wildcard(-sort) foobar/config/*.def.in) {25:10:delegate {25:12:builtin wildcard} [{=flag {25:22:word sort}}] {=list {=path {25:40:word foobar} {25:47:word config} {=glob {25:54:meta *} {25:55:punct .} {25:56:word def} {25:59:punct .} {25:60:word in}}}}}`:`{} {25:10 {25:12:null}}`,

		`26:6:fix3 $/ {26:32:delegate {1:1:def /}}`:testdata_f(`%[1]s/builtins/wildcard {26:32 {=path %[3]s {1:1:raw builtins} {1:1:raw wildcard}}}`),
		`26:6:fix3 $(wildcard(-sort -dir=$//inc) foobar/config/*.def.am) {26:10:delegate {26:12:builtin wildcard} [{=flag {26:22:word sort}} {=pair {=flag {26:28:word dir}} {=path {26:32:delegate {1:1:def /}} {26:35:word inc}}}] {=list {=path {26:40:word foobar} {26:47:word config} {=glob {26:54:meta *} {26:55:punct .} {26:56:word def} {26:59:punct .} {26:60:word am}}}}}`:`{=file foobar/config/a.def.am} {26:10 {=file foobar/config/a.def.am}}`,

		`27:6:fix4 $/ {27:32:delegate {1:1:def /}}`:testdata_f(`%[1]s/builtins/wildcard {27:32 {=path %[3]s {1:1:raw builtins} {1:1:raw wildcard}}}`),
		`27:6:fix4 $(wildcard(-sort -dir=$//inc) foobar/config/*.def.in) {27:10:delegate {27:12:builtin wildcard} [{=flag {27:22:word sort}} {=pair {=flag {27:28:word dir}} {=path {27:32:delegate {1:1:def /}} {27:35:word inc}}}] {=list {=path {27:40:word foobar} {27:47:word config} {=glob {27:54:meta *} {27:55:punct .} {27:56:word def} {27:59:punct .} {27:60:word in}}}}}`:`{=file foobar/config/a.def.in} {=file foobar/config/b.def.in} {=list {27:10 {=file foobar/config/a.def.in}} {27:10 {=file foobar/config/b.def.in}}}`,
	},
}

var checkstrs__wildcard = map[string]map[string]any{
	"loader.go": map[string]any{
		`17:6:val1 $/ {17:32:delegate {1:1:def /}}`:testdata_f(`{17:32 {=path %[3]s {1:1:raw builtins} {1:1:raw wildcard}}} %[1]s/builtins/wildcard`),
		`18:6:val2 $/ {18:32:delegate {1:1:def /}}`:testdata_f(`{18:32 {=path %[3]s {1:1:raw builtins} {1:1:raw wildcard}}} %[1]s/builtins/wildcard`),
		`19:6:val3 $/ {19:32:delegate {1:1:def /}}`:testdata_f(`{19:32 {=path %[3]s {1:1:raw builtins} {1:1:raw wildcard}}} %[1]s/builtins/wildcard`),
		`26:6:fix3 $/ {26:32:delegate {1:1:def /}}`:testdata_f(`{26:32 {=path %[3]s {1:1:raw builtins} {1:1:raw wildcard}}} %[1]s/builtins/wildcard`),
		`27:6:fix4 $/ {27:32:delegate {1:1:def /}}`:testdata_f(`{27:32 {=path %[3]s {1:1:raw builtins} {1:1:raw wildcard}}} %[1]s/builtins/wildcard`),
	},
}
