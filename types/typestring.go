//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package types

import (
        "bytes"
        "fmt"
)

// A Qualifier controls how named module-level objects are printed in
// calls to TypeString, ObjectString, and SelectionString.
//
// These three formatting routines call the Qualifier for each
// module-level object O, and if the Qualifier returns a non-empty
// string p, the object is printed in the form p.O.
// If it returns an empty string, only the object name O is printed.
//
// Using a nil Qualifier is equivalent to using (*Module).Path: the
// object is qualified by the import path, e.g., "encoding/json.Marshal".
//
type Qualifier func(*Module) string

// RelativeTo(pkg) returns a Qualifier that fully qualifies members of
// all modules other than pkg.
func RelativeTo(pkg *Module) Qualifier {
	if pkg == nil {
		return nil
	}
	return func(other *Module) string {
		if pkg == other {
			return "" // same module; unqualified
		}
		return other.path //other.Path()
	}
}

// TypeString returns the string representation of typ.
// The Qualifier controls the printing of
// module-level objects, and may be nil.
func TypeString(typ Type, qf Qualifier) string {
	var buf bytes.Buffer
	WriteType(&buf, typ, qf)
	return buf.String()
}

// WriteType writes the string representation of typ to buf.
// The Qualifier controls the printing of
// module-level objects, and may be nil.
func WriteType(buf *bytes.Buffer, typ Type, qf Qualifier) {
	writeType(buf, typ, qf, make([]Type, 8))
}

func writeType(buf *bytes.Buffer, typ Type, qf Qualifier, visited []Type) {
	// Theoretically, this is a quadratic lookup algorithm, but in
	// practice deeply nested composite types with unnamed component
	// types are uncommon. This code is likely more efficient than
	// using a map.
	for _, t := range visited {
		if t == typ {
			fmt.Fprintf(buf, "○%T", typ) // cycle to typ
			return
		}
	}
	visited = append(visited, typ)

	switch t := typ.(type) {
	case nil:
		buf.WriteString("<nil>")

	case *Core:
		buf.WriteString(t.name)

	case *Basic:
		buf.WriteString(t.name)

        case *Composite:
		buf.WriteString(t.name)

	case *Named:
		buf.WriteString(t.name)

	default:
		// For externally defined implementations of Type.
		buf.WriteString(t.String())
	}
}
