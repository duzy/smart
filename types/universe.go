//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

// This file sets up the global scope and the root project/module.

package types

import (
        "github.com/duzy/smart/token"
)

var (
	Universe *Scope
	Unsafe   *Module
)

var Types = []*Basic {
	Invalid:  {Invalid, 0, "invalid type"},
        
        Int:      {Int, IsInteger, "int"},
        Float:    {Float, IsFloat, "float"},
        DateTime: {DateTime, IsDateTime, "datetime"},
        Date:     {Date, IsDate, "date"},
        Time:     {Time, IsTime, "time"},
        Uri:      {Uri, IsUri, "uri"},
        String:   {String, IsString, "string"},

        None:     {None, IsNone, "none"},
}

func init() {
        Universe = NewScope(nil, token.NoPos, token.NoPos, "universe")
        Unsafe = NewModule("unsafe", "unsafe")
        Unsafe.complete = true
}
