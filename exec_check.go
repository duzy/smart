//
//  Copyright (C) 2012-2024, Duzy Chan <code@extbit.io>, all rights reserverd.
//
package smart

import (
	"regexp"
	"strings"
	"io/ioutil"
)

const (
	d_run_sh = false
	d_exec_recipe = false
)

var nv = struct{}{}

func sizeof_map(ctx *exec_ctx, os, sh string, sm []string) (_ map[int]map[string]struct{}) {
	switch os {
	case "darwin":
		return map[int]map[string]struct{}{
			0: map[string]struct{}{
				"__INT64":nv, "__INT64_T":nv, "CLOCKID_T":nv, "TIMER_T":nv,
			},
			1: map[string]struct{}{
				"_BOOL":nv, "BOOL":nv, "CHAR":nv, "SIGNED_CHAR":nv, "CONST_CHAR":nv,
				"SA_FAMILY_T":nv,
			},
			2: map[string]struct{}{
				"SHORT":nv, "MODE_T":nv, "NLINK_T":nv,
			},
			4: map[string]struct{}{
				"INT":nv, "UINT32_T":nv, "FLOAT":nv, "BLKSIZE_T":nv, "WCHAR_T":nv,
				"DEV_T":nv, "ATOMIC_INT":nv, "FSBLKCNT_T":nv, "FSFILCNT_T":nv, "GID_T":nv,
				"KEY_T":nv, "PID_T":nv, "ID_T":nv, "SOCKLEN_T":nv, "SUSECONDS_T":nv,
				"UID_T":nv, "USECONDS_T":nv,
			},
			8: map[string]struct{}{
				"LONG":nv, "LONG_DOUBLE":nv, "LONG_LONG":nv, "DOUBLE":nv,
				"UINT64_T":nv, "UINTPTR_T":nv, "INO_T":nv, "VOID_P":nv, "SIZE_T":nv,
				"PTRDIFF_T":nv, "BLKCNT_T":nv, "CLOCK_T":nv, "ATOMIC_UINTPTR_T":nv,
				"OFF_T":nv, "FPOS_T":nv, "PTHREAD_T":nv, "PTHREAD_KEY_T":nv, "SSIZE_T":nv,
				"TIME_T":nv, "VA_LIST":nv,
			},
			16: map[string]struct{}{
				"PTHREAD_CONDATTR_T":nv, "PTHREAD_MUTEXATTR_T":nv, "PTHREAD_ONCE_T":nv,
			},
			24: map[string]struct{}{
				"PTHREAD_RWLOCKATTR_T":nv,
			},
			48: map[string]struct{}{
				"PTHREAD_COND_T":nv,
			},
			64: map[string]struct{}{
				"PTHREAD_ATTR_T":nv, "PTHREAD_MUTEX_T":nv,
			},
			104: map[string]struct{}{
				"SIGINFO_T":nv,
			},
			200: map[string]struct{}{
				"PTHREAD_RWLOCK_T":nv,
			},
		}
	default:
		prompt(ctx, "%v\n", sh)
		errostack(ctx, 2, "%s, status=%d", sm[1], ctx.Status).trace()
	}
	return
}

func alignof_map(ctx *exec_ctx, os, sh string, sm []string) (_ map[int]map[string]struct{}) {
	switch os {
	case "darwin":
		return map[int]map[string]struct{}{
			0: map[string]struct{}{
				"__INT64":nv, "__INT64_T":nv, "CLOCKID_T":nv, "TIMER_T":nv,
			},
			1: map[string]struct{}{
				"_BOOL":nv, "BOOL":nv, "CHAR":nv, "SIGNED_CHAR":nv, "CONST_CHAR":nv,
				"SA_FAMILY_T":nv,
			},
			2: map[string]struct{}{
				"SHORT":nv, "MODE_T":nv, "NLINK_T":nv,
			},
			4: map[string]struct{}{
				"INT":nv, "UINT32_T":nv, "FLOAT":nv, "BLKSIZE_T":nv, "WCHAR_T":nv,
				"DEV_T":nv, "ATOMIC_INT":nv, "FSBLKCNT_T":nv, "FSFILCNT_T":nv, "GID_T":nv,
				"KEY_T":nv, "PID_T":nv, "ID_T":nv, "SOCKLEN_T":nv, "SUSECONDS_T":nv,
				"UID_T":nv, "USECONDS_T":nv,
			},
			8: map[string]struct{}{
				"LONG":nv, "LONG_DOUBLE":nv, "LONG_LONG":nv, "DOUBLE":nv, "VOID_P":nv,
				"UINT64_T":nv, "UINTPTR_T":nv, "INO_T":nv, "SIZE_T":nv, "SSIZE_T":nv,
				"PTRDIFF_T":nv, "BLKCNT_T":nv, "CLOCK_T":nv, "ATOMIC_UINTPTR_T":nv,
				"OFF_T":nv, "FPOS_T":nv, "PTHREAD_T":nv, "PTHREAD_KEY_T":nv,
				"TIME_T":nv, "VA_LIST":nv,
			},
		}
	default:
		prompt(ctx, "%v\n", sh)
		errostack(ctx, 2, "%s, status=%d", sm[1], ctx.Status).trace()
	}
	return
}

var rx_sizeof    = regexp.MustCompile(`^-sizeof-c\+*$`)
var rx_alignof   = regexp.MustCompile(`^-alignof-c\+*$`)
var rx_status    = regexp.MustCompile(`^-(?:command|program)-status$`)
var rx_fn_src    = regexp.MustCompile(`\.(?:log|h|c|c\+\+)$`)
var rx_fn_conf_x = regexp.MustCompile(`^(/.+?/\.configure/.+?)\.x$`)
var rx_fn_confag = regexp.MustCompile(`^/.+?/\.configure/.+?/ALIGNOF_([^.]+?)\.log$`)
var rx_fn_confsz = regexp.MustCompile(`^/.+?/\.configure/.+?/SIZEOF_([^.]+?)\.log$`)
var rx_fn_conftp = regexp.MustCompile(`^\.configure/type/(?:size/SIZEOF|align/ALIGNOF)_(.+?)\.c\+*\.x$`)
var rx_conf_exec = regexp.MustCompile(`^[^ ]+ -c (/.+?/\.configure/.+?\.c\+*)(?:\.x)?(?:\.log)?$`)
var rx_conf_sizf = regexp.MustCompile(`#define SIZE \(sizeof\((.+?)\)\)`)
var rx_conf_alif = regexp.MustCompile(`#define ALIGN \(alignof\((.+?)\)\)`)
var rx_conf_incl = regexp.MustCompile(`# *include +(.+)`)
func (ctx *exec_ctx) run_check(exe *execution) (err error) {
	var c Context = ctx
	var p = _project(c)
	if p.name == "configure.base" {
		var t1 = auto_get(c, "TYPE")
		var t = auto_get(c, "@")
		var l = auto_get(c, "<")
		var r = auto_get(c, ">")
		if l != r {
			d := _entry(c).destiny()
			errostack(c, 5, "%v %v %v %v", d, t, l, r).trace()
		}

		if x, y := l.(*file); y && rx_fn_src.MatchString(x.name) { c = pc(c, x.fullname()) }
		if x, y := t.(*file); y && rx_fn_src.MatchString(x.name) { c = pc(c, x.fullname()) }
		if d_run_sh {
			if _, y := t.(*file); y {
				d := _entry(c).destiny()
				prompt(c, "%s\n", ctx.sh)
				notestack(c, 3, "%v", d).debug(32)
			}
		}

		if f, y := t.(*file); y && t1 != nil {
			var dest = _entry(c).destiny().string(c)
			var tp = t1.string(c)
			var sh = ctx.sh.String()
			var fn = f.fullname()
			if  m1 := rx_fn_conftp.FindStringSubmatch(dest); len(m1) == 2 {
				sm := rx_fn_conf_x.FindStringSubmatch(fn)

				// c = pc(pc(c, sm[1]), fn+".log")

				t := strings.ToUpper(tp)
				t = strings.Replace(t, " ", "_",  -1)
				t = strings.Replace(t, "*", "_P", -1)
				if m1[1] != t {
					prompt(c, "%v\n", sh)
					errostack(c, 3, "%s != %s", t, m1[1]).trace()
				}

				if s, e := ioutil.ReadFile(sm[1]); e != nil {
					prompt(c, "%v\n", sh)
					errostack(c, 2, "%v", e).trace()
				} else if m2 := rx_conf_sizf.FindSubmatch(s); len(m2) == 2 {
					if t := string(m2[1]); tp != t {
						prompt(c, "%v\n", sh)
						errostack(c, 2, "%s != %s", tp, t).trace()
					}
				} else if m2 := rx_conf_alif.FindSubmatch(s); len(m2) == 2 {
					if t := string(m2[1]); tp != t {
						prompt(c, "%v\n", sh)
						errostack(c, 2, "%s != %s", tp, t).trace()
					}
				} else {
					prompt(c, "%v\n", sh)
					errostack(c, 2, "%s %s", tp, m2).trace()
				}
			} else if rx_sizeof.MatchString(dest) {
				sm := rx_fn_confsz.FindStringSubmatch(fn)
				ss := rx_conf_exec.FindStringSubmatch(sh)

				if len(ss) == 2 { c = pc(c, ss[1]) }
				if /* c = pc(c, fn) */; len(sm) == 2 {
					t := strings.ToUpper(tp)
					t  = strings.Replace(t, " ", "_",  -1)
					t  = strings.Replace(t, "*", "_P", -1)
					if sm[1] != t {
						prompt(c, "%v\n", sh)
						errostack(c, 2, "%s != %s", t, sm[1]).trace()
					}

					var chk = sizeof_map(ctx, "darwin", sh, sm)

					if x, y := chk[ctx.Status]; !y {
						prompt(c, "%v\n", sh)
						errostack(c, 2, "%s, status=%d", sm[1], ctx.Status).trace()
					} else if _, y := x[sm[1]]; !y {
						prompt(c, "%v\n", sh)
						errostack(c, 2, "%s, status=%d", sm[1], ctx.Status).trace()
					}
				} else {
					prompt(c, "%v\n", sh)
					errostack(c, 2, "%s", tp).trace()
				}
			} else if rx_alignof.MatchString(dest) {
				sm := rx_fn_confag.FindStringSubmatch(fn)
				ss := rx_conf_exec.FindStringSubmatch(sh)

				if len(ss) == 2 { c = pc(c, ss[1]) }
				if /* c = pc(c, fn) */; len(sm) == 2 {
					t := strings.ToUpper(tp)
					t  = strings.Replace(t, " ", "_",  -1)
					t  = strings.Replace(t, "*", "_P", -1)
					if sm[1] != t {
						prompt(c, "%v\n", sh)
						errostack(c, 2, "%s != %s", t, sm[1]).trace()
					}

					var chk = alignof_map(ctx, "darwin", sh, sm)

					if x, y := chk[ctx.Status]; !y {
						prompt(c, "%v\n", sh)
						errostack(c, 2, "%s, status=%d", sm[1], ctx.Status).trace()
					} else if _, y := x[sm[1]]; !y {
						prompt(c, "%v\n", sh)
						errostack(c, 2, "%s, status=%d", sm[1], ctx.Status).trace()
					}
				} else {
					prompt(c, "%v\n", sh)
					errostack(c, 2, "%s", tp).trace()
				}
			} else if rx_status.MatchString(dest) {
				prompt(c, "%v\n", ctx.sh)
				notestack(c, 2, "%v, status=%v", tp, ctx.Status).debug(2)
			}
		}
	}
	return
}

var rx_conf_inc_log = regexp.MustCompile(`^\.configure/(type(?:/(?:size|align))?|function|symbol|struct-member|var)/([^/]+?)\.log$`)
func (ctx *exec_ctx) exec_check(exe *execution, src *raw) {
	var c Context = ctx
	var p = _project(c)
	if p.name == "configure.base" {
		var t = auto_get(c, "@")
		var l = auto_get(c, "<")
		var r = auto_get(c, ">")
		if l != r {
			d := _entry(c).destiny()
			errostack(c, 5, "%v %v %v %v", d, t, l, r).trace()
		}

		if x, y := l.(*file); y && rx_fn_src.MatchString(x.name) { c = pc(c, x.fullname()) }
		if x, y := t.(*file); y && rx_fn_src.MatchString(x.name) { c = pc(c, x.fullname()) }
		if d_exec_recipe {
			if x, y := t.(*file); y && strings.HasSuffix(x.name, ".rev.log") {
				d := _entry(c).destiny()
				prompt(c, "%s\n", src)
				notestack(c, 3, "%v", d).debug(32)
			}
		}

		if x, y := t.(*file); y && rx_conf_inc_log.MatchString(x.name) {
			if x, y := l.(*file); y && strings.HasSuffix(x.name, ".c") {
				var name, incl string
				var v_name = auto_get(c, "NAME")
				var v_incl = auto_get(c, "INCLUDE")

				if v_name == nil {
					prompt(c, "%v\n", src)
					errostack(c, 2, "NAME is undefined").trace()
				} else if name = v_name.string(c); name == "" {
					prompt(c, "%v\n", src)
					errostack(c, 2, "NAME is empty").trace()
				}

				if v_incl == nil {
					prompt(c, "%v\n", src)
					errostack(c, 2, "%s: INCLUDE is undefined", name).trace()
				} else if incl = v_incl.string(c); incl == "" {
					prompt(c, "%v\n", src)
					errostack(c, 2, "%s: INCLUDE is empty: %v", name, v_incl).trace()
				}

				ninc := 0
				incs := make(map[string]struct{})
				for _, v := range merge(v_incl) { incs[v.string(ctx)] = struct{}{} }

				b, e := ioutil.ReadFile(x.fullname())
				if e != nil {
					prompt(c, "%v\n", src)
					errostack(c, 2, "%v", e).trace()
				}
				if sm := rx_conf_incl.FindAllStringSubmatch(string(b), -1); sm != nil {
					for _, m := range sm {
						if _, y := incs[m[1]]; y {
							ninc += 1
						} else {
							prompt(c, "%v\n", src)
							errostack(c, 2, "no %v", m[0]).trace()
						}
					}
				}
				if ninc != len(incs) {
					prompt(c, "%v\n", src)
					errostack(c, 2, "%v: %v	!= %v", name, ninc, len(incs)).trace()
				}
			}
		} else if y && strings.HasPrefix(x.name, ".configure/library/") && strings.HasSuffix(x.name, ".log") {
			inc := auto_get(c, "INCLUDE")
			notestack(c, 3, "%v %v", inc, p.configure.names()).debug()
		}

		if x, y := t.(*file); y && strings.HasPrefix(x.name, ".configure/") && strings.HasSuffix(x.name, ".x") {
			if s := l.String(); strings.Contains(s, "%") {
				errostack(c, 5, "%v %s", s, ts(l)).trace()
			}
			if s := r.String(); strings.Contains(s, "%") {
				errostack(c, 5, "%v %s", s, ts(r)).trace()
			}

			e := false
			rx := regexp.MustCompile(`\.configure/.*?%+\.[^ ]*`)
			if sm := rx.FindAllStringSubmatch(src.s, -1); sm != nil {
				for _, s := range sm { note(c, "%s", s[0]) }; e = true
			}
			if e {
				errostack(c, 5, "%v", x.name).trace()
			}
		}
	}
}

func (ctx *exec_buffer) check_line(line string, lnum int) {
	var p = _project(ctx)
	if p.name == "configure.base" {
		var t = auto_get(ctx, "@")
		if f, y := t.(*file); y {
			var s = t.String()
			if strings.HasPrefix(s, "{=file .configure/") && strings.HasSuffix(s, ".x}") {
				c := pc(ctx, f.fullname()+".log")
				rx := regexp.MustCompile(`[^:]+: error: no such file or directory: '\.configure/.*?%+\.[^ ]*'`)
				if sm := rx.FindStringSubmatch(line); sm != nil {
					prompt(c, "%v\n", s)
					errostack(c, 3, "%s", sm[0]).trace()
				}
			}
			switch dest := _entry(ctx).destiny().string(ctx); dest {
			case "-sizeof-c", "-sizeof-c++", "-alignof-c", "-alignof-c++", "-program-status", "-command-status":
				var sh = ctx.sh.String()
				if regexp.MustCompile(`^/.+?/bash .+ -c .+`).MatchString(sh) {
					c := pc(ctx, f.fullname(), lnum)
					rx := regexp.MustCompile(`^bash: -.+?: invalid option`)
					if sm := rx.FindStringSubmatch(line); sm != nil {
						prompt(c, "%v\n", sh)
						errostack(c, 3, "%s", sm[0]).trace()
					}
				}
			}
		}
	}
}
