//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints__foreach3 = map[string]map[string]any{
	"loader.go": map[string]any{
		`.test.a {=compound {3:1:punct .} {3:2:word test} {3:6:punct .} {3:7:word a}}`:`.test.a {=compound {3:1:punct .} {3:2:word test} {3:6:punct .} {3:7:word a}}`,
		`.test.x {=compound {4:1:punct .} {4:2:word test} {4:6:punct .} {4:7:word x}}`:`.test.x {=compound {4:1:punct .} {4:2:word test} {4:6:punct .} {4:7:word x}}`,
		`.test.y {=compound {5:1:punct .} {5:2:word test} {5:6:punct .} {5:7:word y}}`:`.test.y {=compound {5:1:punct .} {5:2:word test} {5:6:punct .} {5:7:word y}}`,
		`.test.z {=compound {6:1:punct .} {6:2:word test} {6:6:punct .} {6:7:word z}}`:`.test.z {=compound {6:1:punct .} {6:2:word test} {6:6:punct .} {6:7:word z}}`,
		`.test.if.x {=compound {7:1:punct .} {7:2:word test} {7:6:punct .} {7:7:word if} {7:9:punct .} {7:10:word x}}`:`.test.if.x {=compound {7:1:punct .} {7:2:word test} {7:6:punct .} {7:7:word if} {7:9:punct .} {7:10:word x}}`,
		`.test.if.y {=compound {8:1:punct .} {8:2:word test} {8:6:punct .} {8:7:word if} {8:9:punct .} {8:10:word y}}`:`.test.if.y {=compound {8:1:punct .} {8:2:word test} {8:6:punct .} {8:7:word if} {8:9:punct .} {8:10:word y}}`,
	},
	"check-builtins-foreach3_test.go": map[string]any{
		`18 3:9:.test.a $1 {3:21:delegate {3:22:auto 1}}`:`a {3:21 {1:9:word a}}`,
		`18 3:9:.test.a $2 {3:24:delegate {3:25:auto 2}}`:`b {3:24 {1:9:word b}}`,
		`18 3:9:.test.a $_ {3:27:delegate {3:28:auto _}}`:[]string{`a {3:27 {3:21 {1:9:word a}}}`,`b {3:27 {3:24 {1:9:word b}}}`},
		`18 3:9:.test.a $3 {3:29:delegate {3:30:auto 3}}`:`cc {3:29 {1:9:word cc}}`,
		`18 3:9:.test.a $_$3 {=compound {3:27:delegate {3:28:auto _}} {3:29:delegate {3:30:auto 3}}}`:[]string{
			`acc {=compound {3:27 {3:21 {1:9:word a}}} {3:29 {1:9:word cc}}}`,
			`bcc {=compound {3:27 {3:24 {1:9:word b}}} {3:29 {1:9:word cc}}}`,
		},
		`18 3:9:.test.a $(foreach $1 $2,$_$3) {3:11:delegate {3:13:builtin foreach} {=list {3:21:delegate {3:22:auto 1}} {3:24:delegate {3:25:auto 2}}} {=list {=compound {3:27:delegate {3:28:auto _}} {3:29:delegate {3:30:auto 3}}}}}`:`acc bcc {=list {3:11 {=compound {3:27 {3:21 {1:9:word a}}} {3:29 {1:9:word cc}}}} {3:11 {=compound {3:27 {3:24 {1:9:word b}}} {3:29 {1:9:word cc}}}}}`,

		`25 3:9:.test.a $1 {3:21:delegate {3:22:auto 1}}`:` {3:21 {1:9:none}}`,
		`25 3:9:.test.a $2 {3:24:delegate {3:25:auto 2}}`:`1 2 3 {=list {3:24 {1:9:word 1}} {3:24 {1:9:word 2}} {3:24 {1:9:word 3}}}`,
		`25 3:9:.test.a $_ {3:27:delegate {3:28:auto _}}`:[]string{`1 {3:27 {3:24 {1:9:word 1}}}`,`2 {3:27 {3:24 {1:9:word 2}}}`,`3 {3:27 {3:24 {1:9:word 3}}}`},
		`25 3:9:.test.a $3 {3:29:delegate {3:30:auto 3}}`:`x {3:29 {1:9:word x}}`,
		`25 3:9:.test.a $_$3 {=compound {3:27:delegate {3:28:auto _}} {3:29:delegate {3:30:auto 3}}}`:[]string{`1x {=compound {3:27 {3:24 {1:9:word 1}}} {3:29 {1:9:word x}}}`,`2x {=compound {3:27 {3:24 {1:9:word 2}}} {3:29 {1:9:word x}}}`,`3x {=compound {3:27 {3:24 {1:9:word 3}}} {3:29 {1:9:word x}}}`},
		`25 3:9:.test.a $(foreach $1 $2,$_$3) {3:11:delegate {3:13:builtin foreach} {=list {3:21:delegate {3:22:auto 1}} {3:24:delegate {3:25:auto 2}}} {=list {=compound {3:27:delegate {3:28:auto _}} {3:29:delegate {3:30:auto 3}}}}}`:`1x 2x 3x {=list {3:11 {=compound {3:27 {3:24 {1:9:word 1}}} {3:29 {1:9:word x}}}} {3:11 {=compound {3:27 {3:24 {1:9:word 2}}} {3:29 {1:9:word x}}}} {3:11 {=compound {3:27 {3:24 {1:9:word 3}}} {3:29 {1:9:word x}}}}}`,

		`32 3:9:.test.a $1 {3:21:delegate {3:22:auto 1}}`:`a {3:21 {1:9:word a}}`,
		`32 3:9:.test.a $2 {3:24:delegate {3:25:auto 2}}`:`b {3:24 {1:9:word b}}`,
		`32 3:9:.test.a $_ {3:27:delegate {3:28:auto _}}`:[]string{`a {3:27 {3:21 {1:9:word a}}}`,`b {3:27 {3:24 {1:9:word b}}}`},
		`32 3:9:.test.a $3 {3:29:delegate {3:30:auto 3}}`:`{} {3:29 {3:30:null}}`,
		`32 3:9:.test.a $_$3 {=compound {3:27:delegate {3:28:auto _}} {3:29:delegate {3:30:auto 3}}}`:[]string{`a{} {=compound {3:27 {3:21 {1:9:word a}}} {3:29 {3:30:null}}}`,`b{} {=compound {3:27 {3:24 {1:9:word b}}} {3:29 {3:30:null}}}`},
		`32 3:9:.test.a $(foreach $1 $2,$_$3) {3:11:delegate {3:13:builtin foreach} {=list {3:21:delegate {3:22:auto 1}} {3:24:delegate {3:25:auto 2}}} {=list {=compound {3:27:delegate {3:28:auto _}} {3:29:delegate {3:30:auto 3}}}}}`:`a{} b{} {=list {3:11 {=compound {3:27 {3:21 {1:9:word a}}} {3:29 {3:30:null}}}} {3:11 {=compound {3:27 {3:24 {1:9:word b}}} {3:29 {3:30:null}}}}}`,

		`39 3:9:.test.a $1 {3:21:delegate {3:22:auto 1}}`:`a {3:21 {1:9:word a}}`,
		`39 3:9:.test.a $2 {3:24:delegate {3:25:auto 2}}`:`b {3:24 {1:9:word b}}`,
		`39 3:9:.test.a $_ {3:27:delegate {3:28:auto _}}`:[]string{`a {3:27 {3:21 {1:9:word a}}}`,`b {3:27 {3:24 {1:9:word b}}}`},
		`39 3:9:.test.a $3 {3:29:delegate {3:30:auto 3}}`:`x {3:29 {1:9:word x}}`,
		`39 3:9:.test.a $_$3 {=compound {3:27:delegate {3:28:auto _}} {3:29:delegate {3:30:auto 3}}}`:[]string{
			`ax {=compound {3:27 {3:21 {1:9:word a}}} {3:29 {1:9:word x}}}`,
			`bx {=compound {3:27 {3:24 {1:9:word b}}} {3:29 {1:9:word x}}}`,
		},
		`39 3:9:.test.a $(foreach $1 $2,$_$3) {3:11:delegate {3:13:builtin foreach} {=list {3:21:delegate {3:22:auto 1}} {3:24:delegate {3:25:auto 2}}} {=list {=compound {3:27:delegate {3:28:auto _}} {3:29:delegate {3:30:auto 3}}}}}`:`ax bx {=list {3:11 {=compound {3:27 {3:21 {1:9:word a}}} {3:29 {1:9:word x}}}} {3:11 {=compound {3:27 {3:24 {1:9:word b}}} {3:29 {1:9:word x}}}}}`,

		`55 4:9:.test.x $1 {4:21:delegate {3:22:auto 1}}`:`a {4:21 {1:9:word a}}`,
		`55 4:9:.test.x $2 {4:24:delegate {3:25:auto 2}}`:`{} {4:24 {3:25:null}}`,
		`55 4:9:.test.x $_ {4:52:delegate {3:28:auto _}}`:[]string{`a {4:52 {4:21 {1:9:word a}}}`,``},
		`55 4:9:.test.x .test.$_ {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52:delegate {3:28:auto _}}}`:[]string{
			`.test.a {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52 {4:21 {1:9:word a}}}}`,
		},
		`55 4:9:.test.x &(.test.$_) {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52:delegate {3:28:auto _}}}}`:[]string{
			`&(.test.a) {4:44:closure {3:9:def .test.a}}`,
		},
		`55 4:9:.test.x $(addprefix std=,&(.test.$_)) {4:27:delegate {4:29:builtin addprefix} {=list {=pair {4:39:word std} {4:43}}} {=list {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52:delegate {3:28:auto _}}}}}}`:`std={&(.test.a)} {4:27 {=pair {4:39:word std} {4:44:disjunction {4:44:closure {3:9:def .test.a}}}}}`,
		`55 4:9:.test.x $(foreach $1 $2,$(addprefix std=,&(.test.$_))) {4:11:delegate {4:13:builtin foreach} {=list {4:21:delegate {3:22:auto 1}} {4:24:delegate {3:25:auto 2}}} {=list {4:27:delegate {4:29:builtin addprefix} {=list {=pair {4:39:word std} {4:43}}} {=list {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52:delegate {3:28:auto _}}}}}}}}`:`std={&(.test.a)} {4:11 {4:27 {=pair {4:39:word std} {4:44:disjunction {4:44:closure {3:9:def .test.a}}}}}}`,

		`59 $1 {3:21:delegate {3:22:auto 1}}`:`{} {3:21 {3:22:null}}`,
		`59 $2 {3:24:delegate {3:25:auto 2}}`:`{} {3:24 {3:25:null}}`,
		`59 $(foreach $1 $2,$_$3) {3:11:delegate {3:13:builtin foreach} {=list {3:21:delegate {3:22:auto 1}} {3:24:delegate {3:25:auto 2}}} {=list {=compound {3:27:delegate {3:28:auto _}} {3:29:delegate {3:30:auto 3}}}}}`:`{} {3:11 {3:13:null}}`,
		`59 &(.test.a) {4:44:closure {3:9:def .test.a}}`:`{} {4:44 {3:11 {3:13:null}}}`,

		`61 4:9:.test.x $1 {4:21:delegate {3:22:auto 1}}`:`a {4:21 {1:9:word a}}`,
		`61 4:9:.test.x $2 {4:24:delegate {3:25:auto 2}}`:`{} {4:24 {1:9:null}}`,
		`61 4:9:.test.x $_ {4:52:delegate {3:28:auto _}}`:[]string{`a {4:52 {4:21 {1:9:word a}}}`,},
		`61 4:9:.test.x .test.$_ {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52:delegate {3:28:auto _}}}`:[]string{`.test.a {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52 {4:21 {1:9:word a}}}}`,},
		`61 4:9:.test.x &(.test.$_) {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52:delegate {3:28:auto _}}}}`:`&(.test.a) {4:44:closure {3:9:def .test.a}}`,
		`61 4:9:.test.x $(addprefix std=,&(.test.$_)) {4:27:delegate {4:29:builtin addprefix} {=list {=pair {4:39:word std} {4:43}}} {=list {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52:delegate {3:28:auto _}}}}}}`:`std={&(.test.a)} {4:27 {=pair {4:39:word std} {4:44:disjunction {4:44:closure {3:9:def .test.a}}}}}`,
		`61 4:9:.test.x $(foreach $1 $2,$(addprefix std=,&(.test.$_))) {4:11:delegate {4:13:builtin foreach} {=list {4:21:delegate {3:22:auto 1}} {4:24:delegate {3:25:auto 2}}} {=list {4:27:delegate {4:29:builtin addprefix} {=list {=pair {4:39:word std} {4:43}}} {=list {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52:delegate {3:28:auto _}}}}}}}}`:`std={&(.test.a)} {4:11 {4:27 {=pair {4:39:word std} {4:44:disjunction {4:44:closure {3:9:def .test.a}}}}}}`,

		`65 $1 {3:21:delegate {3:22:auto 1}}`:`{} {3:21 {3:22:null}}`,
		`65 $2 {3:24:delegate {3:25:auto 2}}`:`{} {3:24 {3:25:null}}`,
		`65 $(foreach $1 $2,$_$3) {3:11:delegate {3:13:builtin foreach} {=list {3:21:delegate {3:22:auto 1}} {3:24:delegate {3:25:auto 2}}} {=list {=compound {3:27:delegate {3:28:auto _}} {3:29:delegate {3:30:auto 3}}}}}`:`{} {3:11 {3:13:null}}`,
		`65 &(.test.a) {4:44:closure {3:9:def .test.a}}`:`{} {4:44 {3:11 {3:13:null}}}`,

		`67 4:9:.test.x $1 {4:21:delegate {3:22:auto 1}}`:`{} {4:21 {1:9:null}}`,
		`67 4:9:.test.x $2 {4:24:delegate {3:25:auto 2}}`:`b {4:24 {1:9:word b}}`,
		`67 4:9:.test.x $_ {4:52:delegate {3:28:auto _}}`:`b {4:52 {4:24 {1:9:word b}}}`,
		`67 4:9:.test.x .test.$_ {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52:delegate {3:28:auto _}}}`:`.test.b {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52 {4:24 {1:9:word b}}}}`,
		`67 4:9:.test.x &(.test.$_) {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52:delegate {3:28:auto _}}}}`:`&(.test.b) {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52 {4:24 {1:9:word b}}}}}`,
		`67 4:9:.test.x $(addprefix std=,&(.test.$_)) {4:27:delegate {4:29:builtin addprefix} {=list {=pair {4:39:word std} {4:43}}} {=list {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52:delegate {3:28:auto _}}}}}}`:`std={&(.test.b)} {4:27 {=pair {4:39:word std} {4:44:disjunction {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52 {4:24 {1:9:word b}}}}}}}}`,
		`67 4:9:.test.x $(foreach $1 $2,$(addprefix std=,&(.test.$_))) {4:11:delegate {4:13:builtin foreach} {=list {4:21:delegate {3:22:auto 1}} {4:24:delegate {3:25:auto 2}}} {=list {4:27:delegate {4:29:builtin addprefix} {=list {=pair {4:39:word std} {4:43}}} {=list {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52:delegate {3:28:auto _}}}}}}}}`:`std={&(.test.b)} {4:11 {4:27 {=pair {4:39:word std} {4:44:disjunction {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52 {4:24 {1:9:word b}}}}}}}}}`,

		`71 .test.b {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52 {4:24 {1:9:word b}}}}`:`.test.b {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52 {4:24 {1:9:word b}}}}`,
		`71 &(.test.b) {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52 {4:24 {1:9:word b}}}}}`:`{} {4:44:null}`,

		`73 4:9:.test.x $1 {4:21:delegate {3:22:auto 1}}`:`a {4:21 {1:9:word a}}`,
		`73 4:9:.test.x $2 {4:24:delegate {3:25:auto 2}}`:`b {4:24 {1:9:word b}}`,
		`73 4:9:.test.x $_ {4:52:delegate {3:28:auto _}}`:[]string{`a {4:52 {4:21 {1:9:word a}}}`,`b {4:52 {4:24 {1:9:word b}}}`},
		`73 4:9:.test.x .test.$_ {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52:delegate {3:28:auto _}}}`:[]string{
			`.test.a {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52 {4:21 {1:9:word a}}}}`,
			`.test.b {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52 {4:24 {1:9:word b}}}}`,
		},
		`73 4:9:.test.x &(.test.$_) {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52:delegate {3:28:auto _}}}}`:[]string{
			`&(.test.a) {4:44:closure {3:9:def .test.a}}`,
			`&(.test.b) {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52 {4:24 {1:9:word b}}}}}`,
		},
		`73 4:9:.test.x $(addprefix std=,&(.test.$_)) {4:27:delegate {4:29:builtin addprefix} {=list {=pair {4:39:word std} {4:43}}} {=list {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52:delegate {3:28:auto _}}}}}}`:[]string{
			`std={&(.test.a)} {4:27 {=pair {4:39:word std} {4:44:disjunction {4:44:closure {3:9:def .test.a}}}}}`,
			`std={&(.test.b)} {4:27 {=pair {4:39:word std} {4:44:disjunction {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52 {4:24 {1:9:word b}}}}}}}}`,
		},
		`73 4:9:.test.x $(foreach $1 $2,$(addprefix std=,&(.test.$_))) {4:11:delegate {4:13:builtin foreach} {=list {4:21:delegate {3:22:auto 1}} {4:24:delegate {3:25:auto 2}}} {=list {4:27:delegate {4:29:builtin addprefix} {=list {=pair {4:39:word std} {4:43}}} {=list {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52:delegate {3:28:auto _}}}}}}}}`:`std={&(.test.a)} std={&(.test.b)} {=list {4:11 {4:27 {=pair {4:39:word std} {4:44:disjunction {4:44:closure {3:9:def .test.a}}}}}} {4:11 {4:27 {=pair {4:39:word std} {4:44:disjunction {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52 {4:24 {1:9:word b}}}}}}}}}}`,

		`77 $1 {3:21:delegate {3:22:auto 1}}`:`{} {3:21 {3:22:null}}`,
		`77 $2 {3:24:delegate {3:25:auto 2}}`:`{} {3:24 {3:25:null}}`,
		`77 $(foreach $1 $2,$_$3) {3:11:delegate {3:13:builtin foreach} {=list {3:21:delegate {3:22:auto 1}} {3:24:delegate {3:25:auto 2}}} {=list {=compound {3:27:delegate {3:28:auto _}} {3:29:delegate {3:30:auto 3}}}}}`:`{} {3:11 {3:13:null}}`,
		`77 &(.test.a) {4:44:closure {3:9:def .test.a}}`:`{} {4:44 {3:11 {3:13:null}}}`,
		`77 .test.b {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52 {4:24 {1:9:word b}}}}`:`.test.b {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52 {4:24 {1:9:word b}}}}`,
		`77 &(.test.b) {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52 {4:24 {1:9:word b}}}}}`:`{} {4:44:null}`,

		`79 4:9:.test.x $1 {4:21:delegate {3:22:auto 1}}`:`{} {4:21 {1:9:null}}`,
		`79 4:9:.test.x $2 {4:24:delegate {3:25:auto 2}}`:`if.x if.y if.z {=list {4:24 {1:9:word if.x}} {4:24 {1:9:word if.y}} {4:24 {1:9:word if.z}}}`,
		`79 4:9:.test.x $_ {4:52:delegate {3:28:auto _}}`:[]string{
			`if.x {4:52 {4:24 {1:9:word if.x}}}`,
			`if.y {4:52 {4:24 {1:9:word if.y}}}`,
			`if.z {4:52 {4:24 {1:9:word if.z}}}`,
		},
		`79 4:9:.test.x .test.$_ {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52:delegate {3:28:auto _}}}`:[]string{
			`.test.if.x {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52 {4:24 {1:9:word if.x}}}}`,
			`.test.if.y {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52 {4:24 {1:9:word if.y}}}}`,
			`.test.if.z {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52 {4:24 {1:9:word if.z}}}}`,
		},
		`79 4:9:.test.x &(.test.$_) {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52:delegate {3:28:auto _}}}}`:[]string{
			`&(.test.if.x) {4:44:closure {7:12:def .test.if.x}}`,
			`&(.test.if.y) {4:44:closure {8:12:def .test.if.y}}`,
			`&(.test.if.z) {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52 {4:24 {1:9:word if.z}}}}}`,
		},
		`79 4:9:.test.x $(addprefix std=,&(.test.$_)) {4:27:delegate {4:29:builtin addprefix} {=list {=pair {4:39:word std} {4:43}}} {=list {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52:delegate {3:28:auto _}}}}}}`:[]string{
			`std={&(.test.if.x)} {4:27 {=pair {4:39:word std} {4:44:disjunction {4:44:closure {7:12:def .test.if.x}}}}}`,
			`std={&(.test.if.y)} {4:27 {=pair {4:39:word std} {4:44:disjunction {4:44:closure {8:12:def .test.if.y}}}}}`,
			`std={&(.test.if.z)} {4:27 {=pair {4:39:word std} {4:44:disjunction {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52 {4:24 {1:9:word if.z}}}}}}}}`,
		},
		`79 4:9:.test.x $(foreach $1 $2,$(addprefix std=,&(.test.$_))) {4:11:delegate {4:13:builtin foreach} {=list {4:21:delegate {3:22:auto 1}} {4:24:delegate {3:25:auto 2}}} {=list {4:27:delegate {4:29:builtin addprefix} {=list {=pair {4:39:word std} {4:43}}} {=list {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52:delegate {3:28:auto _}}}}}}}}`:`std={&(.test.if.x)} std={&(.test.if.y)} std={&(.test.if.z)} {=list {4:11 {4:27 {=pair {4:39:word std} {4:44:disjunction {4:44:closure {7:12:def .test.if.x}}}}}} {4:11 {4:27 {=pair {4:39:word std} {4:44:disjunction {4:44:closure {8:12:def .test.if.y}}}}}} {4:11 {4:27 {=pair {4:39:word std} {4:44:disjunction {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52 {4:24 {1:9:word if.z}}}}}}}}}}`,

		`83 &(.test.if.x) {4:44:closure {7:12:def .test.if.x}}`:`xxx {4:44 {7:15:word xxx}}`,
		`83 &(.test.if.y) {4:44:closure {8:12:def .test.if.y}}`:`yyy {4:44 {8:15:word yyy}}`,
		`83 .test.if.z {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52 {4:24 {1:9:word if.z}}}}`:`.test.if.z {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52 {4:24 {1:9:word if.z}}}}`,
		`83 &(.test.if.z) {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52 {4:24 {1:9:word if.z}}}}}`:`{} {4:44:null}`,

		`85 4:9:.test.x $1 {4:21:delegate {3:22:auto 1}}`:`{} {4:21 {1:9:null}}`,
		`85 4:9:.test.x $2 {4:24:delegate {3:25:auto 2}}`:`if.x if.y if.z {=list {4:24 {1:9:word if.x}} {4:24 {1:9:word if.y}} {4:24 {1:9:word if.z}}}`,
		`85 4:9:.test.x $_ {4:52:delegate {3:28:auto _}}`:[]string{
			`if.x {4:52 {4:24 {1:9:word if.x}}}`,
			`if.y {4:52 {4:24 {1:9:word if.y}}}`,
			`if.z {4:52 {4:24 {1:9:word if.z}}}`,
		},
		`85 4:9:.test.x .test.$_ {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52:delegate {3:28:auto _}}}`:[]string{
			`.test.if.x {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52 {4:24 {1:9:word if.x}}}}`,
			`.test.if.y {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52 {4:24 {1:9:word if.y}}}}`,
			`.test.if.z {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52 {4:24 {1:9:word if.z}}}}`,
		},
		`85 4:9:.test.x &(.test.$_) {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52:delegate {3:28:auto _}}}}`:[]string{
			`xxx {4:44 {7:15:word xxx}}`,
			`yyy {4:44 {8:15:word yyy}}`,
			`{} {4:44:null}`,
		},
		`85 4:9:.test.x $(addprefix std=,&(.test.$_)) {4:27:delegate {4:29:builtin addprefix} {=list {=pair {4:39:word std} {4:43}}} {=list {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52:delegate {3:28:auto _}}}}}}`:[]string{
			`std=xxx {4:27 {=pair {4:39:word std} {7:15:word xxx}}}`,
			`std=yyy {4:27 {=pair {4:39:word std} {8:15:word yyy}}}`,
			`{} {4:27 {4:29:null}}`,
		},
		`85 4:9:.test.x $(foreach $1 $2,$(addprefix std=,&(.test.$_))) {4:11:delegate {4:13:builtin foreach} {=list {4:21:delegate {3:22:auto 1}} {4:24:delegate {3:25:auto 2}}} {=list {4:27:delegate {4:29:builtin addprefix} {=list {=pair {4:39:word std} {4:43}}} {=list {4:44:closure {=compound {4:46:punct .} {4:47:word test} {4:51:punct .} {4:52:delegate {3:28:auto _}}}}}}}}`:`std=xxx std=yyy {=list {4:11 {4:27 {=pair {4:39:word std} {7:15:word xxx}}}} {4:11 {4:27 {=pair {4:39:word std} {8:15:word yyy}}}}}`,

		`103 5:9:.test.y $1 {5:21:delegate {3:22:auto 1}}`:`if.x {5:21 {1:9:word if.x}}`,
		`103 5:9:.test.y $2 {5:24:delegate {3:25:auto 2}}`:`{} {5:24 {3:25:null}}`,
		`103 5:9:.test.y $_ {5:40:delegate {3:28:auto _}}`:`if.x {5:40 {5:21 {1:9:word if.x}}}`,
		`103 5:9:.test.y .test.$_ {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}`:`.test.if.x {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40 {5:21 {1:9:word if.x}}}}`,
		`103 5:9:.test.y &(.test.$_) {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}}`:`&(.test.if.x) {5:32:closure {7:12:def .test.if.x}}`,
		`103 5:9:.test.y &(.test.if.x) {5:32:closure {7:12:def .test.if.x}}`:`&(.test.if.x) {5:32:closure {7:12:def .test.if.x}}`,
		`103 5:9:.test.y $(if &(.test.$_),std=&(.test.$_)) {5:27:delegate {5:29:builtin if} {=list {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}}} {=list {=pair {5:44:word std} {5:48:closure {=compound {5:50:punct .} {5:51:word test} {5:55:punct .} {5:56:delegate {3:28:auto _}}}}}}}`:`{} {5:27 {5:29:null}}`,
		`103 5:9:.test.y $(foreach $1 $2,$(if &(.test.$_),std=&(.test.$_))) {5:11:delegate {5:13:builtin foreach} {=list {5:21:delegate {3:22:auto 1}} {5:24:delegate {3:25:auto 2}}} {=list {5:27:delegate {5:29:builtin if} {=list {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}}} {=list {=pair {5:44:word std} {5:48:closure {=compound {5:50:punct .} {5:51:word test} {5:55:punct .} {5:56:delegate {3:28:auto _}}}}}}}}}`:`{} {5:11 {5:13:null}}`,

		`110 5:9:.test.y $1 {5:21:delegate {3:22:auto 1}}`:`if.x {5:21 {1:9:word if.x}}`,
		`110 5:9:.test.y $2 {5:24:delegate {3:25:auto 2}}`:`{} {5:24 {1:9:null}}`,
		`110 5:9:.test.y $_ {5:40:delegate {3:28:auto _}}`:`if.x {5:40 {5:21 {1:9:word if.x}}}`,
		`110 5:9:.test.y .test.$_ {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}`:`.test.if.x {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40 {5:21 {1:9:word if.x}}}}`,
		`110 5:9:.test.y &(.test.$_) {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}}`:`&(.test.if.x) {5:32:closure {7:12:def .test.if.x}}`,
		`110 5:9:.test.y &(.test.if.x) {5:32:closure {7:12:def .test.if.x}}`:`&(.test.if.x) {5:32:closure {7:12:def .test.if.x}}`,
		`110 5:9:.test.y $(if &(.test.$_),std=&(.test.$_)) {5:27:delegate {5:29:builtin if} {=list {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}}} {=list {=pair {5:44:word std} {5:48:closure {=compound {5:50:punct .} {5:51:word test} {5:55:punct .} {5:56:delegate {3:28:auto _}}}}}}}`:`{} {5:27 {5:29:null}}`,
		`110 5:9:.test.y $(foreach $1 $2,$(if &(.test.$_),std=&(.test.$_))) {5:11:delegate {5:13:builtin foreach} {=list {5:21:delegate {3:22:auto 1}} {5:24:delegate {3:25:auto 2}}} {=list {5:27:delegate {5:29:builtin if} {=list {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}}} {=list {=pair {5:44:word std} {5:48:closure {=compound {5:50:punct .} {5:51:word test} {5:55:punct .} {5:56:delegate {3:28:auto _}}}}}}}}}`:`{} {5:11 {5:13:null}}`,

		`117 5:9:.test.y $1 {5:21:delegate {3:22:auto 1}}`:`{} {5:21 {1:9:null}}`,
		`117 5:9:.test.y $2 {5:24:delegate {3:25:auto 2}}`:`if.y {5:24 {1:9:word if.y}}`,
		`117 5:9:.test.y $_ {5:40:delegate {3:28:auto _}}`:`if.y {5:40 {5:24 {1:9:word if.y}}}`,
		`117 5:9:.test.y .test.$_ {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}`:`.test.if.y {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40 {5:24 {1:9:word if.y}}}}`,
		`117 5:9:.test.y &(.test.$_) {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}}`:`&(.test.if.y) {5:32:closure {8:12:def .test.if.y}}`,
		`117 5:9:.test.y &(.test.if.y) {5:32:closure {8:12:def .test.if.y}}`:`&(.test.if.y) {5:32:closure {8:12:def .test.if.y}}`,
		`117 5:9:.test.y $(if &(.test.$_),std=&(.test.$_)) {5:27:delegate {5:29:builtin if} {=list {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}}} {=list {=pair {5:44:word std} {5:48:closure {=compound {5:50:punct .} {5:51:word test} {5:55:punct .} {5:56:delegate {3:28:auto _}}}}}}}`:`{} {5:27 {5:29:null}}`,
		`117 5:9:.test.y $(foreach $1 $2,$(if &(.test.$_),std=&(.test.$_))) {5:11:delegate {5:13:builtin foreach} {=list {5:21:delegate {3:22:auto 1}} {5:24:delegate {3:25:auto 2}}} {=list {5:27:delegate {5:29:builtin if} {=list {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}}} {=list {=pair {5:44:word std} {5:48:closure {=compound {5:50:punct .} {5:51:word test} {5:55:punct .} {5:56:delegate {3:28:auto _}}}}}}}}}`:`{} {5:11 {5:13:null}}`,

		`124 5:9:.test.y $1 {5:21:delegate {3:22:auto 1}}`:`if.x {5:21 {1:9:word if.x}}`,
		`124 5:9:.test.y $2 {5:24:delegate {3:25:auto 2}}`:`if.y {5:24 {1:9:word if.y}}`,
		`124 5:9:.test.y $_ {5:40:delegate {3:28:auto _}}`:[]string{`if.x {5:40 {5:21 {1:9:word if.x}}}`,`if.y {5:40 {5:24 {1:9:word if.y}}}`},
		`124 5:9:.test.y .test.$_ {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}`:[]string{
			`.test.if.x {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40 {5:21 {1:9:word if.x}}}}`,
			`.test.if.y {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40 {5:24 {1:9:word if.y}}}}`,
		},
		`124 5:9:.test.y &(.test.$_) {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}}`:[]string{
			`&(.test.if.x) {5:32:closure {7:12:def .test.if.x}}`,
			`&(.test.if.y) {5:32:closure {8:12:def .test.if.y}}`,
		},
		`124 5:9:.test.y &(.test.if.x) {5:32:closure {7:12:def .test.if.x}}`:`&(.test.if.x) {5:32:closure {7:12:def .test.if.x}}`,
		`124 5:9:.test.y $(if &(.test.$_),std=&(.test.$_)) {5:27:delegate {5:29:builtin if} {=list {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}}} {=list {=pair {5:44:word std} {5:48:closure {=compound {5:50:punct .} {5:51:word test} {5:55:punct .} {5:56:delegate {3:28:auto _}}}}}}}`:`{} {5:27 {5:29:null}}`,
		`124 5:9:.test.y &(.test.if.y) {5:32:closure {8:12:def .test.if.y}}`:`&(.test.if.y) {5:32:closure {8:12:def .test.if.y}}`,
		`124 5:9:.test.y $(foreach $1 $2,$(if &(.test.$_),std=&(.test.$_))) {5:11:delegate {5:13:builtin foreach} {=list {5:21:delegate {3:22:auto 1}} {5:24:delegate {3:25:auto 2}}} {=list {5:27:delegate {5:29:builtin if} {=list {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}}} {=list {=pair {5:44:word std} {5:48:closure {=compound {5:50:punct .} {5:51:word test} {5:55:punct .} {5:56:delegate {3:28:auto _}}}}}}}}}`:`{} {5:11 {5:13:null}}`,

		`131 5:9:.test.y $1 {5:21:delegate {3:22:auto 1}}`:`zzz {5:21 {1:9:word zzz}}`,
		`131 5:9:.test.y $2 {5:24:delegate {3:25:auto 2}}`:`{} {5:24 {1:9:null}}`,
		`131 5:9:.test.y $_ {5:40:delegate {3:28:auto _}}`:`zzz {5:40 {5:21 {1:9:word zzz}}}`,
		`131 5:9:.test.y .test.$_ {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}`:`.test.zzz {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40 {5:21 {1:9:word zzz}}}}`,
		`131 5:9:.test.y &(.test.$_) {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}}`:`&(.test.zzz) {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40 {5:21 {1:9:word zzz}}}}}`,
		`131 5:9:.test.y .test.zzz {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40 {5:21 {1:9:word zzz}}}}`:`.test.zzz {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40 {5:21 {1:9:word zzz}}}}`,
		`131 5:9:.test.y &(.test.zzz) {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40 {5:21 {1:9:word zzz}}}}}`:`&(.test.zzz) {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40 {5:21 {1:9:word zzz}}}}}`,
		`131 5:9:.test.y $(if &(.test.$_),std=&(.test.$_)) {5:27:delegate {5:29:builtin if} {=list {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}}} {=list {=pair {5:44:word std} {5:48:closure {=compound {5:50:punct .} {5:51:word test} {5:55:punct .} {5:56:delegate {3:28:auto _}}}}}}}`:`{} {5:27 {5:29:null}}`,
		`131 5:9:.test.y $(foreach $1 $2,$(if &(.test.$_),std=&(.test.$_))) {5:11:delegate {5:13:builtin foreach} {=list {5:21:delegate {3:22:auto 1}} {5:24:delegate {3:25:auto 2}}} {=list {5:27:delegate {5:29:builtin if} {=list {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}}} {=list {=pair {5:44:word std} {5:48:closure {=compound {5:50:punct .} {5:51:word test} {5:55:punct .} {5:56:delegate {3:28:auto _}}}}}}}}}`:`{} {5:11 {5:13:null}}`,

		`138 5:9:.test.y $1 {5:21:delegate {3:22:auto 1}}`:`{} {5:21 {1:9:null}}`,
		`138 5:9:.test.y $2 {5:24:delegate {3:25:auto 2}}`:`zzz {5:24 {1:9:word zzz}}`,
		`138 5:9:.test.y $_ {5:40:delegate {3:28:auto _}}`:`zzz {5:40 {5:24 {1:9:word zzz}}}`,
		`138 5:9:.test.y .test.$_ {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}`:`.test.zzz {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40 {5:24 {1:9:word zzz}}}}`,
		`138 5:9:.test.y &(.test.$_) {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}}`:`&(.test.zzz) {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40 {5:24 {1:9:word zzz}}}}}`,
		`138 5:9:.test.y .test.zzz {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40 {5:24 {1:9:word zzz}}}}`:`.test.zzz {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40 {5:24 {1:9:word zzz}}}}`,
		`138 5:9:.test.y &(.test.zzz) {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40 {5:24 {1:9:word zzz}}}}}`:`&(.test.zzz) {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40 {5:24 {1:9:word zzz}}}}}`,
		`138 5:9:.test.y $(if &(.test.$_),std=&(.test.$_)) {5:27:delegate {5:29:builtin if} {=list {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}}} {=list {=pair {5:44:word std} {5:48:closure {=compound {5:50:punct .} {5:51:word test} {5:55:punct .} {5:56:delegate {3:28:auto _}}}}}}}`:`{} {5:27 {5:29:null}}`,
		`138 5:9:.test.y $(foreach $1 $2,$(if &(.test.$_),std=&(.test.$_))) {5:11:delegate {5:13:builtin foreach} {=list {5:21:delegate {3:22:auto 1}} {5:24:delegate {3:25:auto 2}}} {=list {5:27:delegate {5:29:builtin if} {=list {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}}} {=list {=pair {5:44:word std} {5:48:closure {=compound {5:50:punct .} {5:51:word test} {5:55:punct .} {5:56:delegate {3:28:auto _}}}}}}}}}`:`{} {5:11 {5:13:null}}`,

		`145 5:9:.test.y $1 {5:21:delegate {3:22:auto 1}}`:`if.x {5:21 {1:9:word if.x}}`,
		`145 5:9:.test.y $2 {5:24:delegate {3:25:auto 2}}`:`{} {5:24 {3:25:null}}`,
		`145 5:9:.test.y $_ {5:40:delegate {3:28:auto _}}`:`if.x {5:40 {5:21 {1:9:word if.x}}}`,
		`145 5:9:.test.y .test.$_ {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}`:`.test.if.x {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40 {5:21 {1:9:word if.x}}}}`,
		`145 5:9:.test.y &(.test.$_) {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}}`:`xxx {5:32 {7:15:word xxx}}`,
		`145 5:9:.test.y $_ {5:56:delegate {3:28:auto _}}`:`if.x {5:56 {5:21 {1:9:word if.x}}}`,
		`145 5:9:.test.y .test.$_ {=compound {5:50:punct .} {5:51:word test} {5:55:punct .} {5:56:delegate {3:28:auto _}}}`:`.test.if.x {=compound {5:50:punct .} {5:51:word test} {5:55:punct .} {5:56 {5:21 {1:9:word if.x}}}}`,
		`145 5:9:.test.y &(.test.$_) {5:48:closure {=compound {5:50:punct .} {5:51:word test} {5:55:punct .} {5:56:delegate {3:28:auto _}}}}`:`xxx {5:48 {7:15:word xxx}}`,
		`145 5:9:.test.y $(if &(.test.$_),std=&(.test.$_)) {5:27:delegate {5:29:builtin if} {=list {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}}} {=list {=pair {5:44:word std} {5:48:closure {=compound {5:50:punct .} {5:51:word test} {5:55:punct .} {5:56:delegate {3:28:auto _}}}}}}}`:`std=xxx {5:27 {=pair {5:44:word std} {5:48 {7:15:word xxx}}}}`,
		`145 5:9:.test.y $(foreach $1 $2,$(if &(.test.$_),std=&(.test.$_))) {5:11:delegate {5:13:builtin foreach} {=list {5:21:delegate {3:22:auto 1}} {5:24:delegate {3:25:auto 2}}} {=list {5:27:delegate {5:29:builtin if} {=list {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}}} {=list {=pair {5:44:word std} {5:48:closure {=compound {5:50:punct .} {5:51:word test} {5:55:punct .} {5:56:delegate {3:28:auto _}}}}}}}}}`:`std=xxx {5:11 {5:27 {=pair {5:44:word std} {5:48 {7:15:word xxx}}}}}`,

		`152 5:9:.test.y $1 {5:21:delegate {3:22:auto 1}}`:`if.x {5:21 {1:9:word if.x}}`,
		`152 5:9:.test.y $2 {5:24:delegate {3:25:auto 2}}`:`{} {5:24 {1:9:null}}`,
		`152 5:9:.test.y $_ {5:40:delegate {3:28:auto _}}`:`if.x {5:40 {5:21 {1:9:word if.x}}}`,
		`152 5:9:.test.y .test.$_ {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}`:`.test.if.x {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40 {5:21 {1:9:word if.x}}}}`,
		`152 5:9:.test.y &(.test.$_) {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}}`:`xxx {5:32 {7:15:word xxx}}`,
		`152 5:9:.test.y $_ {5:56:delegate {3:28:auto _}}`:`if.x {5:56 {5:21 {1:9:word if.x}}}`,
		`152 5:9:.test.y .test.$_ {=compound {5:50:punct .} {5:51:word test} {5:55:punct .} {5:56:delegate {3:28:auto _}}}`:`.test.if.x {=compound {5:50:punct .} {5:51:word test} {5:55:punct .} {5:56 {5:21 {1:9:word if.x}}}}`,
		`152 5:9:.test.y &(.test.$_) {5:48:closure {=compound {5:50:punct .} {5:51:word test} {5:55:punct .} {5:56:delegate {3:28:auto _}}}}`:`xxx {5:48 {7:15:word xxx}}`,
		`152 5:9:.test.y $(if &(.test.$_),std=&(.test.$_)) {5:27:delegate {5:29:builtin if} {=list {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}}} {=list {=pair {5:44:word std} {5:48:closure {=compound {5:50:punct .} {5:51:word test} {5:55:punct .} {5:56:delegate {3:28:auto _}}}}}}}`:`std=xxx {5:27 {=pair {5:44:word std} {5:48 {7:15:word xxx}}}}`,
		`152 5:9:.test.y $(foreach $1 $2,$(if &(.test.$_),std=&(.test.$_))) {5:11:delegate {5:13:builtin foreach} {=list {5:21:delegate {3:22:auto 1}} {5:24:delegate {3:25:auto 2}}} {=list {5:27:delegate {5:29:builtin if} {=list {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}}} {=list {=pair {5:44:word std} {5:48:closure {=compound {5:50:punct .} {5:51:word test} {5:55:punct .} {5:56:delegate {3:28:auto _}}}}}}}}}`:`std=xxx {5:11 {5:27 {=pair {5:44:word std} {5:48 {7:15:word xxx}}}}}`,

		`159 5:9:.test.y $1 {5:21:delegate {3:22:auto 1}}`:`if.x {5:21 {1:9:word if.x}}`,
		`159 5:9:.test.y $2 {5:24:delegate {3:25:auto 2}}`:`if.y {5:24 {1:9:word if.y}}`,
		`159 5:9:.test.y $_ {5:40:delegate {3:28:auto _}}`:[]string{`if.x {5:40 {5:21 {1:9:word if.x}}}`,`if.y {5:40 {5:24 {1:9:word if.y}}}`},
		`159 5:9:.test.y .test.$_ {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}`:[]string{
			`.test.if.x {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40 {5:21 {1:9:word if.x}}}}`,
			`.test.if.y {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40 {5:24 {1:9:word if.y}}}}`,
		},
		`159 5:9:.test.y &(.test.$_) {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}}`:[]string{
			`xxx {5:32 {7:15:word xxx}}`,
			`yyy {5:32 {8:15:word yyy}}`,
		},
		`159 5:9:.test.y $_ {5:56:delegate {3:28:auto _}}`:[]string{`if.x {5:56 {5:21 {1:9:word if.x}}}`,`if.y {5:56 {5:24 {1:9:word if.y}}}`},
		`159 5:9:.test.y .test.$_ {=compound {5:50:punct .} {5:51:word test} {5:55:punct .} {5:56:delegate {3:28:auto _}}}`:[]string{
			`.test.if.x {=compound {5:50:punct .} {5:51:word test} {5:55:punct .} {5:56 {5:21 {1:9:word if.x}}}}`,
			`.test.if.y {=compound {5:50:punct .} {5:51:word test} {5:55:punct .} {5:56 {5:24 {1:9:word if.y}}}}`,
		},
		`159 5:9:.test.y &(.test.$_) {5:48:closure {=compound {5:50:punct .} {5:51:word test} {5:55:punct .} {5:56:delegate {3:28:auto _}}}}`:[]string{
			`xxx {5:48 {7:15:word xxx}}`,
			`yyy {5:48 {8:15:word yyy}}`,
		},
		`159 5:9:.test.y $(if &(.test.$_),std=&(.test.$_)) {5:27:delegate {5:29:builtin if} {=list {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}}} {=list {=pair {5:44:word std} {5:48:closure {=compound {5:50:punct .} {5:51:word test} {5:55:punct .} {5:56:delegate {3:28:auto _}}}}}}}`:[]string{
			`std=xxx {5:27 {=pair {5:44:word std} {5:48 {7:15:word xxx}}}}`,
			`std=yyy {5:27 {=pair {5:44:word std} {5:48 {8:15:word yyy}}}}`,
		},
		`159 5:9:.test.y $(foreach $1 $2,$(if &(.test.$_),std=&(.test.$_))) {5:11:delegate {5:13:builtin foreach} {=list {5:21:delegate {3:22:auto 1}} {5:24:delegate {3:25:auto 2}}} {=list {5:27:delegate {5:29:builtin if} {=list {5:32:closure {=compound {5:34:punct .} {5:35:word test} {5:39:punct .} {5:40:delegate {3:28:auto _}}}}} {=list {=pair {5:44:word std} {5:48:closure {=compound {5:50:punct .} {5:51:word test} {5:55:punct .} {5:56:delegate {3:28:auto _}}}}}}}}}`:`std=xxx std=yyy {=list {5:11 {5:27 {=pair {5:44:word std} {5:48 {7:15:word xxx}}}}} {5:11 {5:27 {=pair {5:44:word std} {5:48 {8:15:word yyy}}}}}}`,

		`178 6:9:.test.z $1 {6:21:delegate {3:22:auto 1}}`:`if.x {6:21 {1:9:word if.x}}`,
		`178 6:9:.test.z $2 {6:24:delegate {3:25:auto 2}}`:`{} {6:24 {3:25:null}}`,
		`178 6:9:.test.z $_ {6:40:delegate {3:28:auto _}}`:`if.x {6:40 {6:21 {1:9:word if.x}}}`,
		`178 6:9:.test.z .test.$_ {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}`:`.test.if.x {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40 {6:21 {1:9:word if.x}}}}`,
		`178 6:9:.test.z $(.test.$_) {6:32:delegate {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}}`:`xxx {6:32 {7:15:word xxx}}`,
		`178 6:9:.test.z $_ {6:56:delegate {3:28:auto _}}`:`if.x {6:56 {6:21 {1:9:word if.x}}}`,
		`178 6:9:.test.z .test.$_ {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}`:`.test.if.x {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56 {6:21 {1:9:word if.x}}}}`,
		`178 6:9:.test.z &(.test.$_) {6:48:closure {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}}`:`&(.test.if.x) {6:48:closure {7:12:def .test.if.x}}`,
		`178 6:9:.test.z $(if $(.test.$_),std=&(.test.$_)) {6:27:delegate {6:29:builtin if} {=list {6:32:delegate {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}}} {=list {=pair {6:44:word std} {6:48:closure {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}}}}}`:`std=&(.test.if.x) {6:27 {=pair {6:44:word std} {6:48:closure {7:12:def .test.if.x}}}}`,
		`178 6:9:.test.z $(foreach $1 $2,$(if $(.test.$_),std=&(.test.$_))) {6:11:delegate {6:13:builtin foreach} {=list {6:21:delegate {3:22:auto 1}} {6:24:delegate {3:25:auto 2}}} {=list {6:27:delegate {6:29:builtin if} {=list {6:32:delegate {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}}} {=list {=pair {6:44:word std} {6:48:closure {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}}}}}}}`:`std=&(.test.if.x) {6:11 {6:27 {=pair {6:44:word std} {6:48:closure {7:12:def .test.if.x}}}}}`,

		`182 &(.test.if.x) {6:48:closure {7:12:def .test.if.x}}`:`xxx {6:48 {7:15:word xxx}}`,

		`185 6:9:.test.z $1 {6:21:delegate {3:22:auto 1}}`:`if.x {6:21 {1:9:word if.x}}`,
		`185 6:9:.test.z $2 {6:24:delegate {3:25:auto 2}}`:`{} {6:24 {1:9:null}}`,
		`185 6:9:.test.z $_ {6:40:delegate {3:28:auto _}}`:`if.x {6:40 {6:21 {1:9:word if.x}}}`,
		`185 6:9:.test.z .test.$_ {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}`:`.test.if.x {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40 {6:21 {1:9:word if.x}}}}`,
		`185 6:9:.test.z $(.test.$_) {6:32:delegate {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}}`:`xxx {6:32 {7:15:word xxx}}`,
		`185 6:9:.test.z $_ {6:56:delegate {3:28:auto _}}`:`if.x {6:56 {6:21 {1:9:word if.x}}}`,
		`185 6:9:.test.z .test.$_ {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}`:`.test.if.x {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56 {6:21 {1:9:word if.x}}}}`,
		`185 6:9:.test.z &(.test.$_) {6:48:closure {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}}`:`&(.test.if.x) {6:48:closure {7:12:def .test.if.x}}`,
		`185 6:9:.test.z $(if $(.test.$_),std=&(.test.$_)) {6:27:delegate {6:29:builtin if} {=list {6:32:delegate {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}}} {=list {=pair {6:44:word std} {6:48:closure {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}}}}}`:`std=&(.test.if.x) {6:27 {=pair {6:44:word std} {6:48:closure {7:12:def .test.if.x}}}}`,
		`185 6:9:.test.z $(foreach $1 $2,$(if $(.test.$_),std=&(.test.$_))) {6:11:delegate {6:13:builtin foreach} {=list {6:21:delegate {3:22:auto 1}} {6:24:delegate {3:25:auto 2}}} {=list {6:27:delegate {6:29:builtin if} {=list {6:32:delegate {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}}} {=list {=pair {6:44:word std} {6:48:closure {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}}}}}}}`:`std=&(.test.if.x) {6:11 {6:27 {=pair {6:44:word std} {6:48:closure {7:12:def .test.if.x}}}}}`,

		`189 &(.test.if.x) {6:48:closure {7:12:def .test.if.x}}`:`xxx {6:48 {7:15:word xxx}}`,

		`192 6:9:.test.z $1 {6:21:delegate {3:22:auto 1}}`:`{} {6:21 {1:9:null}}`,
		`192 6:9:.test.z $2 {6:24:delegate {3:25:auto 2}}`:`if.y {6:24 {1:9:word if.y}}`,
		`192 6:9:.test.z $_ {6:40:delegate {3:28:auto _}}`:`if.y {6:40 {6:24 {1:9:word if.y}}}`,
		`192 6:9:.test.z .test.$_ {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}`:`.test.if.y {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40 {6:24 {1:9:word if.y}}}}`,
		`192 6:9:.test.z $(.test.$_) {6:32:delegate {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}}`:`yyy {6:32 {8:15:word yyy}}`,
		`192 6:9:.test.z $_ {6:56:delegate {3:28:auto _}}`:`if.y {6:56 {6:24 {1:9:word if.y}}}`,
		`192 6:9:.test.z .test.$_ {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}`:`.test.if.y {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56 {6:24 {1:9:word if.y}}}}`,
		`192 6:9:.test.z &(.test.$_) {6:48:closure {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}}`:`&(.test.if.y) {6:48:closure {8:12:def .test.if.y}}`,
		`192 6:9:.test.z $(if $(.test.$_),std=&(.test.$_)) {6:27:delegate {6:29:builtin if} {=list {6:32:delegate {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}}} {=list {=pair {6:44:word std} {6:48:closure {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}}}}}`:`std=&(.test.if.y) {6:27 {=pair {6:44:word std} {6:48:closure {8:12:def .test.if.y}}}}`,
		`192 6:9:.test.z $(foreach $1 $2,$(if $(.test.$_),std=&(.test.$_))) {6:11:delegate {6:13:builtin foreach} {=list {6:21:delegate {3:22:auto 1}} {6:24:delegate {3:25:auto 2}}} {=list {6:27:delegate {6:29:builtin if} {=list {6:32:delegate {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}}} {=list {=pair {6:44:word std} {6:48:closure {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}}}}}}}`:`std=&(.test.if.y) {6:11 {6:27 {=pair {6:44:word std} {6:48:closure {8:12:def .test.if.y}}}}}`,

		`196 &(.test.if.y) {6:48:closure {8:12:def .test.if.y}}`:`yyy {6:48 {8:15:word yyy}}`,

		`199 6:9:.test.z $1 {6:21:delegate {3:22:auto 1}}`:`if.x {6:21 {1:9:word if.x}}`,
		`199 6:9:.test.z $2 {6:24:delegate {3:25:auto 2}}`:`if.y {6:24 {1:9:word if.y}}`,
		`199 6:9:.test.z $_ {6:40:delegate {3:28:auto _}}`:[]string{`if.x {6:40 {6:21 {1:9:word if.x}}}`,`if.y {6:40 {6:24 {1:9:word if.y}}}`},
		`199 6:9:.test.z .test.$_ {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}`:[]string{
			`.test.if.x {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40 {6:21 {1:9:word if.x}}}}`,
			`.test.if.y {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40 {6:24 {1:9:word if.y}}}}`,
		},
		`199 6:9:.test.z $(.test.$_) {6:32:delegate {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}}`:[]string{
			`xxx {6:32 {7:15:word xxx}}`,
			`yyy {6:32 {8:15:word yyy}}`,
		},
		`199 6:9:.test.z $_ {6:56:delegate {3:28:auto _}}`:[]string{`if.x {6:56 {6:21 {1:9:word if.x}}}`,`if.y {6:56 {6:24 {1:9:word if.y}}}`},
		`199 6:9:.test.z .test.$_ {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}`:[]string{
			`.test.if.x {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56 {6:21 {1:9:word if.x}}}}`,
			`.test.if.y {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56 {6:24 {1:9:word if.y}}}}`,
		},
		`199 6:9:.test.z &(.test.$_) {6:48:closure {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}}`:[]string{
			`&(.test.if.x) {6:48:closure {7:12:def .test.if.x}}`,
			`&(.test.if.y) {6:48:closure {8:12:def .test.if.y}}`,
		},
		`199 6:9:.test.z $(if $(.test.$_),std=&(.test.$_)) {6:27:delegate {6:29:builtin if} {=list {6:32:delegate {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}}} {=list {=pair {6:44:word std} {6:48:closure {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}}}}}`:[]string{
			`std=&(.test.if.x) {6:27 {=pair {6:44:word std} {6:48:closure {7:12:def .test.if.x}}}}`,
			`std=&(.test.if.y) {6:27 {=pair {6:44:word std} {6:48:closure {8:12:def .test.if.y}}}}`,
		},
		`199 6:9:.test.z $(foreach $1 $2,$(if $(.test.$_),std=&(.test.$_))) {6:11:delegate {6:13:builtin foreach} {=list {6:21:delegate {3:22:auto 1}} {6:24:delegate {3:25:auto 2}}} {=list {6:27:delegate {6:29:builtin if} {=list {6:32:delegate {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}}} {=list {=pair {6:44:word std} {6:48:closure {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}}}}}}}`:`std=&(.test.if.x) std=&(.test.if.y) {=list {6:11 {6:27 {=pair {6:44:word std} {6:48:closure {7:12:def .test.if.x}}}}} {6:11 {6:27 {=pair {6:44:word std} {6:48:closure {8:12:def .test.if.y}}}}}}`,

		`203 &(.test.if.x) {6:48:closure {7:12:def .test.if.x}}`:`xxx {6:48 {7:15:word xxx}}`,
		`203 &(.test.if.y) {6:48:closure {8:12:def .test.if.y}}`:`yyy {6:48 {8:15:word yyy}}`,

		`206 6:9:.test.z $1 {6:21:delegate {3:22:auto 1}}`:`zzz {6:21 {1:9:word zzz}}`,
		`206 6:9:.test.z $2 {6:24:delegate {3:25:auto 2}}`:`{} {6:24 {1:9:null}}`,
		`206 6:9:.test.z $_ {6:40:delegate {3:28:auto _}}`:`zzz {6:40 {6:21 {1:9:word zzz}}}`,
		`206 6:9:.test.z .test.$_ {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}`:`.test.zzz {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40 {6:21 {1:9:word zzz}}}}`,
		`206 6:9:.test.z $(.test.$_) {6:32:delegate {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}}`:`{} {6:32:null}`,
		`206 6:9:.test.z $(if $(.test.$_),std=&(.test.$_)) {6:27:delegate {6:29:builtin if} {=list {6:32:delegate {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}}} {=list {=pair {6:44:word std} {6:48:closure {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}}}}}`:`{} {6:27 {6:29:null}}`,
		`206 6:9:.test.z $(foreach $1 $2,$(if $(.test.$_),std=&(.test.$_))) {6:11:delegate {6:13:builtin foreach} {=list {6:21:delegate {3:22:auto 1}} {6:24:delegate {3:25:auto 2}}} {=list {6:27:delegate {6:29:builtin if} {=list {6:32:delegate {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}}} {=list {=pair {6:44:word std} {6:48:closure {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}}}}}}}`:`{} {6:11 {6:13:null}}`,

		`213 6:9:.test.z $1 {6:21:delegate {3:22:auto 1}}`:`{} {6:21 {1:9:null}}`,
		`213 6:9:.test.z $2 {6:24:delegate {3:25:auto 2}}`:`zzz {6:24 {1:9:word zzz}}`,
		`213 6:9:.test.z $_ {6:40:delegate {3:28:auto _}}`:`zzz {6:40 {6:24 {1:9:word zzz}}}`,
		`213 6:9:.test.z .test.$_ {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}`:`.test.zzz {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40 {6:24 {1:9:word zzz}}}}`,
		`213 6:9:.test.z $(.test.$_) {6:32:delegate {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}}`:`{} {6:32:null}`,
		`213 6:9:.test.z $(if $(.test.$_),std=&(.test.$_)) {6:27:delegate {6:29:builtin if} {=list {6:32:delegate {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}}} {=list {=pair {6:44:word std} {6:48:closure {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}}}}}`:`{} {6:27 {6:29:null}}`,
		`213 6:9:.test.z $(foreach $1 $2,$(if $(.test.$_),std=&(.test.$_))) {6:11:delegate {6:13:builtin foreach} {=list {6:21:delegate {3:22:auto 1}} {6:24:delegate {3:25:auto 2}}} {=list {6:27:delegate {6:29:builtin if} {=list {6:32:delegate {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}}} {=list {=pair {6:44:word std} {6:48:closure {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}}}}}}}`:`{} {6:11 {6:13:null}}`,

		`220 6:9:.test.z $1 {6:21:delegate {3:22:auto 1}}`:`if.x {6:21 {1:9:word if.x}}`,
		`220 6:9:.test.z $2 {6:24:delegate {3:25:auto 2}}`:`{} {6:24 {3:25:null}}`,
		`220 6:9:.test.z $_ {6:40:delegate {3:28:auto _}}`:`if.x {6:40 {6:21 {1:9:word if.x}}}`,
		`220 6:9:.test.z .test.$_ {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}`:`.test.if.x {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40 {6:21 {1:9:word if.x}}}}`,
		`220 6:9:.test.z $(.test.$_) {6:32:delegate {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}}`:`xxx {6:32 {7:15:word xxx}}`,
		`220 6:9:.test.z $_ {6:56:delegate {3:28:auto _}}`:`if.x {6:56 {6:21 {1:9:word if.x}}}`,
		`220 6:9:.test.z .test.$_ {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}`:`.test.if.x {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56 {6:21 {1:9:word if.x}}}}`,
		`220 6:9:.test.z &(.test.$_) {6:48:closure {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}}`:`xxx {6:48 {7:15:word xxx}}`,
		`220 6:9:.test.z $(if $(.test.$_),std=&(.test.$_)) {6:27:delegate {6:29:builtin if} {=list {6:32:delegate {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}}} {=list {=pair {6:44:word std} {6:48:closure {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}}}}}`:`std=xxx {6:27 {=pair {6:44:word std} {6:48 {7:15:word xxx}}}}`,
		`220 6:9:.test.z $(foreach $1 $2,$(if $(.test.$_),std=&(.test.$_))) {6:11:delegate {6:13:builtin foreach} {=list {6:21:delegate {3:22:auto 1}} {6:24:delegate {3:25:auto 2}}} {=list {6:27:delegate {6:29:builtin if} {=list {6:32:delegate {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}}} {=list {=pair {6:44:word std} {6:48:closure {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}}}}}}}`:`std=xxx {6:11 {6:27 {=pair {6:44:word std} {6:48 {7:15:word xxx}}}}}`,

		`227 6:9:.test.z $1 {6:21:delegate {3:22:auto 1}}`:`if.x {6:21 {1:9:word if.x}}`,
		`227 6:9:.test.z $2 {6:24:delegate {3:25:auto 2}}`:`{} {6:24 {1:9:null}}`,
		`227 6:9:.test.z $_ {6:40:delegate {3:28:auto _}}`:`if.x {6:40 {6:21 {1:9:word if.x}}}`,
		`227 6:9:.test.z .test.$_ {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}`:`.test.if.x {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40 {6:21 {1:9:word if.x}}}}`,
		`227 6:9:.test.z $(.test.$_) {6:32:delegate {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}}`:`xxx {6:32 {7:15:word xxx}}`,
		`227 6:9:.test.z $_ {6:56:delegate {3:28:auto _}}`:`if.x {6:56 {6:21 {1:9:word if.x}}}`,
		`227 6:9:.test.z .test.$_ {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}`:`.test.if.x {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56 {6:21 {1:9:word if.x}}}}`,
		`227 6:9:.test.z &(.test.$_) {6:48:closure {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}}`:`xxx {6:48 {7:15:word xxx}}`,
		`227 6:9:.test.z $(if $(.test.$_),std=&(.test.$_)) {6:27:delegate {6:29:builtin if} {=list {6:32:delegate {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}}} {=list {=pair {6:44:word std} {6:48:closure {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}}}}}`:`std=xxx {6:27 {=pair {6:44:word std} {6:48 {7:15:word xxx}}}}`,
		`227 6:9:.test.z $(foreach $1 $2,$(if $(.test.$_),std=&(.test.$_))) {6:11:delegate {6:13:builtin foreach} {=list {6:21:delegate {3:22:auto 1}} {6:24:delegate {3:25:auto 2}}} {=list {6:27:delegate {6:29:builtin if} {=list {6:32:delegate {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}}} {=list {=pair {6:44:word std} {6:48:closure {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}}}}}}}`:`std=xxx {6:11 {6:27 {=pair {6:44:word std} {6:48 {7:15:word xxx}}}}}`,

		`234 6:9:.test.z $1 {6:21:delegate {3:22:auto 1}}`:`if.x {6:21 {1:9:word if.x}}`,
		`234 6:9:.test.z $2 {6:24:delegate {3:25:auto 2}}`:`if.y {6:24 {1:9:word if.y}}`,
		`234 6:9:.test.z $_ {6:40:delegate {3:28:auto _}}`:[]string{`if.x {6:40 {6:21 {1:9:word if.x}}}`,`if.y {6:40 {6:24 {1:9:word if.y}}}`},
		`234 6:9:.test.z .test.$_ {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}`:[]string{
			`.test.if.x {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40 {6:21 {1:9:word if.x}}}}`,
			`.test.if.y {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40 {6:24 {1:9:word if.y}}}}`,
		},
		`234 6:9:.test.z $(.test.$_) {6:32:delegate {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}}`:[]string{
			`xxx {6:32 {7:15:word xxx}}`,
			`yyy {6:32 {8:15:word yyy}}`,
		},
		`234 6:9:.test.z $_ {6:56:delegate {3:28:auto _}}`:[]string{
			`if.x {6:56 {6:21 {1:9:word if.x}}}`,
			`if.y {6:56 {6:24 {1:9:word if.y}}}`,
		},
		`234 6:9:.test.z .test.$_ {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}`:[]string{
			`.test.if.x {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56 {6:21 {1:9:word if.x}}}}`,
			`.test.if.y {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56 {6:24 {1:9:word if.y}}}}`,
		},
		`234 6:9:.test.z &(.test.$_) {6:48:closure {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}}`:[]string{
			`xxx {6:48 {7:15:word xxx}}`,
			`yyy {6:48 {8:15:word yyy}}`,
		},
		`234 6:9:.test.z $(if $(.test.$_),std=&(.test.$_)) {6:27:delegate {6:29:builtin if} {=list {6:32:delegate {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}}} {=list {=pair {6:44:word std} {6:48:closure {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}}}}}`:[]string{
			`std=xxx {6:27 {=pair {6:44:word std} {6:48 {7:15:word xxx}}}}`,
			`std=yyy {6:27 {=pair {6:44:word std} {6:48 {8:15:word yyy}}}}`,
		},
		`234 6:9:.test.z $(foreach $1 $2,$(if $(.test.$_),std=&(.test.$_))) {6:11:delegate {6:13:builtin foreach} {=list {6:21:delegate {3:22:auto 1}} {6:24:delegate {3:25:auto 2}}} {=list {6:27:delegate {6:29:builtin if} {=list {6:32:delegate {=compound {6:34:punct .} {6:35:word test} {6:39:punct .} {6:40:delegate {3:28:auto _}}}}} {=list {=pair {6:44:word std} {6:48:closure {=compound {6:50:punct .} {6:51:word test} {6:55:punct .} {6:56:delegate {3:28:auto _}}}}}}}}}`:`std=xxx std=yyy {=list {6:11 {6:27 {=pair {6:44:word std} {6:48 {7:15:word xxx}}}}} {6:11 {6:27 {=pair {6:44:word std} {6:48 {8:15:word yyy}}}}}}`,
	},
}

var checkstrs__foreach3 = map[string]map[string]any{
	"check-builtins-foreach3_test.go": map[string]any{
	},
}
