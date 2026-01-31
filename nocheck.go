//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build !checkpoints

package smart

// NOTE: cannot decalre `checkpoints` as `const` because it's compile-time evaled.
var checkpoints = false

func check__string_com(_ Context, _ *compound, _ Value) {}
func check_prefix(_ Context, _ string, _, _ Value, _ *Value) {}
func check_string(_ Context, _ Value, _ Value, _ string) {}
func check_cmp(_ Context, _, _ Value, _ *cmpres) {}
func check_com(_ *comctx, _, _ []Value, _ *[]Value) {}
func check_match(_ Context, _, _ any, _ *bool, _ *any, _ *[]string) {}
func check(_ Context, _, _ Value, _ *Value) {}
