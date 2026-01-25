//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints__xor = map[string]map[string]any{
	"loader.go": map[string]any{
		`val14.1 {=compound {3:1:word val14} {3:6:punct .} {3:7:decimal 1}}`:`val14.1 {=compound {3:1:word val14} {3:6:punct .} {3:7:decimal 1}}`,
		`val14.2 {=compound {4:1:word val14} {4:6:punct .} {4:7:decimal 2}}`:`val14.2 {=compound {4:1:word val14} {4:6:punct .} {4:7:decimal 2}}`,
		`val14.3 {=compound {5:1:word val14} {5:6:punct .} {5:7:decimal 3}}`:`val14.3 {=compound {5:1:word val14} {5:6:punct .} {5:7:decimal 3}}`,
		`val14.4 {=compound {6:1:word val14} {6:6:punct .} {6:7:decimal 4}}`:`val14.4 {=compound {6:1:word val14} {6:6:punct .} {6:7:decimal 4}}`,
		`3:9:val14.1 $(xor {=true},{=true}) {3:12:delegate {3:14:builtin xor} {=list {3:24:true}} {=list {3:33:true}}}`:`{} {3:12:null}`,
		`4:9:val14.2 $(xor {=true},{=false}) {4:12:delegate {4:14:builtin xor} {=list {4:24:true}} {=list {4:34:false}}}`:`{=true} {4:12 {4:34:true}}`,
		`5:9:val14.3 $(xor {=false},{=true}) {5:12:delegate {5:14:builtin xor} {=list {5:25:false}} {=list {5:33:true}}}`:`{=true} {5:12 {5:33:true}}`,
		`6:9:val14.4 $(xor {=false},{=false}) {6:12:delegate {6:14:builtin xor} {=list {6:25:false}} {=list {6:34:false}}}`:`{} {6:12:null}`,
	},
	"check-builtins-xor_test.go": map[string]any{
	},
}

var checkstrs__xor = map[string]map[string]any{
	"check-builtins-xor_test.go": map[string]any{
	},
}
