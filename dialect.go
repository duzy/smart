//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

type interpreter interface {
    evaluate(Context, ...Value) Value
}

var dialects = map[string]interpreter{
    "":       &eval{ },
    "eval":   &eval{ eval:true },
    "value":  &eval{ accumulation:true },
    "shell":  &executor{ cmd:"bash",   opt:"-c", contained:false },
    "python": &executor{ cmd:"python", opt:"-c", contained:false },
    "perl":   &executor{ cmd:"perl",   opt:"-e", contained:false },
    "dock":   &executor{ cmd:"sh",     opt:"-c", contained:true },
    "plain":  &plainInt{},
    "json":   &json{},
    "xml":    &xml{ whitespace:false },
    "yaml":   &yaml{ whitespace:false },
}

func intername(i interpreter) (s string) {
    for k, d := range dialects {
        if d == i { s = k; break }
    }
    return
}
