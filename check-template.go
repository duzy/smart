//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints_template = map[string]map[string]any{
	"loader.go": map[string]any{
		`var.xxx {=compound {6:1:word var} {6:4:punct .} {3:9:word xxx}}`:`var.xxx {=compound {6:1:word var} {6:4:punct .} {3:9:word xxx}}`,
		`var.yyy {=compound {6:1:word var} {6:4:punct .} {3:13:word yyy}}`:`var.yyy {=compound {6:1:word var} {6:4:punct .} {3:13:word yyy}}`,
		`var.zzz {=compound {6:1:word var} {6:4:punct .} {3:17:word zzz}}`:`var.zzz {=compound {6:1:word var} {6:4:punct .} {3:17:word zzz}}`,

		`.test.1 {=compound {13:1:punct .} {13:2:word test} {13:6:punct .} {12:9:decimal 1}}`:`.test.1 {=compound {13:1:punct .} {13:2:word test} {13:6:punct .} {12:9:decimal 1}}`,
		`.test.2 {=compound {13:1:punct .} {13:2:word test} {13:6:punct .} {12:11:decimal 2}}`:`.test.2 {=compound {13:1:punct .} {13:2:word test} {13:6:punct .} {12:11:decimal 2}}`,
		`.test.3 {=compound {13:1:punct .} {13:2:word test} {13:6:punct .} {12:13:decimal 3}}`:`.test.3 {=compound {13:1:punct .} {13:2:word test} {13:6:punct .} {12:13:decimal 3}}`,
		`.test.4 {=compound {13:1:punct .} {13:2:word test} {13:6:punct .} {12:15:decimal 4}}`:`.test.4 {=compound {13:1:punct .} {13:2:word test} {13:6:punct .} {12:15:decimal 4}}`,
		`.test.5 {=compound {13:1:punct .} {13:2:word test} {13:6:punct .} {12:17:decimal 5}}`:`.test.5 {=compound {13:1:punct .} {13:2:word test} {13:6:punct .} {12:17:decimal 5}}`,
		`.test.6 {=compound {13:1:punct .} {13:2:word test} {13:6:punct .} {12:19:decimal 6}}`:`.test.6 {=compound {13:1:punct .} {13:2:word test} {13:6:punct .} {12:19:decimal 6}}`,
		`.test.7 {=compound {13:1:punct .} {13:2:word test} {13:6:punct .} {12:21:decimal 7}}`:`.test.7 {=compound {13:1:punct .} {13:2:word test} {13:6:punct .} {12:21:decimal 7}}`,
		`.test.8 {=compound {13:1:punct .} {13:2:word test} {13:6:punct .} {12:23:decimal 8}}`:`.test.8 {=compound {13:1:punct .} {13:2:word test} {13:6:punct .} {12:23:decimal 8}}`,
		`.test.9 {=compound {13:1:punct .} {13:2:word test} {13:6:punct .} {12:25:decimal 9}}`:`.test.9 {=compound {13:1:punct .} {13:2:word test} {13:6:punct .} {12:25:decimal 9}}`,
		`.test.10 {=compound {13:1:punct .} {13:2:word test} {13:6:punct .} {12:27:decimal 10}}`:`.test.10 {=compound {13:1:punct .} {13:2:word test} {13:6:punct .} {12:27:decimal 10}}`,

		`.test.1 {=compound {17:1:punct .} {17:2:word test} {17:6:punct .} {17:7:decimal 1}}`:`.test.1 {=compound {17:1:punct .} {17:2:word test} {17:6:punct .} {17:7:decimal 1}}`,
		`.test.2 {=compound {21:1:punct .} {21:2:word test} {21:6:punct .} {21:7:decimal 2}}`:`.test.2 {=compound {21:1:punct .} {21:2:word test} {21:6:punct .} {21:7:decimal 2}}`,

		`.test.1 {=compound {29:1:punct .} {29:2:word test} {29:6:punct .} {29:7:decimal 1}}`:`.test.1 {=compound {29:1:punct .} {29:2:word test} {29:6:punct .} {29:7:decimal 1}}`,
		`.test.2 {=compound {33:1:punct .} {33:2:word test} {33:6:punct .} {33:7:decimal 2}}`:`.test.2 {=compound {33:1:punct .} {33:2:word test} {33:6:punct .} {33:7:decimal 2}}`,
		`.test.3 {=compound {25:1:punct .} {25:2:word test} {25:6:punct .} {25:7:decimal 3}}`:`.test.3 {=compound {25:1:punct .} {25:2:word test} {25:6:punct .} {25:7:decimal 3}}`,
		`.test.3 {=compound {37:1:punct .} {37:2:word test} {37:6:punct .} {37:7:decimal 3}}`:`.test.3 {=compound {37:1:punct .} {37:2:word test} {37:6:punct .} {37:7:decimal 3}}`,
		`.test.4 {=compound {48:1:punct .} {48:2:word test} {48:6:punct .} {48:7:decimal 4}}`:`.test.4 {=compound {48:1:punct .} {48:2:word test} {48:6:punct .} {48:7:decimal 4}}`,
		`.test.5 {=compound {53:1:punct .} {53:2:word test} {53:6:punct .} {53:7:decimal 5}}`:`.test.5 {=compound {53:1:punct .} {53:2:word test} {53:6:punct .} {53:7:decimal 5}}`,
		`.test.6 {=compound {58:1:punct .} {58:2:word test} {58:6:punct .} {58:7:decimal 6}}`:`.test.6 {=compound {58:1:punct .} {58:2:word test} {58:6:punct .} {58:7:decimal 6}}`,
		`.test.7 {=compound {63:1:punct .} {63:2:word test} {63:6:punct .} {63:7:decimal 7}}`:`.test.7 {=compound {63:1:punct .} {63:2:word test} {63:6:punct .} {63:7:decimal 7}}`,
		`.test.8 {=compound {68:1:punct .} {68:2:word test} {68:6:punct .} {68:7:decimal 8}}`:`.test.8 {=compound {68:1:punct .} {68:2:word test} {68:6:punct .} {68:7:decimal 8}}`,
		`.test.9 {=compound {73:1:punct .} {73:2:word test} {73:6:punct .} {73:7:decimal 9}}`:`.test.9 {=compound {73:1:punct .} {73:2:word test} {73:6:punct .} {73:7:decimal 9}}`,
		`.test.10 {=compound {78:1:punct .} {78:2:word test} {78:6:punct .} {78:7:decimal 10}}`:`.test.10 {=compound {78:1:punct .} {78:2:word test} {78:6:punct .} {78:7:decimal 10}}`,
		`.test.11 {=compound {83:1:punct .} {83:2:word test} {83:6:punct .} {83:7:decimal 11}}`:`.test.11 {=compound {83:1:punct .} {83:2:word test} {83:6:punct .} {83:7:decimal 11}}`,
		`.test.12 {=compound {88:1:punct .} {88:2:word test} {88:6:punct .} {88:7:decimal 12}}`:`.test.12 {=compound {88:1:punct .} {88:2:word test} {88:6:punct .} {88:7:decimal 12}}`,
		`.test.13 {=compound {93:1:punct .} {93:2:word test} {93:6:punct .} {93:7:decimal 13}}`:`.test.13 {=compound {93:1:punct .} {93:2:word test} {93:6:punct .} {93:7:decimal 13}}`,

		`$(xyz) {5:9:delegate {3:5:def xyz}}`:`{3:5:def xyz} xxx yyy zzz {=list {5:9 {3:9:word xxx}} {5:9 {3:13:word yyy}} {5:9 {3:17:word zzz}}}`,
		`$(xyz) {16:10:delegate {3:5:def xyz}}`:`{3:5:def xyz} xxx yyy zzz {=list {16:10 {3:9:word xxx}} {16:10 {3:13:word yyy}} {16:10 {3:17:word zzz}}}`,
		`$(xyz) {16:11:delegate {3:5:def xyz}}`:`{3:5:def xyz} xxx yyy zzz {=list {16:11 {3:9:word xxx}} {16:11 {3:13:word yyy}} {16:11 {3:17:word zzz}}}`,
		`$(xyz) {28:10:delegate {3:5:def xyz}}`:`{3:5:def xyz} xxx yyy zzz {=list {28:10 {3:9:word xxx}} {28:10 {3:13:word yyy}} {28:10 {3:17:word zzz}}}`,
		`$(xyz) {28:11:delegate {3:5:def xyz}}`:`{3:5:def xyz} xxx yyy zzz {=list {28:11 {3:9:word xxx}} {28:11 {3:13:word yyy}} {28:11 {3:17:word zzz}}}`,

		`$(defs(-capture=1 -sort) {=regex ^var\.([xyz]+)}) {20:9:delegate {20:11:builtin defs} [{=pair {=flag {20:17:word capture}} {20:25:decimal 1}} {=flag {20:28:word sort}}] {=list {20:42:regex ^var\.([xyz]+)}}}`:`{20:11:builtin defs} xxx yyy zzz {=list {20:9 {20:11:word xxx}} {20:9 {20:11:word yyy}} {20:9 {20:11:word zzz}}}`,
		`$(defs(-capture=1 -sort) {=regex ^var\.([xyz]+)}) {32:9:delegate {32:11:builtin defs} [{=pair {=flag {32:17:word capture}} {32:25:decimal 1}} {=flag {32:28:word sort}}] {=list {32:42:regex ^var\.([xyz]+)}}}`:`{32:11:builtin defs} xxx yyy zzz {=list {32:9 {32:11:word xxx}} {32:9 {32:11:word yyy}} {32:9 {32:11:word zzz}}}`,

		`9:6:vars $(defs(-capture=1 -sort) {=regex ^var\.([xyz]+)}) {9:9:delegate {9:11:builtin defs} [{=pair {=flag {9:17:word capture}} {9:25:decimal 1}} {=flag {9:28:word sort}}] {=list {9:43:regex ^var\.([xyz]+)}}}`:`{9:11:builtin defs} xxx yyy zzz {=list {9:9 {9:11:word xxx}} {9:9 {9:11:word yyy}} {9:9 {9:11:word zzz}}}`,
		`10:6:var2 $(defs(-capture=1 -sort) !{=regex ^var\.([xy]+)}) {10:9:delegate {10:11:builtin defs} [{=pair {=flag {10:17:word capture}} {10:25:decimal 1}} {=flag {10:28:word sort}}] {=list {=negative {10:43:regex ^var\.([xy]+)}}}}`:`{10:11:builtin defs} .self .usee var.zzz var2 vars xyz {=list {10:9 {10:11:word .self}} {10:9 {10:11:word .usee}} {10:9 {10:11:word var.zzz}} {10:9 {10:11:word var2}} {10:9 {10:11:word vars}} {10:9 {10:11:word xyz}}}`,
	},
	"check-template_test.go": map[string]any{
	},
}

var checkstrs_template = map[string]map[string]any{
	"loader.go": map[string]any{
	},
	"check-template_test.go": map[string]any{
	},
}
