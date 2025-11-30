//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	"regexp"
	"reflect"
	"strings"
	"bytes"
	"sync"
	"sort"
	"fmt"
	"io"
)

var         name_prefix = regexp.MustCompile(`^((android|darwin|linux|bsd|ios|windows|mingw|[^~]+)~)(.+)$`)
var illegal_name_prefix = regexp.MustCompile(`^use\.(android|darwin|linux|bsd|ios|windows|mingw|[^~]+)~`)

// A scope maintains a set of objects;
// TODO: remote scope struct, use scopeContext instead
type scope struct {
	mutex sync.Mutex
	elems map[string]object
	project *project
	outer *scope
	comment string
}

func newscope(pos Position, outer *scope, owner *project, c string) (s *scope) {
	return &scope{outer:outer, project:owner, comment:c, elems:make(map[string]object)}
}

func (s *scope) has_outer(outer *scope) bool {
	return s.outer != nil && (s.outer == outer || s.outer.has_outer(outer))
}

func (s *scope) copyElems() (result map[string]object) {
	s.mutex.Lock(); defer s.mutex.Unlock()
	result = make(map[string]object, len(s.elems))
	for k, o := range s.elems { result[k] = o }
	return
}

func (s *scope) estr() (res string) {
	for _, o := range s.elems {
		if res != "" { res += " " }
		res += fmt.Sprintf("%v", o)
	}
	return
}

// Names returns the scope's element names in sorted order.
func (s *scope) names() []string {
	s.mutex.Lock() ; defer s.mutex.Unlock()
	var i = 0
	var names = make([]string, len(s.elems))
	for name := range s.elems {
		names[i] = name
		i++
	}
	sort.Strings(names)
	return names
}

// Lookup returns the object in scope s with the given name if such an
// object exists; otherwise the result is nil.
func (s *scope) Lookup(name string) object {
	s.mutex.Lock() ; defer s.mutex.Unlock()
	return s.lookup(name)
}
func (s *scope) lookup(name string) (obj object) {
	if s.elems != nil { obj, _ = s.elems[name] }
	return
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
func (s *scope) find(name string) (res *scope, obj object) {
	if obj = s.lookup(name) ; obj != nil {
		return s,obj
	} else if  s.outer != nil  {
		return s.outer.find(name)
	}
	return
}

func (s *scope) finddef(name string) (d *def) {
	if _, o := s.find(name) ; o != nil { d, _ = o.(*def) }
	return
}

func (s *scope) resolve(name string) (obj object) {
	if false { s.mutex.Lock() ; defer s.mutex.Unlock() }
	_, obj = s.find(name)
	return
}

// Insert attempts to insert an object obj into scope s.
// If s already contains an alternative object alt with
// the same name, Insert leaves s unchanged and returns alt.
// Otherwise it inserts obj, sets the object's outer scope
// if not already set, and returns nil.
func (s *scope) insert(ctx Context, obj object) object {
	s.mutex.Lock(); defer s.mutex.Unlock()

    var ic *ident_ctx
    if ic, ctx = identity(ctx); ic.nil > 0 {
		erro(pc(ctx,obj), "no ident: %v", obj).trace()
	}

	if name := ident(ctx, obj); name == "" {
		erro(pc(ctx,obj), "no ident: %v", obj).trace()
		return nil
	} else if alt := s.elems[name]; alt != nil {
		return alt
	} else {
		s.replace(ctx, name, obj)
		return nil
	}
}

func (s *scope) replace(ctx Context, name string, obj object) {
	switch o := obj.(type) {
	case interface { setscope(string, *scope) }:
		o.setscope(name, s)
	}
	s.elems[name] = obj
}

// WriteTo writes a string representation of the scope to w,
// with the scope elements sorted by name.
// The level of indentation is controlled by n >= 0, with
// n == 0 for no indentation.
func (s *scope) WriteTo(w io.Writer, n int) {
	s.mutex.Lock() ; defer s.mutex.Unlock()

	const ind = ".  "

	var indn  = strings.Repeat(ind, n)
	var indn1 = indn + ind

	fmt.Fprintf(w, "%s%s scope %p {", indn, s.comment, s)

	if len(s.elems) == 0 {
		fmt.Fprintf(w, "}")
		return
	}

	fmt.Fprintln(w)

	for _, name := range s.names() {
		fmt.Fprintf(w, "%s%s\n", indn1, s.elems[name])
	}

	fmt.Fprintf(w, "%s}", indn)
}

// String returns a string representation of the scope, for debugging.
func (s *scope) String() string { return fmt.Sprintf("{=scope %s}", s.string()) }
func (s *scope) string() string {
	var buf bytes.Buffer
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

func (s *scope) projectname(ctx Context, name string, project *project) (p *project, a object) {
	s.mutex.Lock() ; defer s.mutex.Unlock()
	if a = s.elems[name] ; a == nil {
		p = project
		s.replace(ctx, name, p)
	}
	return
}

func (s *scope) builtin(ctx Context, name string, f reflect.Type) (res *builtin, a object) {
	s.mutex.Lock() ; defer s.mutex.Unlock()
	if a = s.elems[name] ; a == nil {
		res = &builtin{knownobject{objbase{scope:s}, name}, f}
		s.replace(ctx, name, res)
	}
	return
}

func (s *scope) _auto(ctx Context, name string) (a *auto, o object) {
	s.mutex.Lock() ; defer s.mutex.Unlock()

	var y bool

	if o, y = s.elems[name]; y && o == nil {
		delete(s.elems, name)
		y = false
	}

	if !y {
		p := _position(ctx)
		a = &auto{knownobject{objbase{valbase{p},s}, name}}
		s.replace(ctx, name, a)
	}
	return
}

func (s *scope) auto(ctx Context, name string) (a *auto) {
	var y bool
	var o object
	if a, o = s._auto(ctx, name); o != nil {
		if a, y = o.(*auto); !y {
			erro(ctx, "name already taken (%s)", typeof(o)).trace()
		}
	}
	return
}

func (s *scope) alias(ctx Context, o object, alias ...string) {
	for _, a := range alias { s.elems[a] = o }
}

func (s *scope) _def(ctx Context, o origin, id any, vals ...Value) (d *def, isNew bool) {
	var pos Position
	var name string
	switch t := id.(type) {
	case Value : name, pos = ident(ctx, t), t.Position()
	case string: name, pos = t, _position(ctx)
	}
	if name == "" {
		errostack(ctx, 3, "empty name: %s : %s", id, ts(id)).trace()
	}
	if checkpoints && illegal_name_prefix.MatchString(name) {
		errostack(ctx, 3, "illegal name: %v", name).trace()
	}

	s.mutex.Lock(); defer s.mutex.Unlock()

	if a, y := s.elems[name]; !y || a == nil {
		d = new(def)
		d.name, d.position, d.scope = name, pos, s
		s.elems[name] = d
		isNew = true
	} else if d, y = a.(*def); d != nil {
		if !d.position.valid() && pos.valid() { d.position = pos }
	}
	if o != defInvalid {
		if d.o == defInvalid { d.o = o } else {
			errostack(ctx, 3, "%v: conflicts origin: %v | %v", ident, d.o, o).trace()
		}
	}
	if vals != nil { d.set(ctx, ease(ctx, vals)) }
	return
}

func (s *scope) def(ctx Context, o origin, ident any, vals ...Value) (d *def) {
	d, _ = s._def(ctx, o, ident, vals...)
	return
}
