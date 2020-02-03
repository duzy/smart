//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

type interpreter interface {
        Evaluate(pc *traversal, args []Value) (Value, error)
}

var dialects = map[string]interpreter{
        "":       &evaluer{ accumulation:false },
        "eval":   &evaluer{ accumulation:false },
        "value":  &evaluer{ accumulation:true },
        "shell":  &executor{ cmd:"bash",   opt:"-c", contained:false }, //&executor_{ "bash", "-c" },
        "python": &executor{ cmd:"python", opt:"-c", contained:false }, //&executor_{ "python", "-c" },
        "perl":   &executor{ cmd:"perl",   opt:"-e", contained:false }, //&executor_{ "perl", "-e" },
        "dock":   &executor{ cmd:"sh",     opt:"-c", contained:true },
        "plain":  &plain{},
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
