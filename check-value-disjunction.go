//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints_value_disjunction = map[string]map[string]any{
	"loader.go": map[string]any{
		`.test {=compound {8:1:punct .} {8:2:word test}}`:`.test {=compound {8:1:punct .} {8:2:word test}}`,
	},
	"check-value-disjunction_test.go": map[string]any{
		`18 3:6:val1 .test {=compound {3:15:punct .} {3:16:word test}}`:`.test {=compound {3:15:punct .} {3:16:word test}}`,
		`18 3:6:val1 &(.test) {3:13:closure {=compound {3:15:punct .} {3:16:word test}}}`:`{8:7:def .test} a b c {=list {3:13 {8:10:word a}} {3:13 {8:12:word b}} {3:13 {8:14:word c}}}`,

		`20 3:6:val1 .test {=compound {3:15:punct .} {3:16:word test}}`:`.test {=compound {3:15:punct .} {3:16:word test}}`,
		`20 3:6:val1 &(.test) {3:13:closure {=compound {3:15:punct .} {3:16:word test}}}`:`{8:7:def .test} &(.test) {3:13:closure {8:7:def .test}}`,

		`24 3:6:val1 &(.test) {3:13:closure {8:7:def .test}}`:`{8:7:def .test} a b c {=list {3:13 {8:10:word a}} {3:13 {8:12:word b}} {3:13 {8:14:word c}}}`,

		`26 3:6:val1 .test {=compound {3:15:punct .} {3:16:word test}}`:`.test {=compound {3:15:punct .} {3:16:word test}}`,
		`26 3:6:val1 &(.test) {3:13:closure {=compound {3:15:punct .} {3:16:word test}}}`:`{8:7:def .test} a b c {=list {3:13 {8:10:word a}} {3:13 {8:12:word b}} {3:13 {8:14:word c}}}`,

		`42 4:6:val2 .test {=compound {4:11:punct .} {4:12:word test}}`:`.test {=compound {4:11:punct .} {4:12:word test}}`,
		`42 4:6:val2 &(.test) {4:9:closure {=compound {4:11:punct .} {4:12:word test}}}`:`{8:7:def .test} a b c {=list {4:9 {8:10:word a}} {4:9 {8:12:word b}} {4:9 {8:14:word c}}}`,

		`44 4:6:val2 .test {=compound {4:11:punct .} {4:12:word test}}`:`.test {=compound {4:11:punct .} {4:12:word test}}`,
		`44 4:6:val2 &(.test) {4:9:closure {=compound {4:11:punct .} {4:12:word test}}}`:`{8:7:def .test} &(.test) {4:9:closure {8:7:def .test}}`,

		`48 4:6:val2 &(.test) {4:9:closure {8:7:def .test}}`:`{8:7:def .test} a b c {=list {4:9 {8:10:word a}} {4:9 {8:12:word b}} {4:9 {8:14:word c}}}`,

		`50 4:6:val2 .test {=compound {4:11:punct .} {4:12:word test}}`:`.test {=compound {4:11:punct .} {4:12:word test}}`,
		`50 4:6:val2 &(.test) {4:9:closure {=compound {4:11:punct .} {4:12:word test}}}`:`{8:7:def .test} a b c {=list {4:9 {8:10:word a}} {4:9 {8:12:word b}} {4:9 {8:14:word c}}}`,

		`66 5:6:val3 .test {=compound {5:18:punct .} {5:19:word test}}`:`.test {=compound {5:18:punct .} {5:19:word test}}`,
		`66 5:6:val3 &(.test) {5:16:closure {=compound {5:18:punct .} {5:19:word test}}}`:`{8:7:def .test} a b c {=list {5:16 {8:10:word a}} {5:16 {8:12:word b}} {5:16 {8:14:word c}}}`,
		`66 5:6:val3 bar{&(.test)} {=compound {5:12:word bar} {5:15:disjunction {5:16:closure {=compound {5:18:punct .} {5:19:word test}}}}}`:`bara barb barc {=list {=compound {5:12:word bar} {5:16 {8:10:word a}}} {=compound {5:12:word bar} {5:16 {8:12:word b}}} {=compound {5:12:word bar} {5:16 {8:14:word c}}}}`,
		`66 5:6:val3 bara {=compound {5:12:word bar} {5:16 {8:10:word a}}}`:`bara {=compound {5:12:word bar} {5:16 {8:10:word a}}}`,
		`66 5:6:val3 barb {=compound {5:12:word bar} {5:16 {8:12:word b}}}`:`barb {=compound {5:12:word bar} {5:16 {8:12:word b}}}`,
		`66 5:6:val3 barc {=compound {5:12:word bar} {5:16 {8:14:word c}}}`:`barc {=compound {5:12:word bar} {5:16 {8:14:word c}}}`,

		`68 5:6:val3 .test {=compound {5:18:punct .} {5:19:word test}}`:`.test {=compound {5:18:punct .} {5:19:word test}}`,
		`68 5:6:val3 &(.test) {5:16:closure {=compound {5:18:punct .} {5:19:word test}}}`:`{8:7:def .test} &(.test) {5:16:closure {8:7:def .test}}`,
		`68 5:6:val3 bar{&(.test)} {=compound {5:12:word bar} {5:15:disjunction {5:16:closure {=compound {5:18:punct .} {5:19:word test}}}}}`:`bar{&(.test)} {=compound {5:12:word bar} {5:16:disjunction {5:16:closure {8:7:def .test}}}}`,

		`72 5:6:val3 &(.test) {5:16:closure {8:7:def .test}}`:`{8:7:def .test} a b c {=list {5:16 {8:10:word a}} {5:16 {8:12:word b}} {5:16 {8:14:word c}}}`,
		`72 5:6:val3 bar{&(.test)} {=compound {5:12:word bar} {5:16:disjunction {5:16:closure {8:7:def .test}}}}`:`bara barb barc {=list {=compound {5:12:word bar} {5:16 {8:10:word a}}} {=compound {5:12:word bar} {5:16 {8:12:word b}}} {=compound {5:12:word bar} {5:16 {8:14:word c}}}}`,
		`72 5:6:val3 bara {=compound {5:12:word bar} {5:16 {8:10:word a}}}`:`bara {=compound {5:12:word bar} {5:16 {8:10:word a}}}`,
		`72 5:6:val3 barb {=compound {5:12:word bar} {5:16 {8:12:word b}}}`:`barb {=compound {5:12:word bar} {5:16 {8:12:word b}}}`,
		`72 5:6:val3 barc {=compound {5:12:word bar} {5:16 {8:14:word c}}}`:`barc {=compound {5:12:word bar} {5:16 {8:14:word c}}}`,

		`74 5:6:val3 .test {=compound {5:18:punct .} {5:19:word test}}`:`.test {=compound {5:18:punct .} {5:19:word test}}`,
		`74 5:6:val3 &(.test) {5:16:closure {=compound {5:18:punct .} {5:19:word test}}}`:`{8:7:def .test} a b c {=list {5:16 {8:10:word a}} {5:16 {8:12:word b}} {5:16 {8:14:word c}}}`,
		`74 5:6:val3 bar{&(.test)} {=compound {5:12:word bar} {5:15:disjunction {5:16:closure {=compound {5:18:punct .} {5:19:word test}}}}}`:`bara barb barc {=list {=compound {5:12:word bar} {5:16 {8:10:word a}}} {=compound {5:12:word bar} {5:16 {8:12:word b}}} {=compound {5:12:word bar} {5:16 {8:14:word c}}}}`,

		`78 5:6:val3 bara {=compound {5:12:word bar} {5:16 {8:10:word a}}}`:`bara {=compound {5:12:word bar} {5:16 {8:10:word a}}}`,
		`78 5:6:val3 barb {=compound {5:12:word bar} {5:16 {8:12:word b}}}`:`barb {=compound {5:12:word bar} {5:16 {8:12:word b}}}`,
		`78 5:6:val3 barc {=compound {5:12:word bar} {5:16 {8:14:word c}}}`:`barc {=compound {5:12:word bar} {5:16 {8:14:word c}}}`,

		`90 6:6:val4 .test {=compound {6:14:punct .} {6:15:word test}}`:`.test {=compound {6:14:punct .} {6:15:word test}}`,
		`90 6:6:val4 &(.test) {6:12:closure {=compound {6:14:punct .} {6:15:word test}}}`:`{8:7:def .test} a b c {=list {6:12 {8:10:word a}} {6:12 {8:12:word b}} {6:12 {8:14:word c}}}`,
		`90 6:6:val4 foo{&(.test)} {=compound {6:8:word foo} {6:11:disjunction {6:12:closure {=compound {6:14:punct .} {6:15:word test}}}}}`:`fooa foob fooc {=list {=compound {6:8:word foo} {6:12 {8:10:word a}}} {=compound {6:8:word foo} {6:12 {8:12:word b}}} {=compound {6:8:word foo} {6:12 {8:14:word c}}}}`,
		`90 6:6:val4 fooa {=compound {6:8:word foo} {6:12 {8:10:word a}}}`:`fooa {=compound {6:8:word foo} {6:12 {8:10:word a}}}`,
		`90 6:6:val4 foob {=compound {6:8:word foo} {6:12 {8:12:word b}}}`:`foob {=compound {6:8:word foo} {6:12 {8:12:word b}}}`,
		`90 6:6:val4 fooc {=compound {6:8:word foo} {6:12 {8:14:word c}}}`:`fooc {=compound {6:8:word foo} {6:12 {8:14:word c}}}`,

		`92 6:6:val4 .test {=compound {6:14:punct .} {6:15:word test}}`:`.test {=compound {6:14:punct .} {6:15:word test}}`,
		`92 6:6:val4 &(.test) {6:12:closure {=compound {6:14:punct .} {6:15:word test}}}`:`{8:7:def .test} &(.test) {6:12:closure {8:7:def .test}}`,
		`92 6:6:val4 foo{&(.test)} {=compound {6:8:word foo} {6:11:disjunction {6:12:closure {=compound {6:14:punct .} {6:15:word test}}}}}`:`foo{&(.test)} {=compound {6:8:word foo} {6:12:disjunction {6:12:closure {8:7:def .test}}}}`,

		`96 6:6:val4 &(.test) {6:12:closure {8:7:def .test}}`:`{8:7:def .test} a b c {=list {6:12 {8:10:word a}} {6:12 {8:12:word b}} {6:12 {8:14:word c}}}`,
		`96 6:6:val4 foo{&(.test)} {=compound {6:8:word foo} {6:12:disjunction {6:12:closure {8:7:def .test}}}}`:`fooa foob fooc {=list {=compound {6:8:word foo} {6:12 {8:10:word a}}} {=compound {6:8:word foo} {6:12 {8:12:word b}}} {=compound {6:8:word foo} {6:12 {8:14:word c}}}}`,
		`96 6:6:val4 fooa {=compound {6:8:word foo} {6:12 {8:10:word a}}}`:`fooa {=compound {6:8:word foo} {6:12 {8:10:word a}}}`,
		`96 6:6:val4 foob {=compound {6:8:word foo} {6:12 {8:12:word b}}}`:`foob {=compound {6:8:word foo} {6:12 {8:12:word b}}}`,
		`96 6:6:val4 fooc {=compound {6:8:word foo} {6:12 {8:14:word c}}}`:`fooc {=compound {6:8:word foo} {6:12 {8:14:word c}}}`,

		`98 6:6:val4 .test {=compound {6:14:punct .} {6:15:word test}}`:`.test {=compound {6:14:punct .} {6:15:word test}}`,
		`98 6:6:val4 &(.test) {6:12:closure {=compound {6:14:punct .} {6:15:word test}}}`:`{8:7:def .test} a b c {=list {6:12 {8:10:word a}} {6:12 {8:12:word b}} {6:12 {8:14:word c}}}`,
		`98 6:6:val4 foo{&(.test)} {=compound {6:8:word foo} {6:11:disjunction {6:12:closure {=compound {6:14:punct .} {6:15:word test}}}}}`:`fooa foob fooc {=list {=compound {6:8:word foo} {6:12 {8:10:word a}}} {=compound {6:8:word foo} {6:12 {8:12:word b}}} {=compound {6:8:word foo} {6:12 {8:14:word c}}}}`,

		`102 6:6:val4 fooa {=compound {6:8:word foo} {6:12 {8:10:word a}}}`:`fooa {=compound {6:8:word foo} {6:12 {8:10:word a}}}`,
		`102 6:6:val4 foob {=compound {6:8:word foo} {6:12 {8:12:word b}}}`:`foob {=compound {6:8:word foo} {6:12 {8:12:word b}}}`,
		`102 6:6:val4 fooc {=compound {6:8:word foo} {6:12 {8:14:word c}}}`:`fooc {=compound {6:8:word foo} {6:12 {8:14:word c}}}`,
	},
}

var checkstrs_value_disjunction = map[string]map[string]any{
}
