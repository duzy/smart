//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints__logic = map[string]map[string]any{
	"loader.go": map[string]any{
		`3:6:val1 &(none) {3:14:closure {3:16:word none}}`:`&(none) {3:14:closure {3:16:word none}}`,
		`3:6:val1 $(or &(none),a) {3:9:delegate {3:11:builtin or} {=list {3:14:closure {3:16:word none}}} {=list {3:22:word a}}}`:`a {3:9 {3:22:word a}}`,
		`4:6:val2 $(or a,&(none)) {4:9:delegate {4:11:builtin or} {=list {4:14:word a}} {=list {4:16:closure {4:18:word none}}}}`:`a {4:9 {4:14:word a}}`,
		`6:6:val4 &(none) {6:14:closure {6:16:word none}}`:`&(none) {6:14:closure {6:16:word none}}`,
		`6:6:val4 $(or &(none),a) {6:9:delegate {6:11:builtin or} {=list {6:14:closure {6:16:word none}}} {=list {6:22:word a}}}`:`a {6:9 {6:22:word a}}`,
		`7:6:val5 $(or a,&(none)) {7:9:delegate {7:11:builtin or} {=list {7:14:word a}} {=list {7:16:closure {7:18:word none}}}}`:`a {7:9 {7:14:word a}}`,
		`11:4:x1 &(variant) {11:28:closure {11:30:word variant}}`:`{} {11:28:null}`,
		`11:4:x1 $(base &(variant)) {11:21:delegate {11:23:builtin base} {=list {11:28:closure {11:30:word variant}}}}`:`{} {11:21 {11:23:null}}`,
		`11:4:x1 $(or $(base &(variant)),bootstrap) {11:16:delegate {11:18:builtin or} {=list {11:21:delegate {11:23:builtin base} {=list {11:28:closure {11:30:word variant}}}}} {=list {11:40:word bootstrap}}}`:`bootstrap {11:16 {11:40:word bootstrap}}`,
		`12:4:x2 &(variant) {12:28:closure {12:30:word variant}}`:`{} {12:28:null}`,
		`12:4:x2 $(base &(variant)) {12:21:delegate {12:23:builtin base} {=list {12:28:closure {12:30:word variant}}}}`:`{} {12:21 {12:23:null}}`,
		`12:4:x2 $(or $(base &(variant)),bootstrap) {12:16:delegate {12:18:builtin or} {=list {12:21:delegate {12:23:builtin base} {=list {12:28:closure {12:30:word variant}}}}} {=list {12:40:word bootstrap}}}`:`bootstrap {12:16 {12:40:word bootstrap}}`,
		`13:4:x3 &(variant) {13:28:closure {13:30:word variant}}`:`{} {13:28:null}`,
		`13:4:x3 $(base &(variant)) {13:21:delegate {13:23:builtin base} {=list {13:28:closure {13:30:word variant}}}}`:`{} {13:21 {13:23:null}}`,
		`13:4:x3 $(or $(base &(variant)),bootstrap) {13:16:delegate {13:18:builtin or} {=list {13:21:delegate {13:23:builtin base} {=list {13:28:closure {13:30:word variant}}}}} {=list {13:40:word bootstrap}}}`:`bootstrap {13:16 {13:40:word bootstrap}}`,
		`14:4:x4 &(variant) {14:28:closure {14:30:word variant}}`:`{} {14:28:null}`,
		`14:4:x4 $(base &(variant)) {14:21:delegate {14:23:builtin base} {=list {14:28:closure {14:30:word variant}}}}`:`'.' {14:21 {14:28:strlit '.'}}`,
		`14:4:x4 $(or $(base &(variant)),bootstrap) {14:16:delegate {14:18:builtin or} {=list {14:21:delegate {14:23:builtin base} {=list {14:28:closure {14:30:word variant}}}}} {=list {14:40:word bootstrap}}}`:`'.' {14:16 {14:21 {14:28:strlit '.'}}}`,
		`15:4:x5 &(variant) {15:28:closure {15:30:word variant}}`:`{} {15:28:null}`,
		`15:4:x5 $(or &(variant),bootstrap) {15:16:delegate {15:18:builtin or} {=list {15:28:closure {15:30:word variant}}} {=list {15:40:word bootstrap}}}`:`bootstrap {15:16 {15:40:word bootstrap}}`,
		`15:4:x5 $(base $(or &(variant),bootstrap)) {15:7:delegate {15:9:builtin base} {=list {15:16:delegate {15:18:builtin or} {=list {15:28:closure {15:30:word variant}}} {=list {15:40:word bootstrap}}}}}`:`'bootstrap' {15:7 {15:16:strlit 'bootstrap'}}`,
	},
	"check-builtins-logic_test.go": map[string]any{
		`42 5:7:val3 &(none) {5:14:closure {5:16:word none}}`:`{} {5:14:null}`,
		`42 5:7:val3 $(or &(none),a) {5:9:delegate {5:11:builtin or} {=list {5:14:closure {5:16:word none}}} {=list {5:22:word a}}}`:`a {5:9 {5:22:word a}}`,
		`80 8:7:val6 $1 {8:15:delegate {8:16:auto 1}}`:`{} {8:15 {8:16:null}}`,
		`80 8:7:val6 $(and $1,$2,$3) {8:9:delegate {8:11:builtin and} {=list {8:15:delegate {8:16:auto 1}}} {=list {8:18:delegate {8:19:auto 2}}} {=list {8:21:delegate {8:22:auto 3}}}}`:`{} {8:9:null}`,
		`82 8:7:val6 $1 {8:15:delegate {8:16:auto 1}}`:`a {8:15 {1:9:word a}}`,
		`82 8:7:val6 $2 {8:18:delegate {8:19:auto 2}}`:`b {8:18 {1:9:word b}}`,
		`82 8:7:val6 $3 {8:21:delegate {8:22:auto 3}}`:`c {8:21 {1:9:word c}}`,
		`82 8:7:val6 $(and $1,$2,$3) {8:9:delegate {8:11:builtin and} {=list {8:15:delegate {8:16:auto 1}}} {=list {8:18:delegate {8:19:auto 2}}} {=list {8:21:delegate {8:22:auto 3}}}}`:`c {8:9 {8:21 {1:9:word c}}}`,
		`98 10:5:x0 &(variant) {10:28:closure {10:30:word variant}}`:`{} {10:28:null}`,
		`98 10:5:x0 $(base &(variant)) {10:21:delegate {10:23:builtin base} {=list {10:28:closure {10:30:word variant}}}}`:`{} {10:21 {10:23:null}}`,
		`98 10:5:x0 $(or $(base &(variant)),bootstrap) {10:16:delegate {10:18:builtin or} {=list {10:21:delegate {10:23:builtin base} {=list {10:28:closure {10:30:word variant}}}}} {=list {10:40:word bootstrap}}}`:`bootstrap {10:16 {10:40:word bootstrap}}`,
		`100 10:5:x0 &(variant) {10:28:closure {10:30:word variant}}`:`{} {10:28:null}`,
		`100 10:5:x0 $(base &(variant)) {10:21:delegate {10:23:builtin base} {=list {10:28:closure {10:30:word variant}}}}`:`{} {10:21 {10:23:null}}`,
		`100 10:5:x0 $(or $(base &(variant)),bootstrap) {10:16:delegate {10:18:builtin or} {=list {10:21:delegate {10:23:builtin base} {=list {10:28:closure {10:30:word variant}}}}} {=list {10:40:word bootstrap}}}`:`bootstrap {10:16 {10:40:word bootstrap}}`,
		`146 14:5:x4 &(variant) {14:28:closure {14:30:word variant}}`:`{} {14:28:null}`,
		`146 14:5:x4 $(base &(variant)) {14:21:delegate {14:23:builtin base} {=list {14:28:closure {14:30:word variant}}}}`:`{} {14:21 {14:23:null}}`,
		`146 14:5:x4 $(or $(base &(variant)),bootstrap) {14:16:delegate {14:18:builtin or} {=list {14:21:delegate {14:23:builtin base} {=list {14:28:closure {14:30:word variant}}}}} {=list {14:40:word bootstrap}}}`:`bootstrap {14:16 {14:40:word bootstrap}}`,
		`158 15:5:x5 &(variant) {15:28:closure {15:30:word variant}}`:`{} {15:28:null}`,
		`158 15:5:x5 $(or &(variant),bootstrap) {15:16:delegate {15:18:builtin or} {=list {15:28:closure {15:30:word variant}}} {=list {15:40:word bootstrap}}}`:`bootstrap {15:16 {15:40:word bootstrap}}`,
		`158 15:5:x5 $(base $(or &(variant),bootstrap)) {15:7:delegate {15:9:builtin base} {=list {15:16:delegate {15:18:builtin or} {=list {15:28:closure {15:30:word variant}}} {=list {15:40:word bootstrap}}}}}`:`bootstrap {15:7 {15:16:word bootstrap}}`,
	},
}

var checkstrs__logic = map[string]map[string]any{
	"loader.go": map[string]any{
		`11:4:x1 &(variant) {11:28:closure {11:30:word variant}}`:`{11:28:null} `,
		`12:4:x2 &(variant) {12:28:closure {12:30:word variant}}`:`{12:28:null} `,
		`13:4:x3 &(variant) {13:28:closure {13:30:word variant}}`:`{13:28:null} `,
		`14:4:x4 &(variant) {14:28:closure {14:30:word variant}}`:`{14:28:null} `,
		`15:4:x5 $(or &(variant),bootstrap) {15:16:delegate {15:18:builtin or} {=list {15:28:closure {15:30:word variant}}} {=list {15:40:word bootstrap}}}`:`{15:16 {15:40:word bootstrap}} bootstrap`,
	},
	"check-builtins-logic_test.go": map[string]any{
		`42 5:7:val3 $(or &(none),a) {5:9:delegate {5:11:builtin or} {=list {5:14:closure {5:16:word none}}} {=list {5:22:word a}}}`:`{5:9 {5:22:word a}} a`,
		`80 8:7:val6 $(and $1,$2,$3) {8:9:delegate {8:11:builtin and} {=list {8:15:delegate {8:16:auto 1}}} {=list {8:18:delegate {8:19:auto 2}}} {=list {8:21:delegate {8:22:auto 3}}}}`:`{8:9:null} `,
		`98 10:5:x0 &(variant) {10:28:closure {10:30:word variant}}`:`{10:28:null} `,
		`100 10:5:x0 &(variant) {10:28:closure {10:30:word variant}}`:`{10:28:null} `,
		`100 10:5:x0 $(or $(base &(variant)),bootstrap) {10:16:delegate {10:18:builtin or} {=list {10:21:delegate {10:23:builtin base} {=list {10:28:closure {10:30:word variant}}}}} {=list {10:40:word bootstrap}}}`:`{10:16 {10:40:word bootstrap}} bootstrap`,
		`146 14:5:x4 &(variant) {14:28:closure {14:30:word variant}}`:`{14:28:null} `,
		`146 14:5:x4 $(or $(base &(variant)),bootstrap) {14:16:delegate {14:18:builtin or} {=list {14:21:delegate {14:23:builtin base} {=list {14:28:closure {14:30:word variant}}}}} {=list {14:40:word bootstrap}}}`:`{14:16 {14:40:word bootstrap}} bootstrap`,
		`158 15:5:x5 $(or &(variant),bootstrap) {15:16:delegate {15:18:builtin or} {=list {15:28:closure {15:30:word variant}}} {=list {15:40:word bootstrap}}}`:`{15:16 {15:40:word bootstrap}} bootstrap`,
		`158 15:5:x5 $(base $(or &(variant),bootstrap)) {15:7:delegate {15:9:builtin base} {=list {15:16:delegate {15:18:builtin or} {=list {15:28:closure {15:30:word variant}}} {=list {15:40:word bootstrap}}}}}`:`{15:7 {15:16:word bootstrap}} bootstrap`,
	},
}
