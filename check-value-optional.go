//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

import (
	"fmt"
)

var checkpoints_value_optional_foo = map[string]map[string]any{
	"loader.go": map[string]any{
		`1120 3:6:name $({=self foo}) {3:9:delegate {1:8:self foo}}`:`{1:8:self foo} {=self foo} {3:9 {1:8:self foo}}`,
	},
}

var checkpoints_value_optional = map[string]map[string]any{
	"loader.go": map[string]any{
		`1120 19:7:val0 $({=project foo}) {19:10:delegate {1:8:project foo}}`:sfmt(`{%[1]s/value/optional/foo/do.smart:1:8:project foo} {=project foo} {19:10 {%[1]s/value/optional/foo/do.smart:1:8:project foo}}`,testdata_dir),
		`1120 20:7:val1 $(name?) {20:10:delegate {=cond {20:12:word name}}}`:`{20:12:word name} {} {20:10:null}`,
		`1120 21:7:val2 $({=project foo}→name?) {21:10:delegate {=cond {=arrow {1:8:project foo}→{21:18:word name}}}}`:fmt.Sprintf(`{21:12 {%[1]s/value/optional/foo/do.smart:3:6:def name}} {=self foo} {21:10 {%[1]s/value/optional/foo/do.smart:3:9 {%[1]s/value/optional/foo/do.smart:1:8:self foo}}}`,testdata_dir),
		`1120 22:7:val3 $({=project foo}→baz?) {22:10:delegate {=cond {=arrow {1:8:project foo}→{22:18:word baz}}}}`:`{22:12:null} {} {22:10:null}`,
		`1120 23:7:val4 $(fo?→bar) {23:10:delegate {=arrow {=cond {23:12:word fo}}→{23:18:word bar}}}`:`{23:12:null} {} {23:10:null}`,
		`1120 24:7:val5 $(fo?→bar?) {24:10:delegate {=cond {=arrow {=cond {24:12:word fo}}→{24:18:word bar}}}}`:`{24:12:null} {} {24:10:null}`,
		`1120 25:7:val6 $({=project foo}) {25:20:delegate {1:8:project foo}}`:sfmt(`{%[1]s/value/optional/foo/do.smart:1:8:project foo} {=project foo} {25:20 {%[1]s/value/optional/foo/do.smart:1:8:project foo}}`,testdata_dir),
		`1120 25:7:val6 $_ {25:29:delegate {11:27:auto _}}`:sfmt(`{25:12:def _} {=project foo} {25:29 {25:20 {%[1]s/value/optional/foo/do.smart:1:8:project foo}}}`,testdata_dir),
		`1120 25:7:val6 $($_→name) {25:27:delegate {=arrow {25:29:delegate {11:27:auto _}}→{25:34:word name}}}`:sfmt(`{25:29 {%[1]s/value/optional/foo/do.smart:3:6:def name}} {=self foo} {25:27 {%[1]s/value/optional/foo/do.smart:3:9 {%[1]s/value/optional/foo/do.smart:1:8:self foo}}}`,testdata_dir),
		`1120 25:7:val6 $(foreach $({=project foo}),$($_→name)) {25:10:delegate {25:12:builtin foreach} {=list {25:20:delegate {1:8:project foo}}} {=list {25:27:delegate {=arrow {25:29:delegate {11:27:auto _}}→{25:34:word name}}}}}`:fmt.Sprintf(`{25:12:builtin foreach} {=self foo} {25:10 {25:27 {%[1]s/value/optional/foo/do.smart:3:9 {%[1]s/value/optional/foo/do.smart:1:8:self foo}}}}`,testdata_dir),
		`1120 26:7:val7 $({=project foo}) {26:20:delegate {1:8:project foo}}`:sfmt(`{%[1]s/value/optional/foo/do.smart:1:8:project foo} {=project foo} {26:20 {%[1]s/value/optional/foo/do.smart:1:8:project foo}}`,testdata_dir),
		`1120 26:7:val7 $_ {26:29:delegate {11:27:auto _}}`:sfmt(`{26:12:def _} {=project foo} {26:29 {26:20 {%[1]s/value/optional/foo/do.smart:1:8:project foo}}}`,testdata_dir),
		`1120 26:7:val7 $($_→name?) {26:27:delegate {=cond {=arrow {26:29:delegate {11:27:auto _}}→{26:34:word name}}}}`:fmt.Sprintf(`{26:29 {%[1]s/value/optional/foo/do.smart:3:6:def name}} {=self foo} {26:27 {%[1]s/value/optional/foo/do.smart:3:9 {%[1]s/value/optional/foo/do.smart:1:8:self foo}}}`,testdata_dir),
		`1120 26:7:val7 $(foreach $({=project foo}),$($_→name?)) {26:10:delegate {26:12:builtin foreach} {=list {26:20:delegate {1:8:project foo}}} {=list {26:27:delegate {=cond {=arrow {26:29:delegate {11:27:auto _}}→{26:34:word name}}}}}}`:fmt.Sprintf(`{26:12:builtin foreach} {=self foo} {26:10 {26:27 {%[1]s/value/optional/foo/do.smart:3:9 {%[1]s/value/optional/foo/do.smart:1:8:self foo}}}}`,testdata_dir),
		`1120 27:7:val8 $({=project foo}) {27:20:delegate {1:8:project foo}}`:sfmt(`{%[1]s/value/optional/foo/do.smart:1:8:project foo} {=project foo} {27:20 {%[1]s/value/optional/foo/do.smart:1:8:project foo}}`,testdata_dir),
		`1120 27:7:val8 $_ {27:29:delegate {11:27:auto _}}`:sfmt(`{27:12:def _} {=project foo} {27:29 {27:20 {%[1]s/value/optional/foo/do.smart:1:8:project foo}}}`,testdata_dir),
		`1120 27:7:val8 $($_→bar?) {27:27:delegate {=cond {=arrow {27:29:delegate {11:27:auto _}}→{27:34:word bar}}}}`:`{27:29:null} {} {27:27:null}`,
		`1120 27:7:val8 $(foreach $({=project foo}),$($_→bar?)) {27:10:delegate {27:12:builtin foreach} {=list {27:20:delegate {1:8:project foo}}} {=list {27:27:delegate {=cond {=arrow {27:29:delegate {11:27:auto _}}→{27:34:word bar}}}}}}`:`{27:12:builtin foreach} {} {27:10 {27:12:null}}`,
		`1120 28:7:val9 $({=project foo}→name→xxxx?) {28:10:delegate {=cond {=arrow {=arrow {1:8:project foo}→{28:18:word name}}→{28:25:word xxxx}}}}`:`{28:12:null} {} {28:10:null}`,
		`1120 29:7:val10 $({=project foo}→name→item?) {29:10:delegate {=cond {=arrow {=arrow {1:8:project foo}→{29:18:word name}}→{29:25:word item}}}}`:sfmt(`{29:12 {%[1]s/value/optional/foo/do.smart:4:6:def item}} {=yes} {29:10 {%[1]s/value/optional/foo/do.smart:4:14:yes}}`,testdata_dir),
		`1120 30:7:val11 $({=project foo}→name?→item?) {30:10:delegate {=cond {=arrow {=cond {=arrow {1:8:project foo}→{30:18:word name}}}→{30:26:word item}}}}`:sfmt(`{30:12 {%[1]s/value/optional/foo/do.smart:4:6:def item}} {=yes} {30:10 {%[1]s/value/optional/foo/do.smart:4:14:yes}}`,testdata_dir),
		`1120 31:7:val12 $(foo?→name?→item?) {31:10:delegate {=cond {=arrow {=cond {=arrow {=cond {31:12:word foo}}→{31:19:word name}}}→{31:27:word item}}}}`:sfmt(`{31:12 {%[1]s/value/optional/foo/do.smart:4:6:def item}} {=yes} {31:10 {%[1]s/value/optional/foo/do.smart:4:14:yes}}`,testdata_dir),
	},
}
