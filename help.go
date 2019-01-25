//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "fmt"
)

func do_helpscreen() {
        fmt.Fprintf(stderr, `Build your projects smartly.

Usage:

    smart -help[(arguments)]
    smart -configure[(arguments)]
    smart -reconfigure[(arguments)]
    smart -clean[(arguments)]

Basic:

   -h
   -help
    Display this help screen.

   -c
   -configure
    Configure all projects underneath the work directory.

   -r
   -reconfigure
    Reconfigures all projects underneath the work directory.

   -l
   -clean
    Clean things already built previously.

Configuration:

Issues:

    * https://github.com/extbit/smart/issues
    * https://bugs.extbit.io/smart/report (not ready yet)

TODO:
    * display configure option list
    * display help entry list
`)
}
