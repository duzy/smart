//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        //yaml_enc "encoding/yaml"
        //"strings"
        //"io"
        "fmt"
)

type YAML struct { Value }
func (p *YAML) String() string { return "(yaml " + p.Value.String() + ")" }
func (p *YAML) cmp(ctx Context, v Value) (res cmpres) {
        if a, ok := v.(*YAML); ok {
                assert(ok, "value is not YAML")
                res = p.Value.cmp(ctx, a.Value)
        }
        return
}

/*
   TODO: implement the yaml format:

   yaml{books(number=3
       book(id=1 title('book one'))
       book(id=2 title('book two'))
       book(id=3 title('   abc   '))
   )}
 */
func DecodeYAML(ctx Context, source string, ws bool) (result Value, err error) {
        err = fmt.Errorf("DecodeYAML not implemented yet")
        return 
}

type yaml struct { whitespace bool }
func (p *yaml) Evaluate(ctx Context, args ...Value) (result Value, err error) {
        var source string
        if source, err = multiline(ctx, ctx.program().recipes...); err != nil {
                erro(ctx, "%v", err).debug(1)
                return
        } else if result, err = DecodeYAML(ctx, source, p.whitespace); err == nil {
                result = &YAML{ result }
        } else {
                result = &YAML{ MakeNone(ctx.Position()) }
                erro(ctx, "%v", err).debug(1)
        }
        return
}
