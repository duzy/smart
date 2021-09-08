//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	"strings"
	"bytes"
	"sync"
	"sort"
	"fmt"
	"io"
)

// A Scope maintains a set of objects;
type Scope struct {
	mutex *sync.Mutex
	position Position
	project *Project
	outer *Scope
	elems map[string]Object
	comment string
}

func NewScope(pos Position, outer *Scope, project *Project, comment string) *Scope {
	scope := &Scope{
		mutex: new(sync.Mutex),
		position: pos,
		project: project,
		outer: outer,
		elems: make(map[string]Object),
		comment: comment,
	}
	return scope
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
func (s *Scope) lookup(name string) (obj Object) { return s.elems[name] }
func (s *Scope) Lookup(name string) (obj Object) {
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
func (s *Scope) findouter(name string) (*Scope, Object) {
	if false {
		for p := s; p != nil; p = p.outer {
			if obj := p.Lookup(name); obj != nil /*&& (!pos.IsValid() || obj.scopePos() <= pos)*/ {
				return p, obj
			}
		}
	} else {
		if s.outer != nil {
			if p, obj := s.outer.Find(name); obj != nil {
				return p, obj
			}
		}
	}
	return nil, nil
}

func (s *Scope) Find(name string) (*Scope, Object) {
	s.mutex.Lock(); defer s.mutex.Unlock()
	if obj := s.lookup(name); obj == nil {
		return s.findouter(name)
	} else {
		return s, obj
	}
}

func (s *Scope) Resolve(name string) (obj Object) {
	_, obj = s.Find(name)
	return
}

// Insert attempts to insert an object obj into scope s.
// If s already contains an alternative object alt with
// the same name, Insert leaves s unchanged and returns alt.
// Otherwise it inserts obj, sets the object's outer scope
// if not already set, and returns nil.
func (s *Scope) Insert(ctx Context, obj Object) Object {
	s.mutex.Lock(); defer s.mutex.Unlock()
	var name = obj.Name()
	if alt := s.elems[name]; alt != nil {
		return alt
	}
	s.replace(ctx, name, obj)
	return nil
}

func (s *Scope) replace(ctx Context, name string, obj Object) {
	if s.elems[name] = obj; obj.DeclScope() == nil {
		obj.redecl(ctx, s)
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
func (s *Scope) String() string {
	var buf bytes.Buffer
	s.WriteTo(&buf, 0)
	return buf.String()
}

func (s *Scope) FindDef(name string) (def *Def) {
	if _, sym := s.Find(name); sym != nil {
		def, _ = sym.(*Def)
	}
	return
}

func (scope *Scope) ProjectName(ctx Context, owner *Project, name string, project *Project) (pn *ProjectName, alt Object) {
	scope.mutex.Lock(); defer scope.mutex.Unlock()
	if alt = scope.elems[name]; alt == nil {
		pn = &ProjectName{
			knownobject{
				objbase{
					scope: scope,
					owner: owner,
				}, name,
			},
			project,
		}
		scope.replace(ctx, name, pn)
	}
	return
}

func (scope *Scope) ScopeName(ctx Context, owner *Project, name string, s *Scope) (sn *ScopeName, alt Object) {
	scope.mutex.Lock(); defer scope.mutex.Unlock()
	if alt = scope.elems[name]; alt == nil {
		sn = &ScopeName{
			knownobject{
				objbase{
					scope: scope,
					owner: owner,
				}, name,
			},
			s,
		}
		scope.replace(ctx, name, sn)
	}
	return
}

func (scope *Scope) Builtin(ctx Context, name string, f BuiltinFunc) (bui *Builtin, alt Object) {
	scope.mutex.Lock(); defer scope.mutex.Unlock()
	if alt = scope.elems[name]; alt == nil {
		bui = &Builtin{
			knownobject{
				objbase{
					scope: scope,
					owner: nil,
				}, name,
			},
			builtinFlag(0), f,
		}
		scope.replace(ctx, name, bui)
	}
	return
}

func (scope *Scope) define(ctx Context, owner *Project, name string, value Value) (def *Def, alt Object) {
	var okay bool
	scope.mutex.Lock(); defer scope.mutex.Unlock()
	if alt, okay = scope.elems[name]; okay && alt == nil {
		delete(scope.elems, name)
		okay = false
	}
	if !okay {
		def = &Def{
			knownobject: knownobject{
				objbase{
					scope: scope,
					owner: owner,
				}, name,
			},
			origin: Origin(0), value: value,
		}
		scope.replace(ctx, name, def)
	}
	return
}
