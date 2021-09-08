//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

// This file sets up the global scope and the root project/module.

package smart

import (
        "runtime"
        "strconv"
        "sync"
        "time"
        "fmt"
        "os"
)

const maxNumVarVal = 9

var (
        _context defaultContext
	universe *Scope
)

func defineUniverseBuiltins(ctx Context) {
        for name, f := range builtins {
                if v, alt := universe.Builtin(ctx, name, f); alt != nil {
                        panic(fmt.Sprintf("builtin '%s' already defined", name))
                } else {
                        v.flag |= builtinFunction
                }
        }
        for name, f := range commands {
                if v, alt := universe.Builtin(ctx, name, f); alt != nil {
                        panic(fmt.Sprintf("builtin '%s' already defined (command)", name))
                } else {
                        v.flag |= builtinCommand
                }
        }
}

func init() {
        var err error
        if _context.workdir, err = os.Getwd(); err != nil {
                diag.errorAt(_context.Position(), "%v", err).debug(6)
                return
        }

        var (
                ctx Context = &_context
                pos Position = ctx.Position()
                bin = MakeString(pos, os.Args[0])
                args = MakeList(pos)
        )
        for _, a := range os.Args[1:] {
                args.Elems = append(args.Elems, MakeString(pos, a))
        }

        universe = NewScope(pos, nil, nil, "universe")
        _, _ = universe.define(ctx, nil, "SMART.ARGS", args)
        _, _ = universe.define(ctx, nil, "SMART.BIN", bin)
        _, _ = universe.define(ctx, nil, "SMART", bin)
        
        defineUniverseBuiltins(ctx)
}

// IsUniverse checks if the scope is universe.
func IsUniverse(scope *Scope) bool { return scope == universe }

// A Globe represents a global execution context. 
type Globe struct {
        scope  *Scope
	os     *Project
        main   *Project
        _timestamps map[string]time.Time
        _timestampx *sync.Mutex
}

// Scope returns the globe scope.
func (g *Globe) Scope() *Scope { return g.scope }

// Main returns the main project.
func (g *Globe) Main() *Project { return g.main }

func (g *Globe) SetScopeOuter(scope *Scope) { scope.outer = g.scope }

func (g *Globe) timestamp(s string) (t time.Time) {
  g._timestampx.Lock(); defer g._timestampx.Unlock()
  t, _ = g._timestamps[s]
  return
}

func (g *Globe) stamp(s string, t time.Time) {
  g._timestampx.Lock(); defer g._timestampx.Unlock()
  g._timestamps[s] = t
}

// project returns a new Project for the given project path and name;
// the name must not be the blank identifier.
// The project is not complete and contains no explicit imports.
func (g *Globe) project(ctx Context, outer *Scope, absPath, relPath, tmpPath, spec, name string) (m *Project) {
        if outer == nil { outer = g.scope }

        var pos = ctx.Position()
	m = &Project{
                position: pos,
                absPath: absPath,
                relPath: relPath, 
                tmpPath: tmpPath,
                using: new(usinglist),
                self: new(ProjectName),
                spec: spec,
                name: name,
        }

        m.scope = NewScope(pos, outer, m, fmt.Sprintf("project %q", name))
        m.self.name = name
        m.self.scope = m.scope
        m.self.owner = m
        m.self.project = m
        m.using.name = "usee"
        m.using.scope = m.scope
        m.using.owner = m

        if g.main == nil && spec != "" && name != "@" && name != "~" {
                for outer != nil && outer != g.scope {
                        if p := outer.project; p != nil && p.Name() == "@" {
                                return
                        }
                        outer = outer.outer
                }

                g.main = m

                var none = &None{valbase{pos}}

                def, _ := g.scope.define(ctx, m, "_", none)
                if enable_assertions { assert(def != nil, "'$_' is nil") }

                for i := 1; i <= maxNumVarVal; i += 1 {
                        def, _ := g.scope.define(ctx, m, strconv.Itoa(i), none)
                        if enable_assertions { assert(def != nil, "'$%d' is nil", i) }
                }
        }
        return
}

// NewGlobe creates a new Globe context.
func NewGlobe(ctx Context, name string) (g *Globe) {
        var pos = ctx.Position()
        g = &Globe{
                scope: NewScope(pos, universe, nil, fmt.Sprintf("globe %q", name)),
                _timestamps: make(map[string]time.Time),
                _timestampx: new(sync.Mutex),
        }

        var absPath, relPath, tmpPath, spec string
        // TODO: determines absPath, relPath, tmpPath, spec
        g.os = g.project(ctx, nil, absPath, relPath, tmpPath, spec, runtime.GOOS)
        //g.os.scope.define(g.os, "name", &None{})
        return g
}
