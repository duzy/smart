//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints_valcache = map[string]map[string]any{
	"loader.go": map[string]any{
		`.deps {=compound {4:17:punct .} {4:18:word deps}}`:`.deps {=compound {4:17:punct .} {4:18:word deps}}`,
		`.tmp {=compound {4:53:punct .} {4:54:word tmp}}`:`.tmp {=compound {4:53:punct .} {4:54:word tmp}}`,
		`bar.c {=compound {5:10:word bar} {5:13:punct .} {5:14:word c}}`:`bar.c {=compound {5:10:word bar} {5:13:punct .} {5:14:word c}}`,
		`bar.c++ {=compound {5:20:word bar} {5:23:punct .} {5:24:word c++}}`:`bar.c++ {=compound {5:20:word bar} {5:23:punct .} {5:24:word c++}}`,
		`foo.c {=compound {5:28:word foo} {5:31:punct .} {5:32:word c}}`:`foo.c {=compound {5:28:word foo} {5:31:punct .} {5:32:word c}}`,
		`foo.c++ {=compound {5:34:word foo} {5:37:punct .} {5:38:word c++}}`:`foo.c++ {=compound {5:34:word foo} {5:37:punct .} {5:38:word c++}}`,

		`&(gen) {4:40:closure {4:42:word gen}}`:`&(gen) {4:40:closure {4:42:word gen}}`,

		`$/ {4:50:delegate {1:1:def /}}`:testdata_f(`%[1]s/valcache {4:50 {=path %[3]s {1:1:word valcache}}}`),
		`$/ {5:45:delegate {1:1:def /}}`:testdata_f(`%[1]s/valcache {5:45 {=path %[3]s {1:1:word valcache}}}`),

		`10:6:val2 foo.c++ {=compound {10:9:word foo} {10:12:punct .} {10:13:word c++}}`:`foo.c++ {=compound {10:9:word foo} {10:12:punct .} {10:13:word c++}}`,
		`11:6:val3 foo.o {=compound {11:9:word foo} {11:12:punct .} {11:13:word o}}`:`foo.o {=compound {11:9:word foo} {11:12:punct .} {11:13:word o}}`,
		`12:6:val4 .deps {=compound {12:9:punct .} {12:10:word deps}}`:`.deps {=compound {12:9:punct .} {12:10:word deps}}`,

		`16:5:gen x.gen {=compound {16:8:word x} {16:9:punct .} {16:10:word gen}}`:`x.gen {=compound {16:8:word x} {16:9:punct .} {16:10:word gen}}`,
		`16:5:gen y.gen {=compound {16:14:word y} {16:15:punct .} {16:16:word gen}}`:`y.gen {=compound {16:14:word y} {16:15:punct .} {16:16:word gen}}`,
		`16:5:gen z.gen {=compound {16:24:word z} {16:25:punct .} {16:26:word gen}}`:`z.gen {=compound {16:24:word z} {16:25:punct .} {16:26:word gen}}`,

		`17:10:nexists1 .tmp {=compound {4:53:punct .} {4:54:word tmp}}`:`.tmp {=compound {4:53:punct .} {4:54:word tmp}}`,
		`17:10:nexists1 .deps {=compound {4:17:punct .} {4:18:word deps}}`:`.deps {=compound {4:17:punct .} {4:18:word deps}}`,
		`17:10:nexists1 x.gen {=compound {16:8:word x} {16:9:punct .} {16:10:word gen}}`:`x.gen {=compound {16:8:word x} {16:9:punct .} {16:10:word gen}}`,
		`17:10:nexists1 y.gen {=compound {16:14:word y} {16:15:punct .} {16:16:word gen}}`:`y.gen {=compound {16:14:word y} {16:15:punct .} {16:16:word gen}}`,
		`17:10:nexists1 z.gen {=compound {16:24:word z} {16:25:punct .} {16:26:word gen}}`:`z.gen {=compound {16:24:word z} {16:25:punct .} {16:26:word gen}}`,
		`17:10:nexists1 &(gen) {4:40:closure {4:42:word gen}}`:`x.gen y.gen foo/z.gen {=list {4:40 {=compound {16:8:word x} {16:9:punct .} {16:10:word gen}}} {4:40 {=compound {16:14:word y} {16:15:punct .} {16:16:word gen}}} {4:40 {=path {16:20:word foo} {=compound {16:24:word z} {16:25:punct .} {16:26:word gen}}}}}`,
		`17:10:nexists1 bar.c {=compound {5:10:word bar} {5:13:punct .} {5:14:word c}}`:`bar.c {=compound {5:10:word bar} {5:13:punct .} {5:14:word c}}`,
		`17:10:nexists1 bar.c++ {=compound {5:20:word bar} {5:23:punct .} {5:24:word c++}}`:`bar.c++ {=compound {5:20:word bar} {5:23:punct .} {5:24:word c++}}`,
		`17:10:nexists1 foo.c {=compound {5:28:word foo} {5:31:punct .} {5:32:word c}}`:`foo.c {=compound {5:28:word foo} {5:31:punct .} {5:32:word c}}`,
		`17:10:nexists1 foo.c++ {=compound {5:34:word foo} {5:37:punct .} {5:38:word c++}}`:`foo.c++ {=compound {5:34:word foo} {5:37:punct .} {5:38:word c++}}`,
		`17:10:nexists1 $(wildcard(-sort -missing) *.gen) {17:13:delegate {17:15:builtin wildcard} [{=flag {17:25:word sort}} {=flag {17:31:word missing}}] {=list {=glob {17:40:meta *} {17:41:punct .} {17:42:word gen}}}}`:`{=file x.gen} {=file y.gen} {=list {17:13 {=file x.gen}} {17:13 {=file y.gen}}}`,

		`18:10:_exists1 .tmp {=compound {4:53:punct .} {4:54:word tmp}}`:`.tmp {=compound {4:53:punct .} {4:54:word tmp}}`,
		`18:10:_exists1 .deps {=compound {4:17:punct .} {4:18:word deps}}`:`.deps {=compound {4:17:punct .} {4:18:word deps}}`,
		`18:10:_exists1 x.gen {=compound {16:8:word x} {16:9:punct .} {16:10:word gen}}`:`x.gen {=compound {16:8:word x} {16:9:punct .} {16:10:word gen}}`,
		`18:10:_exists1 y.gen {=compound {16:14:word y} {16:15:punct .} {16:16:word gen}}`:`y.gen {=compound {16:14:word y} {16:15:punct .} {16:16:word gen}}`,
		`18:10:_exists1 z.gen {=compound {16:24:word z} {16:25:punct .} {16:26:word gen}}`:`z.gen {=compound {16:24:word z} {16:25:punct .} {16:26:word gen}}`,
		`18:10:_exists1 &(gen) {4:40:closure {4:42:word gen}}`:`x.gen y.gen foo/z.gen {=list {4:40 {=compound {16:8:word x} {16:9:punct .} {16:10:word gen}}} {4:40 {=compound {16:14:word y} {16:15:punct .} {16:16:word gen}}} {4:40 {=path {16:20:word foo} {=compound {16:24:word z} {16:25:punct .} {16:26:word gen}}}}}`,
		`18:10:_exists1 bar.c {=compound {5:10:word bar} {5:13:punct .} {5:14:word c}}`:`bar.c {=compound {5:10:word bar} {5:13:punct .} {5:14:word c}}`,
		`18:10:_exists1 bar.c++ {=compound {5:20:word bar} {5:23:punct .} {5:24:word c++}}`:`bar.c++ {=compound {5:20:word bar} {5:23:punct .} {5:24:word c++}}`,
		`18:10:_exists1 foo.c {=compound {5:28:word foo} {5:31:punct .} {5:32:word c}}`:`foo.c {=compound {5:28:word foo} {5:31:punct .} {5:32:word c}}`,
		`18:10:_exists1 foo.c++ {=compound {5:34:word foo} {5:37:punct .} {5:38:word c++}}`:`foo.c++ {=compound {5:34:word foo} {5:37:punct .} {5:38:word c++}}`,
		`18:10:_exists1 $(wildcard(-sort) *.gen) {18:13:delegate {18:15:builtin wildcard} [{=flag {18:25:word sort}}] {=list {=glob {18:40:meta *} {18:41:punct .} {18:42:word gen}}}}`:`{=file x.gen} {18:13 {=file x.gen}}`,

		`16:5:gen a.gen {=compound {20:8:word a} {20:9:punct .} {20:10:word gen}}`:`a.gen {=compound {20:8:word a} {20:9:punct .} {20:10:word gen}}`,
		`16:5:gen b.gen {=compound {20:14:word b} {20:15:punct .} {20:16:word gen}}`:`b.gen {=compound {20:14:word b} {20:15:punct .} {20:16:word gen}}`,
		`16:5:gen c.gen {=compound {20:24:word c} {20:25:punct .} {20:26:word gen}}`:`c.gen {=compound {20:24:word c} {20:25:punct .} {20:26:word gen}}`,

		`21:10:nexists2 .tmp {=compound {4:53:punct .} {4:54:word tmp}}`:`.tmp {=compound {4:53:punct .} {4:54:word tmp}}`,
		`21:10:nexists2 .deps {=compound {4:17:punct .} {4:18:word deps}}`:`.deps {=compound {4:17:punct .} {4:18:word deps}}`,
		`21:10:nexists2 a.gen {=compound {20:8:word a} {20:9:punct .} {20:10:word gen}}`:`a.gen {=compound {20:8:word a} {20:9:punct .} {20:10:word gen}}`,
		`21:10:nexists2 b.gen {=compound {20:14:word b} {20:15:punct .} {20:16:word gen}}`:`b.gen {=compound {20:14:word b} {20:15:punct .} {20:16:word gen}}`,
		`21:10:nexists2 c.gen {=compound {20:24:word c} {20:25:punct .} {20:26:word gen}}`:`c.gen {=compound {20:24:word c} {20:25:punct .} {20:26:word gen}}`,
		`21:10:nexists2 &(gen) {4:40:closure {4:42:word gen}}`:`a.gen b.gen foo/c.gen {=list {4:40 {=compound {20:8:word a} {20:9:punct .} {20:10:word gen}}} {4:40 {=compound {20:14:word b} {20:15:punct .} {20:16:word gen}}} {4:40 {=path {20:20:word foo} {=compound {20:24:word c} {20:25:punct .} {20:26:word gen}}}}}`,
		`21:10:nexists2 bar.c {=compound {5:10:word bar} {5:13:punct .} {5:14:word c}}`:`bar.c {=compound {5:10:word bar} {5:13:punct .} {5:14:word c}}`,
		`21:10:nexists2 bar.c++ {=compound {5:20:word bar} {5:23:punct .} {5:24:word c++}}`:`bar.c++ {=compound {5:20:word bar} {5:23:punct .} {5:24:word c++}}`,
		`21:10:nexists2 foo.c {=compound {5:28:word foo} {5:31:punct .} {5:32:word c}}`:`foo.c {=compound {5:28:word foo} {5:31:punct .} {5:32:word c}}`,
		`21:10:nexists2 foo.c++ {=compound {5:34:word foo} {5:37:punct .} {5:38:word c++}}`:`foo.c++ {=compound {5:34:word foo} {5:37:punct .} {5:38:word c++}}`,
		`21:10:nexists2 $(wildcard(-sort -missing) **.gen) {21:13:delegate {21:15:builtin wildcard} [{=flag {21:25:word sort}} {=flag {21:31:word missing}}] {=list {=glob {21:40:meta **} {21:42:punct .} {21:43:word gen}}}}`:`{=file a.gen} {=file b.gen} {=file foo/c.gen} {=list {21:13 {=file a.gen}} {21:13 {=file b.gen}} {21:13 {=file foo/c.gen}}}`,

		`22:10:_exists2 .tmp {=compound {4:53:punct .} {4:54:word tmp}}`:`.tmp {=compound {4:53:punct .} {4:54:word tmp}}`,
		`22:10:_exists2 .deps {=compound {4:17:punct .} {4:18:word deps}}`:`.deps {=compound {4:17:punct .} {4:18:word deps}}`,
		`22:10:_exists2 a.gen {=compound {20:8:word a} {20:9:punct .} {20:10:word gen}}`:`a.gen {=compound {20:8:word a} {20:9:punct .} {20:10:word gen}}`,
		`22:10:_exists2 b.gen {=compound {20:14:word b} {20:15:punct .} {20:16:word gen}}`:`b.gen {=compound {20:14:word b} {20:15:punct .} {20:16:word gen}}`,
		`22:10:_exists2 c.gen {=compound {20:24:word c} {20:25:punct .} {20:26:word gen}}`:`c.gen {=compound {20:24:word c} {20:25:punct .} {20:26:word gen}}`,
		`22:10:_exists2 &(gen) {4:40:closure {4:42:word gen}}`:`a.gen b.gen foo/c.gen {=list {4:40 {=compound {20:8:word a} {20:9:punct .} {20:10:word gen}}} {4:40 {=compound {20:14:word b} {20:15:punct .} {20:16:word gen}}} {4:40 {=path {20:20:word foo} {=compound {20:24:word c} {20:25:punct .} {20:26:word gen}}}}}`,
		`22:10:_exists2 bar.c {=compound {5:10:word bar} {5:13:punct .} {5:14:word c}}`:`bar.c {=compound {5:10:word bar} {5:13:punct .} {5:14:word c}}`,
		`22:10:_exists2 bar.c++ {=compound {5:20:word bar} {5:23:punct .} {5:24:word c++}}`:`bar.c++ {=compound {5:20:word bar} {5:23:punct .} {5:24:word c++}}`,
		`22:10:_exists2 foo.c {=compound {5:28:word foo} {5:31:punct .} {5:32:word c}}`:`foo.c {=compound {5:28:word foo} {5:31:punct .} {5:32:word c}}`,
		`22:10:_exists2 foo.c++ {=compound {5:34:word foo} {5:37:punct .} {5:38:word c++}}`:`foo.c++ {=compound {5:34:word foo} {5:37:punct .} {5:38:word c++}}`,
		`22:10:_exists2 $(wildcard(-sort) **.gen) {22:13:delegate {22:15:builtin wildcard} [{=flag {22:25:word sort}}] {=list {=glob {22:40:meta **} {22:42:punct .} {22:43:word gen}}}}`:`{=file foo/c.gen} {22:13 {=file foo/c.gen}}`,

		`16:5:gen x.c++ {=compound {24:8:word x} {24:9:punct .} {24:10:word c++}}`:`x.c++ {=compound {24:8:word x} {24:9:punct .} {24:10:word c++}}`,
		`16:5:gen y.c++ {=compound {24:14:word y} {24:15:punct .} {24:16:word c++}}`:`y.c++ {=compound {24:14:word y} {24:15:punct .} {24:16:word c++}}`,
		`16:5:gen z.c {=compound {24:24:word z} {24:25:punct .} {24:26:word c}}`:`z.c {=compound {24:24:word z} {24:25:punct .} {24:26:word c}}`,

		`25:9:sources .tmp {=compound {4:53:punct .} {4:54:word tmp}}`:`.tmp {=compound {4:53:punct .} {4:54:word tmp}}`,
		`25:9:sources .deps {=compound {4:17:punct .} {4:18:word deps}}`:`.deps {=compound {4:17:punct .} {4:18:word deps}}`,
		`25:9:sources x.c++ {=compound {24:8:word x} {24:9:punct .} {24:10:word c++}}`:`x.c++ {=compound {24:8:word x} {24:9:punct .} {24:10:word c++}}`,
		`25:9:sources y.c++ {=compound {24:14:word y} {24:15:punct .} {24:16:word c++}}`:`y.c++ {=compound {24:14:word y} {24:15:punct .} {24:16:word c++}}`,
		`25:9:sources z.c {=compound {24:24:word z} {24:25:punct .} {24:26:word c}}`:`z.c {=compound {24:24:word z} {24:25:punct .} {24:26:word c}}`,
		`25:9:sources &(gen) {4:40:closure {4:42:word gen}}`:`x.c++ y.c++ foo/z.c {=list {4:40 {=compound {24:8:word x} {24:9:punct .} {24:10:word c++}}} {4:40 {=compound {24:14:word y} {24:15:punct .} {24:16:word c++}}} {4:40 {=path {24:20:word foo} {=compound {24:24:word z} {24:25:punct .} {24:26:word c}}}}}`,
		`25:9:sources bar.c {=compound {5:10:word bar} {5:13:punct .} {5:14:word c}}`:`bar.c {=compound {5:10:word bar} {5:13:punct .} {5:14:word c}}`,
		`25:9:sources bar.c++ {=compound {5:20:word bar} {5:23:punct .} {5:24:word c++}}`:`bar.c++ {=compound {5:20:word bar} {5:23:punct .} {5:24:word c++}}`,
		`25:9:sources foo.c {=compound {5:28:word foo} {5:31:punct .} {5:32:word c}}`:`foo.c {=compound {5:28:word foo} {5:31:punct .} {5:32:word c}}`,
		`25:9:sources foo.c++ {=compound {5:34:word foo} {5:37:punct .} {5:38:word c++}}`:`foo.c++ {=compound {5:34:word foo} {5:37:punct .} {5:38:word c++}}`,
		`25:9:sources $(wildcard(-sort -missing) **.c **.c++) {25:12:delegate {25:14:builtin wildcard} [{=flag {25:24:word sort}} {=flag {25:30:word missing}}] {=list {=glob {25:39:meta **} {25:41:punct .} {25:42:word c}} {=glob {25:44:meta **} {25:46:punct .} {25:47:word c++}}}}`:`{=file foo.c} {=file foo.c++} {=file foo/bar.c} {=file foo/bar.c++} {=file foo/z.c} {=file x.c++} {=file y.c++} {=list {25:12 {=file foo.c}} {25:12 {=file foo.c++}} {25:12 {=file foo/bar.c}} {25:12 {=file foo/bar.c++}} {25:12 {=file foo/z.c}} {25:12 {=file x.c++}} {25:12 {=file y.c++}}}`,

		`26:9:objects .c {=compound {26:40:punct .} {26:41:word c}}`:`.c {=compound {26:40:punct .} {26:41:word c}}`,
		`26:9:objects .c++ {=compound {26:44:punct .} {26:45:word c++}}`:`.c++ {=compound {26:44:punct .} {26:45:word c++}}`,
		`26:9:objects .o {=compound {26:50:punct .} {26:51:word o}}`:`.o {=compound {26:50:punct .} {26:51:word o}}`,
		`26:9:objects $(sources) {26:53:delegate {25:9:def sources}}`:`{=file foo.c} {=file foo.c++} {=file foo/bar.c} {=file foo/bar.c++} {=file foo/z.c} {=file x.c++} {=file y.c++} {=list {26:53 {25:12 {=file foo.c}}} {26:53 {25:12 {=file foo.c++}}} {26:53 {25:12 {=file foo/bar.c}}} {26:53 {25:12 {=file foo/bar.c++}}} {26:53 {25:12 {=file foo/z.c}}} {26:53 {25:12 {=file x.c++}}} {26:53 {25:12 {=file y.c++}}}}`,
		`26:9:objects x.c++ {=compound {24:8:word x} {24:9:punct .} {24:10:word c++}}`:`x.c++ {=compound {24:8:word x} {24:9:punct .} {24:10:word c++}}`,
		`26:9:objects y.c++ {=compound {24:14:word y} {24:15:punct .} {24:16:word c++}}`:`y.c++ {=compound {24:14:word y} {24:15:punct .} {24:16:word c++}}`,
		`26:9:objects z.c {=compound {24:24:word z} {24:25:punct .} {24:26:word c}}`:`z.c {=compound {24:24:word z} {24:25:punct .} {24:26:word c}}`,
		`26:9:objects &(gen) {4:40:closure {4:42:word gen}}`:`x.c++ y.c++ foo/z.c {=list {4:40 {=compound {24:8:word x} {24:9:punct .} {24:10:word c++}}} {4:40 {=compound {24:14:word y} {24:15:punct .} {24:16:word c++}}} {4:40 {=path {24:20:word foo} {=compound {24:24:word z} {24:25:punct .} {24:26:word c}}}}}`,
		`26:9:objects .tmp {=compound {4:53:punct .} {4:54:word tmp}}`:`.tmp {=compound {4:53:punct .} {4:54:word tmp}}`,
	},
}

var checkstrs_valcache = map[string]map[string]any{
	"loader.go": map[string]any{
		`17:10:nexists1 &(gen) {4:40:closure {4:42:word gen}}`:`{=list {4:40 {=compound {16:8:word x} {16:9:punct .} {16:10:word gen}}} {4:40 {=compound {16:14:word y} {16:15:punct .} {16:16:word gen}}} {4:40 {=path {16:20:word foo} {=compound {16:24:word z} {16:25:punct .} {16:26:word gen}}}}} x.gen y.gen foo/z.gen`,
		`26:9:objects $(sources) {26:53:delegate {25:9:def sources}}`:`{=list {26:53 {25:12 {=file foo.c}}} {26:53 {25:12 {=file foo.c++}}} {26:53 {25:12 {=file foo/bar.c}}} {26:53 {25:12 {=file foo/bar.c++}}} {26:53 {25:12 {=file foo/z.c}}} {26:53 {25:12 {=file x.c++}}} {26:53 {25:12 {=file y.c++}}}} foo.c foo.c++ foo/bar.c foo/bar.c++ foo/z.c x.c++ y.c++`,
	},
}

var checkpoints_cache = map[string]map[string]string{
	"closure":map[string]string{
		`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}}} &(gen)`:`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}},&:{}} [{}]`,
		`{foo:{*:{bar:{0:foo/*/bar}}}} &(gen)`:`{foo:{*:{bar:{0:foo/*/bar}}},&:{}} [{}]`,
	},
	"compound":map[string]string{
		`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}},&:{0:&(gen)},foo:{bar:{.:{c:{0:foo/bar.c},c++:{0:foo/bar.c++}}}}} foo.c`:`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}},&:{0:&(gen)},foo:{bar:{.:{c:{0:foo/bar.c},c++:{0:foo/bar.c++}}},.:{c:{}}}} [{}]`,
		`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}},&:{0:&(gen)},foo:{bar:{.:{c:{0:foo/bar.c},c++:{0:foo/bar.c++}}},.:{c:{0:foo.c}}}} foo.c++`:`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}},&:{0:&(gen)},foo:{bar:{.:{c:{0:foo/bar.c},c++:{0:foo/bar.c++}}},.:{c:{0:foo.c},c++:{}}}} [{}]`,
		`{foo:{.:{c:{0:foo.c}}}} foo.c++`:`{foo:{.:{c:{0:foo.c},c++:{}}}} [{}]`,
		`{} foo.c`:`{foo:{.:{c:{}}}} [{}]`,
	},
	"path":map[string]string{
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}}}}} foo/bar/v?.h`:`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}},v:{?:{.:{h:{}}}}}}} [{}]`,
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}},v:{?:{.:{h:{0:foo/bar/v?.h}}}}}}} foo/*.h`:`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}},v:{?:{.:{h:{0:foo/bar/v?.h}}}}},*:{h:{.:{}}}}} [{}]`,
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}},v:{?:{.:{h:{0:foo/bar/v?.h}}}}},*:{h:{.:{0:foo/*.h}}}}} foo/**.hh`:`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}},v:{?:{.:{h:{0:foo/bar/v?.h}}}}},*:{h:{.:{0:foo/*.h}}},**:{h:{h:{.:{}}}}}} [{}]`,
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}},v:{?:{.:{h:{0:foo/bar/v?.h}}}}},*:{h:{.:{0:foo/*.h}}},**:{h:{h:{.:{0:foo/**.hh}}}}}} foo???/x.h`:`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}},v:{?:{.:{h:{0:foo/bar/v?.h}}}}},*:{h:{.:{0:foo/*.h}}},**:{h:{h:{.:{0:foo/**.hh}}}},?:{?:{?:{x:{.:{h:{}}}}}}}} [{}]`,
		`{foo:{b:{*:{v:{*:{h:{.:{0:foo/b*/v*.h}}}}}}}} foo/x*y.h`:`{foo:{b:{*:{v:{*:{h:{.:{0:foo/b*/v*.h}}}}}},x:{*:{h:{.:{y:{}}}}}}} [{}]`,
		`{foo:{b:{*:{v:{*:{h:{.:{0:foo/b*/v*.h}}}}}},x:{*:{h:{.:{y:{0:foo/x*y.h}}}}}}} f*?/x.h`:`{foo:{b:{*:{v:{*:{h:{.:{0:foo/b*/v*.h}}}}}},x:{*:{h:{.:{y:{0:foo/x*y.h}}}}}},f:{*?:{x:{.:{h:{}}}}}} [{}]`,
		`{foo:{*:{bar:{0:foo/*/bar}}}} &(gen)`:`{foo:{*:{bar:{0:foo/*/bar}}},&:{}} [{}]`,
		`{foo:{.:{c:{0:foo.c},c++:{0:foo.c++}}}} foo/bar.c`:`{foo:{.:{c:{0:foo.c},c++:{0:foo.c++}},bar:{.:{c:{}}}}} [{}]`,
		`{foo:{.:{c:{0:foo.c},c++:{0:foo.c++}},bar:{.:{c:{0:foo/bar.c}}}}} foo/bar.c++`:`{foo:{.:{c:{0:foo.c},c++:{0:foo.c++}},bar:{.:{c:{0:foo/bar.c},c++:{}}}}} [{}]`,
		`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}}} .deps/??/??/??????????`:`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{}}}}}}}}}}}}}}}}} [{}]`,
		`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}},&:{0:&(gen)}} foo/bar.c`:`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}},&:{0:&(gen)},foo:{bar:{.:{c:{}}}}} [{}]`,
		`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}},&:{0:&(gen)},foo:{bar:{.:{c:{0:foo/bar.c}}}}} foo/bar.c++`:`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}},&:{0:&(gen)},foo:{bar:{.:{c:{0:foo/bar.c},c++:{}}}}} [{}]`,
		`{*:{+:{+:{c:{.:{0:*.c++}}}}},**:{c:{.:{0:**.c}}},?:{?:{?:{0:???}}}} foo/*.c++`:`{*:{+:{+:{c:{.:{0:*.c++}}}}},**:{c:{.:{0:**.c}}},?:{?:{?:{0:???}}},foo:{*:{+:{+:{c:{.:{}}}}}}} [{}]`,
		`{*:{+:{+:{c:{.:{0:*.c++}}}}},**:{c:{.:{0:**.c}}},?:{?:{?:{0:???}}},foo:{*:{+:{+:{c:{.:{0:foo/*.c++}}}}}}} foo/*.xx`:`{*:{+:{+:{c:{.:{0:*.c++}}}}},**:{c:{.:{0:**.c}}},?:{?:{?:{0:???}}},foo:{*:{+:{+:{c:{.:{0:foo/*.c++}}}},x:{x:{.:{}}}}}} [{}]`,
		`{*:{+:{+:{c:{.:{0:*.c++}}}}},**:{c:{.:{0:**.c}}},?:{?:{?:{0:???}}},foo:{*:{+:{+:{c:{.:{0:foo/*.c++}}}},x:{x:{.:{0:foo/*.xx}}}}}} foo/*.yy`:`{*:{+:{+:{c:{.:{0:*.c++}}}}},**:{c:{.:{0:**.c}}},?:{?:{?:{0:???}}},foo:{*:{+:{+:{c:{.:{0:foo/*.c++}}}},x:{x:{.:{0:foo/*.xx}}},y:{y:{.:{}}}}}} [{}]`,
		`{*:{+:{+:{c:{.:{0:*.c++}}}}},**:{c:{.:{0:**.c}}},?:{?:{?:{0:???}}},foo:{*:{+:{+:{c:{.:{0:foo/*.c++}}}},x:{x:{.:{0:foo/*.xx}}},y:{y:{.:{0:foo/*.yy}}}}}} foo/??/???.c++`:`{*:{+:{+:{c:{.:{0:*.c++}}}}},**:{c:{.:{0:**.c}}},?:{?:{?:{0:???}}},foo:{*:{+:{+:{c:{.:{0:foo/*.c++}}}},x:{x:{.:{0:foo/*.xx}}},y:{y:{.:{0:foo/*.yy}}}},?:{?:{?:{?:{?:{.:{.:{c:{+:{+:{}}}}}}}}}}}} [{}]`,
		`{*:{+:{+:{c:{.:{0:*.c++}}}}},**:{c:{.:{0:**.c}}},?:{?:{?:{0:???}}},foo:{*:{+:{+:{c:{.:{0:foo/*.c++}}}},x:{x:{.:{0:foo/*.xx}}},y:{y:{.:{0:foo/*.yy}}}},?:{?:{?:{?:{?:{.:{.:{c:{+:{+:{0:foo/??/???.c++}}}}}}}}}}}} foo/*zzz`:`{*:{+:{+:{c:{.:{0:*.c++}}}}},**:{c:{.:{0:**.c}}},?:{?:{?:{0:???}}},foo:{*:{+:{+:{c:{.:{0:foo/*.c++}}}},x:{x:{.:{0:foo/*.xx}}},y:{y:{.:{0:foo/*.yy}}},z:{z:{z:{}}}},?:{?:{?:{?:{?:{.:{.:{c:{+:{+:{0:foo/??/???.c++}}}}}}}}}}}} [{}]`,
		`{*:{+:{+:{c:{.:{0:*.c++}}}}},**:{c:{.:{0:**.c}}},?:{?:{?:{0:???}}},foo:{*:{+:{+:{c:{.:{0:foo/*.c++}}}},x:{x:{.:{0:foo/*.xx}}},y:{y:{.:{0:foo/*.yy}}},z:{z:{z:{0:foo/*zzz}}}},?:{?:{?:{?:{?:{.:{.:{c:{+:{+:{0:foo/??/???.c++}}}}}}}}}}}} foo/**z`:`{*:{+:{+:{c:{.:{0:*.c++}}}}},**:{c:{.:{0:**.c}}},?:{?:{?:{0:???}}},foo:{*:{+:{+:{c:{.:{0:foo/*.c++}}}},x:{x:{.:{0:foo/*.xx}}},y:{y:{.:{0:foo/*.yy}}},z:{z:{z:{0:foo/*zzz}}}},?:{?:{?:{?:{?:{.:{.:{c:{+:{+:{0:foo/??/???.c++}}}}}}}}}},**:{z:{}}}} [{}]`,
		`{*:{+:{+:{c:{.:{0:*.c++}}}}},**:{c:{.:{0:**.c}}},?:{?:{?:{0:???}}},foo:{*:{+:{+:{c:{.:{0:foo/*.c++}}}},x:{x:{.:{0:foo/*.xx}}},y:{y:{.:{0:foo/*.yy}}},z:{z:{z:{0:foo/*zzz}}}},?:{?:{?:{?:{?:{.:{.:{c:{+:{+:{0:foo/??/???.c++}}}}}}}}}},**:{z:{0:foo/**z}}}} foo/?????.o`:`{*:{+:{+:{c:{.:{0:*.c++}}}}},**:{c:{.:{0:**.c}}},?:{?:{?:{0:???}}},foo:{*:{+:{+:{c:{.:{0:foo/*.c++}}}},x:{x:{.:{0:foo/*.xx}}},y:{y:{.:{0:foo/*.yy}}},z:{z:{z:{0:foo/*zzz}}}},?:{?:{?:{?:{?:{.:{.:{c:{+:{+:{0:foo/??/???.c++}}},o:{}}}}}}}},**:{z:{0:foo/**z}}}} [{}]`,
		`{} foo/bar/zz/x.h`:`{foo:{bar:{zz:{x:{.:{h:{}}}}}}} [{}]`,
		`{} foo/b*/v*.h`:`{foo:{b:{*:{v:{*:{h:{.:{}}}}}}}} [{}]`,
		`{} foo/*/bar`:`{foo:{*:{bar:{}}}} [{}]`,
	},
	"globpat":map[string]string{
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}}}}} **.h`:`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}}}},**:{h:{.:{}}}} [{}]`,
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}}}},**:{h:{.:{0:**.h}}}} **.def.am`:`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}}}},**:{h:{.:{0:**.h}},m:{a:{.:{f:{e:{d:{.:{}}}}}}}}} [{}]`,
		`{*:{g:{o:{l:{.:{0:*.log}}}}}} **.o`:`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{}}}} [{}]`,
		`{*:{+:{+:{c:{.:{0:*.c++}}}}}} **.c`:`{*:{+:{+:{c:{.:{0:*.c++}}}}},**:{c:{.:{}}}} [{}]`,
		`{*:{+:{+:{c:{.:{0:*.c++}}}}},**:{c:{.:{0:**.c}}}} ???`:`{*:{+:{+:{c:{.:{0:*.c++}}}}},**:{c:{.:{0:**.c}}},?:{?:{?:{}}}} [{}]`,
		`{} *.log`:`{*:{g:{o:{l:{.:{}}}}}} [{}]`,
		`{} *.c++`:`{*:{+:{+:{c:{.:{}}}}}} [{}]`,
	},
	"string":map[string]string{
		`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}},&:{0:&(gen)},foo:{bar:{.:{c:{0:foo/bar.c},c++:{0:foo/bar.c++}}}}} foo.c`:`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}},&:{0:&(gen)},foo:{bar:{.:{c:{0:foo/bar.c},c++:{0:foo/bar.c++}}},.:{c:{}}}} [{}]`,
		`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}},&:{0:&(gen)},foo:{bar:{.:{c:{0:foo/bar.c},c++:{0:foo/bar.c++}}},.:{c:{0:foo.c}}}} foo.c++`:`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}},&:{0:&(gen)},foo:{bar:{.:{c:{0:foo/bar.c},c++:{0:foo/bar.c++}}},.:{c:{0:foo.c},c++:{}}}} [{}]`,
		`{foo:{0:foo}} bar`:`{foo:{0:foo},bar:{}} [{}]`,
		`{} foo`:`{foo:{}} [{}]`,
	},
	"word":map[string]string{
		`{foo:{0:foo}} bar`:`{foo:{0:foo},bar:{}} [{}]`,
		`{} foo`:`{foo:{}} [{}]`,
	},
}
func check_cache(ctx Context, k any, c0 string, c *valcache, r []*valcache) {
	var (
		ks = sf("%v %s", c0, k)
		vs = sf("%v %v", c, r)
		rs, y = checkpoints_cache[typeof(k)][ks]
	)
	if !y {
		debug(ctx, "%s: `%s`:`%s`,", typeof(k), ks, vs, trace{})
	} else if vs != rs {
		debug(ctx, _f("%s: %v", typeof(k), ks), _f("%v != %v", vs, rs), trace{})
	}
}

var checkpoints_uncache = map[string]map[string]string{
	"path":map[string]string{
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}}}},**:{h:{.:{0:**.h}},m:{a:{.:{f:{e:{d:{.:{0:**.def.am}}}}}}}}} foobar/config/*.def.am`:`[{0:**.def.am}]`,
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}}}},**:{h:{.:{0:**.h}},m:{a:{.:{f:{e:{d:{.:{0:**.def.am}}}}}}}}} foobar/config/*.def.in`:`[]`,
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}},v:{?:{.:{h:{0:foo/bar/v?.h}}}}},*:{h:{.:{0:foo/*.h}}},**:{h:{h:{.:{0:foo/**.hh}}}},?:{?:{?:{x:{.:{h:{0:foo???/x.h}}}}}}}} foo/*.h`:`[{0:foo/*.h}]`,
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}},v:{?:{.:{h:{0:foo/bar/v?.h}}}}},*:{h:{.:{0:foo/*.h}}},**:{h:{h:{.:{0:foo/**.hh}}}},?:{?:{?:{x:{.:{h:{0:foo???/x.h}}}}}}}} foo/bar/*.h`:`[{0:foo/bar/v?.h}]`,
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}},v:{?:{.:{h:{0:foo/bar/v?.h}}}}},*:{h:{.:{0:foo/*.h}}},**:{h:{h:{.:{0:foo/**.hh}}}},?:{?:{?:{x:{.:{h:{0:foo???/x.h}}}}}}}} foo/bar/*/*.h`:`[{0:foo/bar/zz/x.h}]`,
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}},v:{?:{.:{h:{0:foo/bar/v?.h}}}}},*:{h:{.:{0:foo/*.h}}},**:{h:{h:{.:{0:foo/**.hh}}}},?:{?:{?:{x:{.:{h:{0:foo???/x.h}}}}}}}} foo/bar/zz/?.h`:`[{0:foo/bar/zz/x.h}]`,
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}},v:{?:{.:{h:{0:foo/bar/v?.h}}}}},*:{h:{.:{0:foo/*.h}}},**:{h:{h:{.:{0:foo/**.hh}}}},?:{?:{?:{x:{.:{h:{0:foo???/x.h}}}}}}}} foo/bar/z?/?.h`:`[{0:foo/bar/zz/x.h}]`,
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}},v:{?:{.:{h:{0:foo/bar/v?.h}}}}},*:{h:{.:{0:foo/*.h}}},**:{h:{h:{.:{0:foo/**.hh}}}},?:{?:{?:{x:{.:{h:{0:foo???/x.h}}}}}}}} foo/bar/v1.h`:`[{0:foo/bar/v?.h}]`,
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}},v:{?:{.:{h:{0:foo/bar/v?.h}}}}},*:{h:{.:{0:foo/*.h}}},**:{h:{h:{.:{0:foo/**.hh}}}},?:{?:{?:{x:{.:{h:{0:foo???/x.h}}}}}}}} foo/bar/v2.h`:`[{0:foo/bar/v?.h}]`,
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}},v:{?:{.:{h:{0:foo/bar/v?.h}}}}},*:{h:{.:{0:foo/*.h}}},**:{h:{h:{.:{0:foo/**.hh}}}},?:{?:{?:{x:{.:{h:{0:foo???/x.h}}}}}}}} foo/bar/*.hh`:`[{0:foo/**.hh}]`,
		`{foo:{b:{*:{v:{*:{h:{.:{0:foo/b*/v*.h}}}}}},x:{*:{h:{.:{y:{0:foo/x*y.h}}}}}},f:{*?:{x:{.:{h:{0:f*?/x.h}}}}}} foo/bar/v1.h`:`[{0:foo/b*/v*.h}]`,
		`{foo:{b:{*:{v:{*:{h:{.:{0:foo/b*/v*.h}}}}}},x:{*:{h:{.:{y:{0:foo/x*y.h}}}}}},f:{*?:{x:{.:{h:{0:f*?/x.h}}}}}} foo/bar/v2.h`:`[{0:foo/b*/v*.h}]`,
	},
	"globpat":map[string]string{
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}}}},**:{h:{.:{0:**.h}},m:{a:{.:{f:{e:{d:{.:{0:**.def.am}}}}}}}}} *.h` :`[{0:**.h}]`,
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}}}},**:{h:{.:{0:**.h}},m:{a:{.:{f:{e:{d:{.:{0:**.def.am}}}}}}}}} **.h`:`[{0:foo/bar/zz/x.h} {0:**.h}]`,
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}}}},**:{h:{.:{0:**.h}},m:{a:{.:{f:{e:{d:{.:{0:**.def.am}}}}}}}}} *.def.am` :`[{0:**.def.am}]`,
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}}}},**:{h:{.:{0:**.h}},m:{a:{.:{f:{e:{d:{.:{0:**.def.am}}}}}}}}} **.def.am`:`[{0:**.def.am}]`,
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}},v:{?:{.:{h:{0:foo/bar/v?.h}}}}},*:{h:{.:{0:foo/*.h}}},**:{h:{h:{.:{0:foo/**.hh}}}},?:{?:{?:{x:{.:{h:{0:foo???/x.h}}}}}}}} *.h`:`[]`,
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}},v:{?:{.:{h:{0:foo/bar/v?.h}}}}},*:{h:{.:{0:foo/*.h}}},**:{h:{h:{.:{0:foo/**.hh}}}},?:{?:{?:{x:{.:{h:{0:foo???/x.h}}}}}}}} **.h`:`[{0:foo/bar/zz/x.h} {0:foo/bar/v?.h} {0:foo/*.h} {0:foo???/x.h}]`,
		`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}},&:{0:&(gen)},foo:{bar:{.:{c:{0:foo/bar.c},c++:{0:foo/bar.c++}}},.:{c:{0:foo.c},c++:{0:foo.c++}}}} **.c++`:`[{0:&(gen)} {0:foo/bar.c++} {0:foo.c++}]`,
		`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}},&:{0:&(gen)},foo:{bar:{.:{c:{0:foo/bar.c},c++:{0:foo/bar.c++}}},.:{c:{0:foo.c},c++:{0:foo.c++}}}} **.c`  :`[{0:&(gen)} {0:foo/bar.c} {0:foo.c}]`,
		`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}},&:{0:&(gen)},foo:{bar:{.:{c:{0:foo/bar.c},c++:{0:foo/bar.c++}}},.:{c:{0:foo.c},c++:{0:foo.c++}}}} *.gen` :`[{0:&(gen)}]`,
		`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}},&:{0:&(gen)},foo:{bar:{.:{c:{0:foo/bar.c},c++:{0:foo/bar.c++}}},.:{c:{0:foo.c},c++:{0:foo.c++}}}} **.gen`:`[{0:&(gen)}]`,
	},
	"compound":map[string]string{
		`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}},&:{0:&(gen)},foo:{bar:{.:{c:{0:foo/bar.c},c++:{0:foo/bar.c++}}},.:{c:{0:foo.c},c++:{0:foo.c++}}}} foo.o`:`[{0:**.o}]`,
	},
	"string":map[string]string{
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}}}},**:{h:{.:{0:**.h}},m:{a:{.:{f:{e:{d:{.:{0:**.def.am}}}}}}}}} **.h`:`[{0:**.h}]`,
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}}}},**:{h:{.:{0:**.h}},m:{a:{.:{f:{e:{d:{.:{0:**.def.am}}}}}}}}} **.def.am`:`[{0:**.def.am}]`,
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}}}},**:{h:{.:{0:**.h}},m:{a:{.:{f:{e:{d:{.:{0:**.def.am}}}}}}}}} foobar.h`:`[{0:**.h}]`,
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}}}},**:{h:{.:{0:**.h}},m:{a:{.:{f:{e:{d:{.:{0:**.def.am}}}}}}}}} foobar.def.am`:`[{0:**.def.am}]`,
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}}}},**:{h:{.:{0:**.h}},m:{a:{.:{f:{e:{d:{.:{0:**.def.am}}}}}}}}} foo/a/b/c/bar.h`:`[{0:**.h}]`,
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}}}},**:{h:{.:{0:**.h}},m:{a:{.:{f:{e:{d:{.:{0:**.def.am}}}}}}}}} foo/bar/zz/x.h`:`[{0:foo/bar/zz/x.h}]`,
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}}}},**:{h:{.:{0:**.h}},m:{a:{.:{f:{e:{d:{.:{0:**.def.am}}}}}}}}} foo/1/2/3/bar.def.am`:`[{0:**.def.am}]`,
		`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}},&:{0:&(gen)},foo:{bar:{.:{c:{0:foo/bar.c},c++:{0:foo/bar.c++}}},.:{c:{0:foo.c},c++:{0:foo.c++}}}} foo.o`:`[{0:**.o}]`,
		`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}},&:{0:&(gen)},foo:{bar:{.:{c:{0:foo/bar.c},c++:{0:foo/bar.c++}}},.:{c:{0:foo.c},c++:{0:foo.c++}}}} foo/bar.o`:`[{0:**.o}]`,
		`{} bar`:`[]`, `{} foo`:`[]`, `{} foobar`:`[]`,
	},
	"[]string":map[string]string{
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}}}},**:{h:{.:{0:**.h}},m:{a:{.:{f:{e:{d:{.:{0:**.def.am}}}}}}}}} [foo a b c bar.h]`:`[{0:**.h}]`,
		`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}}}},**:{h:{.:{0:**.h}},m:{a:{.:{f:{e:{d:{.:{0:**.def.am}}}}}}}}} [foo 1 2 3 bar.def.am]`:`[{0:**.def.am}]`,
	},
	"word":map[string]string{
		`{} bar`:`[]`, `{} foo`:`[]`, `{} foobar`:`[]`,
	},
}
func check_uncache(ctx Context, k any, c0 string, c *valcache, r []*valcache) {
	var (
		t = typeof(k)
		ks = sf("%v %s", c0, k)
		vs = sf("%v", r)
		rs, y = checkpoints_uncache[t][ks]
	)
	if stop := (callstack{stop:"smart.hit"}); !y {
		debug(ctx, _f("%s: `%s`:`%s`,", t, ks, vs), stop, trace{})
	} else if vs != rs {
		debug(ctx, _f("%s: %v", t, ks), _f("%s → %v != %v", k, vs, rs), stop, trace{})
	}
}

var checkpoints_unmap = map[string]string{
	`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}}}},**:{h:{.:{0:**.h}},m:{a:{.:{f:{e:{d:{.:{0:**.def.am}}}}}}}}} *.h` :`[{0:**.h}]`,
	`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}}}},**:{h:{.:{0:**.h}},m:{a:{.:{f:{e:{d:{.:{0:**.def.am}}}}}}}}} **.h`:`[{0:foo/bar/zz/x.h} {0:**.h}]`,
	`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}}}},**:{h:{.:{0:**.h}},m:{a:{.:{f:{e:{d:{.:{0:**.def.am}}}}}}}}} *.def.am` :`[{0:**.def.am}]`,
	`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}}}},**:{h:{.:{0:**.h}},m:{a:{.:{f:{e:{d:{.:{0:**.def.am}}}}}}}}} **.def.am`:`[{0:**.def.am}]`,
	`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}}}},**:{h:{.:{0:**.h}},m:{a:{.:{f:{e:{d:{.:{0:**.def.am}}}}}}}}} foobar/config/*.def.am`:`[{0:**.def.am}]`,
	`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}}}},**:{h:{.:{0:**.h}},m:{a:{.:{f:{e:{d:{.:{0:**.def.am}}}}}}}}} foobar/config/*.def.in`:`[]`,
	`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}},v:{?:{.:{h:{0:foo/bar/v?.h}}}}},*:{h:{.:{0:foo/*.h}}},**:{h:{h:{.:{0:foo/**.hh}}}},?:{?:{?:{x:{.:{h:{0:foo???/x.h}}}}}}}} *.h`:`[]`,
	`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}},v:{?:{.:{h:{0:foo/bar/v?.h}}}}},*:{h:{.:{0:foo/*.h}}},**:{h:{h:{.:{0:foo/**.hh}}}},?:{?:{?:{x:{.:{h:{0:foo???/x.h}}}}}}}} **.h`:`[{0:foo/bar/zz/x.h} {0:foo/bar/v?.h} {0:foo/*.h} {0:foo???/x.h}]`,
	`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}},v:{?:{.:{h:{0:foo/bar/v?.h}}}}},*:{h:{.:{0:foo/*.h}}},**:{h:{h:{.:{0:foo/**.hh}}}},?:{?:{?:{x:{.:{h:{0:foo???/x.h}}}}}}}} foo/*.h`:`[{0:foo/*.h}]`,
	`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}},v:{?:{.:{h:{0:foo/bar/v?.h}}}}},*:{h:{.:{0:foo/*.h}}},**:{h:{h:{.:{0:foo/**.hh}}}},?:{?:{?:{x:{.:{h:{0:foo???/x.h}}}}}}}} foo/bar/*.h`:`[{0:foo/bar/v?.h}]`,
	`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}},v:{?:{.:{h:{0:foo/bar/v?.h}}}}},*:{h:{.:{0:foo/*.h}}},**:{h:{h:{.:{0:foo/**.hh}}}},?:{?:{?:{x:{.:{h:{0:foo???/x.h}}}}}}}} foo/bar/*/*.h`:`[{0:foo/bar/zz/x.h}]`,
	`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}},v:{?:{.:{h:{0:foo/bar/v?.h}}}}},*:{h:{.:{0:foo/*.h}}},**:{h:{h:{.:{0:foo/**.hh}}}},?:{?:{?:{x:{.:{h:{0:foo???/x.h}}}}}}}} foo/bar/zz/?.h`:`[{0:foo/bar/zz/x.h}]`,
	`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}},v:{?:{.:{h:{0:foo/bar/v?.h}}}}},*:{h:{.:{0:foo/*.h}}},**:{h:{h:{.:{0:foo/**.hh}}}},?:{?:{?:{x:{.:{h:{0:foo???/x.h}}}}}}}} foo/bar/z?/?.h`:`[{0:foo/bar/zz/x.h}]`,
	`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}},v:{?:{.:{h:{0:foo/bar/v?.h}}}}},*:{h:{.:{0:foo/*.h}}},**:{h:{h:{.:{0:foo/**.hh}}}},?:{?:{?:{x:{.:{h:{0:foo???/x.h}}}}}}}} foo/bar/v1.h`:`[{0:foo/bar/v?.h}]`,
	`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}},v:{?:{.:{h:{0:foo/bar/v?.h}}}}},*:{h:{.:{0:foo/*.h}}},**:{h:{h:{.:{0:foo/**.hh}}}},?:{?:{?:{x:{.:{h:{0:foo???/x.h}}}}}}}} foo/bar/v2.h`:`[{0:foo/bar/v?.h}]`,
	`{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}},v:{?:{.:{h:{0:foo/bar/v?.h}}}}},*:{h:{.:{0:foo/*.h}}},**:{h:{h:{.:{0:foo/**.hh}}}},?:{?:{?:{x:{.:{h:{0:foo???/x.h}}}}}}}} foo/bar/*.hh`:`[{0:foo/**.hh}]`,
	`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}},&:{0:&(gen)},foo:{bar:{.:{c:{0:foo/bar.c},c++:{0:foo/bar.c++}}},.:{c:{0:foo.c},c++:{0:foo.c++}}}} **.c++`   :`[{0:&(gen)} {0:foo/bar.c++} {0:foo.c++}]`,
	`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}},&:{0:&(gen)},foo:{bar:{.:{c:{0:foo/bar.c},c++:{0:foo/bar.c++}}},.:{c:{0:foo.c},c++:{0:foo.c++}}}} **.c`     :`[{0:&(gen)} {0:foo/bar.c} {0:foo.c}]`,
	`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}},&:{0:&(gen)},foo:{bar:{.:{c:{0:foo/bar.c},c++:{0:foo/bar.c++}}},.:{c:{0:foo.c},c++:{0:foo.c++}}}} *.gen`    :`[{0:&(gen)}]`,
	`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}},&:{0:&(gen)},foo:{bar:{.:{c:{0:foo/bar.c},c++:{0:foo/bar.c++}}},.:{c:{0:foo.c},c++:{0:foo.c++}}}} **.gen`   :`[{0:&(gen)}]`,
	`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}},&:{0:&(gen)},foo:{bar:{.:{c:{0:foo/bar.c},c++:{0:foo/bar.c++}}},.:{c:{0:foo.c},c++:{0:foo.c++}}}} foo.o`    :`[{0:**.o}]`,
	`{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}},&:{0:&(gen)},foo:{bar:{.:{c:{0:foo/bar.c},c++:{0:foo/bar.c++}}},.:{c:{0:foo.c},c++:{0:foo.c++}}}} foo/bar.o`:`[{0:**.o}]`,
	`{} foobar`:`[]`, `{} foo`:`[]`, `{} bar`:`[]`,
}
func check_unmap(ctx *uncache_t, key any, c *valcache, x []*valcache) {
	var (
		t = typeof(key)
		ks = sf("%v %v", c, key)
		vs = sf("%v", x)
		rs, y = checkpoints_unmap[ks]
		ca = callstack{stop:"smart.unmap[...]"}
	)
	if !y {
		debug(ctx, "%s: `%s`:`%s`,", t, ks, vs, ca, trace{})
	} else if vs != rs {
		debug(ctx, _f("%s: %v", t, ks), _f("%v != %v", vs, rs), ca, trace{})
	}
}

func unmap_check(ctx *uncache_t, key any, c, x *valcache) {
	switch _project(ctx).name {
	case "configure.base":
		switch x := key.(type) {
		case flag:
			var cc, y = c.v[MINUS.String()]
			if !y {
				if truly(ctx, unmap_uncheck_y{}) { break }
				debug(ctx, "%v %v %v", do(ctx, propUncache), x, c.ks(), trace{})
			}
			if _, y = cc.v[x.Value.String()]; !y {
				debug(ctx, "%v", x.Value, trace{})
			}
		}
	}
	switch do(ctx, get_include_spec{}) {
	case "configure/.base/.template":
		debug(pc(ctx,key), "%v %v", tv(key), c, trace{})
	}
}
