//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints__addsuffix = map[string]map[string]any{
	"check-builtins-addprefix_test.go": map[string]any{
	},
}

var checkstrs__addsuffix = map[string]map[string]any{
}

var prefix__addsuffix = map[string]map[string]string{
	"check-builtins-addprefix_test.go": map[string]string{
		`134 {8:28:null} {8:20:word foo}`:`foo {8:20:word foo}`,
	},
}

var suffix__addsuffix = map[string]map[string]string{
}
