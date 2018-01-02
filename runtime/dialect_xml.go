//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        "github.com/duzy/smart/types"
        "github.com/duzy/smart/values"
        "encoding/xml"
        "strings"
        "io"
)

/*
<books number="3">
  <book id="1">
    <title>book one</title>
  </book>
  <book id="2">
    <title>book two</title>
  </book>
  <book id="3"> <title>  abc  </title> </book>
</books>

Converted into:

(
        books number=3
        (
                book id=1
                (title 'book one')
        )
        (
                book id=2
                (title 'book two')
        )
        (
                book id=3
                (title '  abc  ')
        )
) */
func DecodeXML(source string, ws bool) (result types.Value, err error) {
        var (
                stack []*types.Group
                nodes []*types.Group
        )
        xd := xml.NewDecoder(strings.NewReader(source))
        var tok xml.Token
        for tok, err = xd.Token(); err == nil; tok, err = xd.Token() {
                switch elem := tok.(type) {
                case xml.ProcInst:
                        // TODO: ...
                case xml.StartElement:
                        nn := values.Group(values.Bareword(elem.Name.Local))
                        for _, a := range elem.Attr {
                                var k, v types.Value
                                k = values.Bareword(a.Name.Local)
                                v = values.String(a.Value)
                                if s := a.Name.Space; s != "" {
                                        k = values.Group(values.String(s), k)
                                }
                                nn.Append(values.Pair(k, v))
                        }
                        if x := len(stack); x > 0 {
                                stack[x-1].Append(nn)
                        } else {
                                nodes = append(nodes, nn)
                        }
                        stack = append(stack, nn)
                case xml.EndElement:
                        if x := len(stack); x > 0 {
                                stack = stack[0:x-1]
                        } else {
                                // FIXME: report illegal xml
                        }
                case xml.CharData:
                        if x := len(stack); x > 0 {
                                node, s := stack[x-1], string(elem)
                                if ws {
                                        node.Append(values.String(s))
                                } else {
                                        if s = strings.TrimSpace(s); s != "" {
                                                node.Append(values.String(s))
                                        }
                                }
                        }
                case xml.Directive:
                        // TODO: ...
                case xml.Comment:
                        // TODO: ...
                }
        }
        if x := len(nodes); x > 1 {
                g := values.Group()
                for _, node := range nodes {
                        g.Append(node)
                }
                result = g
        } else if x == 1 {
                result = nodes[0]
        }
        if err == io.EOF {
                err = nil // all done
        }
        return
}

type dialectXml struct {
        polyInterpreter
        whitespace bool
}

func (t *dialectXml) dialect() string { return "xml" }
func (t *dialectXml) evaluate(prog *Program, args []types.Value, recipes []types.Value) (result types.Value, err error) {
        var source = joinRecipesString(recipes...)
        if result, err = DecodeXML(source, t.whitespace); err == nil {
                result = &types.XML{ result }
        } else {
                result = &types.XML{ values.None }
        }
        return
}
