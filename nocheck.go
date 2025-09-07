//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build !checkpoints

package smart

// NOTE: cannot decalre `checkpoints` as `const` because it's compile-time evaled.
var checkpoints = false

func check_pre_suf(_ Context, _ string, _, _ Value, _ *Value) {}
func check_cmp(_ Context, _, _ Value, _ *cmpres) {}
func check(_ Context, _, _ Value, _ *Value) {}
