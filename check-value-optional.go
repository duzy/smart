//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints_value_optional_foo = map[string]map[string]any{
	"loader.go": map[string]any{
		`3:6:name $({=self foo}) {3:9:delegate {1:8:self foo}}`:`{1:8:self foo} {=self foo} {3:9 {1:8:self foo}}`,
	},
}

var checkpoints_value_optional = map[string]map[string]any{
	"loader.go": map[string]any{
		`19:7:val0 $({=project foo}) {19:10:delegate {1:8:project foo}}`:testdata_fmt(`{%[1]s/value/optional/foo/do.smart:1:8:project foo} {=project foo} {19:10 {%[1]s/value/optional/foo/do.smart:1:8:project foo}}`),

		`20:7:val1 $(name?) {20:10:delegate {=glob {20:12:word name} {20:16:meta ?}}}`:`{=glob {20:12:word name} {20:16:meta ?}} {} {20:10:null}`,

		`21:7:val2 $({=project foo}→name?) {21:10:delegate {=arrow {1:8:project foo}→{=glob {21:16:word name} {21:20:meta ?}}}}`:testdata_fmt(`{21:12 {21:16 {%[1]s/value/optional/foo/do.smart:3:6:def name}}} {=self foo} {21:10 {%[1]s/value/optional/foo/do.smart:3:9 {%[1]s/value/optional/foo/do.smart:1:8:self foo}}}`),
		`22:7:val3 $({=project foo}→baz?) {22:10:delegate {=arrow {1:8:project foo}→{=glob {22:16:word baz} {22:19:meta ?}}}}`:`{22:12 {22:16:null}} {} {22:10:null}`,

		`23:7:val4 $(fo?→bar) {23:10:delegate {=arrow {=glob {23:12:word fo} {23:14:meta ?}}→{23:16:word bar}}}`:`{23:12 {23:16:null}} {} {23:10:null}`,
		`24:7:val5 $(fo?→bar?) {24:10:delegate {=arrow {=glob {24:12:word fo} {24:14:meta ?}}→{=glob {24:16:word bar} {24:19:meta ?}}}}`:`{24:12 {24:16:null}} {} {24:10:null}`,

		`25:7:val6 $({=project foo}) {25:20:delegate {1:8:project foo}}`:testdata_fmt(`{%[1]s/value/optional/foo/do.smart:1:8:project foo} {=project foo} {25:20 {%[1]s/value/optional/foo/do.smart:1:8:project foo}}`),
		`25:7:val6 $_ {25:29:delegate {11:27:auto _}}`:testdata_fmt(`{11:27:auto _} {=project foo} {25:29 {25:20 {%[1]s/value/optional/foo/do.smart:1:8:project foo}}}`),
		`25:7:val6 $($_→name) {25:27:delegate {=arrow {25:29:delegate {11:27:auto _}}→{25:32:word name}}}`:testdata_fmt(`{25:29 {25:32 {%[1]s/value/optional/foo/do.smart:3:6:def name}}} {=self foo} {25:27 {%[1]s/value/optional/foo/do.smart:3:9 {%[1]s/value/optional/foo/do.smart:1:8:self foo}}}`),
		`25:7:val6 $(foreach $({=project foo}),$($_→name)) {25:10:delegate {25:12:builtin foreach} {=list {25:20:delegate {1:8:project foo}}} {=list {25:27:delegate {=arrow {25:29:delegate {11:27:auto _}}→{25:32:word name}}}}}`:testdata_fmt(`{25:12:builtin foreach} {=self foo} {25:10 {25:27 {%[1]s/value/optional/foo/do.smart:3:9 {%[1]s/value/optional/foo/do.smart:1:8:self foo}}}}`),

		`26:7:val7 $({=project foo}) {26:20:delegate {1:8:project foo}}`:testdata_fmt(`{%[1]s/value/optional/foo/do.smart:1:8:project foo} {=project foo} {26:20 {%[1]s/value/optional/foo/do.smart:1:8:project foo}}`),
		`26:7:val7 $_ {26:29:delegate {11:27:auto _}}`:testdata_fmt(`{11:27:auto _} {=project foo} {26:29 {26:20 {%[1]s/value/optional/foo/do.smart:1:8:project foo}}}`),
		`26:7:val7 $($_→name?) {26:27:delegate {=arrow {26:29:delegate {11:27:auto _}}→{=glob {26:32:word name} {26:36:meta ?}}}}`:testdata_fmt(`{26:29 {26:32 {%[1]s/value/optional/foo/do.smart:3:6:def name}}} {=self foo} {26:27 {%[1]s/value/optional/foo/do.smart:3:9 {%[1]s/value/optional/foo/do.smart:1:8:self foo}}}`),
		`26:7:val7 $(foreach $({=project foo}),$($_→name?)) {26:10:delegate {26:12:builtin foreach} {=list {26:20:delegate {1:8:project foo}}} {=list {26:27:delegate {=arrow {26:29:delegate {11:27:auto _}}→{=glob {26:32:word name} {26:36:meta ?}}}}}}`:testdata_fmt(`{26:12:builtin foreach} {=self foo} {26:10 {26:27 {%[1]s/value/optional/foo/do.smart:3:9 {%[1]s/value/optional/foo/do.smart:1:8:self foo}}}}`),

		`27:7:val8 $({=project foo}) {27:20:delegate {1:8:project foo}}`:testdata_fmt(`{%[1]s/value/optional/foo/do.smart:1:8:project foo} {=project foo} {27:20 {%[1]s/value/optional/foo/do.smart:1:8:project foo}}`),
		`27:7:val8 $_ {27:29:delegate {11:27:auto _}}`:testdata_fmt(`{11:27:auto _} {=project foo} {27:29 {27:20 {%[1]s/value/optional/foo/do.smart:1:8:project foo}}}`),
		`27:7:val8 $($_→bar?) {27:27:delegate {=arrow {27:29:delegate {11:27:auto _}}→{=glob {27:32:word bar} {27:35:meta ?}}}}`:`{27:29 {27:32:null}} {} {27:27:null}`,
		`27:7:val8 $(foreach $({=project foo}),$($_→bar?)) {27:10:delegate {27:12:builtin foreach} {=list {27:20:delegate {1:8:project foo}}} {=list {27:27:delegate {=arrow {27:29:delegate {11:27:auto _}}→{=glob {27:32:word bar} {27:35:meta ?}}}}}}`:`{27:12:builtin foreach} {} {27:10 {27:12:null}}`,

		`28:7:val9 $({=project foo}→name→xxxx?) {28:10:delegate {=arrow {=arrow {1:8:project foo}→{28:16:word name}}→{=glob {28:21:word xxxx} {28:25:meta ?}}}}`:`{28:12 {28:21:null}} {} {28:10:null}`,

		`29:7:val10 $({=project foo}→name→item?) {29:10:delegate {=arrow {=arrow {1:8:project foo}→{29:16:word name}}→{=glob {29:21:word item} {29:25:meta ?}}}}`:testdata_fmt(`{29:12 {29:21 {%[1]s/value/optional/foo/do.smart:4:6:def item}}} {=yes} {29:10 {%[1]s/value/optional/foo/do.smart:4:14:yes}}`),

		`30:7:val11 $({=project foo}→name?→item?) {30:10:delegate {=arrow {=arrow {1:8:project foo}→{=glob {30:16:word name} {30:20:meta ?}}}→{=glob {30:22:word item} {30:26:meta ?}}}}`:testdata_fmt(`{30:12 {30:22 {%[1]s/value/optional/foo/do.smart:4:6:def item}}} {=yes} {30:10 {%[1]s/value/optional/foo/do.smart:4:14:yes}}`),

		`31:7:val12 $(foo?→name?→item?) {31:10:delegate {=arrow {=arrow {=glob {31:12:word foo} {31:15:meta ?}}→{=glob {31:17:word name} {31:21:meta ?}}}→{=glob {31:23:word item} {31:27:meta ?}}}}`:testdata_fmt(`{31:12 {31:23 {%[1]s/value/optional/foo/do.smart:4:6:def item}}} {=yes} {31:10 {%[1]s/value/optional/foo/do.smart:4:14:yes}}`),
	},
}

var checkstrs_value_optional_foo = map[string]map[string]any{
}

var checkstrs_value_optional = map[string]map[string]any{
}
