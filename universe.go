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
	universe *Scope
)

func defUniverseBuiltins() {
        for name, f := range builtins {
                if _, alt := universe.builtin(name, f); alt != nil {
                        panic(fmt.Sprintf("builtin '%s' already defined", name))
                }
        }
}

func init() {
        universe = NewScope(nil, nil, "universe")

        var pos Position
        bin, args := &String{trivial{pos},os.Args[0]}, new(List)
        for _, a := range os.Args[1:] {
                args.Elems = append(args.Elems, &String{trivial{pos},a})
        }
        _, _ = universe.define(nil, "SMART.BIN", bin)
        _, _ = universe.define(nil, "SMART.ARGS", args)
        _, _ = universe.define(nil, "SMART", bin)
        
        defUniverseBuiltins()
}

// IsUniverse checks if the scope is universe.
func IsUniverse(scope *Scope) bool {
        return scope == universe
}

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

func (g *Globe) SetScopeOuter(scope *Scope) {
        scope.outer = g.scope
}

func (g *Globe) timestamp(s string) (t time.Time) {
        g._timestampx.Lock()
        t, _ = g._timestamps[s]
        g._timestampx.Unlock()
        return
}

func (g *Globe) stamp(s string, t time.Time) {
        g._timestampx.Lock()
        g._timestamps[s] = t
        g._timestampx.Unlock()
}

// project returns a new Project for the given project path and name;
// the name must not be the blank identifier.
// The project is not complete and contains no explicit imports.
func (g *Globe) project(outer *Scope, absPath, relPath, tmpPath, spec, name string) (m *Project) {
        if outer == nil {
                outer = g.scope
        }

	m = &Project{
                absPath: absPath,
                relPath: relPath, 
                tmpPath: tmpPath,
                using: new(usinglist),
                self: new(ProjectName),
                spec: spec,
                name: name,
        }

        m.scope = NewScope(outer, m, fmt.Sprintf("project %q", name))
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

                def, _ := g.scope.define(m, "_", &None{})
                if enable_assertions { assert(def != nil, "'$_' is nil") }

                for i := 1; i <= maxNumVarVal; i += 1 {
                        def, _ := g.scope.define(m, strconv.Itoa(i), &None{})
                        if enable_assertions { assert(def != nil, "'$%d' is nil", i) }
                }
        }
        return
}

// NewGlobe creates a new Globe context.
func NewGlobe(name string) (g *Globe) {
        g = &Globe{
                scope: NewScope(universe, nil, fmt.Sprintf("globe %q", name)),
                _timestamps: make(map[string]time.Time),
                _timestampx: new(sync.Mutex),
        }

        var absPath, relPath, tmpPath, spec string
        // TODO: determines absPath, relPath, tmpPath, spec
        g.os = g.project(nil, absPath, relPath, tmpPath, spec, runtime.GOOS)
        //g.os.scope.define(g.os, "name", &None{})
        return g
}
