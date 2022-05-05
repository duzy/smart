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
        _context.Context = &_context // self context for diagnostic

        var err error
        if _context.workdir, err = os.Getwd(); err != nil {
                erro(&_context, "%v", err).debug(6)
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
        _, _ = universe.define(ctx, DefVoid, "SMART.ARGS", args)
        _, _ = universe.define(ctx, DefVoid, "SMART.BIN", bin)
        _, _ = universe.define(ctx, DefVoid, "SMART", bin)
        
        defineUniverseBuiltins(ctx)
}

// IsUniverse checks if the scope is universe.
func IsUniverse(scope *Scope) bool { return scope == universe }

// A Globe represents a global execution context. 
type Globe struct {
        scope  *Scope
	os     *Project
        main   *Project
        projects    map[string]*Project // all projects

        args        map[Value][]Value
        flagEntries map[string][]Entry
        flags []*Flag
        pairs []*Pair
        goals   *Def
        mode    *Def
}

// Scope returns the globe scope.
func (g *Globe) Scope() *Scope { return g.scope }

// Main returns the main project.
func (g *Globe) Main() *Project { return g.main }

func (g *Globe) SetScopeOuter(scope *Scope) { scope.outer = g.scope }

func (g *Globe) AddFlagEntry(name string, entry Entry) {
  flags, _ := g.flagEntries[name]
  flags     = append(flags, entry)
  g.flagEntries[name] = flags
  return
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
                        if p := outer.project; p != nil && p.name == "@" {
                                return
                        }
                        outer = outer.outer
                }
                var (
                        none = MakeNone(pos)
                        def, _ = g.scope.define(ctx, DefAuto, "_", none)
                )
                if enable_assertions { assert(def != nil, "'$_' is nil") }
                for i := 0; i <= maxNumVarVal; i += 1 {
                        def, _ := g.scope.define(ctx, DefAuto, strconv.Itoa(i), none)
                        if enable_assertions { assert(def != nil, "'$%d' is nil", i) }
                }
                g.main = m
        }
        return
}

// NewGlobe creates a new Globe context.
func NewGlobe(ctx Context, name string) (g *Globe) {
        var pos = ctx.Position()
        g = &Globe{
                scope: NewScope(pos, universe, nil, fmt.Sprintf("globe %q", name)),
                args: make(map[Value][]Value),
                flagEntries: make(map[string][]Entry),
                //_timestamps: make(map[string]time.Time),
                //_timestampx: new(sync.Mutex),
        }

        var absPath, relPath, tmpPath, spec string
        // TODO: determines absPath, relPath, tmpPath, spec
        g.os = g.project(ctx, nil, absPath, relPath, tmpPath, spec, runtime.GOOS)
        //g.os.scope.define(g.os, "name", &None{})
        return g
}
