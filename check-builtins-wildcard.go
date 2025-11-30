//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints__wildcard = map[string]map[string]any{
	"loader.go": map[string]any{
		`$/ {4:24:delegate {1:1:def /}}`:`{1:1:def /} /Volumes/workspace/go/src/extbit.io/smart/testdata/builtins/wildcard {4:24 {=path {1:1:punct ROOT} {1:1:word Volumes} {1:1:word workspace} {1:1:word go} {1:1:word src} {1:1:word extbit.io} {1:1:word smart} {1:1:word testdata} {1:1:word builtins} {1:1:word wildcard}}}`,

		`7:5:top $/ {7:8:delegate {1:1:def /}}`:`{1:1:def /} /Volumes/workspace/go/src/extbit.io/smart/testdata/builtins/wildcard {7:8 {=path {1:1:punct ROOT} {1:1:word Volumes} {1:1:word workspace} {1:1:word go} {1:1:word src} {1:1:word extbit.io} {1:1:word smart} {1:1:word testdata} {1:1:word builtins} {1:1:word wildcard}}}`,
		`8:5:inc $/ {8:8:delegate {1:1:def /}}`:`{1:1:def /} /Volumes/workspace/go/src/extbit.io/smart/testdata/builtins/wildcard {8:8 {=path {1:1:punct ROOT} {1:1:word Volumes} {1:1:word workspace} {1:1:word go} {1:1:word src} {1:1:word extbit.io} {1:1:word smart} {1:1:word testdata} {1:1:word builtins} {1:1:word wildcard}}}`,

		`17:6:val1 $/ {17:26:delegate {1:1:def /}}`:`{1:1:def /} /Volumes/workspace/go/src/extbit.io/smart/testdata/builtins/wildcard {17:26 {=path {1:1:punct ROOT} {1:1:word Volumes} {1:1:word workspace} {1:1:word go} {1:1:word src} {1:1:word extbit.io} {1:1:word smart} {1:1:word testdata} {1:1:word builtins} {1:1:word wildcard}}}`,
		`17:6:val1 $(wildcard(-dir=$/) inc/*.h inc/*/*.h inc/*/*/*.h) {17:10:delegate {17:12:builtin wildcard} [{=pair {=flag {17:22:word dir}} {17:26:delegate {1:1:def /}}}] {=list {=path {17:30:word inc} {=glob {17:34:meta *} {17:35:punct .} {17:36:word h}}} {=path {17:38:word inc} {=glob {17:42:meta *}} {=glob {17:44:meta *} {17:45:punct .} {17:46:word h}}} {=path {17:48:word inc} {=glob {17:52:meta *}} {=glob {17:54:meta *}} {=glob {17:56:meta *} {17:57:punct .} {17:58:word h}}}}}`:sorted_strings{
			`{17:12:builtin wildcard} {=file inc/bar.h} {=file inc/foo.h} {=file inc/foo/bar/v1.h} {=file inc/foo/bar/v2.h} {=file inc/foo/v1.h} {=file inc/foo/v2.h} {=list {17:10 {=file inc/bar.h}} {17:10 {=file inc/foo.h}} {17:10 {=file inc/foo/bar/v1.h}} {17:10 {=file inc/foo/bar/v2.h}} {17:10 {=file inc/foo/v1.h}} {17:10 {=file inc/foo/v2.h}}}`,
		},

		`18:6:val2 $/ {18:26:delegate {1:1:def /}}`:`{1:1:def /} /Volumes/workspace/go/src/extbit.io/smart/testdata/builtins/wildcard {18:26 {=path {1:1:punct ROOT} {1:1:word Volumes} {1:1:word workspace} {1:1:word go} {1:1:word src} {1:1:word extbit.io} {1:1:word smart} {1:1:word testdata} {1:1:word builtins} {1:1:word wildcard}}}`,
		`18:6:val2 $(wildcard(-dir=$/) inc/**.h) {18:10:delegate {18:12:builtin wildcard} [{=pair {=flag {18:22:word dir}} {18:26:delegate {1:1:def /}}}] {=list {=path {18:30:word inc} {=glob {18:34:meta **} {18:36:punct .} {18:37:word h}}}}}`:sorted_strings{
			`{18:12:builtin wildcard} {=file inc/bar.h} {=file inc/foo.h} {=file inc/foo/bar/v1.h} {=file inc/foo/bar/v2.h} {=file inc/foo/bar/zz/x.h} {=file inc/foo/v1.h} {=file inc/foo/v2.h} {=list {18:10 {=file inc/bar.h}} {18:10 {=file inc/foo.h}} {18:10 {=file inc/foo/bar/v1.h}} {18:10 {=file inc/foo/bar/v2.h}} {18:10 {=file inc/foo/bar/zz/x.h}} {18:10 {=file inc/foo/v1.h}} {18:10 {=file inc/foo/v2.h}}}`,
		},

		`19:6:val3 $/ {19:26:delegate {1:1:def /}}`:`{1:1:def /} /Volumes/workspace/go/src/extbit.io/smart/testdata/builtins/wildcard {19:26 {=path {1:1:punct ROOT} {1:1:word Volumes} {1:1:word workspace} {1:1:word go} {1:1:word src} {1:1:word extbit.io} {1:1:word smart} {1:1:word testdata} {1:1:word builtins} {1:1:word wildcard}}}`,
		`19:6:val3 $(wildcard(-dir=$//inc) **.h) {19:10:delegate {19:12:builtin wildcard} [{=pair {=flag {19:22:word dir}} {=path {19:26:delegate {1:1:def /}} {19:29:word inc}}}] {=list {=glob {19:34:meta **} {19:36:punct .} {19:37:word h}}}}`:sorted_strings{
			`{19:12:builtin wildcard} {=file bar.h} {=file foo.h} {=file foo/bar/v1.h} {=file foo/bar/v2.h} {=file foo/bar/zz/x.h} {=file foo/v1.h} {=file foo/v2.h} {=list {19:10 {=file bar.h}} {19:10 {=file foo.h}} {19:10 {=file foo/bar/v1.h}} {19:10 {=file foo/bar/v2.h}} {19:10 {=file foo/bar/zz/x.h}} {19:10 {=file foo/v1.h}} {19:10 {=file foo/v2.h}}}`,
		},

		`20:6:val4 $(wildcard **.h) {20:10:delegate {20:12:builtin wildcard} {=list {=glob {20:21:meta **} {20:23:punct .} {20:24:word h}}}}`:sorted_strings{
			`{20:12:builtin wildcard} {=file bar.h} {=file foo.h} {=file foo/bar/v1.h} {=file foo/bar/v2.h} {=file foo/bar/zz/x.h} {=file foo/v1.h} {=file foo/v2.h} {=list {20:10 {=file bar.h}} {20:10 {=file foo.h}} {20:10 {=file foo/bar/v1.h}} {20:10 {=file foo/bar/v2.h}} {20:10 {=file foo/bar/zz/x.h}} {20:10 {=file foo/v1.h}} {20:10 {=file foo/v2.h}}}`,
		},
		`21:6:val5 $(wildcard *.h) {21:10:delegate {21:12:builtin wildcard} {=list {=glob {21:22:meta *} {21:23:punct .} {21:24:word h}}}}`:sorted_strings{
			`{21:12:builtin wildcard} {=file bar.h} {=file foo.h} {=list {21:10 {=file bar.h}} {21:10 {=file foo.h}}}`,
		},
		`24:6:fix1 $(wildcard foobar/config/*.def.am) {24:10:delegate {24:12:builtin wildcard} {=list {=path {24:34:word foobar} {24:41:word config} {=glob {24:48:meta *} {24:49:punct .} {24:50:word def} {24:53:punct .} {24:54:word am}}}}}`:[]string{
			`{24:12:builtin wildcard} {=file foobar/config/a.def.am} {24:10 {=file foobar/config/a.def.am}}`,
		},
		`25:6:fix2 $(wildcard foobar/config/*.def.in) {25:10:delegate {25:12:builtin wildcard} {=list {=path {25:34:word foobar} {25:41:word config} {=glob {25:48:meta *} {25:49:punct .} {25:50:word def} {25:53:punct .} {25:54:word in}}}}}`:[]string{
			`{25:12:builtin wildcard} {} {25:10 {25:12:null}}`,
		},

		`26:6:fix3 $/ {26:26:delegate {1:1:def /}}`:`{1:1:def /} /Volumes/workspace/go/src/extbit.io/smart/testdata/builtins/wildcard {26:26 {=path {1:1:punct ROOT} {1:1:word Volumes} {1:1:word workspace} {1:1:word go} {1:1:word src} {1:1:word extbit.io} {1:1:word smart} {1:1:word testdata} {1:1:word builtins} {1:1:word wildcard}}}`,
		`26:6:fix3 $(wildcard(-dir=$//inc) foobar/config/*.def.am) {26:10:delegate {26:12:builtin wildcard} [{=pair {=flag {26:22:word dir}} {=path {26:26:delegate {1:1:def /}} {26:29:word inc}}}] {=list {=path {26:34:word foobar} {26:41:word config} {=glob {26:48:meta *} {26:49:punct .} {26:50:word def} {26:53:punct .} {26:54:word am}}}}}`:`{26:12:builtin wildcard} {=file foobar/config/a.def.am} {26:10 {=file foobar/config/a.def.am}}`,

		`27:6:fix4 $/ {27:26:delegate {1:1:def /}}`:`{1:1:def /} /Volumes/workspace/go/src/extbit.io/smart/testdata/builtins/wildcard {27:26 {=path {1:1:punct ROOT} {1:1:word Volumes} {1:1:word workspace} {1:1:word go} {1:1:word src} {1:1:word extbit.io} {1:1:word smart} {1:1:word testdata} {1:1:word builtins} {1:1:word wildcard}}}`,
		`27:6:fix4 $(wildcard(-dir=$//inc) foobar/config/*.def.in) {27:10:delegate {27:12:builtin wildcard} [{=pair {=flag {27:22:word dir}} {=path {27:26:delegate {1:1:def /}} {27:29:word inc}}}] {=list {=path {27:34:word foobar} {27:41:word config} {=glob {27:48:meta *} {27:49:punct .} {27:50:word def} {27:53:punct .} {27:54:word in}}}}}`:[]string{
			`{27:12:builtin wildcard} {=file foobar/config/a.def.in} {=file foobar/config/b.def.in} {=list {27:10 {=file foobar/config/a.def.in}} {27:10 {=file foobar/config/b.def.in}}}`,
			`{27:12:builtin wildcard} {=file foobar/config/b.def.in} {=file foobar/config/a.def.in} {=list {27:10 {=file foobar/config/b.def.in}} {27:10 {=file foobar/config/a.def.in}}}`,
		},
	},
}

var checkstrs__wildcard = map[string]map[string]any{
	"loader.go": map[string]any{
		`17:6:val1 $/ {17:26:delegate {1:1:def /}}`:`{17:26 {=path {1:1:punct ROOT} {1:1:word Volumes} {1:1:word workspace} {1:1:word go} {1:1:word src} {1:1:word extbit.io} {1:1:word smart} {1:1:word testdata} {1:1:word builtins} {1:1:word wildcard}}} /Volumes/workspace/go/src/extbit.io/smart/testdata/builtins/wildcard`,
		`18:6:val2 $/ {18:26:delegate {1:1:def /}}`:`{18:26 {=path {1:1:punct ROOT} {1:1:word Volumes} {1:1:word workspace} {1:1:word go} {1:1:word src} {1:1:word extbit.io} {1:1:word smart} {1:1:word testdata} {1:1:word builtins} {1:1:word wildcard}}} /Volumes/workspace/go/src/extbit.io/smart/testdata/builtins/wildcard`,
		`19:6:val3 $/ {19:26:delegate {1:1:def /}}`:`{19:26 {=path {1:1:punct ROOT} {1:1:word Volumes} {1:1:word workspace} {1:1:word go} {1:1:word src} {1:1:word extbit.io} {1:1:word smart} {1:1:word testdata} {1:1:word builtins} {1:1:word wildcard}}} /Volumes/workspace/go/src/extbit.io/smart/testdata/builtins/wildcard`,
		`26:6:fix3 $/ {26:26:delegate {1:1:def /}}`:`{26:26 {=path {1:1:punct ROOT} {1:1:word Volumes} {1:1:word workspace} {1:1:word go} {1:1:word src} {1:1:word extbit.io} {1:1:word smart} {1:1:word testdata} {1:1:word builtins} {1:1:word wildcard}}} /Volumes/workspace/go/src/extbit.io/smart/testdata/builtins/wildcard`,
		`27:6:fix4 $/ {27:26:delegate {1:1:def /}}`:`{27:26 {=path {1:1:punct ROOT} {1:1:word Volumes} {1:1:word workspace} {1:1:word go} {1:1:word src} {1:1:word extbit.io} {1:1:word smart} {1:1:word testdata} {1:1:word builtins} {1:1:word wildcard}}} /Volumes/workspace/go/src/extbit.io/smart/testdata/builtins/wildcard`,
	},
}
