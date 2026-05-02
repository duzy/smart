//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build !checkpoints

package smart

import (
	"regexp"
)

const checkpoints = false

func check__string_com(_ Context, _ *compound, _ Value) {}
func check_prefix(_ Context, _ string, _, _ Value, _ *Value) {}
func check_string(_ Context, _ any) func(*Value, *string) { return nil }
func check_cache(_ Context, _ any, _ string, _ *valcache, _ []*valcache) {}
func check_uncache(_ Context, _ any, _ string, _ *valcache, _ []*valcache) {}
func check_unmap(_ *uncache_t, _ any, _ *valcache, _ []*valcache) {}
func check_match(_ Context, _, _ Value) func(*bool, *Value, *Value, *[]Value) { return nil }
func check_matchGlobScalar(_ Context, _, _ Value, _ bool) func(*bool, *Value, *Value, *[]Value) { return nil }
func check_matchCompComp(_ Context, _, _ []Value, _ bool) func(*bool, *[]Value, *[]Value, *int, *int) { return nil }
func check_matchGlobComp(_ Context, _, _ []Value, _ bool) func(*bool, *[]Value, *[]Value, *[]Value, *int, *int, *token) { return nil }
func check_matchGlobPath(_ Context, _, _ []Value, _ bool) func(*bool, *[]Value, *[]Value, *[]Value, *int, *int, *token) { return nil }
func check_matchPathPath(_ Context, _, _ []Value, _ bool) func(*bool, *[]Value, *[]Value, *[]Value, *int, *int) { return nil }
func check_cmp(_ Context, _, _ any) func(*cmpres) { return nil }
func check_com(_ *comctx, _, _ []Value, _ *[]Value) {}
func check(_ Context, _, _ Value, _ ...Value) {}

func (*plainint) evaluate_check(ctx Context, args, recipes []Value, p *plain) {}
func (*execution) evaluate_check(ctx Context, i interpreter, args []Value, res Value) {}
func (*exec_buffer) check_line(line string, lnum int) {}
func (*exec_ctx) run_check(exe *execution) (err error) { return nil }
func (*exec_ctx) sources_check(cc Context, i int, rv Value, s string) {}
func (*exec_ctx) exec_check(i int, src *raw, e error) {}
func (*program) execute_check(ctx *execution, result *Value) {}
func (*program) execute_check_0(ctx *execution) {}
func (*program) execute_check_1(ctx *execution) {}
func (*configurecontext) execute_check(ctx *execution, e entry, p *project, s *string, _d **def) {}

func (*modifier_updatefile) x_check(_ Value, _, _ string, _ []Value, _ any) {}

func (*__trimprefix) check(_, _, _ Value) {}
func (*__grep) check(_ *regexp.Regexp, _ string, _, _ Value) {}

func (p *compiler) check_ident(_ Context, _ string, _ Symbol) {}
func (p *compiler) configure_val_check(_ *execution, _ Symbol, _ Value, _ []Value, _, _ *diagpoint) {}

func tempdir_check(ctx Context, p *project, d *def, s string) {}
func tempfile_check(ctx Context, p *project, name, d string, f *file) {}
