//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	"reflect"
	"strings"
	"bytes"
	"sync"
	"sort"
	"fmt"
	"io"
)

// A Scope maintains a set of objects;
type Scope struct { // TODO: remote Scope struct, use scopeContext instead
	mutex sync.Mutex

	elems map[string]Object
	position Position
	project *Project
	outer *Scope
	comment string
}

func NewScope(pos Position, outer *Scope, project *Project, comment string) *Scope {
	return &Scope{
		elems: make(map[string]Object),
		position: pos,
		project: project,
		outer: outer,
		comment: comment,
	}
}

func (s *Scope) hasOuter(outer *Scope) bool {
	return s.outer != nil && (s.outer == outer || s.outer.hasOuter(outer))
}

func (s *Scope) copyElems() (result map[string]Object) {
	s.mutex.Lock(); defer s.mutex.Unlock()
	result = make(map[string]Object, len(s.elems))
	for k, o := range s.elems { result[k] = o }
	return
}

func (s *Scope) Comment() string { return s.comment }

// Outer returns the scope's containing (outer) scope.
//func (s *Scope) Outer() *Scope { return s.outer }

// Len() returns the number of scope elements.
func (s *Scope) Len() int {
	s.mutex.Lock(); defer s.mutex.Unlock()
	return len(s.elems)
}

// Names returns the scope's element names in sorted order.
func (s *Scope) Names() []string {
	s.mutex.Lock(); defer s.mutex.Unlock()
	names := make([]string, len(s.elems))
	i := 0
	for name := range s.elems {
		names[i] = name
		i++
	}
	sort.Strings(names)
	return names
}

// Project returns the project where this scope is existed.
//func (s *Scope) Project() *Project { return s.project }

// Lookup returns the object in scope s with the given name if such an
// object exists; otherwise the result is nil.
func (s *Scope) lookup(name string) (obj Object) {
	if s.elems != nil { obj = s.elems[name] }
	return
}
func (s *Scope) Lookup(name string) Object {
	s.mutex.Lock(); defer s.mutex.Unlock()
	return s.lookup(name)
}

// findouter follows the outer chain of scopes starting with s until
// it finds a scope where Lookup(name) returns a non-nil object, and then
// returns that scope and object. If no such scope and object exists, the
// result is (nil, nil).
//
// Note that obj.Outer() may be different from the returned scope if the
// object was inserted into the scope and already had a outer at that
// time (see Insert, below). This can only happen for dot-imported objects
// whose scope is the scope of the package that exported them.
func (s *Scope) find(name string) (res *Scope, obj Object) {
	if false { s.mutex.Lock(); defer s.mutex.Unlock() }
	if obj = s.lookup(name); obj == nil && s.outer != nil {
		if false { for t := s.outer; t != nil; t = s.outer {
			if t == s { panic(name) }
		}}
		if p, o := s.outer.find(name); o != nil {
			res, obj = p, o
		}
	}
	return s, obj
}

func (s *Scope) resolve(name string) (obj Object) {
	_, obj = s.find(name)
	return
}

// Insert attempts to insert an object obj into scope s.
// If s already contains an alternative object alt with
// the same name, Insert leaves s unchanged and returns alt.
// Otherwise it inserts obj, sets the object's outer scope
// if not already set, and returns nil.
func (s *Scope) insert(ctx Context, obj Object) Object {
	s.mutex.Lock(); defer s.mutex.Unlock()
	var name = obj.name(ctx)
	if alt := s.elems[name]; alt != nil {
		return alt
	}
	s.replace(ctx, name, obj)
	return nil
}

func (s *Scope) replace(ctx Context, name string, obj Object) {
	if s.elems[name] = obj; obj.declScope() == nil {
		obj.rescope(ctx, s)
	}
}

// WriteTo writes a string representation of the scope to w,
// with the scope elements sorted by name.
// The level of indentation is controlled by n >= 0, with
// n == 0 for no indentation.
func (s *Scope) WriteTo(w io.Writer, n int) {
	s.mutex.Lock(); defer s.mutex.Unlock()

	const ind = ".  "
	indn := strings.Repeat(ind, n)

	fmt.Fprintf(w, "%s%s scope %p {", indn, s.comment, s)
	if len(s.elems) == 0 {
		fmt.Fprintf(w, "}")
		return
	}

	fmt.Fprintln(w)
	indn1 := indn + ind
	for _, name := range s.Names() {
		fmt.Fprintf(w, "%s%s\n", indn1, s.elems[name])
	}

	fmt.Fprintf(w, "%s}", indn)
}

// String returns a string representation of the scope, for debugging.
func (s *Scope) String() string { return fmt.Sprintf("scope{%s}", s.string()) }
func (s *Scope) string() string {
	var buf bytes.Buffer //s.WriteTo(&buf, 0)
	if s.outer != nil {
		if false {
			fmt.Fprintf(&buf, "%s → %s", s.outer.string(), s.comment)
		} else {
			fmt.Fprintf(&buf, "%s ← %s", s.comment, s.outer.string())
		}
	} else {
		fmt.Fprintf(&buf, "%s", s.comment)
	}
	return buf.String()
}

func (s *Scope) FindDef(name string) (res *def) {
	if _, sym := s.find(name); sym != nil {
		res, _ = sym.(*def)
	}
	return
}

func (scope *Scope) projectname(ctx Context, name string, project *Project) (pn *projectname, alt Object) {
	scope.mutex.Lock(); defer scope.mutex.Unlock()
	if alt = scope.elems[name]; alt == nil {
		pn = &projectname{ project, scope }
		scope.replace(ctx, name, pn)
	}
	return
}

func (scope *Scope) scopename(ctx Context, name string, s *Scope) (sn *scopename, alt Object) {
	scope.mutex.Lock(); defer scope.mutex.Unlock()
	if alt = scope.elems[name]; alt == nil {
		sn = &scopename{ s, name } // TODO: scope,
		scope.replace(ctx, name, sn)
	}
	return
}

func (scope *Scope) builtin(ctx Context, name string, f reflect.Type) (bui *builtin, alt Object) {
	scope.mutex.Lock(); defer scope.mutex.Unlock()
	if alt = scope.elems[name]; alt == nil {
		bui = &builtin{
			knownobject{
				objbase{
					scope_: scope,
					owner_: nil,
				}, name,
			}, f,
		}
		scope.replace(ctx, name, bui)
	}
	return
}

func (scope *Scope) auto2(ctx Context, name string) (a *auto, alt Object) {
	scope.mutex.Lock(); defer scope.mutex.Unlock()

	var okay bool
	if alt, okay = scope.elems[name]; okay && alt == nil {
		delete(scope.elems, name)
		okay = false
	}
	if !okay {
		p := ctx.Position()
		a = &auto{
			knownobject{
				objbase{valbase{p}, scope, ctx.Project()},
				name,
			},
		}
		scope.replace(ctx, name, a)
	}
	return
}

func (scope *Scope) auto(ctx Context, name string, alias ...string) (a *auto) {
	var t Object
	if a, t = scope.auto2(ctx, name); t != nil { var y bool
		if a, y = t.(*auto)	; !y { erro(ctx, "name '%s' already taken: %T", t) }
	}
	if a != nil { for _, s := range alias {
		scope.replace(ctx, s, a)
	}}
	return
}

func (scope *Scope) define(ctx Context, origin Origin, name string, value Value) (d *def, alt Object) {
	var okay bool
	scope.mutex.Lock(); defer scope.mutex.Unlock()
	if alt, okay = scope.elems[name]; okay && alt == nil {
		delete(scope.elems, name)
		okay = false
	}
	if !okay {
		d = &def{
			origin: origin, value: value,
			knownobject: knownobject{
				objbase{
					scope_: scope,
					owner_: scope.project, //ctx.Project(),
				}, name,
			},
		}
		scope.replace(ctx, name, d)
	}
	return
}
