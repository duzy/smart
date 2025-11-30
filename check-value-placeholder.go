//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints_value_placeholder = map[string]map[string]any{
	"loader.go": map[string]any{
		`foo1 {=compound {10:1:word foo} {9:24:raw 1}}`:`foo1 {=compound {10:1:word foo} {9:24:raw 1}}`,
		`foo2 {=compound {10:1:word foo} {9:24:raw 2}}`:`foo2 {=compound {10:1:word foo} {9:24:raw 2}}`,
		`foo3 {=compound {10:1:word foo} {9:24:raw 3}}`:`foo3 {=compound {10:1:word foo} {9:24:raw 3}}`,
		`foo4 {=compound {10:1:word foo} {9:24:raw 4}}`:`foo4 {=compound {10:1:word foo} {9:24:raw 4}}`,
		`foo5 {=compound {10:1:word foo} {9:24:raw 5}}`:`foo5 {=compound {10:1:word foo} {9:24:raw 5}}`,

		`6:6:val4 $_ {6:31:delegate {4:32:auto _}}`:[]string{
			`{4:32:auto _} a {6:31 {6:19:word a}}`,
			`{4:32:auto _} b {6:31 {6:21:word b}}`,
			`{4:32:auto _} c {6:31 {6:23:word c}}`,
			`{4:32:auto _} d {6:31 {6:25:word d}}`,
			`{4:32:auto _} e {6:31 {6:27:word e}}`,
			`{4:32:auto _} f {6:31 {6:29:word f}}`,
		},
		`6:6:val4 $(foreach a b c d e f,$_) {6:9:delegate {6:11:builtin foreach} {=list {6:19:word a} {6:21:word b} {6:23:word c} {6:25:word d} {6:27:word e} {6:29:word f}} {=list {6:31:delegate {4:32:auto _}}}}`:`{6:11:builtin foreach} a b c d e f {=list {6:9 {6:31 {6:19:word a}}} {6:9 {6:31 {6:21:word b}}} {6:9 {6:31 {6:23:word c}}} {6:9 {6:31 {6:25:word d}}} {6:9 {6:31 {6:27:word e}}} {6:9 {6:31 {6:29:word f}}}}`,

		`7:6:val5 $1 {7:19:delegate {3:10:auto 1}}`:`{3:10:auto 1} {} {7:19 {3:10:null}}`,
		`7:6:val5 $2 {7:22:delegate {3:13:auto 2}}`:`{3:13:auto 2} {} {7:22 {3:13:null}}`,
		`7:6:val5 $3 {7:25:delegate {3:16:auto 3}}`:`{3:16:auto 3} {} {7:25 {3:16:null}}`,
		`7:6:val5 $4 {7:28:delegate {3:19:auto 4}}`:`{3:19:auto 4} {} {7:28 {3:19:null}}`,
		`7:6:val5 $5 {7:31:delegate {3:22:auto 5}}`:`{3:22:auto 5} {} {7:31 {3:22:null}}`,
		`7:6:val5 $6 {7:34:delegate {3:25:auto 6}}`:`{3:25:auto 6} {} {7:34 {3:25:null}}`,
		`7:6:val5 $7 {7:37:delegate {3:28:auto 7}}`:`{3:28:auto 7} {} {7:37 {3:28:null}}`,
		`7:6:val5 $8 {7:40:delegate {3:31:auto 8}}`:`{3:31:auto 8} {} {7:40 {3:31:null}}`,
		`7:6:val5 $9 {7:43:delegate {3:34:auto 9}}`:`{3:34:auto 9} {} {7:43 {3:34:null}}`,
		`7:6:val5 $_ {7:46:delegate {4:32:auto _}}`:[]string{
			`{4:32:auto _} {} {7:46 {7:19 {3:10:null}}}`,
			`{4:32:auto _} {} {7:46 {7:22 {3:13:null}}}`,
			`{4:32:auto _} {} {7:46 {7:25 {3:16:null}}}`,
			`{4:32:auto _} {} {7:46 {7:28 {3:19:null}}}`,
			`{4:32:auto _} {} {7:46 {7:31 {3:22:null}}}`,
			`{4:32:auto _} {} {7:46 {7:34 {3:25:null}}}`,
			`{4:32:auto _} {} {7:46 {7:37 {3:28:null}}}`,
			`{4:32:auto _} {} {7:46 {7:40 {3:31:null}}}`,
			`{4:32:auto _} {} {7:46 {7:43 {3:34:null}}}`,
		},
		`7:6:val5 $(foreach $1 $2 $3 $4 $5 $6 $7 $8 $9,$_) {7:9:delegate {7:11:builtin foreach} {=list {7:19:delegate {3:10:auto 1}} {7:22:delegate {3:13:auto 2}} {7:25:delegate {3:16:auto 3}} {7:28:delegate {3:19:auto 4}} {7:31:delegate {3:22:auto 5}} {7:34:delegate {3:25:auto 6}} {7:37:delegate {3:28:auto 7}} {7:40:delegate {3:31:auto 8}} {7:43:delegate {3:34:auto 9}}} {=list {7:46:delegate {4:32:auto _}}}}`:`{7:11:builtin foreach} {} {7:9 {7:11:null}}`,
	},
	"check-value-placeholder_test.go": map[string]any{
		`16 3:7:val1 $1 {3:9:delegate {3:10:auto 1}}`:`{3:10:auto 1} 1 {3:9 {1:9:word 1}}`,
		`16 3:7:val1 $2 {3:12:delegate {3:13:auto 2}}`:`{3:13:auto 2} 2 {3:12 {1:9:word 2}}`,
		`16 3:7:val1 $3 {3:15:delegate {3:16:auto 3}}`:`{3:16:auto 3} 3 {3:15 {1:9:word 3}}`,
		`16 3:7:val1 $4 {3:18:delegate {3:19:auto 4}}`:`{3:19:auto 4} 4 {3:18 {1:9:word 4}}`,
		`16 3:7:val1 $5 {3:21:delegate {3:22:auto 5}}`:`{3:22:auto 5} 5 {3:21 {1:9:word 5}}`,
		`16 3:7:val1 $6 {3:24:delegate {3:25:auto 6}}`:`{3:25:auto 6} 6 {3:24 {1:9:word 6}}`,
		`16 3:7:val1 $7 {3:27:delegate {3:28:auto 7}}`:`{3:28:auto 7} 7 {3:27 {1:9:word 7}}`,
		`16 3:7:val1 $8 {3:30:delegate {3:31:auto 8}}`:`{3:31:auto 8} 8 {3:30 {1:9:word 8}}`,
		`16 3:7:val1 $9 {3:33:delegate {3:34:auto 9}}`:`{3:34:auto 9} 9 {3:33 {1:9:word 9}}`,

		`28 4:7:val2 $_ {4:31:delegate {4:32:auto _}}`:[]string{
			`{4:32:auto _} a {4:31 {4:19:word a}}`,
			`{4:32:auto _} b {4:31 {4:21:word b}}`,
			`{4:32:auto _} c {4:31 {4:23:word c}}`,
			`{4:32:auto _} d {4:31 {4:25:word d}}`,
			`{4:32:auto _} e {4:31 {4:27:word e}}`,
			`{4:32:auto _} f {4:31 {4:29:word f}}`,
		},
		`28 4:7:val2 $(foreach a b c d e f,$_) {4:9:delegate {4:11:builtin foreach} {=list {4:19:word a} {4:21:word b} {4:23:word c} {4:25:word d} {4:27:word e} {4:29:word f}} {=list {4:31:delegate {4:32:auto _}}}}`:`{4:11:builtin foreach} a b c d e f {=list {4:9 {4:31 {4:19:word a}}} {4:9 {4:31 {4:21:word b}}} {4:9 {4:31 {4:23:word c}}} {4:9 {4:31 {4:25:word d}}} {4:9 {4:31 {4:27:word e}}} {4:9 {4:31 {4:29:word f}}}}`,

		`40 5:7:val3 $1 {5:19:delegate {3:10:auto 1}}`:`{3:10:auto 1} 1 {5:19 {1:9:word 1}}`,
		`40 5:7:val3 $2 {5:22:delegate {3:13:auto 2}}`:`{3:13:auto 2} 2 {5:22 {1:9:word 2}}`,
		`40 5:7:val3 $3 {5:25:delegate {3:16:auto 3}}`:`{3:16:auto 3} 3 {5:25 {1:9:word 3}}`,
		`40 5:7:val3 $4 {5:28:delegate {3:19:auto 4}}`:`{3:19:auto 4} 4 {5:28 {1:9:word 4}}`,
		`40 5:7:val3 $5 {5:31:delegate {3:22:auto 5}}`:`{3:22:auto 5} 5 {5:31 {1:9:word 5}}`,
		`40 5:7:val3 $6 {5:34:delegate {3:25:auto 6}}`:`{3:25:auto 6} 6 {5:34 {1:9:word 6}}`,
		`40 5:7:val3 $7 {5:37:delegate {3:28:auto 7}}`:`{3:28:auto 7} 7 {5:37 {1:9:word 7}}`,
		`40 5:7:val3 $8 {5:40:delegate {3:31:auto 8}}`:`{3:31:auto 8} 8 {5:40 {1:9:word 8}}`,
		`40 5:7:val3 $9 {5:43:delegate {3:34:auto 9}}`:`{3:34:auto 9} 9 {5:43 {1:9:word 9}}`,
		`40 5:7:val3 $_ {5:46:delegate {4:32:auto _}}`:[]string{
			`{4:32:auto _} 1 {5:46 {5:19 {1:9:word 1}}}`,
			`{4:32:auto _} 2 {5:46 {5:22 {1:9:word 2}}}`,
			`{4:32:auto _} 3 {5:46 {5:25 {1:9:word 3}}}`,
			`{4:32:auto _} 4 {5:46 {5:28 {1:9:word 4}}}`,
			`{4:32:auto _} 5 {5:46 {5:31 {1:9:word 5}}}`,
			`{4:32:auto _} 6 {5:46 {5:34 {1:9:word 6}}}`,
			`{4:32:auto _} 7 {5:46 {5:37 {1:9:word 7}}}`,
			`{4:32:auto _} 8 {5:46 {5:40 {1:9:word 8}}}`,
			`{4:32:auto _} 9 {5:46 {5:43 {1:9:word 9}}}`,
		},
		`40 5:7:val3 $(foreach $1 $2 $3 $4 $5 $6 $7 $8 $9,$_) {5:9:delegate {5:11:builtin foreach} {=list {5:19:delegate {3:10:auto 1}} {5:22:delegate {3:13:auto 2}} {5:25:delegate {3:16:auto 3}} {5:28:delegate {3:19:auto 4}} {5:31:delegate {3:22:auto 5}} {5:34:delegate {3:25:auto 6}} {5:37:delegate {3:28:auto 7}} {5:40:delegate {3:31:auto 8}} {5:43:delegate {3:34:auto 9}}} {=list {5:46:delegate {4:32:auto _}}}}`:`{5:11:builtin foreach} 1 2 3 4 5 6 7 8 9 {=list {5:9 {5:46 {5:19 {1:9:word 1}}}} {5:9 {5:46 {5:22 {1:9:word 2}}}} {5:9 {5:46 {5:25 {1:9:word 3}}}} {5:9 {5:46 {5:28 {1:9:word 4}}}} {5:9 {5:46 {5:31 {1:9:word 5}}}} {5:9 {5:46 {5:34 {1:9:word 6}}}} {5:9 {5:46 {5:37 {1:9:word 7}}}} {5:9 {5:46 {5:40 {1:9:word 8}}}} {5:9 {5:46 {5:43 {1:9:word 9}}}}}`,
		`40 5:7:val3 $(foreach 1 2 3 4 5 6 7 8 9,$_) {5:9:delegate {5:11:builtin foreach} {=list {5:19 {1:9:word 1}} {5:22 {1:9:word 2}} {5:25 {1:9:word 3}} {5:28 {1:9:word 4}} {5:31 {1:9:word 5}} {5:34 {1:9:word 6}} {5:37 {1:9:word 7}} {5:40 {1:9:word 8}} {5:43 {1:9:word 9}}} {=list {5:46:delegate {4:32:auto _}}}}`:`{5:11:builtin foreach} 1 2 3 4 5 6 7 8 9 {=list {5:9 {5:46 {5:19 {1:9:word 1}}}} {5:9 {5:46 {5:22 {1:9:word 2}}}} {5:9 {5:46 {5:25 {1:9:word 3}}}} {5:9 {5:46 {5:28 {1:9:word 4}}}} {5:9 {5:46 {5:31 {1:9:word 5}}}} {5:9 {5:46 {5:34 {1:9:word 6}}}} {5:9 {5:46 {5:37 {1:9:word 7}}}} {5:9 {5:46 {5:40 {1:9:word 8}}}} {5:9 {5:46 {5:43 {1:9:word 9}}}}}`,
	},
}

var checkstrs_value_placeholder = map[string]map[string]any{
}
