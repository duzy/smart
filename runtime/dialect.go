//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        "github.com/duzy/smart/types"
        "bytes"
        "fmt"
)

func joinRecipesString(recipes... types.Value) string {
        var (
                x = len(recipes)-1
                s = new(bytes.Buffer)
        )
        for n, recipe := range recipes {
                if fmt.Fprint(s, recipe.Strval()); n < x {
                        fmt.Fprint(s, "\n")
                }
        }
        return s.String()
}
