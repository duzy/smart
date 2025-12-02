//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints_valcache = map[string]map[string]any{
	"loader.go": map[string]any{
		`.deps {=compound {4:17:punct .} {4:18:word deps}}`:`.deps {=compound {4:17:punct .} {4:18:word deps}}`,
		`.tmp {=compound {4:53:punct .} {4:54:word tmp}}`:`.tmp {=compound {4:53:punct .} {4:54:word tmp}}`,

		`&(gen) {4:40:closure {4:42:word gen}}`:`{4:42:word gen} &(gen) {4:40:closure {4:42:word gen}}`,

		`$/ {4:50:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/valcache {4:50 {=path %[3]s {1:1:word valcache}}}`),
	},
	"check-valcache_test.go": map[string]any{
	},
}

var checkstrs_valcache = map[string]map[string]any{
	"loader.go": map[string]any{
	},
	"check-valcache_test.go": map[string]any{
	},
}
