//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints__if = map[string]map[string]any{
	"loader.go": map[string]any{
		`7:5:x3 $(if {=yes},yes,no) {5:8:delegate {5:10:builtin if} {=list {5:18:yes}} {=list {5:20:word yes}} {=list {5:24:word no}}}`:`yes {5:8 {5:20:word yes}}`,
		`7:5:x3 $(x1) {7:8:delegate {5:6:def x1}}`:`yes {7:8 {5:8 {5:20:word yes}}}`,

		`8:5:x4 $(if {=no},yes,no) {6:8:delegate {6:10:builtin if} {=list {6:17:no}} {=list {6:19:word yes}} {=list {6:23:word no}}}`:`no {6:8 {6:23:word no}}`,
		`8:5:x4 $(x2) {8:8:delegate {6:6:def x2}}`:`no {8:8 {6:8 {6:23:word no}}}`,

		`9:5:x5 $(if {=yes},yes,no) {9:8:delegate {9:10:builtin if} {=list {9:18:yes}} {=list {9:20:word yes}} {=list {9:24:word no}}}`:`yes {9:8 {9:20:word yes}}`,

		`10:5:x6 $(if {=no},yes,no) {10:8:delegate {10:10:builtin if} {=list {10:17:no}} {=list {10:19:word yes}} {=list {10:23:word no}}}`:`no {10:8 {10:23:word no}}`,

		`11:5:x7 &(none) {11:13:closure {11:15:word none}}`:`&(none) {11:13:closure {11:15:word none}}`,
		`11:5:x7 $(if &(none),yes,no) {11:8:delegate {11:10:builtin if} {=list {11:13:closure {11:15:word none}}} {=list {11:21:word yes}} {=list {11:25:word no}}}`:`no {11:8 {11:25:word no}}`,

		`12:5:x8 &(some) {12:13:closure {3:6:def some}}`:`&(some) {12:13:closure {3:6:def some}}`,
		`12:5:x8 $(if &(some),yes,no) {12:8:delegate {12:10:builtin if} {=list {12:13:closure {3:6:def some}}} {=list {12:21:word yes}} {=list {12:25:word no}}}`:`no {12:8 {12:25:word no}}`,

		`13:4:x9 &(none) {13:13:closure {13:15:word none}}`:`{} {13:13:null}`,
		`13:4:x9 $(if &(none),yes,no) {13:8:delegate {13:10:builtin if} {=list {13:13:closure {13:15:word none}}} {=list {13:21:word yes}} {=list {13:25:word no}}}`:`no {13:8 {13:25:word no}}`,

		`16:5:x11 $(ifarg 1,yes,no) {16:8:delegate {16:10:builtin ifarg} {=list {16:16:decimal 1}} {=list {16:18:word yes}} {=list {16:22:word no}}}`:`no {16:8 {16:22:word no}}`,

		`18:5:x12 $(ifdef none,yes,no) {18:8:delegate {18:10:builtin ifdef} {=list {18:16:word none}} {=list {18:21:word yes}} {=list {18:25:word no}}}`:`no {18:8 {18:25:word no}}`,

		`20:5:x81 &(some) {20:14:closure {3:6:def some}}`:`thing {20:14 {3:9:word thing}}`,
		`20:5:x81 $(if &(some),yes,no) {20:9:delegate {20:11:builtin if} {=list {20:14:closure {3:6:def some}}} {=list {20:22:word yes}} {=list {20:26:word no}}}`:`yes {20:9 {20:22:word yes}}`,
	},
	"check-builtins-if_test.go": map[string]any{
		`19 5:6:x1 $(if {=yes},yes,no) {5:8:delegate {5:10:builtin if} {=list {5:18:yes}} {=list {5:20:word yes}} {=list {5:24:word no}}}`:`yes {5:8 {5:20:word yes}}`,
		`30 6:6:x2 $(if {=no},yes,no) {6:8:delegate {6:10:builtin if} {=list {6:17:no}} {=list {6:19:word yes}} {=list {6:23:word no}}}`:`no {6:8 {6:23:word no}}`,
		`122 15:6:x10 $(ifarg 1,yes,no) {15:8:delegate {15:10:builtin ifarg} {=list {15:16:decimal 1}} {=list {15:18:word yes}} {=list {15:22:word no}}}`:`no {15:8 {15:22:word no}}`,
		`124 15:6:x10 $(ifarg 1,yes,no) {15:8:delegate {15:10:builtin ifarg} {=list {15:16:decimal 1}} {=list {15:18:word yes}} {=list {15:22:word no}}}`:`yes {15:8 {15:18:word yes}}`,
		`134 15:6:x10 $(ifarg 1,yes,no) {15:8:delegate {15:10:builtin ifarg} {=list {15:16:decimal 1}} {=list {15:18:word yes}} {=list {15:22:word no}}}`:`no {15:8 {15:22:word no}}`,
		`136 15:6:x10 $(ifarg 1,yes,no) {15:8:delegate {15:10:builtin ifarg} {=list {15:16:decimal 1}} {=list {15:18:word yes}} {=list {15:22:word no}}}`:`yes {15:8 {15:18:word yes}}`,
	},
}

var checkstrs__if = map[string]map[string]any{
	"check-builtins-if_test.go": map[string]any{
		`19 5:6:x1 $(if {=yes},yes,no) {5:8:delegate {5:10:builtin if} {=list {5:18:yes}} {=list {5:20:word yes}} {=list {5:24:word no}}}`:`{5:8 {5:20:word yes}} yes`,
		`30 6:6:x2 $(if {=no},yes,no) {6:8:delegate {6:10:builtin if} {=list {6:17:no}} {=list {6:19:word yes}} {=list {6:23:word no}}}`:`{6:8 {6:23:word no}} no`,
		`122 15:6:x10 $(ifarg 1,yes,no) {15:8:delegate {15:10:builtin ifarg} {=list {15:16:decimal 1}} {=list {15:18:word yes}} {=list {15:22:word no}}}`:`{15:8 {15:22:word no}} no`,
		`134 15:6:x10 $(ifarg 1,yes,no) {15:8:delegate {15:10:builtin ifarg} {=list {15:16:decimal 1}} {=list {15:18:word yes}} {=list {15:22:word no}}}`:`{15:8 {15:22:word no}} no`,
	},
}
