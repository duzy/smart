package smartest

import (
        "regexp"
        //"bytes"
        //"fmt"
)

var (
        reEnteringLine = regexp.MustCompile(`smart: Entering directory '(.+?)'\n`)
        reLeavingLine = regexp.MustCompile(`smart:  Leaving directory '(.+?)'\n`)
)

func extractTextOutput(bv []byte) (t, a, b, h, f []byte) {
        var beg, end int
        if id := reEnteringLine.FindSubmatchIndex(bv); len(id) == 4 {
                h, a, beg = bv[:id[0]], bv[id[2]:id[3]], id[1]
        }
        if ids := reLeavingLine.FindAllSubmatchIndex(bv, -1); len(ids) > 0 {
                var id = ids[len(ids)-1]
                b, end, f = bv[id[2]:id[3]], id[0], bv[id[1]:]
        }
        if 0 < beg && beg < end && end < len(bv) {
                t = bv[beg:end]
        }
        return
}
