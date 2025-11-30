//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints__foreach1 = map[string]map[string]any{
	"loader.go": map[string]any{
		`.test.x {=compound {3:1:punct .} {3:2:word test} {3:6:punct .} {3:7:word x}}`:`.test.x {=compound {3:1:punct .} {3:2:word test} {3:6:punct .} {3:7:word x}}`,
		`.test.1 {=compound {6:1:punct .} {6:2:word test} {6:6:punct .} {6:7:decimal 1}}`:`.test.1 {=compound {6:1:punct .} {6:2:word test} {6:6:punct .} {6:7:decimal 1}}`,
		`.test.s {=compound {8:1:punct .} {8:2:word test} {8:6:punct .} {8:7:word s}}`:`.test.s {=compound {8:1:punct .} {8:2:word test} {8:6:punct .} {8:7:word s}}`,
		`.test.A {=compound {10:1:punct .} {10:2:word test} {10:6:punct .} {10:7:word A}}`:`.test.A {=compound {10:1:punct .} {10:2:word test} {10:6:punct .} {10:7:word A}}`,
		`.test.a {=compound {11:1:punct .} {11:2:word test} {11:6:punct .} {11:7:word a}}`:`.test.a {=compound {11:1:punct .} {11:2:word test} {11:6:punct .} {11:7:word a}}`,
		`.test.B {=compound {12:1:punct .} {12:2:word test} {12:6:punct .} {12:7:word B}}`:`.test.B {=compound {12:1:punct .} {12:2:word test} {12:6:punct .} {12:7:word B}}`,
		`.test.b {=compound {13:1:punct .} {13:2:word test} {13:6:punct .} {13:7:word b}}`:`.test.b {=compound {13:1:punct .} {13:2:word test} {13:6:punct .} {13:7:word b}}`,
		`.test~foo {=compound {14:1:punct .} {14:2:word test} {14:6:punct ~} {14:7:word foo}}`:`.test~foo {=compound {14:1:punct .} {14:2:word test} {14:6:punct ~} {14:7:word foo}}`,
		`.test~foo.A {=compound {15:1:punct .} {15:2:word test} {15:6:punct ~} {15:7:word foo} {15:10:punct .} {15:11:word A}}`:`.test~foo.A {=compound {15:1:punct .} {15:2:word test} {15:6:punct ~} {15:7:word foo} {15:10:punct .} {15:11:word A}}`,
		`.test~foo.a {=compound {16:1:punct .} {16:2:word test} {16:6:punct ~} {16:7:word foo} {16:10:punct .} {16:11:word a}}`:`.test~foo.a {=compound {16:1:punct .} {16:2:word test} {16:6:punct ~} {16:7:word foo} {16:10:punct .} {16:11:word a}}`,
		`.test~foo.B {=compound {17:1:punct .} {17:2:word test} {17:6:punct ~} {17:7:word foo} {17:10:punct .} {17:11:word B}}`:`.test~foo.B {=compound {17:1:punct .} {17:2:word test} {17:6:punct ~} {17:7:word foo} {17:10:punct .} {17:11:word B}}`,
		`.test~foo.b {=compound {18:1:punct .} {18:2:word test} {18:6:punct ~} {18:7:word foo} {18:10:punct .} {18:11:word b}}`:`.test~foo.b {=compound {18:1:punct .} {18:2:word test} {18:6:punct ~} {18:7:word foo} {18:10:punct .} {18:11:word b}}`,
		`10:13:.test.A test-A {=compound {10:16:word test} {=flag {10:21:word A}}}`:`test-A {=compound {10:16:word test} {=flag {10:21:word A}}}`,
		`11:13:.test.a test-a {=compound {11:16:word test} {=flag {11:21:word a}}}`:`test-a {=compound {11:16:word test} {=flag {11:21:word a}}}`,
		`12:13:.test.B test-B {=compound {12:16:word test} {=flag {12:21:word B}}}`:`test-B {=compound {12:16:word test} {=flag {12:21:word B}}}`,
		`13:13:.test.b test-b {=compound {13:16:word test} {=flag {13:21:word b}}}`:`test-b {=compound {13:16:word test} {=flag {13:21:word b}}}`,
		`14:13:.test~foo test-foo {=compound {14:16:word test} {=flag {14:21:word foo}}}`:`test-foo {=compound {14:16:word test} {=flag {14:21:word foo}}}`,
		`15:13:.test~foo.A test-foo-A {=compound {15:16:word test} {=flag {=compound {15:21:word foo} {=flag {15:25:word A}}}}}`:`test-foo-A {=compound {15:16:word test} {=flag {=compound {15:21:word foo} {=flag {15:25:word A}}}}}`,
		`16:13:.test~foo.a test-foo-a {=compound {16:16:word test} {=flag {=compound {16:21:word foo} {=flag {16:25:word a}}}}}`:`test-foo-a {=compound {16:16:word test} {=flag {=compound {16:21:word foo} {=flag {16:25:word a}}}}}`,
		`17:13:.test~foo.B test-foo-B {=compound {17:16:word test} {=flag {=compound {17:21:word foo} {=flag {17:25:word B}}}}}`:`test-foo-B {=compound {17:16:word test} {=flag {=compound {17:21:word foo} {=flag {17:25:word B}}}}}`,
		`18:13:.test~foo.b test-foo-b {=compound {18:16:word test} {=flag {=compound {18:21:word foo} {=flag {18:25:word b}}}}}`:`test-foo-b {=compound {18:16:word test} {=flag {=compound {18:21:word foo} {=flag {18:25:word b}}}}}`,
	},
	"check-builtins-foreach1_test.go": map[string]any{
		`21 $1 {6:21:delegate {4:16:auto 1}}`:`{4:16:auto 1} {} {6:21 {4:16:null}}`,
		`21 $1 {4:15:delegate {4:16:auto 1}}`:`{4:16:auto 1} {} {4:15 {6:21 {4:16:null}}}`,
		`21 $2 {4:18:delegate {4:19:auto 2}}`:`{4:19:auto 2} B b {=list {4:18 {6:24:word B}} {4:18 {6:26:word b}}}`,
		`21 .test.s {=compound {3:13:punct .} {3:14:word test} {3:18:punct .} {3:19:word s}}`:`.test.s {=compound {3:13:punct .} {3:14:word test} {3:18:punct .} {3:19:word s}}`,
		`21 .test.s {=compound {3:38:punct .} {3:39:word test} {3:43:punct .} {3:44:word s}}`:`.test.s {=compound {3:38:punct .} {3:39:word test} {3:43:punct .} {3:44:word s}}`,
		`21 &(.test.s) {3:11:closure {=compound {3:13:punct .} {3:14:word test} {3:18:punct .} {3:19:word s}}}`:`{8:13:def .test.s} foo {3:11 {8:16:word foo}}`,
		`21 &(.test.s) {3:36:closure {=compound {3:38:punct .} {3:39:word test} {3:43:punct .} {3:44:word s}}}`:`{8:13:def .test.s} foo {3:36 {8:16:word foo}}`,
		`21 $(value .test~&(.test.s)) {3:22:delegate {3:24:builtin value} {=list {=compound {3:30:punct .} {3:31:word test} {3:35:punct ~} {3:36:closure {=compound {3:38:punct .} {3:39:word test} {3:43:punct .} {3:44:word s}}}}}}`:`{3:24:builtin value} test-foo {3:22 {=compound {14:16:word test} {=flag {14:21:word foo}}}}`,

		`21 $_ {4:29:delegate {4:30:auto _}}`:[]string{`{4:30:auto _} B {4:29 {4:18 {6:24:word B}}}`,`{4:30:auto _} b {4:29 {4:18 {6:26:word b}}}`},
		`21 .test.$_ {=compound {4:23:punct .} {4:24:word test} {4:28:punct .} {4:29:delegate {4:30:auto _}}}`:[]string{`.test.B {=compound {4:23:punct .} {4:24:word test} {4:28:punct .} {4:29 {4:18 {6:24:word B}}}}`,`.test.b {=compound {4:23:punct .} {4:24:word test} {4:28:punct .} {4:29 {4:18 {6:26:word b}}}}`},
		`21 test-B {=compound {12:16:word test} {=flag {12:21:word B}}}`:`test-B {=compound {12:16:word test} {=flag {12:21:word B}}}`,
		`21 &(.test.$_) {4:21:closure {=compound {4:23:punct .} {4:24:word test} {4:28:punct .} {4:29:delegate {4:30:auto _}}}}`:[]string{
			`{12:13:def .test.B} test-B {4:21 {=compound {12:16:word test} {=flag {12:21:word B}}}}`,
			`{13:13:def .test.b} test-b {4:21 {=compound {13:16:word test} {=flag {13:21:word b}}}}`,
		},
		`21 .test.s {=compound {4:43:punct .} {4:44:word test} {4:48:punct .} {4:49:word s}}`:`.test.s {=compound {4:43:punct .} {4:44:word test} {4:48:punct .} {4:49:word s}}`,
		`21 &(.test.s) {4:41:closure {=compound {4:43:punct .} {4:44:word test} {4:48:punct .} {4:49:word s}}}`:`{8:13:def .test.s} foo {4:41 {8:16:word foo}}`,

		`21 $_ {4:52:delegate {4:30:auto _}}`:[]string{`{4:30:auto _} B {4:52 {4:18 {6:24:word B}}}`,`{4:30:auto _} b {4:52 {4:18 {6:26:word b}}}`},
		`21 .test~&(.test.s).$_ {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {=compound {4:43:punct .} {4:44:word test} {4:48:punct .} {4:49:word s}}} {4:51:punct .} {4:52:delegate {4:30:auto _}}}`:[]string{
			`.test~foo.B {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41 {8:16:word foo}} {4:51:punct .} {4:52 {4:18 {6:24:word B}}}}`,
			`.test~foo.b {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41 {8:16:word foo}} {4:51:punct .} {4:52 {4:18 {6:26:word b}}}}`,
		},
		`21 &(.test~&(.test.s).$_) {4:33:closure {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {=compound {4:43:punct .} {4:44:word test} {4:48:punct .} {4:49:word s}}} {4:51:punct .} {4:52:delegate {4:30:auto _}}}}`:[]string{
			`{17:13:def .test~foo.B} test-foo-B {4:33 {=compound {17:16:word test} {=flag {=compound {17:21:word foo} {=flag {17:25:word B}}}}}}`,
			`{18:13:def .test~foo.b} test-foo-b {4:33 {=compound {18:16:word test} {=flag {=compound {18:21:word foo} {=flag {18:25:word b}}}}}}`,
		},
		`21 $(foreach $1 $2,&(.test.$_) &(.test~&(.test.s).$_)) {4:5:delegate {4:7:builtin foreach} {=list {4:15:delegate {4:16:auto 1}} {4:18:delegate {4:19:auto 2}}} {=list {4:21:closure {=compound {4:23:punct .} {4:24:word test} {4:28:punct .} {4:29:delegate {4:30:auto _}}}} {4:33:closure {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {=compound {4:43:punct .} {4:44:word test} {4:48:punct .} {4:49:word s}}} {4:51:punct .} {4:52:delegate {4:30:auto _}}}}}}`:[]string{
			`{4:7:builtin foreach} test-B test-foo-B test-b test-foo-b {=list {4:5 {4:21 {=compound {12:16:word test} {=flag {12:21:word B}}}}} {4:5 {4:33 {=compound {17:16:word test} {=flag {=compound {17:21:word foo} {=flag {17:25:word B}}}}}}} {4:5 {4:21 {=compound {13:16:word test} {=flag {13:21:word b}}}}} {4:5 {4:33 {=compound {18:16:word test} {=flag {=compound {18:21:word foo} {=flag {18:25:word b}}}}}}}}`,
		},
		`21 test-foo-B {=compound {17:16:word test} {=flag {=compound {17:21:word foo} {=flag {17:25:word B}}}}}`:`test-foo-B {=compound {17:16:word test} {=flag {=compound {17:21:word foo} {=flag {17:25:word B}}}}}`,
		`21 test-b {=compound {13:16:word test} {=flag {13:21:word b}}}`:`test-b {=compound {13:16:word test} {=flag {13:21:word b}}}`,
		`21 test-foo-b {=compound {18:16:word test} {=flag {=compound {18:21:word foo} {=flag {18:25:word b}}}}}`:`test-foo-b {=compound {18:16:word test} {=flag {=compound {18:21:word foo} {=flag {18:25:word b}}}}}`,
		`21 $(.test.x $1,B b) {6:11:delegate {3:9:def .test.x} {=list {6:21:delegate {4:16:auto 1}}} {=list {6:24:word B} {6:26:word b}}}`:`{3:9:def .test.x} foo test-foo test-B test-foo-B test-b test-foo-b {=list {6:11 {3:11 {8:16:word foo}}} {6:11 {3:22 {=compound {14:16:word test} {=flag {14:21:word foo}}}}} {6:11 {4:5 {4:21 {=compound {12:16:word test} {=flag {12:21:word B}}}}}} {6:11 {4:5 {4:33 {=compound {17:16:word test} {=flag {=compound {17:21:word foo} {=flag {17:25:word B}}}}}}}} {6:11 {4:5 {4:21 {=compound {13:16:word test} {=flag {13:21:word b}}}}}} {6:11 {4:5 {4:33 {=compound {18:16:word test} {=flag {=compound {18:21:word foo} {=flag {18:25:word b}}}}}}}}}`,

		`23 6:9:.test.1 $1 {6:21:delegate {4:16:auto 1}}`:`{4:16:auto 1} a {6:21 {1:9:word a}}`,
		`23 6:9:.test.1 .test.s {=compound {3:13:punct .} {3:14:word test} {3:18:punct .} {3:19:word s}}`:`.test.s {=compound {3:13:punct .} {3:14:word test} {3:18:punct .} {3:19:word s}}`,
		`23 6:9:.test.1 &(.test.s) {3:11:closure {=compound {3:13:punct .} {3:14:word test} {3:18:punct .} {3:19:word s}}}`:`{8:13:def .test.s} &(.test.s) {3:11:closure {8:13:def .test.s}}`,
		`23 6:9:.test.1 .test.s {=compound {3:38:punct .} {3:39:word test} {3:43:punct .} {3:44:word s}}`:`.test.s {=compound {3:38:punct .} {3:39:word test} {3:43:punct .} {3:44:word s}}`,
		`23 6:9:.test.1 &(.test.s) {3:36:closure {=compound {3:38:punct .} {3:39:word test} {3:43:punct .} {3:44:word s}}}`:`{8:13:def .test.s} foo {3:36 {8:16:word foo}}`,
		`23 6:9:.test.1 $(value .test~&(.test.s)) {3:22:delegate {3:24:builtin value} {=list {=compound {3:30:punct .} {3:31:word test} {3:35:punct ~} {3:36:closure {=compound {3:38:punct .} {3:39:word test} {3:43:punct .} {3:44:word s}}}}}}`:`{3:24:builtin value} test-foo {3:22 {=compound {14:16:word test} {=flag {14:21:word foo}}}}`,

		`23 6:9:.test.1 $1 {4:15:delegate {4:16:auto 1}}`:`{4:16:auto 1} a {4:15 {6:21 {1:9:word a}}}`,
		`23 6:9:.test.1 $2 {4:18:delegate {4:19:auto 2}}`:`{4:19:auto 2} B b {=list {4:18 {6:24:word B}} {4:18 {6:26:word b}}}`,
		`23 6:9:.test.1 $_ {4:29:delegate {4:30:auto _}}`:[]string{
			`{4:30:auto _} a {4:29 {4:15 {6:21 {1:9:word a}}}}`,
			`{4:30:auto _} B {4:29 {4:18 {6:24:word B}}}`,
			`{4:30:auto _} b {4:29 {4:18 {6:26:word b}}}`,
		},
		`23 6:9:.test.1 .test.$_ {=compound {4:23:punct .} {4:24:word test} {4:28:punct .} {4:29:delegate {4:30:auto _}}}`:[]string{
			`.test.a {=compound {4:23:punct .} {4:24:word test} {4:28:punct .} {4:29 {4:15 {6:21 {1:9:word a}}}}}`,
			`.test.B {=compound {4:23:punct .} {4:24:word test} {4:28:punct .} {4:29 {4:18 {6:24:word B}}}}`,
			`.test.b {=compound {4:23:punct .} {4:24:word test} {4:28:punct .} {4:29 {4:18 {6:26:word b}}}}`,
		},
		`23 6:9:.test.1 &(.test.$_) {4:21:closure {=compound {4:23:punct .} {4:24:word test} {4:28:punct .} {4:29:delegate {4:30:auto _}}}}`:[]string{
			`{11:13:def .test.a} &(.test.a) {4:21:closure {11:13:def .test.a}}`,
			`{12:13:def .test.B} &(.test.B) {4:21:closure {12:13:def .test.B}}`,
			`{13:13:def .test.b} &(.test.b) {4:21:closure {13:13:def .test.b}}`,
		},
		`23 6:9:.test.1 .test.s {=compound {4:43:punct .} {4:44:word test} {4:48:punct .} {4:49:word s}}`:`.test.s {=compound {4:43:punct .} {4:44:word test} {4:48:punct .} {4:49:word s}}`,
		`23 6:9:.test.1 &(.test.s) {4:41:closure {=compound {4:43:punct .} {4:44:word test} {4:48:punct .} {4:49:word s}}}`:`{8:13:def .test.s} &(.test.s) {4:41:closure {8:13:def .test.s}}`,

		`23 6:9:.test.1 $_ {4:52:delegate {4:30:auto _}}`:[]string{
			`{4:30:auto _} a {4:52 {4:15 {6:21 {1:9:word a}}}}`,
			`{4:30:auto _} B {4:52 {4:18 {6:24:word B}}}`,
			`{4:30:auto _} b {4:29 {4:18 {6:26:word b}}}`,
			`{4:30:auto _} b {4:52 {4:18 {6:26:word b}}}`,
		},
		`23 6:9:.test.1 .test~&(.test.s).$_ {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {=compound {4:43:punct .} {4:44:word test} {4:48:punct .} {4:49:word s}}} {4:51:punct .} {4:52:delegate {4:30:auto _}}}`:[]string{
			`.test~&(.test.s).a {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {8:13:def .test.s}} {4:51:punct .} {4:52 {4:15 {6:21 {1:9:word a}}}}}`,
			`.test~&(.test.s).B {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {8:13:def .test.s}} {4:51:punct .} {4:52 {4:18 {6:24:word B}}}}`,
			`.test~&(.test.s).b {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {8:13:def .test.s}} {4:51:punct .} {4:52 {4:18 {6:26:word b}}}}`,
		},
		`23 6:9:.test.1 &(.test~&(.test.s).$_) {4:33:closure {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {=compound {4:43:punct .} {4:44:word test} {4:48:punct .} {4:49:word s}}} {4:51:punct .} {4:52:delegate {4:30:auto _}}}}`:[]string{
			`{=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {8:13:def .test.s}} {4:51:punct .} {4:52 {4:15 {6:21 {1:9:word a}}}}} &(.test~&(.test.s).a) {4:33:closure {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {8:13:def .test.s}} {4:51:punct .} {4:52 {4:15 {6:21 {1:9:word a}}}}}}`,
			`{=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {8:13:def .test.s}} {4:51:punct .} {4:52 {4:18 {6:24:word B}}}} &(.test~&(.test.s).B) {4:33:closure {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {8:13:def .test.s}} {4:51:punct .} {4:52 {4:18 {6:24:word B}}}}}`,
			`{=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {8:13:def .test.s}} {4:51:punct .} {4:52 {4:18 {6:26:word b}}}} &(.test~&(.test.s).b) {4:33:closure {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {8:13:def .test.s}} {4:51:punct .} {4:52 {4:18 {6:26:word b}}}}}`,
		},
		`23 6:9:.test.1 $(foreach $1 $2,&(.test.$_) &(.test~&(.test.s).$_)) {4:5:delegate {4:7:builtin foreach} {=list {4:15:delegate {4:16:auto 1}} {4:18:delegate {4:19:auto 2}}} {=list {4:21:closure {=compound {4:23:punct .} {4:24:word test} {4:28:punct .} {4:29:delegate {4:30:auto _}}}} {4:33:closure {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {=compound {4:43:punct .} {4:44:word test} {4:48:punct .} {4:49:word s}}} {4:51:punct .} {4:52:delegate {4:30:auto _}}}}}}`:[]string{
			`{4:7:builtin foreach} &(.test.a) &(.test~&(.test.s).a) &(.test.B) &(.test~&(.test.s).B) &(.test.b) &(.test~&(.test.s).b) {=list {4:5 {4:21:closure {11:13:def .test.a}}} {4:5 {4:33:closure {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {8:13:def .test.s}} {4:51:punct .} {4:52 {4:15 {6:21 {1:9:word a}}}}}}} {4:5 {4:21:closure {12:13:def .test.B}}} {4:5 {4:33:closure {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {8:13:def .test.s}} {4:51:punct .} {4:52 {4:18 {6:24:word B}}}}}} {4:5 {4:21:closure {13:13:def .test.b}}} {4:5 {4:33:closure {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {8:13:def .test.s}} {4:51:punct .} {4:52 {4:18 {6:26:word b}}}}}}}`,
		},
		`23 6:9:.test.1 $(.test.x $1,B b) {6:11:delegate {3:9:def .test.x} {=list {6:21:delegate {4:16:auto 1}}} {=list {6:24:word B} {6:26:word b}}}`:[]string{
			`{3:9:def .test.x} &(.test.s) test-foo &(.test.a) &(.test~&(.test.s).a) &(.test.B) &(.test~&(.test.s).B) &(.test.b) &(.test~&(.test.s).b) {=list {6:11 {3:11:closure {8:13:def .test.s}}} {6:11 {3:22 {=compound {14:16:word test} {=flag {14:21:word foo}}}}} {6:11 {4:5 {4:21:closure {11:13:def .test.a}}}} {6:11 {4:5 {4:33:closure {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {8:13:def .test.s}} {4:51:punct .} {4:52 {4:15 {6:21 {1:9:word a}}}}}}}} {6:11 {4:5 {4:21:closure {12:13:def .test.B}}}} {6:11 {4:5 {4:33:closure {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {8:13:def .test.s}} {4:51:punct .} {4:52 {4:18 {6:24:word B}}}}}}} {6:11 {4:5 {4:21:closure {13:13:def .test.b}}}} {6:11 {4:5 {4:33:closure {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {8:13:def .test.s}} {4:51:punct .} {4:52 {4:18 {6:26:word b}}}}}}}}`,
		},

		`27 &(.test.s) {3:11:closure {8:13:def .test.s}}`:`{8:13:def .test.s} foo {3:11 {8:16:word foo}}`,
		`27 test-a {=compound {11:16:word test} {=flag {11:21:word a}}}`:`test-a {=compound {11:16:word test} {=flag {11:21:word a}}}`,
		`27 &(.test.a) {4:21:closure {11:13:def .test.a}}`:`{11:13:def .test.a} test-a {4:21 {=compound {11:16:word test} {=flag {11:21:word a}}}}`,

		`27 &(.test.s) {4:41:closure {8:13:def .test.s}}`:`{8:13:def .test.s} foo {4:41 {8:16:word foo}}`,
		`27 .test~&(.test.s).a {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {8:13:def .test.s}} {4:51:punct .} {4:52 {4:15 {6:21 {1:9:word a}}}}}`:`.test~foo.a {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41 {8:16:word foo}} {4:51:punct .} {4:52 {4:15 {6:21 {1:9:word a}}}}}`,
		`27 test-foo-a {=compound {16:16:word test} {=flag {=compound {16:21:word foo} {=flag {16:25:word a}}}}}`:`test-foo-a {=compound {16:16:word test} {=flag {=compound {16:21:word foo} {=flag {16:25:word a}}}}}`,
		`27 &(.test~&(.test.s).a) {4:33:closure {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {8:13:def .test.s}} {4:51:punct .} {4:52 {4:15 {6:21 {1:9:word a}}}}}}`:`{16:13:def .test~foo.a} test-foo-a {4:33 {=compound {16:16:word test} {=flag {=compound {16:21:word foo} {=flag {16:25:word a}}}}}}`,

		`27 test-B {=compound {12:16:word test} {=flag {12:21:word B}}}`:`test-B {=compound {12:16:word test} {=flag {12:21:word B}}}`,
		`27 &(.test.B) {4:21:closure {12:13:def .test.B}}`:`{12:13:def .test.B} test-B {4:21 {=compound {12:16:word test} {=flag {12:21:word B}}}}`,
		`27 .test~&(.test.s).B {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {8:13:def .test.s}} {4:51:punct .} {4:52 {4:18 {6:24:word B}}}}`:`.test~foo.B {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41 {8:16:word foo}} {4:51:punct .} {4:52 {4:18 {6:24:word B}}}}`,
		`27 test-foo-B {=compound {17:16:word test} {=flag {=compound {17:21:word foo} {=flag {17:25:word B}}}}}`:`test-foo-B {=compound {17:16:word test} {=flag {=compound {17:21:word foo} {=flag {17:25:word B}}}}}`,
		`27 &(.test~&(.test.s).B) {4:33:closure {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {8:13:def .test.s}} {4:51:punct .} {4:52 {4:18 {6:24:word B}}}}}`:`{17:13:def .test~foo.B} test-foo-B {4:33 {=compound {17:16:word test} {=flag {=compound {17:21:word foo} {=flag {17:25:word B}}}}}}`,

		`27 test-b {=compound {13:16:word test} {=flag {13:21:word b}}}`:`test-b {=compound {13:16:word test} {=flag {13:21:word b}}}`,
		`27 &(.test.b) {4:21:closure {13:13:def .test.b}}`:`{13:13:def .test.b} test-b {4:21 {=compound {13:16:word test} {=flag {13:21:word b}}}}`,
		`27 .test~&(.test.s).b {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {8:13:def .test.s}} {4:51:punct .} {4:52 {4:18 {6:26:word b}}}}`:`.test~foo.b {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41 {8:16:word foo}} {4:51:punct .} {4:52 {4:18 {6:26:word b}}}}`,
		`27 test-foo-b {=compound {18:16:word test} {=flag {=compound {18:21:word foo} {=flag {18:25:word b}}}}}`:`test-foo-b {=compound {18:16:word test} {=flag {=compound {18:21:word foo} {=flag {18:25:word b}}}}}`,
		`27 &(.test~&(.test.s).b) {4:33:closure {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {8:13:def .test.s}} {4:51:punct .} {4:52 {4:18 {6:26:word b}}}}}`:`{18:13:def .test~foo.b} test-foo-b {4:33 {=compound {18:16:word test} {=flag {=compound {18:21:word foo} {=flag {18:25:word b}}}}}}`,
	},
}

var checkstrs__foreach1 = map[string]map[string]any{
	"check-builtins-foreach1_test.go": map[string]any{
		`21 0:0: &(.test.$_) {4:23:closure {=compound {4:31:punct .} {4:32:word test} {4:36:punct .} {4:37:delegate {4:38:auto _}}}}`:`{4:23:null} `,
		`21 0:0: &(.test~&(.test.s).$_) {4:43:closure {=compound {4:51:punct .} {4:52:word test} {4:56:punct ~} {4:57:closure {=compound {4:59:punct .} {4:60:word test} {4:64:punct .} {4:65:word s}}} {4:67:punct .} {4:68:delegate {4:38:auto _}}}}`:`{4:43:null} `,
		`21 0:0: $(.test.x $1,B b) {6:11:delegate {3:9:def .test.x} {=list {6:21:delegate {4:16:auto 1}}} {=list {6:24:word B} {6:26:word b}}}`:`{=list {6:11 {3:11 {8:16:word foo}}} {6:11 {3:22 {=compound {14:16:word test} {=flag {14:21:word foo}}}}} {6:11 {4:5 {4:21 {=compound {12:16:word test} {=flag {12:21:word B}}}}}} {6:11 {4:5 {4:33 {=compound {17:16:word test} {=flag {=compound {17:21:word foo} {=flag {17:25:word B}}}}}}}} {6:11 {4:5 {4:21 {=compound {13:16:word test} {=flag {13:21:word b}}}}}} {6:11 {4:5 {4:33 {=compound {18:16:word test} {=flag {=compound {18:21:word foo} {=flag {18:25:word b}}}}}}}}} foo test-foo test-B test-foo-B test-b test-foo-b`,
		`27 0:0: &(.test.s) {3:11:closure {8:13:def .test.s}}`:`{3:11 {8:16:word foo}} foo`,
		`27 0:0: &(.test.a) {4:21:closure {11:13:def .test.a}}`:`{4:21 {=compound {11:16:word test} {=flag {11:21:word a}}}} test-a`,
		`27 0:0: &(.test~&(.test.s).a) {4:33:closure {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {8:13:def .test.s}} {4:51:punct .} {4:52 {4:15 {6:21 {1:9:word a}}}}}}`:`{4:33 {=compound {16:16:word test} {=flag {=compound {16:21:word foo} {=flag {16:25:word a}}}}}} test-foo-a`,
		`27 0:0: &(.test.B) {4:21:closure {12:13:def .test.B}}`:`{4:21 {=compound {12:16:word test} {=flag {12:21:word B}}}} test-B`,
		`27 0:0: &(.test~&(.test.s).B) {4:33:closure {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {8:13:def .test.s}} {4:51:punct .} {4:52 {4:18 {6:24:word B}}}}}`:`{4:33 {=compound {17:16:word test} {=flag {=compound {17:21:word foo} {=flag {17:25:word B}}}}}} test-foo-B`,
		`27 0:0: &(.test.b) {4:21:closure {13:13:def .test.b}}`:`{4:21 {=compound {13:16:word test} {=flag {13:21:word b}}}} test-b`,
		`27 0:0: &(.test~&(.test.s).b) {4:33:closure {=compound {4:35:punct .} {4:36:word test} {4:40:punct ~} {4:41:closure {8:13:def .test.s}} {4:51:punct .} {4:52 {4:18 {6:26:word b}}}}}`:`{4:33 {=compound {18:16:word test} {=flag {=compound {18:21:word foo} {=flag {18:25:word b}}}}}} test-foo-b`,
	},
}
