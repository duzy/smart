//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints_valcache1 = map[string]map[string]any{
	"loader.go": map[string]any{
		`foo.c {=compound {4:6:word foo} {4:9:punct .} {4:10:word c}}`:`foo.c {=compound {4:6:word foo} {4:9:punct .} {4:10:word c}}`,
	},
	"check-valcache1_test.go": map[string]any{
	},
}

var checkstrs_valcache1 = map[string]map[string]any{
	"loader.go": map[string]any{
	},
	"check-valcache1_test.go": map[string]any{
	},
}
