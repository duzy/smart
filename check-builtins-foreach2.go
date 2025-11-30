//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints__foreach2 = map[string]map[string]any{
	"loader.go": map[string]any{
		`.test.foreach.a {=compound {3:1:punct .} {3:2:word test} {3:6:punct .} {3:7:word foreach} {3:14:punct .} {3:15:word a}}`:`.test.foreach.a {=compound {3:1:punct .} {3:2:word test} {3:6:punct .} {3:7:word foreach} {3:14:punct .} {3:15:word a}}`,
		`.test.foreach.b {=compound {4:1:punct .} {4:2:word test} {4:6:punct .} {4:7:word foreach} {4:14:punct .} {4:15:word b}}`:`.test.foreach.b {=compound {4:1:punct .} {4:2:word test} {4:6:punct .} {4:7:word foreach} {4:14:punct .} {4:15:word b}}`,
		`.test.foreach.c {=compound {5:1:punct .} {5:2:word test} {5:6:punct .} {5:7:word foreach} {5:14:punct .} {5:15:word c}}`:`.test.foreach.c {=compound {5:1:punct .} {5:2:word test} {5:6:punct .} {5:7:word foreach} {5:14:punct .} {5:15:word c}}`,
		`.test.1 {=compound {9:1:punct .} {9:2:word test} {9:6:punct .} {9:7:decimal 1}}`:`.test.1 {=compound {9:1:punct .} {9:2:word test} {9:6:punct .} {9:7:decimal 1}}`,
		`.test.2 {=compound {12:1:punct .} {12:2:word test} {12:6:punct .} {12:7:decimal 2}}`:`.test.2 {=compound {12:1:punct .} {12:2:word test} {12:6:punct .} {12:7:decimal 2}}`,
		`.test.foreach.x.1 {=compound {13:1:punct .} {13:2:word test} {13:6:punct .} {13:7:word foreach} {13:14:punct .} {13:15:word x} {13:16:punct .} {13:17:decimal 1}}`:`.test.foreach.x.1 {=compound {13:1:punct .} {13:2:word test} {13:6:punct .} {13:7:word foreach} {13:14:punct .} {13:15:word x} {13:16:punct .} {13:17:decimal 1}}`,
		`.test.foreach.x.2 {=compound {14:1:punct .} {14:2:word test} {14:6:punct .} {14:7:word foreach} {14:14:punct .} {14:15:word x} {14:16:punct .} {14:17:decimal 2}}`:`.test.foreach.x.2 {=compound {14:1:punct .} {14:2:word test} {14:6:punct .} {14:7:word foreach} {14:14:punct .} {14:15:word x} {14:16:punct .} {14:17:decimal 2}}`,
		`.test.foreach.x {=compound {15:1:punct .} {15:2:word test} {15:6:punct .} {15:7:word foreach} {15:14:punct .} {15:15:word x}}`:`.test.foreach.x {=compound {15:1:punct .} {15:2:word test} {15:6:punct .} {15:7:word foreach} {15:14:punct .} {15:15:word x}}`,
		`.test.3 {=compound {18:1:punct .} {18:2:word test} {18:6:punct .} {18:7:decimal 3}}`:`.test.3 {=compound {18:1:punct .} {18:2:word test} {18:6:punct .} {18:7:decimal 3}}`,
		`.test.foreach.x.3 {=compound {19:1:punct .} {19:2:word test} {19:6:punct .} {19:7:word foreach} {19:14:punct .} {19:15:word x} {19:16:punct .} {19:17:decimal 3}}`:`.test.foreach.x.3 {=compound {19:1:punct .} {19:2:word test} {19:6:punct .} {19:7:word foreach} {19:14:punct .} {19:15:word x} {19:16:punct .} {19:17:decimal 3}}`,
		`.test.foreach.x.4 {=compound {20:1:punct .} {20:2:word test} {20:6:punct .} {20:7:word foreach} {20:14:punct .} {20:15:word x} {20:16:punct .} {20:17:decimal 4}}`:`.test.foreach.x.4 {=compound {20:1:punct .} {20:2:word test} {20:6:punct .} {20:7:word foreach} {20:14:punct .} {20:15:word x} {20:16:punct .} {20:17:decimal 4}}`,
		`.test.foreach.x {=compound {21:1:punct .} {21:2:word test} {21:6:punct .} {21:7:word foreach} {21:14:punct .} {21:15:word x}}`:`.test.foreach.x {=compound {21:1:punct .} {21:2:word test} {21:6:punct .} {21:7:word foreach} {21:14:punct .} {21:15:word x}}`,
		`.test.foreach.d {=compound {24:1:punct .} {24:2:word test} {24:6:punct .} {24:7:word foreach} {24:14:punct .} {24:15:word d}}`:`.test.foreach.d {=compound {24:1:punct .} {24:2:word test} {24:6:punct .} {24:7:word foreach} {24:14:punct .} {24:15:word d}}`,
		`.test.foreach.d.1 {=compound {25:1:punct .} {25:2:word test} {25:6:punct .} {25:7:word foreach} {25:14:punct .} {25:15:word d} {25:16:punct .} {25:17:decimal 1}}`:`.test.foreach.d.1 {=compound {25:1:punct .} {25:2:word test} {25:6:punct .} {25:7:word foreach} {25:14:punct .} {25:15:word d} {25:16:punct .} {25:17:decimal 1}}`,
		`.test.foreach.d.2 {=compound {26:1:punct .} {26:2:word test} {26:6:punct .} {26:7:word foreach} {26:14:punct .} {26:15:word d} {26:16:punct .} {26:17:decimal 2}}`:`.test.foreach.d.2 {=compound {26:1:punct .} {26:2:word test} {26:6:punct .} {26:7:word foreach} {26:14:punct .} {26:15:word d} {26:16:punct .} {26:17:decimal 2}}`,
		`.test.4 {=compound {27:1:punct .} {27:2:word test} {27:6:punct .} {27:7:decimal 4}}`:`.test.4 {=compound {27:1:punct .} {27:2:word test} {27:6:punct .} {27:7:decimal 4}}`,
	},
	"check-builtins-foreach2_test.go": map[string]any{
		`44 $1 {4:38:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {4:38 {3:30:null}}`,
		`44 x$1 {=compound {4:37:word x} {4:38:delegate {3:30:auto 1}}}`:`x{} {=compound {4:37:word x} {4:38 {3:30:null}}}`,

		`44 $1 {4:42:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {4:42 {3:30:null}}`,
		`44 y$1 {=compound {4:41:word y} {4:42:delegate {3:30:auto 1}}}`:`y{} {=compound {4:41:word y} {4:42 {3:30:null}}}`,

		`44 $1 {4:46:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {4:46 {3:30:null}}`,
		`44 z$1 {=compound {4:45:word z} {4:46:delegate {3:30:auto 1}}}`:`z{} {=compound {4:45:word z} {4:46 {3:30:null}}}`,

		`44 $2 {4:51:delegate {3:43:auto 2}}`:`{3:43:auto 2} {} {4:51 {3:43:null}}`,
		`44 xx$2 {=compound {4:49:word xx} {4:51:delegate {3:43:auto 2}}}`:`xx{} {=compound {4:49:word xx} {4:51 {3:43:null}}}`,

		`44 $2 {4:56:delegate {3:43:auto 2}}`:`{3:43:auto 2} {} {4:56 {3:43:null}}`,
		`44 yy$2 {=compound {4:54:word yy} {4:56:delegate {3:43:auto 2}}}`:`yy{} {=compound {4:54:word yy} {4:56 {3:43:null}}}`,

		`44 x{} {=compound {4:37:word x} {4:38 {3:30:null}}}`:`x{} {=compound {4:37:word x} {4:38 {3:30:null}}}`,
		`44 y{} {=compound {4:41:word y} {4:42 {3:30:null}}}`:`y{} {=compound {4:41:word y} {4:42 {3:30:null}}}`,
		`44 z{} {=compound {4:45:word z} {4:46 {3:30:null}}}`:`z{} {=compound {4:45:word z} {4:46 {3:30:null}}}`,

		`44 $1 {3:29:delegate {3:30:auto 1}}`:`{3:30:auto 1} x{} y{} z{} {=list {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}} {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}} {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}`,

		`44 xx{} {=compound {4:49:word xx} {4:51 {3:43:null}}}`:`xx{} {=compound {4:49:word xx} {4:51 {3:43:null}}}`,
		`44 yy{} {=compound {4:54:word yy} {4:56 {3:43:null}}}`:`yy{} {=compound {4:54:word yy} {4:56 {3:43:null}}}`,

		`44 $2 {3:42:delegate {3:43:auto 2}}`:`{3:43:auto 2} xx{} yy{} {=list {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}} {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}`,
		`44 $_ {3:46:delegate {3:47:auto _}}`:[]string{
			`{3:47:auto _} xx{} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}`,
			`{3:47:auto _} yy{} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}`,
		},
		`44 a$_ {=compound {3:45:word a} {3:46:delegate {3:47:auto _}}}`:[]string{
			`axx{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}`,
			`ayy{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}`,
		},
		`44 $(foreach $2,a$_) {3:32:delegate {3:34:builtin foreach} {=list {3:42:delegate {3:43:auto 2}}} {=list {=compound {3:45:word a} {3:46:delegate {3:47:auto _}}}}}`:`{3:34:builtin foreach} axx{} ayy{} {=list {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}} {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}`,

		`44 $_ {3:51:delegate {3:47:auto _}}`:[]string{
			`{3:47:auto _} x{} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}`,
			`{3:47:auto _} y{} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}`,
			`{3:47:auto _} z{} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}`,
			`{3:47:auto _} axx{} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}`,
			`{3:47:auto _} ayy{} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}`,
		},
		`44 b$_ {=compound {3:50:word b} {3:51:delegate {3:47:auto _}}}`:[]string{
			`bx{} {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}`,
			`by{} {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}`,
			`bz{} {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}`,
			`baxx{} {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}`,
			`bayy{} {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}`,
		},

		`44 axx{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}`:`axx{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}`,
		`44 ayy{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}`:`ayy{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}`,

		`44 $(foreach $1 $(foreach $2,a$_),b$_) {3:19:delegate {3:21:builtin foreach} {=list {3:29:delegate {3:30:auto 1}} {3:32:delegate {3:34:builtin foreach} {=list {3:42:delegate {3:43:auto 2}}} {=list {=compound {3:45:word a} {3:46:delegate {3:47:auto _}}}}}} {=list {=compound {3:50:word b} {3:51:delegate {3:47:auto _}}}}}`:`{3:21:builtin foreach} bx{} by{} bz{} baxx{} bayy{} {=list {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}} {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}} {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}} {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}} {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}}}`,
		`44 $(.test.foreach.a x$1 y$1 z$1,xx$2 yy$2) {4:19:delegate {3:17:def .test.foreach.a} {=list {=compound {4:37:word x} {4:38:delegate {3:30:auto 1}}} {=compound {4:41:word y} {4:42:delegate {3:30:auto 1}}} {=compound {4:45:word z} {4:46:delegate {3:30:auto 1}}}} {=list {=compound {4:49:word xx} {4:51:delegate {3:43:auto 2}}} {=compound {4:54:word yy} {4:56:delegate {3:43:auto 2}}}}}`:`{3:17:def .test.foreach.a} bx{} by{} bz{} baxx{} bayy{} {=list {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}}} {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}}} {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}}} {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}}} {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}}}}`,
		`44 $(.test.foreach.b) {9:11:delegate {4:17:def .test.foreach.b}}`:`{4:17:def .test.foreach.b} bx{} by{} bz{} baxx{} bayy{} {=list {9:11 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}}}} {9:11 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}}}} {9:11 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}}}} {9:11 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}}}} {9:11 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}}}}}`,

		`46 9:9:.test.1 $1 {4:38:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {4:38 {3:30:null}}`,
		`46 9:9:.test.1 x$1 {=compound {4:37:word x} {4:38:delegate {3:30:auto 1}}}`:`x{} {=compound {4:37:word x} {4:38 {3:30:null}}}`,

		`46 9:9:.test.1 $1 {4:42:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {4:42 {3:30:null}}`,
		`46 9:9:.test.1 y$1 {=compound {4:41:word y} {4:42:delegate {3:30:auto 1}}}`:`y{} {=compound {4:41:word y} {4:42 {3:30:null}}}`,

		`46 9:9:.test.1 $1 {4:46:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {4:46 {3:30:null}}`,
		`46 9:9:.test.1 z$1 {=compound {4:45:word z} {4:46:delegate {3:30:auto 1}}}`:`z{} {=compound {4:45:word z} {4:46 {3:30:null}}}`,

		`46 9:9:.test.1 $2 {4:51:delegate {3:43:auto 2}}`:`{3:43:auto 2} {} {4:51 {3:43:null}}`,
		`46 9:9:.test.1 xx$2 {=compound {4:49:word xx} {4:51:delegate {3:43:auto 2}}}`:`xx{} {=compound {4:49:word xx} {4:51 {3:43:null}}}`,

		`46 9:9:.test.1 $2 {4:56:delegate {3:43:auto 2}}`:`{3:43:auto 2} {} {4:56 {3:43:null}}`,
		`46 9:9:.test.1 yy$2 {=compound {4:54:word yy} {4:56:delegate {3:43:auto 2}}}`:`yy{} {=compound {4:54:word yy} {4:56 {3:43:null}}}`,

		`46 9:9:.test.1 x{} {=compound {4:37:word x} {4:38 {3:30:null}}}`:`x{} {=compound {4:37:word x} {4:38 {3:30:null}}}`,
		`46 9:9:.test.1 y{} {=compound {4:41:word y} {4:42 {3:30:null}}}`:`y{} {=compound {4:41:word y} {4:42 {3:30:null}}}`,
		`46 9:9:.test.1 z{} {=compound {4:45:word z} {4:46 {3:30:null}}}`:`z{} {=compound {4:45:word z} {4:46 {3:30:null}}}`,

		`46 9:9:.test.1 $1 {3:29:delegate {3:30:auto 1}}`:`{3:30:auto 1} x{} y{} z{} {=list {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}} {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}} {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}`,

		`46 9:9:.test.1 xx{} {=compound {4:49:word xx} {4:51 {3:43:null}}}`:`xx{} {=compound {4:49:word xx} {4:51 {3:43:null}}}`,
		`46 9:9:.test.1 yy{} {=compound {4:54:word yy} {4:56 {3:43:null}}}`:`yy{} {=compound {4:54:word yy} {4:56 {3:43:null}}}`,

		`46 9:9:.test.1 $2 {3:42:delegate {3:43:auto 2}}`:`{3:43:auto 2} xx{} yy{} {=list {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}} {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}`,

		`46 9:9:.test.1 $_ {3:46:delegate {3:47:auto _}}`:[]string{
			`{3:47:auto _} xx{} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}`,
			`{3:47:auto _} yy{} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}`,
		},
		`46 9:9:.test.1 a$_ {=compound {3:45:word a} {3:46:delegate {3:47:auto _}}}`:[]string{
			`axx{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}`,
			`ayy{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}`,
		},

		`46 9:9:.test.1 $(foreach $2,a$_) {3:32:delegate {3:34:builtin foreach} {=list {3:42:delegate {3:43:auto 2}}} {=list {=compound {3:45:word a} {3:46:delegate {3:47:auto _}}}}}`:`{3:34:builtin foreach} axx{} ayy{} {=list {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}} {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}`,

		`46 9:9:.test.1 $_ {3:51:delegate {3:47:auto _}}`:[]string{
			`{3:47:auto _} x{} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}`,
			`{3:47:auto _} y{} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}`,
			`{3:47:auto _} z{} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}`,
			`{3:47:auto _} axx{} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}`,
			`{3:47:auto _} ayy{} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}`,
		},
		`46 9:9:.test.1 b$_ {=compound {3:50:word b} {3:51:delegate {3:47:auto _}}}`:[]string{
			`bx{} {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}`,
			`by{} {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}`,
			`bz{} {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}`,
			`baxx{} {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}`,
			`bayy{} {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}`,
		},

		`46 9:9:.test.1 axx{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}`:`axx{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}`,
		`46 9:9:.test.1 ayy{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}`:`ayy{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}`,

		`46 9:9:.test.1 $(foreach $1 $(foreach $2,a$_),b$_) {3:19:delegate {3:21:builtin foreach} {=list {3:29:delegate {3:30:auto 1}} {3:32:delegate {3:34:builtin foreach} {=list {3:42:delegate {3:43:auto 2}}} {=list {=compound {3:45:word a} {3:46:delegate {3:47:auto _}}}}}} {=list {=compound {3:50:word b} {3:51:delegate {3:47:auto _}}}}}`:`{3:21:builtin foreach} bx{} by{} bz{} baxx{} bayy{} {=list {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}} {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}} {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}} {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}} {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}}}`,
		`46 9:9:.test.1 $(.test.foreach.a x$1 y$1 z$1,xx$2 yy$2) {4:19:delegate {3:17:def .test.foreach.a} {=list {=compound {4:37:word x} {4:38:delegate {3:30:auto 1}}} {=compound {4:41:word y} {4:42:delegate {3:30:auto 1}}} {=compound {4:45:word z} {4:46:delegate {3:30:auto 1}}}} {=list {=compound {4:49:word xx} {4:51:delegate {3:43:auto 2}}} {=compound {4:54:word yy} {4:56:delegate {3:43:auto 2}}}}}`:`{3:17:def .test.foreach.a} bx{} by{} bz{} baxx{} bayy{} {=list {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}}} {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}}} {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}}} {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}}} {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}}}}`,
		`46 9:9:.test.1 $(.test.foreach.b) {9:11:delegate {4:17:def .test.foreach.b}}`:`{4:17:def .test.foreach.b} bx{} by{} bz{} baxx{} bayy{} {=list {9:11 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}}}} {9:11 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}}}} {9:11 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}}}} {9:11 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}}}} {9:11 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}}}}}`,

		`61 12:9:.test.2 $1 {4:38:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {4:38 {3:30:null}}`,
		`61 12:9:.test.2 x$1 {=compound {4:37:word x} {4:38:delegate {3:30:auto 1}}}`:`x{} {=compound {4:37:word x} {4:38 {3:30:null}}}`,

		`61 12:9:.test.2 $1 {4:42:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {4:42 {3:30:null}}`,
		`61 12:9:.test.2 y$1 {=compound {4:41:word y} {4:42:delegate {3:30:auto 1}}}`:`y{} {=compound {4:41:word y} {4:42 {3:30:null}}}`,

		`61 12:9:.test.2 $1 {4:46:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {4:46 {3:30:null}}`,
		`61 12:9:.test.2 z$1 {=compound {4:45:word z} {4:46:delegate {3:30:auto 1}}}`:`z{} {=compound {4:45:word z} {4:46 {3:30:null}}}`,

		`61 12:9:.test.2 $2 {4:51:delegate {3:43:auto 2}}`:`{3:43:auto 2} {} {4:51 {3:43:null}}`,
		`61 12:9:.test.2 xx$2 {=compound {4:49:word xx} {4:51:delegate {3:43:auto 2}}}`:`xx{} {=compound {4:49:word xx} {4:51 {3:43:null}}}`,

		`61 12:9:.test.2 $2 {4:56:delegate {3:43:auto 2}}`:`{3:43:auto 2} {} {4:56 {3:43:null}}`,
		`61 12:9:.test.2 yy$2 {=compound {4:54:word yy} {4:56:delegate {3:43:auto 2}}}`:`yy{} {=compound {4:54:word yy} {4:56 {3:43:null}}}`,

		`61 12:9:.test.2 x{} {=compound {4:37:word x} {4:38 {3:30:null}}}`:`x{} {=compound {4:37:word x} {4:38 {3:30:null}}}`,
		`61 12:9:.test.2 y{} {=compound {4:41:word y} {4:42 {3:30:null}}}`:`y{} {=compound {4:41:word y} {4:42 {3:30:null}}}`,
		`61 12:9:.test.2 z{} {=compound {4:45:word z} {4:46 {3:30:null}}}`:`z{} {=compound {4:45:word z} {4:46 {3:30:null}}}`,

		`61 12:9:.test.2 $1 {3:29:delegate {3:30:auto 1}}`:`{3:30:auto 1} x{} y{} z{} {=list {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}} {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}} {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}`,

		`61 12:9:.test.2 xx{} {=compound {4:49:word xx} {4:51 {3:43:null}}}`:`xx{} {=compound {4:49:word xx} {4:51 {3:43:null}}}`,
		`61 12:9:.test.2 yy{} {=compound {4:54:word yy} {4:56 {3:43:null}}}`:`yy{} {=compound {4:54:word yy} {4:56 {3:43:null}}}`,

		`61 12:9:.test.2 $2 {3:42:delegate {3:43:auto 2}}`:`{3:43:auto 2} xx{} yy{} {=list {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}} {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}`,
		`61 12:9:.test.2 $_ {3:46:delegate {3:47:auto _}}`:[]string{
			`{3:47:auto _} xx{} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}`,
			`{3:47:auto _} yy{} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}`,
		},
		`61 12:9:.test.2 a$_ {=compound {3:45:word a} {3:46:delegate {3:47:auto _}}}`:[]string{
			`axx{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}`,
			`ayy{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}`,
		},
		`61 12:9:.test.2 $(foreach $2,a$_) {3:32:delegate {3:34:builtin foreach} {=list {3:42:delegate {3:43:auto 2}}} {=list {=compound {3:45:word a} {3:46:delegate {3:47:auto _}}}}}`:`{3:34:builtin foreach} axx{} ayy{} {=list {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}} {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}`,
		`61 12:9:.test.2 $_ {3:51:delegate {3:47:auto _}}`:[]string{
			`{3:47:auto _} x{} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}`,
			`{3:47:auto _} y{} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}`,
			`{3:47:auto _} z{} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}`,
			`{3:47:auto _} axx{} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}`,
			`{3:47:auto _} ayy{} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}`,
		},
		`61 12:9:.test.2 b$_ {=compound {3:50:word b} {3:51:delegate {3:47:auto _}}}`:[]string{
			`bx{} {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}`,
			`by{} {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}`,
			`bz{} {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}`,
			`baxx{} {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}`,
			`bayy{} {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}`,
		},

		`61 12:9:.test.2 axx{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}`:`axx{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}`,
		`61 12:9:.test.2 ayy{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}`:`ayy{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}`,

		`61 12:9:.test.2 $(foreach $1 $(foreach $2,a$_),b$_) {3:19:delegate {3:21:builtin foreach} {=list {3:29:delegate {3:30:auto 1}} {3:32:delegate {3:34:builtin foreach} {=list {3:42:delegate {3:43:auto 2}}} {=list {=compound {3:45:word a} {3:46:delegate {3:47:auto _}}}}}} {=list {=compound {3:50:word b} {3:51:delegate {3:47:auto _}}}}}`:`{3:21:builtin foreach} bx{} by{} bz{} baxx{} bayy{} {=list {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}} {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}} {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}} {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}} {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}}}`,
		`61 12:9:.test.2 $(.test.foreach.a x$1 y$1 z$1,xx$2 yy$2) {4:19:delegate {3:17:def .test.foreach.a} {=list {=compound {4:37:word x} {4:38:delegate {3:30:auto 1}}} {=compound {4:41:word y} {4:42:delegate {3:30:auto 1}}} {=compound {4:45:word z} {4:46:delegate {3:30:auto 1}}}} {=list {=compound {4:49:word xx} {4:51:delegate {3:43:auto 2}}} {=compound {4:54:word yy} {4:56:delegate {3:43:auto 2}}}}}`:`{3:17:def .test.foreach.a} bx{} by{} bz{} baxx{} bayy{} {=list {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}}} {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}}} {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}}} {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}}} {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}}}}`,
		`61 12:9:.test.2 $(.test.foreach.b) {5:19:delegate {4:17:def .test.foreach.b}}`:`{4:17:def .test.foreach.b} bx{} by{} bz{} baxx{} bayy{} {=list {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}}}} {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}}}} {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}}}} {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}}}} {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}}}}}`,

		`61 12:9:.test.2 .test.foreach.x {=compound {6:17:punct .} {6:18:word test} {6:22:punct .} {6:23:word foreach} {6:30:punct .} {6:31:word x}}`:`.test.foreach.x {=compound {6:17:punct .} {6:18:word test} {6:22:punct .} {6:23:word foreach} {6:30:punct .} {6:31:word x}}`,
		`61 12:9:.test.2 &(.test.foreach.x) {6:15:closure {=compound {6:17:punct .} {6:18:word test} {6:22:punct .} {6:23:word foreach} {6:30:punct .} {6:31:word x}}}`:`{15:17:def .test.foreach.x} &(.test.foreach.x) {6:15:closure {15:17:def .test.foreach.x}}`,

		`61 12:9:.test.2 $1 {6:44:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {6:44 {3:30:null}}`,
		`61 12:9:.test.2 $2 {6:47:delegate {3:43:auto 2}}`:`{3:43:auto 2} {} {6:47 {3:43:null}}`,
		`61 12:9:.test.2 $(foreach $1 $2,&(.test.foreach.x.$_)) {6:34:delegate {6:36:builtin foreach} {=list {6:44:delegate {3:30:auto 1}} {6:47:delegate {3:43:auto 2}}} {=list {6:50:closure {=compound {6:52:punct .} {6:53:word test} {6:57:punct .} {6:58:word foreach} {6:65:punct .} {6:66:word x} {6:67:punct .} {6:68:delegate {3:47:auto _}}}}}}`:`{6:36:builtin foreach} {} {6:34 {6:36:null}}`,
		`61 12:9:.test.2 &(.test.foreach.x) {6:15:closure {15:17:def .test.foreach.x}}`:`{15:17:def .test.foreach.x} &(.test.foreach.x) {6:15:closure {15:17:def .test.foreach.x}}`,
		`61 12:9:.test.2 $_ {6:75:delegate {3:47:auto _}}`:`{3:47:auto _} {&(.test.foreach.x)} {6:75 {6:15:disjunction {6:15:closure {15:17:def .test.foreach.x}}}}`,
		`61 12:9:.test.2 x$_ {=compound {6:74:word x} {6:75:delegate {3:47:auto _}}}`:`x{&(.test.foreach.x)} {=compound {6:74:word x} {6:75 {6:15:disjunction {6:15:closure {15:17:def .test.foreach.x}}}}}`,
		`61 12:9:.test.2 $(foreach &(.test.foreach.x) $(foreach $1 $2,&(.test.foreach.x.$_)),-x$_) {6:5:delegate {6:7:builtin foreach} {=list {6:15:closure {=compound {6:17:punct .} {6:18:word test} {6:22:punct .} {6:23:word foreach} {6:30:punct .} {6:31:word x}}} {6:34:delegate {6:36:builtin foreach} {=list {6:44:delegate {3:30:auto 1}} {6:47:delegate {3:43:auto 2}}} {=list {6:50:closure {=compound {6:52:punct .} {6:53:word test} {6:57:punct .} {6:58:word foreach} {6:65:punct .} {6:66:word x} {6:67:punct .} {6:68:delegate {3:47:auto _}}}}}}} {=list {=flag {=compound {6:74:word x} {6:75:delegate {3:47:auto _}}}}}}`:`{6:7:builtin foreach} -x{&(.test.foreach.x)} {6:5 {=flag {=compound {6:74:word x} {6:75 {6:15:disjunction {6:15:closure {15:17:def .test.foreach.x}}}}}}}`,
		`61 12:9:.test.2 $(.test.foreach.c) {12:11:delegate {5:17:def .test.foreach.c}}`:`{5:17:def .test.foreach.c} bx{} by{} bz{} baxx{} bayy{} -x{&(.test.foreach.x)} {=list {12:11 {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}}}}} {12:11 {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}}}}} {12:11 {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}}}}} {12:11 {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}}}}} {12:11 {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}}}}} {12:11 {6:5 {=flag {=compound {6:74:word x} {6:75 {6:15:disjunction {6:15:closure {15:17:def .test.foreach.x}}}}}}}}}`,

		`65 &(.test.foreach.x) {6:15:closure {15:17:def .test.foreach.x}}`:`{15:17:def .test.foreach.x} vw {6:15 {21:21:word vw}}`,

		`76 $1 {18:29:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {18:29 {3:30:null}}`,

		`76 $1 {4:38:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {4:38 {3:30:null}}`,
		`76 x$1 {=compound {4:37:word x} {4:38:delegate {3:30:auto 1}}}`:`x{} {=compound {4:37:word x} {4:38 {3:30:null}}}`,

		`76 $1 {4:42:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {4:42 {3:30:null}}`,
		`76 y$1 {=compound {4:41:word y} {4:42:delegate {3:30:auto 1}}}`:`y{} {=compound {4:41:word y} {4:42 {3:30:null}}}`,

		`76 $1 {4:46:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {4:46 {3:30:null}}`,
		`76 z$1 {=compound {4:45:word z} {4:46:delegate {3:30:auto 1}}}`:`z{} {=compound {4:45:word z} {4:46 {3:30:null}}}`,

		`76 $2 {4:51:delegate {3:43:auto 2}}`:`{3:43:auto 2} {} {4:51 {3:43:null}}`,
		`76 xx$2 {=compound {4:49:word xx} {4:51:delegate {3:43:auto 2}}}`:`xx{} {=compound {4:49:word xx} {4:51 {3:43:null}}}`,

		`76 $2 {4:56:delegate {3:43:auto 2}}`:`{3:43:auto 2} {} {4:56 {3:43:null}}`,
		`76 yy$2 {=compound {4:54:word yy} {4:56:delegate {3:43:auto 2}}}`:`yy{} {=compound {4:54:word yy} {4:56 {3:43:null}}}`,

		`76 x{} {=compound {4:37:word x} {4:38 {3:30:null}}}`:`x{} {=compound {4:37:word x} {4:38 {3:30:null}}}`,
		`76 y{} {=compound {4:41:word y} {4:42 {3:30:null}}}`:`y{} {=compound {4:41:word y} {4:42 {3:30:null}}}`,
		`76 z{} {=compound {4:45:word z} {4:46 {3:30:null}}}`:`z{} {=compound {4:45:word z} {4:46 {3:30:null}}}`,

		`76 $1 {3:29:delegate {3:30:auto 1}}`:`{3:30:auto 1} x{} y{} z{} {=list {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}} {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}} {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}`,

		`76 xx{} {=compound {4:49:word xx} {4:51 {3:43:null}}}`:`xx{} {=compound {4:49:word xx} {4:51 {3:43:null}}}`,
		`76 yy{} {=compound {4:54:word yy} {4:56 {3:43:null}}}`:`yy{} {=compound {4:54:word yy} {4:56 {3:43:null}}}`,

		`76 $2 {3:42:delegate {3:43:auto 2}}`:`{3:43:auto 2} xx{} yy{} {=list {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}} {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}`,

		`76 $_ {3:46:delegate {3:47:auto _}}`:[]string{
			`{3:47:auto _} xx{} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}`,
			`{3:47:auto _} yy{} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}`,
		},
		`76 a$_ {=compound {3:45:word a} {3:46:delegate {3:47:auto _}}}`:[]string{
			`axx{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}`,
			`ayy{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}`,
		},

		`76 $(foreach $2,a$_) {3:32:delegate {3:34:builtin foreach} {=list {3:42:delegate {3:43:auto 2}}} {=list {=compound {3:45:word a} {3:46:delegate {3:47:auto _}}}}}`:`{3:34:builtin foreach} axx{} ayy{} {=list {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}} {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}`,
		`76 $_ {3:51:delegate {3:47:auto _}}`:[]string{
			`{3:47:auto _} x{} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}`,
			`{3:47:auto _} y{} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}`,
			`{3:47:auto _} z{} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}`,
			`{3:47:auto _} axx{} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}`,
			`{3:47:auto _} ayy{} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}`,
		},
		`76 b$_ {=compound {3:50:word b} {3:51:delegate {3:47:auto _}}}`:[]string{
			`bx{} {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}`,
			`by{} {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}`,
			`bz{} {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}`,
			`baxx{} {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}`,
			`bayy{} {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}`,
		},

		`76 axx{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}`:`axx{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}`,
		`76 ayy{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}`:`ayy{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}`,

		`76 $(foreach $1 $(foreach $2,a$_),b$_) {3:19:delegate {3:21:builtin foreach} {=list {3:29:delegate {3:30:auto 1}} {3:32:delegate {3:34:builtin foreach} {=list {3:42:delegate {3:43:auto 2}}} {=list {=compound {3:45:word a} {3:46:delegate {3:47:auto _}}}}}} {=list {=compound {3:50:word b} {3:51:delegate {3:47:auto _}}}}}`:`{3:21:builtin foreach} bx{} by{} bz{} baxx{} bayy{} {=list {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}} {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}} {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}} {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}} {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}}}`,
		`76 $(.test.foreach.a x$1 y$1 z$1,xx$2 yy$2) {4:19:delegate {3:17:def .test.foreach.a} {=list {=compound {4:37:word x} {4:38:delegate {3:30:auto 1}}} {=compound {4:41:word y} {4:42:delegate {3:30:auto 1}}} {=compound {4:45:word z} {4:46:delegate {3:30:auto 1}}}} {=list {=compound {4:49:word xx} {4:51:delegate {3:43:auto 2}}} {=compound {4:54:word yy} {4:56:delegate {3:43:auto 2}}}}}`:`{3:17:def .test.foreach.a} bx{} by{} bz{} baxx{} bayy{} {=list {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}}} {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}}} {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}}} {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}}} {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}}}}`,
		`76 $(.test.foreach.b) {5:19:delegate {4:17:def .test.foreach.b}}`:`{4:17:def .test.foreach.b} bx{} by{} bz{} baxx{} bayy{} {=list {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}}}} {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}}}} {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}}}} {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}}}} {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}}}}}`,

		`76 .test.foreach.x {=compound {6:17:punct .} {6:18:word test} {6:22:punct .} {6:23:word foreach} {6:30:punct .} {6:31:word x}}`:`.test.foreach.x {=compound {6:17:punct .} {6:18:word test} {6:22:punct .} {6:23:word foreach} {6:30:punct .} {6:31:word x}}`,
		`76 &(.test.foreach.x) {6:15:closure {=compound {6:17:punct .} {6:18:word test} {6:22:punct .} {6:23:word foreach} {6:30:punct .} {6:31:word x}}}`:`{15:17:def .test.foreach.x} vw {6:15 {21:21:word vw}}`,
		`76 $1 {6:44:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {6:44 {18:29 {3:30:null}}}`,
		`76 $2 {6:47:delegate {3:43:auto 2}}`:`{3:43:auto 2} 4 {6:47 {18:32:decimal 4}}`,
		`76 $_ {6:68:delegate {3:47:auto _}}`:`{3:47:auto _} 4 {6:68 {6:47 {18:32:decimal 4}}}`,
		`76 .test.foreach.x.$_ {=compound {6:52:punct .} {6:53:word test} {6:57:punct .} {6:58:word foreach} {6:65:punct .} {6:66:word x} {6:67:punct .} {6:68:delegate {3:47:auto _}}}`:`.test.foreach.x.4 {=compound {6:52:punct .} {6:53:word test} {6:57:punct .} {6:58:word foreach} {6:65:punct .} {6:66:word x} {6:67:punct .} {6:68 {6:47 {18:32:decimal 4}}}}`,
		`76 $1 {20:22:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {20:22 {3:30:null}}`,
		`76 $2 {20:24:delegate {3:43:auto 2}}`:`{3:43:auto 2} {} {20:24 {3:43:null}}`,
		`76 W$1$2 {=compound {20:21:word W} {20:22:delegate {3:30:auto 1}} {20:24:delegate {3:43:auto 2}}}`:`W{}{} {=compound {20:21:word W} {20:22 {3:30:null}} {20:24 {3:43:null}}}`,
		`76 &(.test.foreach.x.$_) {6:50:closure {=compound {6:52:punct .} {6:53:word test} {6:57:punct .} {6:58:word foreach} {6:65:punct .} {6:66:word x} {6:67:punct .} {6:68:delegate {3:47:auto _}}}}`:`{20:19:def .test.foreach.x.4} W{}{} {6:50 {=compound {20:21:word W} {20:22 {3:30:null}} {20:24 {3:43:null}}}}`,
		`76 $(foreach $1 $2,&(.test.foreach.x.$_)) {6:34:delegate {6:36:builtin foreach} {=list {6:44:delegate {3:30:auto 1}} {6:47:delegate {3:43:auto 2}}} {=list {6:50:closure {=compound {6:52:punct .} {6:53:word test} {6:57:punct .} {6:58:word foreach} {6:65:punct .} {6:66:word x} {6:67:punct .} {6:68:delegate {3:47:auto _}}}}}}`:`{6:36:builtin foreach} W{}{} {6:34 {6:50 {=compound {20:21:word W} {20:22 {3:30:null}} {20:24 {3:43:null}}}}}`,
		`76 $_ {6:75:delegate {3:47:auto _}}`:[]string{
			`{3:47:auto _} vw {6:75 {6:15 {21:21:word vw}}}`,
			`{3:47:auto _} W{}{} {6:75 {6:34 {6:50 {=compound {20:21:word W} {20:22 {3:30:null}} {20:24 {3:43:null}}}}}}`,
		},
		`76 x$_ {=compound {6:74:word x} {6:75:delegate {3:47:auto _}}}`:[]string{
			`xvw {=compound {6:74:word x} {6:75 {6:15 {21:21:word vw}}}}`,
			`xW{}{} {=compound {6:74:word x} {6:75 {6:34 {6:50 {=compound {20:21:word W} {20:22 {3:30:null}} {20:24 {3:43:null}}}}}}}`,
		},
		`76 W{}{} {=compound {20:21:word W} {20:22 {3:30:null}} {20:24 {3:43:null}}}`:`W{}{} {=compound {20:21:word W} {20:22 {3:30:null}} {20:24 {3:43:null}}}`,
		`76 $(foreach &(.test.foreach.x) $(foreach $1 $2,&(.test.foreach.x.$_)),-x$_) {6:5:delegate {6:7:builtin foreach} {=list {6:15:closure {=compound {6:17:punct .} {6:18:word test} {6:22:punct .} {6:23:word foreach} {6:30:punct .} {6:31:word x}}} {6:34:delegate {6:36:builtin foreach} {=list {6:44:delegate {3:30:auto 1}} {6:47:delegate {3:43:auto 2}}} {=list {6:50:closure {=compound {6:52:punct .} {6:53:word test} {6:57:punct .} {6:58:word foreach} {6:65:punct .} {6:66:word x} {6:67:punct .} {6:68:delegate {3:47:auto _}}}}}}} {=list {=flag {=compound {6:74:word x} {6:75:delegate {3:47:auto _}}}}}}`:`{6:7:builtin foreach} -xvw -xW{}{} {=list {6:5 {=flag {=compound {6:74:word x} {6:75 {6:15 {21:21:word vw}}}}}} {6:5 {=flag {=compound {6:74:word x} {6:75 {6:34 {6:50 {=compound {20:21:word W} {20:22 {3:30:null}} {20:24 {3:43:null}}}}}}}}}}`,
		`76 $(.test.foreach.c $1,4) {18:11:delegate {5:17:def .test.foreach.c} {=list {18:29:delegate {3:30:auto 1}}} {=list {18:32:decimal 4}}}`:`{5:17:def .test.foreach.c} bx{} by{} bz{} baxx{} bayy{} -xvw -xW{}{} {=list {18:11 {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}}}}} {18:11 {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}}}}} {18:11 {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}}}}} {18:11 {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}}}}} {18:11 {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}}}}} {18:11 {6:5 {=flag {=compound {6:74:word x} {6:75 {6:15 {21:21:word vw}}}}}}} {18:11 {6:5 {=flag {=compound {6:74:word x} {6:75 {6:34 {6:50 {=compound {20:21:word W} {20:22 {3:30:null}} {20:24 {3:43:null}}}}}}}}}}}`,

		`78 18:9:.test.3 $1 {18:29:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {18:29 {3:30:null}}`,
		`78 18:9:.test.3 $1 {4:38:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {4:38 {3:30:null}}`,
		`78 18:9:.test.3 x$1 {=compound {4:37:word x} {4:38:delegate {3:30:auto 1}}}`:`x{} {=compound {4:37:word x} {4:38 {3:30:null}}}`,

		`78 18:9:.test.3 $1 {4:42:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {4:42 {3:30:null}}`,
		`78 18:9:.test.3 y$1 {=compound {4:41:word y} {4:42:delegate {3:30:auto 1}}}`:`y{} {=compound {4:41:word y} {4:42 {3:30:null}}}`,

		`78 18:9:.test.3 $1 {4:46:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {4:46 {3:30:null}}`,
		`78 18:9:.test.3 z$1 {=compound {4:45:word z} {4:46:delegate {3:30:auto 1}}}`:`z{} {=compound {4:45:word z} {4:46 {3:30:null}}}`,

		`78 18:9:.test.3 $2 {4:51:delegate {3:43:auto 2}}`:`{3:43:auto 2} {} {4:51 {3:43:null}}`,
		`78 18:9:.test.3 xx$2 {=compound {4:49:word xx} {4:51:delegate {3:43:auto 2}}}`:`xx{} {=compound {4:49:word xx} {4:51 {3:43:null}}}`,

		`78 18:9:.test.3 $2 {4:56:delegate {3:43:auto 2}}`:`{3:43:auto 2} {} {4:56 {3:43:null}}`,
		`78 18:9:.test.3 yy$2 {=compound {4:54:word yy} {4:56:delegate {3:43:auto 2}}}`:`yy{} {=compound {4:54:word yy} {4:56 {3:43:null}}}`,

		`78 18:9:.test.3 x{} {=compound {4:37:word x} {4:38 {3:30:null}}}`:`x{} {=compound {4:37:word x} {4:38 {3:30:null}}}`,
		`78 18:9:.test.3 y{} {=compound {4:41:word y} {4:42 {3:30:null}}}`:`y{} {=compound {4:41:word y} {4:42 {3:30:null}}}`,
		`78 18:9:.test.3 z{} {=compound {4:45:word z} {4:46 {3:30:null}}}`:`z{} {=compound {4:45:word z} {4:46 {3:30:null}}}`,

		`78 18:9:.test.3 $1 {3:29:delegate {3:30:auto 1}}`:`{3:30:auto 1} x{} y{} z{} {=list {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}} {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}} {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}`,

		`78 18:9:.test.3 xx{} {=compound {4:49:word xx} {4:51 {3:43:null}}}`:`xx{} {=compound {4:49:word xx} {4:51 {3:43:null}}}`,
		`78 18:9:.test.3 yy{} {=compound {4:54:word yy} {4:56 {3:43:null}}}`:`yy{} {=compound {4:54:word yy} {4:56 {3:43:null}}}`,

		`78 18:9:.test.3 $2 {3:42:delegate {3:43:auto 2}}`:`{3:43:auto 2} xx{} yy{} {=list {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}} {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}`,

		`78 18:9:.test.3 $_ {3:46:delegate {3:47:auto _}}`:[]string{
			`{3:47:auto _} xx{} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}`,
			`{3:47:auto _} yy{} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}`,
		},
		`78 18:9:.test.3 a$_ {=compound {3:45:word a} {3:46:delegate {3:47:auto _}}}`:[]string{
			`axx{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}`,
			`ayy{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}`,
		},
		`78 18:9:.test.3 $(foreach $2,a$_) {3:32:delegate {3:34:builtin foreach} {=list {3:42:delegate {3:43:auto 2}}} {=list {=compound {3:45:word a} {3:46:delegate {3:47:auto _}}}}}`:`{3:34:builtin foreach} axx{} ayy{} {=list {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}} {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}`,
		`78 18:9:.test.3 $_ {3:51:delegate {3:47:auto _}}`:[]string{
			`{3:47:auto _} x{} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}`,
			`{3:47:auto _} y{} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}`,
			`{3:47:auto _} z{} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}`,
			`{3:47:auto _} axx{} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}`,
			`{3:47:auto _} ayy{} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}`,
		},
		`78 18:9:.test.3 b$_ {=compound {3:50:word b} {3:51:delegate {3:47:auto _}}}`:[]string{
			`bx{} {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}`,
			`by{} {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}`,
			`bz{} {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}`,
			`baxx{} {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}`,
			`bayy{} {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}`,
		},

		`78 18:9:.test.3 axx{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}`:`axx{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}`,
		`78 18:9:.test.3 ayy{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}`:`ayy{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}`,

		`78 18:9:.test.3 $(foreach $1 $(foreach $2,a$_),b$_) {3:19:delegate {3:21:builtin foreach} {=list {3:29:delegate {3:30:auto 1}} {3:32:delegate {3:34:builtin foreach} {=list {3:42:delegate {3:43:auto 2}}} {=list {=compound {3:45:word a} {3:46:delegate {3:47:auto _}}}}}} {=list {=compound {3:50:word b} {3:51:delegate {3:47:auto _}}}}}`:`{3:21:builtin foreach} bx{} by{} bz{} baxx{} bayy{} {=list {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}} {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}} {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}} {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}} {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}}}`,
		`78 18:9:.test.3 $(.test.foreach.a x$1 y$1 z$1,xx$2 yy$2) {4:19:delegate {3:17:def .test.foreach.a} {=list {=compound {4:37:word x} {4:38:delegate {3:30:auto 1}}} {=compound {4:41:word y} {4:42:delegate {3:30:auto 1}}} {=compound {4:45:word z} {4:46:delegate {3:30:auto 1}}}} {=list {=compound {4:49:word xx} {4:51:delegate {3:43:auto 2}}} {=compound {4:54:word yy} {4:56:delegate {3:43:auto 2}}}}}`:`{3:17:def .test.foreach.a} bx{} by{} bz{} baxx{} bayy{} {=list {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}}} {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}}} {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}}} {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}}} {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}}}}`,
		`78 18:9:.test.3 $(.test.foreach.b) {5:19:delegate {4:17:def .test.foreach.b}}`:`{4:17:def .test.foreach.b} bx{} by{} bz{} baxx{} bayy{} {=list {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}}}} {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}}}} {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}}}} {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}}}} {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}}}}}`,

		`78 18:9:.test.3 .test.foreach.x {=compound {6:17:punct .} {6:18:word test} {6:22:punct .} {6:23:word foreach} {6:30:punct .} {6:31:word x}}`:`.test.foreach.x {=compound {6:17:punct .} {6:18:word test} {6:22:punct .} {6:23:word foreach} {6:30:punct .} {6:31:word x}}`,
		`78 18:9:.test.3 &(.test.foreach.x) {6:15:closure {=compound {6:17:punct .} {6:18:word test} {6:22:punct .} {6:23:word foreach} {6:30:punct .} {6:31:word x}}}`:`{15:17:def .test.foreach.x} &(.test.foreach.x) {6:15:closure {15:17:def .test.foreach.x}}`,

		`78 18:9:.test.3 $1 {6:44:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {6:44 {18:29 {3:30:null}}}`,
		`78 18:9:.test.3 $2 {6:47:delegate {3:43:auto 2}}`:`{3:43:auto 2} 4 {6:47 {18:32:decimal 4}}`,
		`78 18:9:.test.3 $_ {6:68:delegate {3:47:auto _}}`:[]string{
			`{3:47:auto _} 4 {6:68 {6:47 {18:32:decimal 4}}}`,
		},
		`78 18:9:.test.3 .test.foreach.x.$_ {=compound {6:52:punct .} {6:53:word test} {6:57:punct .} {6:58:word foreach} {6:65:punct .} {6:66:word x} {6:67:punct .} {6:68:delegate {3:47:auto _}}}`:[]string{
			`.test.foreach.x.4 {=compound {6:52:punct .} {6:53:word test} {6:57:punct .} {6:58:word foreach} {6:65:punct .} {6:66:word x} {6:67:punct .} {6:68 {6:47 {18:32:decimal 4}}}}`,
		},
		`78 18:9:.test.3 &(.test.foreach.x.$_) {6:50:closure {=compound {6:52:punct .} {6:53:word test} {6:57:punct .} {6:58:word foreach} {6:65:punct .} {6:66:word x} {6:67:punct .} {6:68:delegate {3:47:auto _}}}}`:[]string{
			`{20:19:def .test.foreach.x.4} &(.test.foreach.x.4) {6:50:closure {20:19:def .test.foreach.x.4}}`,
		},
		`78 18:9:.test.3 $(foreach $1 $2,&(.test.foreach.x.$_)) {6:34:delegate {6:36:builtin foreach} {=list {6:44:delegate {3:30:auto 1}} {6:47:delegate {3:43:auto 2}}} {=list {6:50:closure {=compound {6:52:punct .} {6:53:word test} {6:57:punct .} {6:58:word foreach} {6:65:punct .} {6:66:word x} {6:67:punct .} {6:68:delegate {3:47:auto _}}}}}}`:[]string{
			`{6:36:builtin foreach} &(.test.foreach.x.4) {6:34 {6:50:closure {20:19:def .test.foreach.x.4}}}`,
		},
		`78 18:9:.test.3 &(.test.foreach.x) {6:15:closure {15:17:def .test.foreach.x}}`:`{15:17:def .test.foreach.x} &(.test.foreach.x) {6:15:closure {15:17:def .test.foreach.x}}`,
		`78 18:9:.test.3 $_ {6:75:delegate {3:47:auto _}}`:[]string{
			`{3:47:auto _} {&(.test.foreach.x)} {6:75 {6:15:disjunction {6:15:closure {15:17:def .test.foreach.x}}}}`,
			`{3:47:auto _} {&(.test.foreach.x.4)} {6:75 {6:34 {6:50:disjunction {6:50:closure {20:19:def .test.foreach.x.4}}}}}`,
		},
		`78 18:9:.test.3 x$_ {=compound {6:74:word x} {6:75:delegate {3:47:auto _}}}`:[]string{
			`x{&(.test.foreach.x)} {=compound {6:74:word x} {6:75 {6:15:disjunction {6:15:closure {15:17:def .test.foreach.x}}}}}`,
			`x{&(.test.foreach.x.4)} {=compound {6:74:word x} {6:75 {6:34 {6:50:disjunction {6:50:closure {20:19:def .test.foreach.x.4}}}}}}`,
		},
		`78 18:9:.test.3 &(.test.foreach.x.4) {6:50:closure {20:19:def .test.foreach.x.4}}`:`{20:19:def .test.foreach.x.4} &(.test.foreach.x.4) {6:50:closure {20:19:def .test.foreach.x.4}}`,
		`78 18:9:.test.3 $(foreach &(.test.foreach.x) $(foreach $1 $2,&(.test.foreach.x.$_)),-x$_) {6:5:delegate {6:7:builtin foreach} {=list {6:15:closure {=compound {6:17:punct .} {6:18:word test} {6:22:punct .} {6:23:word foreach} {6:30:punct .} {6:31:word x}}} {6:34:delegate {6:36:builtin foreach} {=list {6:44:delegate {3:30:auto 1}} {6:47:delegate {3:43:auto 2}}} {=list {6:50:closure {=compound {6:52:punct .} {6:53:word test} {6:57:punct .} {6:58:word foreach} {6:65:punct .} {6:66:word x} {6:67:punct .} {6:68:delegate {3:47:auto _}}}}}}} {=list {=flag {=compound {6:74:word x} {6:75:delegate {3:47:auto _}}}}}}`:`{6:7:builtin foreach} -x{&(.test.foreach.x)} -x{&(.test.foreach.x.4)} {=list {6:5 {=flag {=compound {6:74:word x} {6:75 {6:15:disjunction {6:15:closure {15:17:def .test.foreach.x}}}}}}} {6:5 {=flag {=compound {6:74:word x} {6:75 {6:34 {6:50:disjunction {6:50:closure {20:19:def .test.foreach.x.4}}}}}}}}}`,
		`78 18:9:.test.3 $(.test.foreach.c $1,4) {18:11:delegate {5:17:def .test.foreach.c} {=list {18:29:delegate {3:30:auto 1}}} {=list {18:32:decimal 4}}}`:`{5:17:def .test.foreach.c} bx{} by{} bz{} baxx{} bayy{} -x{&(.test.foreach.x)} -x{&(.test.foreach.x.4)} {=list {18:11 {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}}}}} {18:11 {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}}}}} {18:11 {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}}}}} {18:11 {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}}}}} {18:11 {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}}}}} {18:11 {6:5 {=flag {=compound {6:74:word x} {6:75 {6:15:disjunction {6:15:closure {15:17:def .test.foreach.x}}}}}}}} {18:11 {6:5 {=flag {=compound {6:74:word x} {6:75 {6:34 {6:50:disjunction {6:50:closure {20:19:def .test.foreach.x.4}}}}}}}}}}`,

		`82 &(.test.foreach.x) {6:15:closure {15:17:def .test.foreach.x}}`:`{15:17:def .test.foreach.x} vw {6:15 {21:21:word vw}}`,
		`82 $1 {20:22:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {20:22 {3:30:null}}`,
		`82 $2 {20:24:delegate {3:43:auto 2}}`:`{3:43:auto 2} {} {20:24 {3:43:null}}`,
		`82 W$1$2 {=compound {20:21:word W} {20:22:delegate {3:30:auto 1}} {20:24:delegate {3:43:auto 2}}}`:`W{}{} {=compound {20:21:word W} {20:22 {3:30:null}} {20:24 {3:43:null}}}`,
		`82 &(.test.foreach.x.4) {6:50:closure {20:19:def .test.foreach.x.4}}`:`{20:19:def .test.foreach.x.4} W{}{} {6:50 {=compound {20:21:word W} {20:22 {3:30:null}} {20:24 {3:43:null}}}}`,

		`84 18:9:.test.3 $1 {18:29:delegate {3:30:auto 1}}`:`{3:30:auto 1} 3 {18:29 {1:9:word 3}}`,

		`84 18:9:.test.3 $1 {4:38:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {4:38 {3:30:null}}`,
		`84 18:9:.test.3 x$1 {=compound {4:37:word x} {4:38:delegate {3:30:auto 1}}}`:`x{} {=compound {4:37:word x} {4:38 {3:30:null}}}`,

		`84 18:9:.test.3 $1 {4:42:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {4:42 {3:30:null}}`,
		`84 18:9:.test.3 y$1 {=compound {4:41:word y} {4:42:delegate {3:30:auto 1}}}`:`y{} {=compound {4:41:word y} {4:42 {3:30:null}}}`,

		`84 18:9:.test.3 $1 {4:46:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {4:46 {3:30:null}}`,
		`84 18:9:.test.3 z$1 {=compound {4:45:word z} {4:46:delegate {3:30:auto 1}}}`:`z{} {=compound {4:45:word z} {4:46 {3:30:null}}}`,

		`84 18:9:.test.3 $2 {4:51:delegate {3:43:auto 2}}`:`{3:43:auto 2} {} {4:51 {3:43:null}}`,
		`84 18:9:.test.3 xx$2 {=compound {4:49:word xx} {4:51:delegate {3:43:auto 2}}}`:`xx{} {=compound {4:49:word xx} {4:51 {3:43:null}}}`,

		`84 18:9:.test.3 $2 {4:56:delegate {3:43:auto 2}}`:`{3:43:auto 2} {} {4:56 {3:43:null}}`,
		`84 18:9:.test.3 yy$2 {=compound {4:54:word yy} {4:56:delegate {3:43:auto 2}}}`:`yy{} {=compound {4:54:word yy} {4:56 {3:43:null}}}`,

		`84 18:9:.test.3 x{} {=compound {4:37:word x} {4:38 {3:30:null}}}`:`x{} {=compound {4:37:word x} {4:38 {3:30:null}}}`,
		`84 18:9:.test.3 y{} {=compound {4:41:word y} {4:42 {3:30:null}}}`:`y{} {=compound {4:41:word y} {4:42 {3:30:null}}}`,
		`84 18:9:.test.3 z{} {=compound {4:45:word z} {4:46 {3:30:null}}}`:`z{} {=compound {4:45:word z} {4:46 {3:30:null}}}`,

		`84 18:9:.test.3 $1 {3:29:delegate {3:30:auto 1}}`:`{3:30:auto 1} x{} y{} z{} {=list {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}} {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}} {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}`,

		`84 18:9:.test.3 xx{} {=compound {4:49:word xx} {4:51 {3:43:null}}}`:`xx{} {=compound {4:49:word xx} {4:51 {3:43:null}}}`,
		`84 18:9:.test.3 yy{} {=compound {4:54:word yy} {4:56 {3:43:null}}}`:`yy{} {=compound {4:54:word yy} {4:56 {3:43:null}}}`,

		`84 18:9:.test.3 $2 {3:42:delegate {3:43:auto 2}}`:`{3:43:auto 2} xx{} yy{} {=list {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}} {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}`,

		`84 18:9:.test.3 $_ {3:46:delegate {3:47:auto _}}`:[]string{
			`{3:47:auto _} xx{} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}`,
			`{3:47:auto _} yy{} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}`,
		},
		`84 18:9:.test.3 a$_ {=compound {3:45:word a} {3:46:delegate {3:47:auto _}}}`:[]string{
			`axx{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}`,
			`ayy{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}`,
		},
		`84 18:9:.test.3 $(foreach $2,a$_) {3:32:delegate {3:34:builtin foreach} {=list {3:42:delegate {3:43:auto 2}}} {=list {=compound {3:45:word a} {3:46:delegate {3:47:auto _}}}}}`:`{3:34:builtin foreach} axx{} ayy{} {=list {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}} {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}`,
		`84 18:9:.test.3 $_ {3:51:delegate {3:47:auto _}}`:[]string{
			`{3:47:auto _} x{} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}`,
			`{3:47:auto _} y{} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}`,
			`{3:47:auto _} z{} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}`,
			`{3:47:auto _} axx{} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}`,
			`{3:47:auto _} ayy{} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}`,
		},
		`84 18:9:.test.3 b$_ {=compound {3:50:word b} {3:51:delegate {3:47:auto _}}}`:[]string{
			`bx{} {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}`,
			`by{} {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}`,
			`bz{} {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}`,
			`baxx{} {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}`,
			`bayy{} {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}`,
		},

		`84 18:9:.test.3 axx{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}`:`axx{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}`,
		`84 18:9:.test.3 ayy{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}`:`ayy{} {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}`,

		`84 18:9:.test.3 $(foreach $1 $(foreach $2,a$_),b$_) {3:19:delegate {3:21:builtin foreach} {=list {3:29:delegate {3:30:auto 1}} {3:32:delegate {3:34:builtin foreach} {=list {3:42:delegate {3:43:auto 2}}} {=list {=compound {3:45:word a} {3:46:delegate {3:47:auto _}}}}}} {=list {=compound {3:50:word b} {3:51:delegate {3:47:auto _}}}}}`:`{3:21:builtin foreach} bx{} by{} bz{} baxx{} bayy{} {=list {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}} {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}} {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}} {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}} {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}}}`,
		`84 18:9:.test.3 $(.test.foreach.a x$1 y$1 z$1,xx$2 yy$2) {4:19:delegate {3:17:def .test.foreach.a} {=list {=compound {4:37:word x} {4:38:delegate {3:30:auto 1}}} {=compound {4:41:word y} {4:42:delegate {3:30:auto 1}}} {=compound {4:45:word z} {4:46:delegate {3:30:auto 1}}}} {=list {=compound {4:49:word xx} {4:51:delegate {3:43:auto 2}}} {=compound {4:54:word yy} {4:56:delegate {3:43:auto 2}}}}}`:`{3:17:def .test.foreach.a} bx{} by{} bz{} baxx{} bayy{} {=list {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}}} {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}}} {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}}} {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}}} {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}}}}`,
		`84 18:9:.test.3 $(.test.foreach.b) {5:19:delegate {4:17:def .test.foreach.b}}`:`{4:17:def .test.foreach.b} bx{} by{} bz{} baxx{} bayy{} {=list {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}}}} {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}}}} {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}}}} {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}}}} {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}}}}}`,

		`84 18:9:.test.3 .test.foreach.x {=compound {6:17:punct .} {6:18:word test} {6:22:punct .} {6:23:word foreach} {6:30:punct .} {6:31:word x}}`:`.test.foreach.x {=compound {6:17:punct .} {6:18:word test} {6:22:punct .} {6:23:word foreach} {6:30:punct .} {6:31:word x}}`,
		`84 18:9:.test.3 &(.test.foreach.x) {6:15:closure {=compound {6:17:punct .} {6:18:word test} {6:22:punct .} {6:23:word foreach} {6:30:punct .} {6:31:word x}}}`:`{15:17:def .test.foreach.x} &(.test.foreach.x) {6:15:closure {15:17:def .test.foreach.x}}`,

		`84 18:9:.test.3 $1 {6:44:delegate {3:30:auto 1}}`:`{3:30:auto 1} 3 {6:44 {18:29 {1:9:word 3}}}`,
		`84 18:9:.test.3 $2 {6:47:delegate {3:43:auto 2}}`:`{3:43:auto 2} 4 {6:47 {18:32:decimal 4}}`,
		`84 18:9:.test.3 $_ {6:68:delegate {3:47:auto _}}`:[]string{
			`{3:47:auto _} 3 {6:68 {6:44 {18:29 {1:9:word 3}}}}`,
			`{3:47:auto _} 4 {6:68 {6:47 {18:32:decimal 4}}}`,
		},

		`84 18:9:.test.3 .test.foreach.x.$_ {=compound {6:52:punct .} {6:53:word test} {6:57:punct .} {6:58:word foreach} {6:65:punct .} {6:66:word x} {6:67:punct .} {6:68:delegate {3:47:auto _}}}`:[]string{
			`.test.foreach.x.3 {=compound {6:52:punct .} {6:53:word test} {6:57:punct .} {6:58:word foreach} {6:65:punct .} {6:66:word x} {6:67:punct .} {6:68 {6:44 {18:29 {1:9:word 3}}}}}`,
			`.test.foreach.x.4 {=compound {6:52:punct .} {6:53:word test} {6:57:punct .} {6:58:word foreach} {6:65:punct .} {6:66:word x} {6:67:punct .} {6:68 {6:47 {18:32:decimal 4}}}}`,
		},
		`84 18:9:.test.3 &(.test.foreach.x.$_) {6:50:closure {=compound {6:52:punct .} {6:53:word test} {6:57:punct .} {6:58:word foreach} {6:65:punct .} {6:66:word x} {6:67:punct .} {6:68:delegate {3:47:auto _}}}}`:[]string{
			`{19:19:def .test.foreach.x.3} &(.test.foreach.x.3) {6:50:closure {19:19:def .test.foreach.x.3}}`,
			`{20:19:def .test.foreach.x.4} &(.test.foreach.x.4) {6:50:closure {20:19:def .test.foreach.x.4}}`,
		},
		`84 18:9:.test.3 $(foreach $1 $2,&(.test.foreach.x.$_)) {6:34:delegate {6:36:builtin foreach} {=list {6:44:delegate {3:30:auto 1}} {6:47:delegate {3:43:auto 2}}} {=list {6:50:closure {=compound {6:52:punct .} {6:53:word test} {6:57:punct .} {6:58:word foreach} {6:65:punct .} {6:66:word x} {6:67:punct .} {6:68:delegate {3:47:auto _}}}}}}`:`{6:36:builtin foreach} &(.test.foreach.x.3) &(.test.foreach.x.4) {=list {6:34 {6:50:closure {19:19:def .test.foreach.x.3}}} {6:34 {6:50:closure {20:19:def .test.foreach.x.4}}}}`,
		`84 18:9:.test.3 &(.test.foreach.x) {6:15:closure {15:17:def .test.foreach.x}}`:`{15:17:def .test.foreach.x} &(.test.foreach.x) {6:15:closure {15:17:def .test.foreach.x}}`,

		`84 18:9:.test.3 $_ {6:75:delegate {3:47:auto _}}`:[]string{
			`{3:47:auto _} {&(.test.foreach.x)} {6:75 {6:15:disjunction {6:15:closure {15:17:def .test.foreach.x}}}}`,
			`{3:47:auto _} {&(.test.foreach.x.3)} {6:75 {6:34 {6:50:disjunction {6:50:closure {19:19:def .test.foreach.x.3}}}}}`,
			`{3:47:auto _} {&(.test.foreach.x.4)} {6:75 {6:34 {6:50:disjunction {6:50:closure {20:19:def .test.foreach.x.4}}}}}`,
		},
		`84 18:9:.test.3 x$_ {=compound {6:74:word x} {6:75:delegate {3:47:auto _}}}`:[]string{
			`x{&(.test.foreach.x)} {=compound {6:74:word x} {6:75 {6:15:disjunction {6:15:closure {15:17:def .test.foreach.x}}}}}`,
			`x{&(.test.foreach.x.3)} {=compound {6:74:word x} {6:75 {6:34 {6:50:disjunction {6:50:closure {19:19:def .test.foreach.x.3}}}}}}`,
			`x{&(.test.foreach.x.4)} {=compound {6:74:word x} {6:75 {6:34 {6:50:disjunction {6:50:closure {20:19:def .test.foreach.x.4}}}}}}`,
		},

		`84 18:9:.test.3 &(.test.foreach.x.3) {6:50:closure {19:19:def .test.foreach.x.3}}`:`{19:19:def .test.foreach.x.3} &(.test.foreach.x.3) {6:50:closure {19:19:def .test.foreach.x.3}}`,
		`84 18:9:.test.3 &(.test.foreach.x.4) {6:50:closure {20:19:def .test.foreach.x.4}}`:`{20:19:def .test.foreach.x.4} &(.test.foreach.x.4) {6:50:closure {20:19:def .test.foreach.x.4}}`,
		`84 18:9:.test.3 $(foreach &(.test.foreach.x) $(foreach $1 $2,&(.test.foreach.x.$_)),-x$_) {6:5:delegate {6:7:builtin foreach} {=list {6:15:closure {=compound {6:17:punct .} {6:18:word test} {6:22:punct .} {6:23:word foreach} {6:30:punct .} {6:31:word x}}} {6:34:delegate {6:36:builtin foreach} {=list {6:44:delegate {3:30:auto 1}} {6:47:delegate {3:43:auto 2}}} {=list {6:50:closure {=compound {6:52:punct .} {6:53:word test} {6:57:punct .} {6:58:word foreach} {6:65:punct .} {6:66:word x} {6:67:punct .} {6:68:delegate {3:47:auto _}}}}}}} {=list {=flag {=compound {6:74:word x} {6:75:delegate {3:47:auto _}}}}}}`:`{6:7:builtin foreach} -x{&(.test.foreach.x)} -x{&(.test.foreach.x.3)} -x{&(.test.foreach.x.4)} {=list {6:5 {=flag {=compound {6:74:word x} {6:75 {6:15:disjunction {6:15:closure {15:17:def .test.foreach.x}}}}}}} {6:5 {=flag {=compound {6:74:word x} {6:75 {6:34 {6:50:disjunction {6:50:closure {19:19:def .test.foreach.x.3}}}}}}}} {6:5 {=flag {=compound {6:74:word x} {6:75 {6:34 {6:50:disjunction {6:50:closure {20:19:def .test.foreach.x.4}}}}}}}}}`,
		`84 18:9:.test.3 $(.test.foreach.c $1,4) {18:11:delegate {5:17:def .test.foreach.c} {=list {18:29:delegate {3:30:auto 1}}} {=list {18:32:decimal 4}}}`:`{5:17:def .test.foreach.c} bx{} by{} bz{} baxx{} bayy{} -x{&(.test.foreach.x)} -x{&(.test.foreach.x.3)} -x{&(.test.foreach.x.4)} {=list {18:11 {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}}}}} {18:11 {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}}}}} {18:11 {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}}}}} {18:11 {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}}}}} {18:11 {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}}}}} {18:11 {6:5 {=flag {=compound {6:74:word x} {6:75 {6:15:disjunction {6:15:closure {15:17:def .test.foreach.x}}}}}}}} {18:11 {6:5 {=flag {=compound {6:74:word x} {6:75 {6:34 {6:50:disjunction {6:50:closure {19:19:def .test.foreach.x.3}}}}}}}}} {18:11 {6:5 {=flag {=compound {6:74:word x} {6:75 {6:34 {6:50:disjunction {6:50:closure {20:19:def .test.foreach.x.4}}}}}}}}}}`,

		`88 &(.test.foreach.x) {6:15:closure {15:17:def .test.foreach.x}}`:`{15:17:def .test.foreach.x} vw {6:15 {21:21:word vw}}`,
		`88 $1 {19:22:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {19:22 {3:30:null}}`,
		`88 $2 {19:24:delegate {3:43:auto 2}}`:`{3:43:auto 2} {} {19:24 {3:43:null}}`,
		`88 V$1$2 {=compound {19:21:word V} {19:22:delegate {3:30:auto 1}} {19:24:delegate {3:43:auto 2}}}`:`V{}{} {=compound {19:21:word V} {19:22 {3:30:null}} {19:24 {3:43:null}}}`,

		`88 &(.test.foreach.x.3) {6:50:closure {19:19:def .test.foreach.x.3}}`:`{19:19:def .test.foreach.x.3} V{}{} {6:50 {=compound {19:21:word V} {19:22 {3:30:null}} {19:24 {3:43:null}}}}`,
		`88 $1 {20:22:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {20:22 {3:30:null}}`,
		`88 $2 {20:24:delegate {3:43:auto 2}}`:`{3:43:auto 2} {} {20:24 {3:43:null}}`,
		`88 W$1$2 {=compound {20:21:word W} {20:22:delegate {3:30:auto 1}} {20:24:delegate {3:43:auto 2}}}`:`W{}{} {=compound {20:21:word W} {20:22 {3:30:null}} {20:24 {3:43:null}}}`,
		`88 &(.test.foreach.x.4) {6:50:closure {20:19:def .test.foreach.x.4}}`:`{20:19:def .test.foreach.x.4} W{}{} {6:50 {=compound {20:21:word W} {20:22 {3:30:null}} {20:24 {3:43:null}}}}`,

		`99 27:9:.test.4 $1 {24:31:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {24:31 {3:30:null}}`,
		`99 27:9:.test.4 $2 {24:34:delegate {3:43:auto 2}}`:`{3:43:auto 2} {} {24:34 {3:43:null}}`,
		`99 27:9:.test.4 $(foreach $1 $2,$(.test.foreach.d.$_ $3,$4)) {24:21:delegate {24:23:builtin foreach} {=list {24:31:delegate {3:30:auto 1}} {24:34:delegate {3:43:auto 2}}} {=list {24:37:delegate {=compound {24:39:punct .} {24:40:word test} {24:44:punct .} {24:45:word foreach} {24:52:punct .} {24:53:word d} {24:54:punct .} {24:55:delegate {3:47:auto _}}} {=list {24:58:delegate {24:59:auto 3}}} {=list {24:61:delegate {24:62:auto 4}}}}}}`:`{24:23:builtin foreach} {} {24:21 {24:23:null}}`,
		`99 27:9:.test.4 $(.test.foreach.d) {27:11:delegate {24:19:def .test.foreach.d}}`:`{24:19:def .test.foreach.d} {} {27:11 {24:21 {24:23:null}}}`,

		`105 27:9:.test.4 $1 {24:31:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {24:31 {3:30:null}}`,
		`105 27:9:.test.4 $2 {24:34:delegate {3:43:auto 2}}`:`{3:43:auto 2} {} {24:34 {3:43:null}}`,
		`105 27:9:.test.4 $(foreach $1 $2,$(.test.foreach.d.$_ $3,$4)) {24:21:delegate {24:23:builtin foreach} {=list {24:31:delegate {3:30:auto 1}} {24:34:delegate {3:43:auto 2}}} {=list {24:37:delegate {=compound {24:39:punct .} {24:40:word test} {24:44:punct .} {24:45:word foreach} {24:52:punct .} {24:53:word d} {24:54:punct .} {24:55:delegate {3:47:auto _}}} {=list {24:58:delegate {24:59:auto 3}}} {=list {24:61:delegate {24:62:auto 4}}}}}}`:`{24:23:builtin foreach} {} {24:21 {24:23:null}}`,
		`105 27:9:.test.4 $(.test.foreach.d) {27:11:delegate {24:19:def .test.foreach.d}}`:`{24:19:def .test.foreach.d} {} {27:11 {24:21 {24:23:null}}}`,

		`113 24:19:.test.foreach.d $1 {24:31:delegate {3:30:auto 1}}`:`{3:30:auto 1} 1 {24:31 {1:9:word 1}}`,
		`113 24:19:.test.foreach.d $2 {24:34:delegate {3:43:auto 2}}`:`{3:43:auto 2} 2 {24:34 {1:9:word 2}}`,
		`113 24:19:.test.foreach.d $_ {24:55:delegate {3:47:auto _}}`:[]string{`{3:47:auto _} 1 {24:55 {24:31 {1:9:word 1}}}`,`{3:47:auto _} 2 {24:55 {24:34 {1:9:word 2}}}`},
		`113 24:19:.test.foreach.d .test.foreach.d.$_ {=compound {24:39:punct .} {24:40:word test} {24:44:punct .} {24:45:word foreach} {24:52:punct .} {24:53:word d} {24:54:punct .} {24:55:delegate {3:47:auto _}}}`:[]string{
			`.test.foreach.d.1 {=compound {24:39:punct .} {24:40:word test} {24:44:punct .} {24:45:word foreach} {24:52:punct .} {24:53:word d} {24:54:punct .} {24:55 {24:31 {1:9:word 1}}}}`,
			`.test.foreach.d.2 {=compound {24:39:punct .} {24:40:word test} {24:44:punct .} {24:45:word foreach} {24:52:punct .} {24:53:word d} {24:54:punct .} {24:55 {24:34 {1:9:word 2}}}}`,
		},
		`113 24:19:.test.foreach.d $3 {24:58:delegate {24:59:auto 3}}`:`{24:59:auto 3} {} {24:58 {24:59:null}}`,
		`113 24:19:.test.foreach.d $4 {24:61:delegate {24:62:auto 4}}`:`{24:62:auto 4} {} {24:61 {24:62:null}}`,
		`113 24:19:.test.foreach.d $1 {25:31:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {25:31 {24:58 {24:59:null}}}`,
		`113 24:19:.test.foreach.d $2 {25:34:delegate {3:43:auto 2}}`:`{3:43:auto 2} {} {25:34 {24:61 {24:62:null}}}`,
		`113 24:19:.test.foreach.d $(foreach $1 $2,-x$_) {25:21:delegate {25:23:builtin foreach} {=list {25:31:delegate {3:30:auto 1}} {25:34:delegate {3:43:auto 2}}} {=list {=flag {=compound {25:38:word x} {25:39:delegate {3:47:auto _}}}}}}`:`{25:23:builtin foreach} {} {25:21 {25:23:null}}`,
		`113 24:19:.test.foreach.d $(.test.foreach.d.$_ $3,$4) {24:37:delegate {=compound {24:39:punct .} {24:40:word test} {24:44:punct .} {24:45:word foreach} {24:52:punct .} {24:53:word d} {24:54:punct .} {24:55:delegate {3:47:auto _}}} {=list {24:58:delegate {24:59:auto 3}}} {=list {24:61:delegate {24:62:auto 4}}}}`:[]string{
			`{25:19:def .test.foreach.d.1} {} {24:37 {25:21 {25:23:null}}}`,
			`{26:19:def .test.foreach.d.2} {} {24:37 {26:21 {26:23:null}}}`,
		},
		`113 24:19:.test.foreach.d $1 {26:31:delegate {3:30:auto 1}}`:`{3:30:auto 1} {} {26:31 {24:58 {24:59:null}}}`,
		`113 24:19:.test.foreach.d $2 {26:34:delegate {3:43:auto 2}}`:`{3:43:auto 2} {} {26:34 {24:61 {24:62:null}}}`,
		`113 24:19:.test.foreach.d $(foreach $1 $2,-y$_) {26:21:delegate {26:23:builtin foreach} {=list {26:31:delegate {3:30:auto 1}} {26:34:delegate {3:43:auto 2}}} {=list {=flag {=compound {26:38:word y} {26:39:delegate {3:47:auto _}}}}}}`:`{26:23:builtin foreach} {} {26:21 {26:23:null}}`,
		`113 24:19:.test.foreach.d $(foreach $1 $2,$(.test.foreach.d.$_ $3,$4)) {24:21:delegate {24:23:builtin foreach} {=list {24:31:delegate {3:30:auto 1}} {24:34:delegate {3:43:auto 2}}} {=list {24:37:delegate {=compound {24:39:punct .} {24:40:word test} {24:44:punct .} {24:45:word foreach} {24:52:punct .} {24:53:word d} {24:54:punct .} {24:55:delegate {3:47:auto _}}} {=list {24:58:delegate {24:59:auto 3}}} {=list {24:61:delegate {24:62:auto 4}}}}}}`:`{24:23:builtin foreach} {} {24:21 {24:23:null}}`,

		`119 24:19:.test.foreach.d $1 {24:31:delegate {3:30:auto 1}}`:`{3:30:auto 1} 1 {24:31 {1:9:word 1}}`,
		`119 24:19:.test.foreach.d $2 {24:34:delegate {3:43:auto 2}}`:`{3:43:auto 2} 2 {24:34 {1:9:word 2}}`,
		`119 24:19:.test.foreach.d $_ {24:55:delegate {3:47:auto _}}`:[]string{`{3:47:auto _} 1 {24:55 {24:31 {1:9:word 1}}}`,`{3:47:auto _} 2 {24:55 {24:34 {1:9:word 2}}}`},
		`119 24:19:.test.foreach.d .test.foreach.d.$_ {=compound {24:39:punct .} {24:40:word test} {24:44:punct .} {24:45:word foreach} {24:52:punct .} {24:53:word d} {24:54:punct .} {24:55:delegate {3:47:auto _}}}`:[]string{
			`.test.foreach.d.1 {=compound {24:39:punct .} {24:40:word test} {24:44:punct .} {24:45:word foreach} {24:52:punct .} {24:53:word d} {24:54:punct .} {24:55 {24:31 {1:9:word 1}}}}`,
			`.test.foreach.d.2 {=compound {24:39:punct .} {24:40:word test} {24:44:punct .} {24:45:word foreach} {24:52:punct .} {24:53:word d} {24:54:punct .} {24:55 {24:34 {1:9:word 2}}}}`,
		},
		`119 24:19:.test.foreach.d $3 {24:58:delegate {24:59:auto 3}}`:`{24:59:auto 3} a {24:58 {1:9:word a}}`,
		`119 24:19:.test.foreach.d $4 {24:61:delegate {24:62:auto 4}}`:`{24:62:auto 4} b {24:61 {1:9:word b}}`,
		`119 24:19:.test.foreach.d $1 {25:31:delegate {3:30:auto 1}}`:`{3:30:auto 1} a {25:31 {24:58 {1:9:word a}}}`,
		`119 24:19:.test.foreach.d $2 {25:34:delegate {3:43:auto 2}}`:`{3:43:auto 2} b {25:34 {24:61 {1:9:word b}}}`,
		`119 24:19:.test.foreach.d $_ {25:39:delegate {3:47:auto _}}`:[]string{`{3:47:auto _} a {25:39 {25:31 {24:58 {1:9:word a}}}}`,`{3:47:auto _} b {25:39 {25:34 {24:61 {1:9:word b}}}}`},
		`119 24:19:.test.foreach.d x$_ {=compound {25:38:word x} {25:39:delegate {3:47:auto _}}}`:[]string{`xa {=compound {25:38:word x} {25:39 {25:31 {24:58 {1:9:word a}}}}}`,`xb {=compound {25:38:word x} {25:39 {25:34 {24:61 {1:9:word b}}}}}`},
		`119 24:19:.test.foreach.d $(foreach $1 $2,-x$_) {25:21:delegate {25:23:builtin foreach} {=list {25:31:delegate {3:30:auto 1}} {25:34:delegate {3:43:auto 2}}} {=list {=flag {=compound {25:38:word x} {25:39:delegate {3:47:auto _}}}}}}`:[]string{
			`{25:23:builtin foreach} -xa -xb {=list {25:21 {=flag {=compound {25:38:word x} {25:39 {25:31 {24:58 {1:9:word a}}}}}}} {25:21 {=flag {=compound {25:38:word x} {25:39 {25:34 {24:61 {1:9:word b}}}}}}}}`,
		},
		`119 24:19:.test.foreach.d $(.test.foreach.d.$_ $3,$4) {24:37:delegate {=compound {24:39:punct .} {24:40:word test} {24:44:punct .} {24:45:word foreach} {24:52:punct .} {24:53:word d} {24:54:punct .} {24:55:delegate {3:47:auto _}}} {=list {24:58:delegate {24:59:auto 3}}} {=list {24:61:delegate {24:62:auto 4}}}}`:[]string{
			`{25:19:def .test.foreach.d.1} -xa -xb {=list {24:37 {25:21 {=flag {=compound {25:38:word x} {25:39 {25:31 {24:58 {1:9:word a}}}}}}}} {24:37 {25:21 {=flag {=compound {25:38:word x} {25:39 {25:34 {24:61 {1:9:word b}}}}}}}}}`,
			`{26:19:def .test.foreach.d.2} -ya -yb {=list {24:37 {26:21 {=flag {=compound {26:38:word y} {26:39 {26:31 {24:58 {1:9:word a}}}}}}}} {24:37 {26:21 {=flag {=compound {26:38:word y} {26:39 {26:34 {24:61 {1:9:word b}}}}}}}}}`,
		},
		`119 24:19:.test.foreach.d $1 {26:31:delegate {3:30:auto 1}}`:`{3:30:auto 1} a {26:31 {24:58 {1:9:word a}}}`,
		`119 24:19:.test.foreach.d $2 {26:34:delegate {3:43:auto 2}}`:`{3:43:auto 2} b {26:34 {24:61 {1:9:word b}}}`,
		`119 24:19:.test.foreach.d $_ {26:39:delegate {3:47:auto _}}`:[]string{
			`{3:47:auto _} a {26:39 {26:31 {24:58 {1:9:word a}}}}`,
			`{3:47:auto _} b {26:39 {26:34 {24:61 {1:9:word b}}}}`,
		},
		`119 24:19:.test.foreach.d y$_ {=compound {26:38:word y} {26:39:delegate {3:47:auto _}}}`:[]string{
			`ya {=compound {26:38:word y} {26:39 {26:31 {24:58 {1:9:word a}}}}}`,
			`yb {=compound {26:38:word y} {26:39 {26:34 {24:61 {1:9:word b}}}}}`,
		},
		`119 24:19:.test.foreach.d $(foreach $1 $2,-y$_) {26:21:delegate {26:23:builtin foreach} {=list {26:31:delegate {3:30:auto 1}} {26:34:delegate {3:43:auto 2}}} {=list {=flag {=compound {26:38:word y} {26:39:delegate {3:47:auto _}}}}}}`:[]string{
			`{26:23:builtin foreach} -ya -yb {=list {26:21 {=flag {=compound {26:38:word y} {26:39 {26:31 {24:58 {1:9:word a}}}}}}} {26:21 {=flag {=compound {26:38:word y} {26:39 {26:34 {24:61 {1:9:word b}}}}}}}}`,
		},
		`119 24:19:.test.foreach.d $(foreach $1 $2,$(.test.foreach.d.$_ $3,$4)) {24:21:delegate {24:23:builtin foreach} {=list {24:31:delegate {3:30:auto 1}} {24:34:delegate {3:43:auto 2}}} {=list {24:37:delegate {=compound {24:39:punct .} {24:40:word test} {24:44:punct .} {24:45:word foreach} {24:52:punct .} {24:53:word d} {24:54:punct .} {24:55:delegate {3:47:auto _}}} {=list {24:58:delegate {24:59:auto 3}}} {=list {24:61:delegate {24:62:auto 4}}}}}}`:[]string{
			`{24:23:builtin foreach} -xa -xb -ya -yb {=list {24:21 {24:37 {25:21 {=flag {=compound {25:38:word x} {25:39 {25:31 {24:58 {1:9:word a}}}}}}}}} {24:21 {24:37 {25:21 {=flag {=compound {25:38:word x} {25:39 {25:34 {24:61 {1:9:word b}}}}}}}}} {24:21 {24:37 {26:21 {=flag {=compound {26:38:word y} {26:39 {26:31 {24:58 {1:9:word a}}}}}}}}} {24:21 {24:37 {26:21 {=flag {=compound {26:38:word y} {26:39 {26:34 {24:61 {1:9:word b}}}}}}}}}}`,
		},
	},
}

var checkstrs__foreach2 = map[string]map[string]any{
	"check-builtins-foreach2_test.go": map[string]any{
		`44 0:0: $(.test.foreach.b) {9:11:delegate {4:17:def .test.foreach.b}}`:`{=list {9:11 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}}}} {9:11 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}}}} {9:11 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}}}} {9:11 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}}}} {9:11 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}}}}} bx by bz baxx bayy`,
		`76 0:0: $(.test.foreach.c $1,4) {18:11:delegate {5:17:def .test.foreach.c} {=list {18:29:delegate {3:30:auto 1}}} {=list {18:32:decimal 4}}}`:`{=list {18:11 {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:37:word x} {4:38 {3:30:null}}}}}}}}}} {18:11 {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:41:word y} {4:42 {3:30:null}}}}}}}}}} {18:11 {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:29 {=compound {4:45:word z} {4:46 {3:30:null}}}}}}}}}} {18:11 {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:49:word xx} {4:51 {3:43:null}}}}}}}}}}}}} {18:11 {5:19 {4:19 {3:19 {=compound {3:50:word b} {3:51 {3:32 {=compound {3:45:word a} {3:46 {3:42 {=compound {4:54:word yy} {4:56 {3:43:null}}}}}}}}}}}}} {18:11 {6:5 {=flag {=compound {6:74:word x} {6:75 {6:15 {21:21:word vw}}}}}}} {18:11 {6:5 {=flag {=compound {6:74:word x} {6:75 {6:34 {6:50 {=compound {20:21:word W} {20:22 {3:30:null}} {20:24 {3:43:null}}}}}}}}}}} bx by bz baxx bayy -xvw -xW`,
	},
}
