//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

type interpreter interface {
        Evaluate(prog *Program, args []Value) (Value, error)
}

var dialects = map[string]interpreter{
        "":       &evaluer{ accumulation:false },
        "eval":   &evaluer{ accumulation:false },
        "value":  &evaluer{ accumulation:true },
        "shell":  &executor{ "bash", "-c" },
        "python": &executor{ "python", "-c" },
        "perl":   &executor{ "perl", "-e" },
        "dock":   &docker{},
        "plain":  &_plain{},
        "json":   &_json{},
        "xml":    &_xml{ whitespace:false },
        "yaml":   &_yaml{ whitespace:false },
}
