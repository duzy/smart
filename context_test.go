//
//  Copyright (C) 2012-2024, Duzy Chan <code@extbit.io>, all rights reserverd.
//
package smart

import (
	"testing"
)

type fooctx  struct { Context }
type foo1ctx struct {  fooctx }
type foo2ctx struct { *fooctx }

func (p *foo1ctx) inner() Context { return p.fooctx }
func (p *foo2ctx) inner() Context { return p.fooctx }

func testInner(t *testing.T) {
	if i := inner(&fooctx{ &fooctx{} }); i == nil {
		t.Fatalf("inner(fooctx{fooctx})")
	} else if i = inner(&fooctx{}); i != nil {
		t.Fatalf("inner(fooctx{}): %v", i)
	}
	if i := inner(&foo1ctx{}); i == nil {
		t.Fatalf("inner(foo1ctx{fooctx{}})")
	} else if _, y := i.(fooctx); !y {
		t.Fatalf("inner(foo1ctx{fooctx{}}): %T", i)
	} else if i = inner(i); i != nil {
		t.Fatalf("inner(foo1ctx{fooctx{}}): %v", i)
	}
	if i := inner(&foo2ctx{ &fooctx{} }); i == nil {
		t.Fatalf("inner(foo2ctx{fooctx{}})")
	} else if _, y := i.(*fooctx); !y {
		t.Fatalf("inner(foo2ctx{fooctx{}}): %T", i)
	} else if i = inner(i); i != nil {
		t.Fatalf("inner(foo2ctx{fooctx{}}): %v", i)
	}
}
