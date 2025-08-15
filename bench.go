//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    "strings"
    "sync"
    "time"
    "fmt"
    "io"
)

type _benchmark struct {
    tag string
    start, spot time.Time
    spent time.Duration
    num int64
    frames []*_benchmark
}

type _benchspot struct {
    tag string
    n int64
    d time.Duration
    x time.Duration
}

type _sum struct {
    n int64
    d time.Duration
}

var (
    // benchmarkM sync.Mutex
    // benchmark_ *_benchmark = &_benchmark{ tag:"benchmark" }
    benchspotM sync.Mutex
    benchspot = make(map[string]*_benchspot,64)
)

type bencher interface { bench(t time.Time) }
func bench(i bencher, t time.Time) { i.bench(t) }

func (frame *_benchmark) report(w io.Writer, indent int, up *_benchmark) {
    var s string
    if frame.num > 0 {
        var d time.Duration
        if up != nil { d = frame.spot.Sub(up.start) }
        s = fmt.Sprintf("%s (n=%v, d=%v, s=%v) (", frame.tag, frame.num, frame.spent, d)
    } else {
        s = fmt.Sprintf("%s (d=%v) (", frame.tag, frame.spent)
    }
    fprintIndentDots(w, indent, s)
    for _, sub := range frame.frames { sub.report(w, indent + 2, frame) }
    fprintIndentDots(w, indent, ")")
}

func (frame *_benchmark) _tag() string {
    var s = frame.tag
    if i := strings.Index(s, "("); i > 0 { s = s[0:i] }
    return s
}

func (frame *_benchmark) sum(res map[string]_sum) map[string]_sum {
    var t = frame._tag()
    var s = res[t]
    s.n += frame.num
    s.d += frame.spent
    res[t] = s
    for _, p := range frame.frames { p.sum(res) }
    return res
}

func (frame *_benchmark) _summary() (res map[string]_sum) {
    res = make(map[string]_sum,16)
    for _, f := range frame.frames { f.sum(res) }
    return
}

func (frame *_benchmark) summary(w io.Writer) {
    var m = frame._summary()
    var tags []string
    for tag, _ := range m { tags = append(tags, tag) }
    for i := 0; i < len(tags); i += 1 {
        for j := i+1; j < len(tags); j += 1 {
            a, b := tags[i], tags[j]
            if m[a].d < m[b].d {
                tags[i] = b
                tags[j] = a
            }
        }
    }
    for _, tag := range tags {
        var p = m[tag]
        fmt.Fprintf(w, "%s ", tag)
        var i = 40 - len(tag)
        for i > ndots {
            fmt.Fprint(w, dots)
            i -= ndots
        }
        fmt.Fprint(w, dots[0:i])
        fmt.Fprintf(w, "{ %d, %s, %s }\n", p.n, p.d, time.Duration(int64(p.d)/p.n))
    }
}

func benchspot_report(w io.Writer) {
    var tags []string
    for tag, _ := range benchspot { tags = append(tags, tag) }
    for i := 0; i < len(tags); i += 1 {
        for j := i+1; j < len(tags); j += 1 {
            a, b := tags[i], tags[j]
            if benchspot[a].d < benchspot[b].d {
                tags[i] = b
                tags[j] = a
            }
        }
    }
    for _, tag := range tags {
        fe := benchspot[tag]
        fmt.Fprintf(w, "%s ", tag)
        var i = 40 - len(tag)
        for i > ndots {
            fmt.Fprint(w, dots)
            i -= ndots
        }
        fmt.Fprint(w, dots[0:i])
        fmt.Fprintf(w, "{ %d, %s, %s }\n", fe.n, fe.d, fe.x)
    }
}
