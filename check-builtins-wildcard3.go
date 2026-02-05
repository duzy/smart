//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints__wildcard3 = map[string]map[string]any{
	"loader.go": map[string]any{
		`$/ {4:45:delegate {1:1:def /}}`:testdata_f(`%[1]s/builtins/wildcard/3 {4:45 {=path %[3]s {1:1:raw builtins} {1:1:raw wildcard} {1:1:raw 3}}}`),
		`$(dir $/) {4:39:delegate {4:41:builtin dir} {=list {4:45:delegate {1:1:def /}}}}`:testdata_f(`%[1]s/builtins/wildcard {4:39 {=path %[3]s {4:41:raw builtins} {4:41:raw wildcard}}}`,line_column_s("4:41")),
	},
}

var checkstrs__wildcard3 = map[string]map[string]any{
	"loader.go": map[string]any{
		`0:0: $/ {4:45:delegate {1:1:def /}}`:testdata_f(`{4:45 {=path %[3]s {1:1:raw builtins} {1:1:raw wildcard} {1:1:raw 3}}} %[1]s/builtins/wildcard/3`),
	},
}
