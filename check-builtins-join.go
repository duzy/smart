//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints__join = map[string]map[string]any{
	"loader.go": map[string]any{
		`target.arch {=compound {8:1:word target} {8:7:punct .} {8:8:word arch}}`:`target.arch {=compound {8:1:word target} {8:7:punct .} {8:8:word arch}}`,
		`target.vendor {=compound {9:1:word target} {9:7:punct .} {9:8:word vendor}}`:`target.vendor {=compound {9:1:word target} {9:7:punct .} {9:8:word vendor}}`,
		`target.os {=compound {10:1:word target} {10:7:punct .} {10:8:word os}}`:`target.os {=compound {10:1:word target} {10:7:punct .} {10:8:word os}}`,
		`target.abi {=compound {11:1:word target} {11:7:punct .} {11:8:word abi}}`:`target.abi {=compound {11:1:word target} {11:7:punct .} {11:8:word abi}}`,
		`3:6:val1 $(join foo bar xx yy zz,-) {3:9:delegate {3:11:builtin join} {=list {3:16:word foo} {3:20:word bar} {3:24:word xx} {3:27:word yy} {3:30:word zz}} {=list {=flag {3:34}}}}`:`foo-bar-xx-yy-zz {3:9 {=compound {3:16:word foo} {=flag {3:34}} {3:20:word bar} {=flag {3:34}} {3:24:word xx} {=flag {3:34}} {3:27:word yy} {=flag {3:34}} {3:30:word zz}}}`,
		`5:6:val3 $(join &(target.arch) &(target.vendor) &(target.os) &(target.abi),-) {5:9:delegate {5:11:builtin join} {=list {5:16:closure {=compound {5:18:word target} {5:24:punct .} {5:25:word arch}}} {5:31:closure {=compound {5:33:word target} {5:39:punct .} {5:40:word vendor}}} {5:48:closure {=compound {5:50:word target} {5:56:punct .} {5:57:word os}}} {5:61:closure {=compound {5:63:word target} {5:69:punct .} {5:70:word abi}}}} {=list {=flag {5:76}}}}`:`&(target.arch)-&(target.vendor)-&(target.os)-&(target.abi) {5:9 {=compound {5:16:closure {=compound {5:18:word target} {5:24:punct .} {5:25:word arch}}} {=flag {5:76}} {5:31:closure {=compound {5:33:word target} {5:39:punct .} {5:40:word vendor}}} {=flag {5:76}} {5:48:closure {=compound {5:50:word target} {5:56:punct .} {5:57:word os}}} {=flag {5:76}} {5:61:closure {=compound {5:63:word target} {5:69:punct .} {5:70:word abi}}}}}`,
		`6:6:val4 $(conjunct &(target.arch) &(XXX) &(target.vendor) &(target.os) &(target.abi),-) {6:9:delegate {6:11:builtin conjunct} {=list {6:20:closure {=compound {6:22:word target} {6:28:punct .} {6:29:word arch}}} {6:35:closure {6:37:word XXX}} {6:42:closure {=compound {6:44:word target} {6:50:punct .} {6:51:word vendor}}} {6:59:closure {=compound {6:61:word target} {6:67:punct .} {6:68:word os}}} {6:72:closure {=compound {6:74:word target} {6:80:punct .} {6:81:word abi}}}} {=list {=flag {6:87}}}}`:`{{&(target.arch) &(XXX) &(target.vendor) &(target.os) &(target.abi)}-} {6:9 {=conjunction {=list {6:20:closure {=compound {6:22:word target} {6:28:punct .} {6:29:word arch}}} {6:35:closure {6:37:word XXX}} {6:42:closure {=compound {6:44:word target} {6:50:punct .} {6:51:word vendor}}} {6:59:closure {=compound {6:61:word target} {6:67:punct .} {6:68:word os}}} {6:72:closure {=compound {6:74:word target} {6:80:punct .} {6:81:word abi}}}}{=list {=flag {6:87}}}}}`,
	},
	"check-builtins-join_test.go": map[string]any{
		`26 4:7:val2 $(join foo bar xx yy zz,-) {4:9:delegate {4:11:builtin join} {=list {4:16:word foo} {4:20:word bar} {4:24:word xx} {4:27:word yy} {4:30:word zz}} {=list {=flag {4:34}}}}`:`foo-bar-xx-yy-zz {4:9 {=compound {4:16:word foo} {=flag {4:34}} {4:20:word bar} {=flag {4:34}} {4:24:word xx} {=flag {4:34}} {4:27:word yy} {=flag {4:34}} {4:30:word zz}}}`,

		`36 5:6:val3 target.arch {=compound {5:18:word target} {5:24:punct .} {5:25:word arch}}`:`target.arch {=compound {5:18:word target} {5:24:punct .} {5:25:word arch}}`,
		`36 5:6:val3 &(target.arch) {5:16:closure {=compound {5:18:word target} {5:24:punct .} {5:25:word arch}}}`:`foo {5:16 {8:18:word foo}}`,
		`36 5:6:val3 target.vendor {=compound {5:33:word target} {5:39:punct .} {5:40:word vendor}}`:`target.vendor {=compound {5:33:word target} {5:39:punct .} {5:40:word vendor}}`,
		`36 5:6:val3 &(target.vendor) {5:31:closure {=compound {5:33:word target} {5:39:punct .} {5:40:word vendor}}}`:`bar {5:31 {9:18:word bar}}`,
		`36 5:6:val3 target.os {=compound {5:50:word target} {5:56:punct .} {5:57:word os}}`:`target.os {=compound {5:50:word target} {5:56:punct .} {5:57:word os}}`,
		`36 5:6:val3 &(target.os) {5:48:closure {=compound {5:50:word target} {5:56:punct .} {5:57:word os}}}`:`{} {5:48:null}`,
		`36 5:6:val3 target.abi {=compound {5:63:word target} {5:69:punct .} {5:70:word abi}}`:`target.abi {=compound {5:63:word target} {5:69:punct .} {5:70:word abi}}`,
		`36 5:6:val3 &(target.abi) {5:61:closure {=compound {5:63:word target} {5:69:punct .} {5:70:word abi}}}`:`0 {5:61 {11:18:decimal 0}}`,

		`46 6:6:val4 target.arch {=compound {6:22:word target} {6:28:punct .} {6:29:word arch}}`:`target.arch {=compound {6:22:word target} {6:28:punct .} {6:29:word arch}}`,
		`46 6:6:val4 &(target.arch) {6:20:closure {=compound {6:22:word target} {6:28:punct .} {6:29:word arch}}}`:`foo {6:20 {8:18:word foo}}`,
		`46 6:6:val4 &(XXX) {6:35:closure {6:37:word XXX}}`:`{} {6:35:null}`,
		`46 6:6:val4 target.vendor {=compound {6:44:word target} {6:50:punct .} {6:51:word vendor}}`:`target.vendor {=compound {6:44:word target} {6:50:punct .} {6:51:word vendor}}`,
		`46 6:6:val4 &(target.vendor) {6:42:closure {=compound {6:44:word target} {6:50:punct .} {6:51:word vendor}}}`:`bar {6:42 {9:18:word bar}}`,
		`46 6:6:val4 target.os {=compound {6:61:word target} {6:67:punct .} {6:68:word os}}`:`target.os {=compound {6:61:word target} {6:67:punct .} {6:68:word os}}`,
		`46 6:6:val4 &(target.os) {6:59:closure {=compound {6:61:word target} {6:67:punct .} {6:68:word os}}}`:`{} {6:59:null}`,
		`46 6:6:val4 target.abi {=compound {6:74:word target} {6:80:punct .} {6:81:word abi}}`:`target.abi {=compound {6:74:word target} {6:80:punct .} {6:81:word abi}}`,
		`46 6:6:val4 &(target.abi) {6:72:closure {=compound {6:74:word target} {6:80:punct .} {6:81:word abi}}}`:`0 {6:72 {11:18:decimal 0}}`,
	},
}

var checkstrs__join = map[string]map[string]any{
	"check-builtins-join_test.go": map[string]any{
		`26 4:7:val2 $(join foo bar xx yy zz,-) {4:9:delegate {4:11:builtin join} {=list {4:16:word foo} {4:20:word bar} {4:24:word xx} {4:27:word yy} {4:30:word zz}} {=list {=flag {4:34}}}}`:`{4:9 {=compound {4:16:word foo} {=flag {4:34}} {4:20:word bar} {=flag {4:34}} {4:24:word xx} {=flag {4:34}} {4:27:word yy} {=flag {4:34}} {4:30:word zz}}} foo-bar-xx-yy-zz`,
		`46 6:6:val4 &(target.arch) {6:20:closure {=compound {6:22:word target} {6:28:punct .} {6:29:word arch}}}`:`{6:20 {8:18:word foo}} foo`,
		`46 6:6:val4 &(XXX) {6:35:closure {6:37:word XXX}}`:`{6:35:null} `,
		`46 6:6:val4 &(target.vendor) {6:42:closure {=compound {6:44:word target} {6:50:punct .} {6:51:word vendor}}}`:`{6:42 {9:18:word bar}} bar`,
		`46 6:6:val4 &(target.os) {6:59:closure {=compound {6:61:word target} {6:67:punct .} {6:68:word os}}}`:`{6:59:null} `,
		`46 6:6:val4 &(target.abi) {6:72:closure {=compound {6:74:word target} {6:80:punct .} {6:81:word abi}}}`:`{6:72 {11:18:decimal 0}} 0`,
	},
}
