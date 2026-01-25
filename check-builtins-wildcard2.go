//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints__wildcard2 = map[string]map[string]any{
	"loader.go": map[string]any{
		`x.h {=compound {4:32:word x} {4:33:punct .} {4:34:word h}}`:`x.h {=compound {4:32:word x} {4:33:punct .} {4:34:word h}}`,
		`$/ {4:45:delegate {1:1:def /}}`:testdata_f(`%[1]s/builtins/wildcard/2 {4:45 {=path %[3]s {1:1:word builtins} {1:1:word wildcard} {1:1:word 2}}}`),
		`$(dir $/) {4:39:delegate {4:41:builtin dir} {=list {4:45:delegate {1:1:def /}}}}`:testdata_f(`%[1]s/builtins/wildcard {4:39 {=path %[3]s {4:41:word builtins} {4:41:word wildcard}}}`,line_column_s("4:41")),
		`f*? {=compound {4:28:word f} {=glob {4:29:meta *?}}}`:`f*? {=compound {4:28:word f} {=glob {4:29:meta *?}}}`,
	},
}

var checkstrs__wildcard2 = map[string]map[string]any{
	"loader.go": map[string]any{
		`0:0: $/ {4:45:delegate {1:1:def /}}`:testdata_f(`{4:45 {=path %[3]s {1:1:word builtins} {1:1:word wildcard} {1:1:word 2}}} %[1]s/builtins/wildcard/2`),
	},
}
