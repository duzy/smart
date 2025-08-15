//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints_value_4 = map[string]map[string]any{
	"loader.go": map[string]any{
		`1120 9:26:.test.D.c.1 &(.test.x) {18:24:closure {=compound {18:26:punct .} {18:27:word test} {18:31:punct .} {18:32:word x}}}`:`{=compound {18:26:punct .} {18:27:word test} {18:31:punct .} {18:32:word x}} {} {18:24:null}`,
		`1120 9:26:.test.D.c.1 $(.test.x) {26:24:delegate {23:9:def .test.x}}`:`{23:9:def .test.x} .test.v {26:24 {=compound {23:12:punct .} {23:13:word test} {23:17:punct .} {23:18:word v}}}`,
		`1120 9:26:.test.D.c.1 $(value &(.test.x)) {18:16:delegate {18:18:builtin value} {=list {18:24:closure {=compound {18:26:punct .} {18:27:word test} {18:31:punct .} {18:32:word x}}}}}`:`{18:18:builtin value} {} {18:16 {18:18:null}}`,
		`1120 9:26:.test.D.c.1 &(value $(.test.x)) {26:16:closure {26:18:builtin value} {=list {26:24:delegate {23:9:def .test.x}}}}`:`{26:18:builtin value} &(value .test.v) {26:16:closure {26:18:builtin value} {=list {26:24 {=compound {23:12:punct .} {23:13:word test} {23:17:punct .} {23:18:word v}}}}}`,
		`1120 9:26:.test.D.c.1 $(value .test.v) {20:16:delegate {20:18:builtin value} {=list {=compound {20:26:punct .} {20:27:word test} {20:31:punct .} {20:32:word v}}}}`:`{20:18:builtin value} {} {20:16 {20:18:null}}`,
		`1120 9:26:.test.I.c.1 &(.test.x) {29:24:closure {23:9:def .test.x}}`:`{23:9:def .test.x} &(.test.x) {29:24:closure {23:9:def .test.x}}`,
		`1120 9:26:.test.I.c.1 $(.test.x) {31:24:delegate {23:9:def .test.x}}`:`{23:9:def .test.x} .test.v {31:24 {=compound {23:12:punct .} {23:13:word test} {23:17:punct .} {23:18:word v}}}`,
		`1120 9:26:.test.I.c.1 &(.test.x) {33:24:closure {23:9:def .test.x}}`:`{23:9:def .test.x} .test.v {33:24 {=compound {23:12:punct .} {23:13:word test} {23:17:punct .} {23:18:word v}}}`,
		`1120 9:26:.test.I.c.1 $(.test.x) {35:24:delegate {23:9:def .test.x}}`:`{23:9:def .test.x} .test.v {35:24 {=compound {23:12:punct .} {23:13:word test} {23:17:punct .} {23:18:word v}}}`,
		`1120 9:26:.test.I.c.1 &(value &(.test.x)) {29:16:closure {29:18:builtin value} {=list {29:24:closure {23:9:def .test.x}}}}`:`{29:18:builtin value} &(value &(.test.x)) {29:16:closure {29:18:builtin value} {=list {29:24:closure {23:9:def .test.x}}}}`,
		`1120 9:26:.test.I.c.1 &(value $(.test.x)) {31:16:closure {31:18:builtin value} {=list {31:24:delegate {23:9:def .test.x}}}}`:`{31:18:builtin value} &(value .test.v) {31:16:closure {31:18:builtin value} {=list {31:24 {=compound {23:12:punct .} {23:13:word test} {23:17:punct .} {23:18:word v}}}}}`,
		`1120 9:26:.test.I.c.1 $(value &(.test.x)) {33:16:delegate {33:18:builtin value} {=list {33:24:closure {23:9:def .test.x}}}}`:`{33:18:builtin value} xx {33:16 {22:12:word xx}}`,
		`1120 9:26:.test.I.c.1 $(value $(.test.x)) {35:16:delegate {35:18:builtin value} {=list {35:24:delegate {23:9:def .test.x}}}}`:`{35:18:builtin value} xx {35:16 {22:12:word xx}}`,
		`1120 9:26:.test.D.c.1 $1 {37:18:delegate {37:19:auto 1}}`:`{37:15:def 1} {} {37:18 {40:32:null}}`,
		`1120 9:26:.test.D.c.1 $1 {40:32:delegate {37:19:auto 1}}`:`{} {} {40:32:null}`,
		`1120 9:26:.test.D.c.1 $1 {40:51:delegate {37:19:auto 1}}`:`{} {} {40:51:null}`,
		`1120 9:26:.test.D.c.1 $1 {42:26:delegate {37:19:auto 1}}`:`{} {} {42:26:null}`,
		`1120 9:26:.test.D.c.1 &(.test.none) {40:35:closure {=compound {40:37:punct .} {40:38:word test} {40:42:punct .} {40:43:word none}}}`:`{=compound {40:37:punct .} {40:38:word test} {40:42:punct .} {40:43:word none}} &(.test.none) {40:35:closure {=compound {40:37:punct .} {40:38:word test} {40:42:punct .} {40:43:word none}}}`,
		`1120 9:26:.test.D.c.1 $(.test.foreach $1,&(.test.none)) {40:16:delegate {37:15:def .test.foreach} {=list {40:32:delegate {37:19:auto 1}}} {=list {40:35:closure {=compound {40:37:punct .} {40:38:word test} {40:42:punct .} {40:43:word none}}}}}`:`{37:15:def .test.foreach} ({}) {40:16 {=group {37:18 {40:32:null}}}}`,
		`1120 9:26:.test.D.c.1 $(foreach $1,&(.test.x.$_)) {42:16:delegate {42:18:builtin foreach} {=list {42:26:delegate {37:19:auto 1}}} {=list {42:29:closure {=compound {42:31:punct .} {42:32:word test} {42:36:punct .} {42:37:word x} {42:38:punct .} {42:39:delegate {41:40:auto _}}}}}}`:`{42:18:builtin foreach} {} {42:16 {42:18:null}}`,
		`1120 9:26:.test.D.c.1 $1 {42:45:delegate {37:19:auto 1}}`:`{} {} {42:45:null}`,
	},
	"check-value-4_test.go": map[string]any{
		`52 8:27:.test.D.c.0 &(.test.x) {17:24:closure {=compound {17:26:punct .} {17:27:word test} {17:31:punct .} {17:32:word x}}}`:`{=compound {17:26:punct .} {17:27:word test} {17:31:punct .} {17:32:word x}} .test.v {17:24 {=compound {23:12:punct .} {23:13:word test} {23:17:punct .} {23:18:word v}}}`,
		`52 8:27:.test.D.c.0 $(value &(.test.x)) {17:16:delegate {17:18:builtin value} {=list {17:24:closure {=compound {17:26:punct .} {17:27:word test} {17:31:punct .} {17:32:word x}}}}}`:sfmt(`{%[1]s/value/4/do.smart:17:18:builtin value} xx {%[1]s/value/4/do.smart:17:16 {%[1]s/value/4/do.smart:22:12:word xx}}`,testdata_dir),
		`52 8:27:.test.D.c.0 $(value .test.v) {19:16:delegate {19:18:builtin value} {=list {=compound {19:26:punct .} {19:27:word test} {19:31:punct .} {19:32:word v}}}}`:sfmt(`{%[1]s/value/4/do.smart:19:18:builtin value} xx {%[1]s/value/4/do.smart:19:16 {%[1]s/value/4/do.smart:22:12:word xx}}`,testdata_dir),
		`52 8:27:.test.D.c.0 $(.test.x) {25:24:delegate {23:9:def .test.x}}`:`{23:9:def .test.x} .test.v {25:24 {=compound {23:12:punct .} {23:13:word test} {23:17:punct .} {23:18:word v}}}`,
		`52 8:27:.test.D.c.0 &(value $(.test.x)) {25:16:closure {25:18:builtin value} {=list {25:24:delegate {23:9:def .test.x}}}}`:sfmt(`{%[1]s/value/4/do.smart:25:18:builtin value} xx {%[1]s/value/4/do.smart:25:16 {%[1]s/value/4/do.smart:22:12:word xx}}`,testdata_dir),
		`52 8:27:.test.D.c.0 &(.test.none) {39:35:closure {=compound {39:37:punct .} {39:38:word test} {39:42:punct .} {39:43:word none}}}`:sfmt(`{=compound {%[1]s/value/4/do.smart:39:37:punct .} {%[1]s/value/4/do.smart:39:38:word test} {%[1]s/value/4/do.smart:39:42:punct .} {%[1]s/value/4/do.smart:39:43:word none}} {} {%[1]s/value/4/do.smart:39:35:null}`,testdata_dir),
		`52 8:27:.test.D.c.0 $(.test.foreach $1,&(.test.none)) {39:16:delegate {37:15:def .test.foreach} {=list {39:32:delegate {37:19:auto 1}}} {=list {39:35:closure {=compound {39:37:punct .} {39:38:word test} {39:42:punct .} {39:43:word none}}}}}`:sfmt(`{%[1]s/value/4/do.smart:37:15:def .test.foreach} ({}) {%[1]s/value/4/do.smart:39:16 {=group {%[1]s/value/4/do.smart:37:18 {%[1]s/value/4/do.smart:39:32:null}}}}`,testdata_dir),
		`52 8:27:.test.D.c.0 $(foreach $1,&(.test.x.$_)) {41:16:delegate {41:18:builtin foreach} {=list {41:26:delegate {37:19:auto 1}}} {=list {41:29:closure {=compound {41:31:punct .} {41:32:word test} {41:36:punct .} {41:37:word x} {41:38:punct .} {41:39:delegate {41:40:auto _}}}}}}`:sfmt(`{%[1]s/value/4/do.smart:41:18:builtin foreach} {} {%[1]s/value/4/do.smart:41:16 {%[1]s/value/4/do.smart:41:18:null}}`,testdata_dir),
		`52 8:27:.test.D.c.0 $1 {37:18:delegate {37:19:auto 1}}`:[]string{`{37:15:def 1} {} {37:18 {39:32:null}}`},
		`52 8:27:.test.D.c.0 $1 {39:32:delegate {37:19:auto 1}}`:[]string{sfmt(`{} {} {%[1]s/value/4/do.smart:39:32:null}`,testdata_dir)},
		`52 8:27:.test.D.c.0 $1 {39:51:delegate {37:19:auto 1}}`:[]string{sfmt(`{} {} {%[1]s/value/4/do.smart:39:51:null}`,testdata_dir)},
		`52 8:27:.test.D.c.0 $1 {41:26:delegate {37:19:auto 1}}`:[]string{`{} {} {41:26:null}`},
		`52 8:27:.test.D.c.0 $1 {41:45:delegate {37:19:auto 1}}`:[]string{sfmt(`{} {} {%[1]s/value/4/do.smart:41:45:null}`,testdata_dir)},
		`76 9:26:.test.D.c.1 &(value .test.v) {26:16:closure {26:18:builtin value} {=list {26:24 {=compound {23:12:punct .} {23:13:word test} {23:17:punct .} {23:18:word v}}}}}`:sfmt(`{%[1]s/value/4/do.smart:26:18:builtin value} xx {%[1]s/value/4/do.smart:26:16 {%[1]s/value/4/do.smart:22:12:word xx}}`,testdata_dir),
		`128 8:27:.test.I.c.0 &(.test.x) {28:24:closure {23:9:def .test.x}}`:`{23:9:def .test.x} .test.v {28:24 {=compound {23:12:punct .} {23:13:word test} {23:17:punct .} {23:18:word v}}}`,
		`128 8:27:.test.I.c.0 $(.test.x) {30:24:delegate {23:9:def .test.x}}`:`{23:9:def .test.x} .test.v {30:24 {=compound {23:12:punct .} {23:13:word test} {23:17:punct .} {23:18:word v}}}`,
		`128 8:27:.test.I.c.0 &(.test.x) {32:24:closure {23:9:def .test.x}}`:`{23:9:def .test.x} .test.v {32:24 {=compound {23:12:punct .} {23:13:word test} {23:17:punct .} {23:18:word v}}}`,
		`128 8:27:.test.I.c.0 $(.test.x) {34:24:delegate {23:9:def .test.x}}`:`{23:9:def .test.x} .test.v {34:24 {=compound {23:12:punct .} {23:13:word test} {23:17:punct .} {23:18:word v}}}`,
		`128 8:27:.test.I.c.0 &(value &(.test.x)) {28:16:closure {28:18:builtin value} {=list {28:24:closure {23:9:def .test.x}}}}`:sfmt(`{%[1]s/value/4/do.smart:28:18:builtin value} xx {%[1]s/value/4/do.smart:28:16 {%[1]s/value/4/do.smart:22:12:word xx}}`,testdata_dir),
		`128 8:27:.test.I.c.0 &(value $(.test.x)) {30:16:closure {30:18:builtin value} {=list {30:24:delegate {23:9:def .test.x}}}}`:sfmt(`{%[1]s/value/4/do.smart:30:18:builtin value} xx {%[1]s/value/4/do.smart:30:16 {%[1]s/value/4/do.smart:22:12:word xx}}`,testdata_dir),
		`128 8:27:.test.I.c.0 $(value &(.test.x)) {32:16:delegate {32:18:builtin value} {=list {32:24:closure {23:9:def .test.x}}}}`:sfmt(`{%[1]s/value/4/do.smart:32:18:builtin value} xx {%[1]s/value/4/do.smart:32:16 {%[1]s/value/4/do.smart:22:12:word xx}}`,testdata_dir),
		`128 8:27:.test.I.c.0 $(value $(.test.x)) {34:16:delegate {34:18:builtin value} {=list {34:24:delegate {23:9:def .test.x}}}}`:sfmt(`{%[1]s/value/4/do.smart:34:18:builtin value} xx {%[1]s/value/4/do.smart:34:16 {%[1]s/value/4/do.smart:22:12:word xx}}`,testdata_dir),
		`152 9:26:.test.I.c.1 &(.test.x) {29:24:closure {23:9:def .test.x}}`:`{23:9:def .test.x} .test.v {29:24 {=compound {23:12:punct .} {23:13:word test} {23:17:punct .} {23:18:word v}}}`,
		`152 9:26:.test.I.c.1 &(value &(.test.x)) {29:16:closure {29:18:builtin value} {=list {29:24:closure {23:9:def .test.x}}}}`:sfmt(`{%[1]s/value/4/do.smart:29:18:builtin value} xx {%[1]s/value/4/do.smart:29:16 {%[1]s/value/4/do.smart:22:12:word xx}}`,testdata_dir),
		`152 9:26:.test.I.c.1 &(value .test.v) {31:16:closure {31:18:builtin value} {=list {31:24 {=compound {23:12:punct .} {23:13:word test} {23:17:punct .} {23:18:word v}}}}}`:sfmt(`{%[1]s/value/4/do.smart:31:18:builtin value} xx {%[1]s/value/4/do.smart:31:16 {%[1]s/value/4/do.smart:22:12:word xx}}`,testdata_dir),
	},
}
