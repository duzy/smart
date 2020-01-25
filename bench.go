//
//  Copyright (C) 2012-2019, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "sync"
        "time"
        "fmt"
)

const optionEnableBenchmarks = true

type _benchmark struct {
        tag string
        spot time.Time
        spent time.Duration
        num int64
        frames []*_benchmark
}

var benchmarkM sync.Mutex
var benchmark *_benchmark = &_benchmark{ tag:"benchmark" }

func spot(tag string) (save *_benchmark) {
        //benchmarkM.Lock(); defer benchmarkM.Unlock()
        var t = time.Now()
        save = benchmark
        for _, frame := range save.frames {
                if frame.tag == tag {
                        benchmark = frame
                        benchmark.spot = t
                        return
                }
        }
        benchmark = &_benchmark{ tag, t, 0, 0, nil }
        save.frames = append(save.frames, benchmark)
        return 
}

func bench(previous *_benchmark) {
        //benchmarkM.Lock(); defer benchmarkM.Unlock()
        benchmark.num += 1
        benchmark.spent += time.Now().Sub(benchmark.spot)
        benchmark = previous
}

func (frame *_benchmark) report(indent int) {
        var s string
        if frame.num > 0 {
                s = fmt.Sprintf("%s (%v/%v) (", frame.tag, frame.num, frame.spent)
        } else {
                s = fmt.Sprintf("%s (%v) (", frame.tag, frame.spent)
        }
        printIndentDots(indent, s)
        for _, sub := range frame.frames { sub.report(indent + 2) }
        printIndentDots(indent, ")")
}
