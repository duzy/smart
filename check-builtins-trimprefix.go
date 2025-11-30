//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints__trimprefix = map[string]map[string]any{
	"loader.go": map[string]any{
		`3:3:p $/ {3:6:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimprefix {3:6 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}}`),

		`10:6:val1 $/ {10:39:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimprefix {10:39 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}}`),
		`10:6:val1 $(dir $/) {10:32:delegate {10:34:builtin dir} {=list {10:39:delegate {1:1:def /}}}}`:testdata_fmt(`{10:34:builtin dir} %[1]s/builtins {10:32 {=path %[3]s {10:34:word builtins}}}`,line_column_s("10:34")),
		`10:6:val1 $/ {10:43:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimprefix {10:43 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}}`),
		`10:6:val1 $(trim-prefix $(dir $/),$/) {10:9:delegate {10:11:builtin trim-prefix} {=list {10:32:delegate {10:34:builtin dir} {=list {10:39:delegate {1:1:def /}}}}} {=list {10:43:delegate {1:1:def /}}}}`:`{10:11:builtin trim-prefix} /trimprefix {10:9 {=path {10:11:punct ROOT} {10:11:word trimprefix}}}`,

		`11:6:val2 $(pat0) {11:23:delegate {5:6:def pat0}}`:`{5:6:def pat0} **/testdata {11:23 {=path {=glob {5:10:meta **}} {5:13:word testdata}}}`,
		`11:6:val2 $/ {11:43:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimprefix {11:43 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}}`),
		`11:6:val2 $/ {11:39:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimprefix {11:39 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}}`),
		`11:6:val2 $(dir2 $/) {11:32:delegate {11:34:builtin dir2} {=list {11:39:delegate {1:1:def /}}}}`:testdata_fmt(`{11:34:builtin dir2} %[1]s {11:32 {=path %[3]s}}`,line_column_s("11:34")),
		`11:6:val2 $(trim-prefix $(pat0)/ $(dir2 $/),$/) {11:9:delegate {11:11:builtin trim-prefix} {=list {=path {11:23:delegate {5:6:def pat0}} {11:31:punct TAIL}} {11:32:delegate {11:34:builtin dir2} {=list {11:39:delegate {1:1:def /}}}}} {=list {11:43:delegate {1:1:def /}}}}`:`{11:11:builtin trim-prefix} builtins/trimprefix {11:9 {=path {11:11:word builtins} {11:11:word trimprefix}}}`,

		`12:6:val3 $(pat0) {12:23:delegate {5:6:def pat0}}`:`{5:6:def pat0} **/testdata {12:23 {=path {=glob {5:10:meta **}} {5:13:word testdata}}}`,
		`12:6:val3 $/ {12:32:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimprefix {12:32 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}}`),
		`12:6:val3 $(trim-prefix $(pat0)/,$/) {12:9:delegate {12:11:builtin trim-prefix} {=list {=path {12:23:delegate {5:6:def pat0}} {12:31:punct TAIL}}} {=list {12:32:delegate {1:1:def /}}}}`:`{12:11:builtin trim-prefix} builtins/trimprefix {12:9 {=path {12:11:word builtins} {12:11:word trimprefix}}}`,

		`13:6:val4 $(pat1) {13:23:delegate {6:6:def pat1}}`:`{6:6:def pat1} %%/testdata {13:23 {=path {=percpat {6:10} {=percpat {6:12} {6:12}}} {6:13:word testdata}}}`,
		`13:6:val4 $/ {13:32:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimprefix {13:32 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}}`),
		`13:6:val4 $(trim-prefix $(pat1)/,$/) {13:9:delegate {13:11:builtin trim-prefix} {=list {=path {13:23:delegate {6:6:def pat1}} {13:31:punct TAIL}}} {=list {13:32:delegate {1:1:def /}}}}`:`{13:11:builtin trim-prefix} builtins/trimprefix {13:9 {=path {13:11:word builtins} {13:11:word trimprefix}}}`,

		`14:6:val5 $(pat2) {14:23:delegate {7:6:def pat2}}`:`{7:6:def pat2} /**/testdata {14:23 {=path {7:9:punct ROOT} {=glob {7:10:meta **}} {7:13:word testdata}}}`,
		`14:6:val5 $/ {14:32:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimprefix {14:32 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}}`),
		`14:6:val5 $(trim-prefix $(pat2)/,$/) {14:9:delegate {14:11:builtin trim-prefix} {=list {=path {14:23:delegate {7:6:def pat2}} {14:31:punct TAIL}}} {=list {14:32:delegate {1:1:def /}}}}`:`{14:11:builtin trim-prefix} builtins/trimprefix {14:9 {=path {14:11:word builtins} {14:11:word trimprefix}}}`,

		`15:6:val6 $(pat3) {15:23:delegate {8:6:def pat3}}`:`{8:6:def pat3} /%%/testdata {15:23 {=path {8:9:punct ROOT} {=percpat {8:10} {=percpat {8:12} {8:12}}} {8:13:word testdata}}}`,
		`15:6:val6 $/ {15:32:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimprefix {15:32 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}}`),
		`15:6:val6 $(trim-prefix $(pat3)/,$/) {15:9:delegate {15:11:builtin trim-prefix} {=list {=path {15:23:delegate {8:6:def pat3}} {15:31:punct TAIL}}} {=list {15:32:delegate {1:1:def /}}}}`:`{15:11:builtin trim-prefix} builtins/trimprefix {15:9 {=path {15:11:word builtins} {15:11:word trimprefix}}}`,

		`20:6:val7 $(pat4) {20:23:delegate {17:6:def pat4}}`:`{17:6:def pat4} *?/testdata {20:23 {=path {=glob {17:10:meta *?}} {17:13:word testdata}}}`,
		`20:6:val7 $/ {20:32:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimprefix {20:32 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}}`),
		`20:6:val7 $(trim-prefix $(pat4)/,$/) {20:9:delegate {20:11:builtin trim-prefix} {=list {=path {20:23:delegate {17:6:def pat4}} {20:31:punct TAIL}}} {=list {20:32:delegate {1:1:def /}}}}`:`{20:11:builtin trim-prefix} builtins/trimprefix {20:9 {=path {20:11:word builtins} {20:11:word trimprefix}}}`,

		`21:6:val8 $(pat5) {21:23:delegate {18:6:def pat5}}`:`{18:6:def pat5} /*?/testdata {21:23 {=path {18:9:punct ROOT} {=glob {18:10:meta *?}} {18:13:word testdata}}}`,
		`21:6:val8 $/ {21:32:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimprefix {21:32 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}}`),
		`21:6:val8 $(trim-prefix $(pat5)/,$/) {21:9:delegate {21:11:builtin trim-prefix} {=list {=path {21:23:delegate {18:6:def pat5}} {21:31:punct TAIL}}} {=list {21:32:delegate {1:1:def /}}}}`:`{21:11:builtin trim-prefix} builtins/trimprefix {21:9 {=path {21:11:word builtins} {21:11:word trimprefix}}}`,

		`22:6:val9 $/ {22:37:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimprefix {22:37 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}}`),
		`22:6:val9 $(trim-prefix /*?/testdata/,$/) {22:9:delegate {22:11:builtin trim-prefix} {=list {=path {22:23:punct ROOT} {=glob {22:24:meta *?}} {22:27:word testdata} {22:36:punct TAIL}}} {=list {22:37:delegate {1:1:def /}}}}`:`{22:11:builtin trim-prefix} builtins/trimprefix {22:9 {=path {22:11:word builtins} {22:11:word trimprefix}}}`,

		`24:7:val10 $/ {24:47:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimprefix {24:47 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}}`),
		`24:7:val10 $(trim-prefix /**/testdata /builtins,$/) {24:10:delegate {24:12:builtin trim-prefix} {=list {=path {24:24:punct ROOT} {=glob {24:25:meta **}} {24:28:word testdata}} {=path {24:37:punct ROOT} {24:38:word builtins}}} {=list {24:47:delegate {1:1:def /}}}}`:`{24:12:builtin trim-prefix} /trimprefix {24:10 {=path {24:12:punct ROOT} {24:12:word trimprefix}}}`,

		`25:7:val11 $/ {25:47:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimprefix {25:47 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}}`),
		`25:7:val11 $(trim-prefix /**/testdata/ builtins,$/) {25:10:delegate {25:12:builtin trim-prefix} {=list {=path {25:24:punct ROOT} {=glob {25:25:meta **}} {25:28:word testdata} {25:37:punct TAIL}} {25:38:word builtins}} {=list {25:47:delegate {1:1:def /}}}}`:`{25:12:builtin trim-prefix} /trimprefix {25:10 {=path {25:12:punct ROOT} {25:12:word trimprefix}}}`,

		`26:7:val12 $/ {26:40:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimprefix {26:40 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}}`),
		`26:7:val12 $(trim-prefix /**/ trimprefix,$/) {26:10:delegate {26:12:builtin trim-prefix} {=list {=path {26:24:punct ROOT} {=glob {26:25:meta **}} {26:28:punct TAIL}} {26:29:word trimprefix}} {=list {26:40:delegate {1:1:def /}}}}`:`{26:12:builtin trim-prefix} {} {26:10 {26:40:null}}`,

		`27:7:val13 $/ {27:41:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimprefix {27:41 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}}`),
		`27:7:val13 $(trim-prefix /**/ trimprefix/,$//) {27:10:delegate {27:12:builtin trim-prefix} {=list {=path {27:24:punct ROOT} {=glob {27:25:meta **}} {27:28:punct TAIL}} {=path {27:29:word trimprefix} {27:40:punct TAIL}}} {=list {=path {27:41:delegate {1:1:def /}} {27:44:punct TAIL}}}}`:`{27:12:builtin trim-prefix} {} {27:10 {27:41:null}}`,

		`28:7:val14 $/ {28:40:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimprefix {28:40 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}}`),
		`28:7:val14 $(trim-prefix **/testdata/**,$/) {28:10:delegate {28:12:builtin trim-prefix} {=list {=path {=glob {28:25:meta **}} {28:28:word testdata} {=glob {28:37:meta **}}}} {=list {28:40:delegate {1:1:def /}}}}`:`{28:12:builtin trim-prefix} {} {28:10 {28:40:null}}`,

		`29:7:val15 $/ {29:41:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimprefix {29:41 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}}`),
		`29:7:val15 $(trim-prefix **/testdata/**/,$//) {29:10:delegate {29:12:builtin trim-prefix} {=list {=path {=glob {29:25:meta **}} {29:28:word testdata} {=glob {29:37:meta **}} {29:40:punct TAIL}}} {=list {=path {29:41:delegate {1:1:def /}} {29:44:punct TAIL}}}}`:`{29:12:builtin trim-prefix} {} {29:10 {29:41:null}}`,

		`30:7:val16 $/ {30:40:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimprefix {30:40 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}}`),
		`30:7:val16 $(trim-prefix /**/testdata/**,$/) {30:10:delegate {30:12:builtin trim-prefix} {=list {=path {30:24:punct ROOT} {=glob {30:25:meta **}} {30:28:word testdata} {=glob {30:37:meta **}}}} {=list {30:40:delegate {1:1:def /}}}}`:`{30:12:builtin trim-prefix} {} {30:10 {30:40:null}}`,

		`31:7:val17 $/ {31:41:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimprefix {31:41 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}}`),
		`31:7:val17 $(trim-prefix /**/testdata/**/,$//) {31:10:delegate {31:12:builtin trim-prefix} {=list {=path {31:24:punct ROOT} {=glob {31:25:meta **}} {31:28:word testdata} {=glob {31:37:meta **}} {31:40:punct TAIL}}} {=list {=path {31:41:delegate {1:1:def /}} {31:44:punct TAIL}}}}`:`{31:12:builtin trim-prefix} {} {31:10 {31:41:null}}`,

		`33:7:val18 $/ {33:40:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimprefix {33:40 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}}`),
		`33:7:val18 $(trim-prefix *?/testdata/*?,$/) {33:10:delegate {33:12:builtin trim-prefix} {=list {=path {=glob {33:25:meta *?}} {33:28:word testdata} {=glob {33:37:meta *?}}}} {=list {33:40:delegate {1:1:def /}}}}`:`{33:12:builtin trim-prefix} {} {33:10 {33:40:null}}`,

		`34:7:val19 $/ {34:41:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimprefix {34:41 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}}`),
		`34:7:val19 $(trim-prefix *?/testdata/*?/,$//) {34:10:delegate {34:12:builtin trim-prefix} {=list {=path {=glob {34:25:meta *?}} {34:28:word testdata} {=glob {34:37:meta *?}} {34:40:punct TAIL}}} {=list {=path {34:41:delegate {1:1:def /}} {34:44:punct TAIL}}}}`:`{34:12:builtin trim-prefix} {} {34:10 {34:41:null}}`,

		`35:7:val20 $/ {35:40:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimprefix {35:40 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}}`),
		`35:7:val20 $(trim-prefix /*?/testdata/*?,$/) {35:10:delegate {35:12:builtin trim-prefix} {=list {=path {35:24:punct ROOT} {=glob {35:25:meta *?}} {35:28:word testdata} {=glob {35:37:meta *?}}}} {=list {35:40:delegate {1:1:def /}}}}`:`{35:12:builtin trim-prefix} {} {35:10 {35:40:null}}`,

		`36:7:val21 $/ {36:41:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimprefix {36:41 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}}`),
		`36:7:val21 $(trim-prefix /*?/testdata/*?/,$//) {36:10:delegate {36:12:builtin trim-prefix} {=list {=path {36:24:punct ROOT} {=glob {36:25:meta *?}} {36:28:word testdata} {=glob {36:37:meta *?}} {36:40:punct TAIL}}} {=list {=path {36:41:delegate {1:1:def /}} {36:44:punct TAIL}}}}`:`{36:12:builtin trim-prefix} {} {36:10 {36:41:null}}`,

		`37:7:val22 $/ {37:38:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimprefix {37:38 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}}`),
		`37:7:val22 $(trim-prefix /*?/,$//) {37:10:delegate {37:12:builtin trim-prefix} {=list {=path {37:33:punct ROOT} {=glob {37:34:meta *?}} {37:37:punct TAIL}}} {=list {=path {37:38:delegate {1:1:def /}} {37:41:punct TAIL}}}}`:`{37:12:builtin trim-prefix} {} {37:10 {37:38:null}}`,

		`38:7:val23 $/ {38:38:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimprefix {38:38 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}}`),
		`38:7:val23 $(trim-prefix /*?/test*/*?/,$//) {38:10:delegate {38:12:builtin trim-prefix} {=list {=path {38:24:punct ROOT} {=glob {38:25:meta *?}} {=glob {38:28:word test} {38:32:meta *}} {=glob {38:34:meta *?}} {38:37:punct TAIL}}} {=list {=path {38:38:delegate {1:1:def /}} {38:41:punct TAIL}}}}`:`{38:12:builtin trim-prefix} {} {38:10 {38:38:null}}`,

		`39:7:val24 $/ {39:38:delegate {1:1:def /}}`:`{1:1:def /} /Volumes/workspace/go/src/extbit.io/smart/testdata/builtins/trimprefix {39:38 {=path {1:1:punct ROOT} {1:1:word Volumes} {1:1:word workspace} {1:1:word go} {1:1:word src} {1:1:word extbit.io} {1:1:word smart} {1:1:word testdata} {1:1:word builtins} {1:1:word trimprefix}}}`,
		`39:7:val24 $(trim-prefix /*?/test*/**/,$//) {39:10:delegate {39:12:builtin trim-prefix} {=list {=path {39:24:punct ROOT} {=glob {39:25:meta *?}} {=glob {39:28:word test} {39:32:meta *}} {=glob {39:34:meta **}} {39:37:punct TAIL}}} {=list {=path {39:38:delegate {1:1:def /}} {39:41:punct TAIL}}}}`:`{39:12:builtin trim-prefix} {} {39:10 {39:38:null}}`,

		`40:7:val25 $/ {40:38:delegate {1:1:def /}}`:`{1:1:def /} /Volumes/workspace/go/src/extbit.io/smart/testdata/builtins/trimprefix {40:38 {=path {1:1:punct ROOT} {1:1:word Volumes} {1:1:word workspace} {1:1:word go} {1:1:word src} {1:1:word extbit.io} {1:1:word smart} {1:1:word testdata} {1:1:word builtins} {1:1:word trimprefix}}}`,
		`40:7:val25 $(trim-prefix /*?/*data/*?/,$//) {40:10:delegate {40:12:builtin trim-prefix} {=list {=path {40:24:punct ROOT} {=glob {40:25:meta *?}} {=glob {40:28:meta *} {40:29:word data}} {=glob {40:34:meta *?}} {40:37:punct TAIL}}} {=list {=path {40:38:delegate {1:1:def /}} {40:41:punct TAIL}}}}`:`{40:12:builtin trim-prefix} {} {40:10 {40:38:null}}`,

		`41:7:val26 $/ {41:38:delegate {1:1:def /}}`:`{1:1:def /} /Volumes/workspace/go/src/extbit.io/smart/testdata/builtins/trimprefix {41:38 {=path {1:1:punct ROOT} {1:1:word Volumes} {1:1:word workspace} {1:1:word go} {1:1:word src} {1:1:word extbit.io} {1:1:word smart} {1:1:word testdata} {1:1:word builtins} {1:1:word trimprefix}}}`,
		`41:7:val26 $(trim-prefix /**/*data/*?/,$//) {41:10:delegate {41:12:builtin trim-prefix} {=list {=path {41:24:punct ROOT} {=glob {41:25:meta **}} {=glob {41:28:meta *} {41:29:word data}} {=glob {41:34:meta *?}} {41:37:punct TAIL}}} {=list {=path {41:38:delegate {1:1:def /}} {41:41:punct TAIL}}}}`:`{41:12:builtin trim-prefix} {} {41:10 {41:38:null}}`,

		`42:7:val27 $/ {42:36:delegate {1:1:def /}}`:`{1:1:def /} /Volumes/workspace/go/src/extbit.io/smart/testdata/builtins/trimprefix {42:36 {=path {1:1:punct ROOT} {1:1:word Volumes} {1:1:word workspace} {1:1:word go} {1:1:word src} {1:1:word extbit.io} {1:1:word smart} {1:1:word testdata} {1:1:word builtins} {1:1:word trimprefix}}}`,
		`42:7:val27 $(trim-prefix /*?/t*a/*?/,$//) {42:10:delegate {42:12:builtin trim-prefix} {=list {=path {42:24:punct ROOT} {=glob {42:25:meta *?}} {=glob {42:28:word t} {42:29:meta *} {42:30:word a}} {=glob {42:32:meta *?}} {42:35:punct TAIL}}} {=list {=path {42:36:delegate {1:1:def /}} {42:39:punct TAIL}}}}`:`{42:12:builtin trim-prefix} {} {42:10 {42:36:null}}`,
	},
}

var checkstrs__trimprefix = map[string]map[string]any{
	"loader.go": map[string]any{
		`10:6:val1 $/ {10:39:delegate {1:1:def /}}`:testdata_fmt(`{10:39 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}} %[1]s/builtins/trimprefix`),
		`10:6:val1 $(dir $/) {10:32:delegate {10:34:builtin dir} {=list {10:39:delegate {1:1:def /}}}}`:`{10:32 {=path {10:34:punct ROOT} {10:34:word Volumes} {10:34:word workspace} {10:34:word go} {10:34:word src} {10:34:word extbit.io} {10:34:word smart} {10:34:word testdata} {10:34:word builtins}}} %[1]s/builtins`,
		`10:6:val1 $/ {10:43:delegate {1:1:def /}}`:`{10:43 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}} %[1]s/builtins/trimprefix`,

		`11:6:val2 $(pat0) {11:23:delegate {5:6:def pat0}}`:`{11:23 {=path {=glob {5:10:meta **}} {5:13:word testdata}}} **/testdata`,
		`11:6:val2 $/ {11:39:delegate {1:1:def /}}`:testdata_fmt(`{11:39 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}} %[1]s/builtins/trimprefix`),
		`11:6:val2 $/ {11:43:delegate {1:1:def /}}`:testdata_fmt(`{11:43 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}} %[1]s/builtins/trimprefix`),

		`12:6:val3 $(pat0) {12:23:delegate {5:6:def pat0}}`:`{12:23 {=path {=glob {5:10:meta **}} {5:13:word testdata}}} **/testdata`,
		`12:6:val3 $/ {12:32:delegate {1:1:def /}}`:testdata_fmt(`{12:32 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}} %[1]s/builtins/trimprefix`),

		`13:6:val4 $(pat1) {13:23:delegate {6:6:def pat1}}`:`{13:23 {=path {=percpat {6:10} {=percpat {6:12} {6:12}}} {6:13:word testdata}}} %%/testdata`,
		`13:6:val4 $/ {13:32:delegate {1:1:def /}}`:testdata_fmt(`{13:32 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}} %[1]s/builtins/trimprefix`),

		`14:6:val5 $(pat2) {14:23:delegate {7:6:def pat2}}`:`{14:23 {=path {7:9:punct ROOT} {=glob {7:10:meta **}} {7:13:word testdata}}} /**/testdata`,
		`14:6:val5 $/ {14:32:delegate {1:1:def /}}`:testdata_fmt(`{14:32 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}} %[1]s/builtins/trimprefix`),

		`15:6:val6 $(pat3) {15:23:delegate {8:6:def pat3}}`:`{15:23 {=path {8:9:punct ROOT} {=percpat {8:10} {=percpat {8:12} {8:12}}} {8:13:word testdata}}} /%%/testdata`,
		`15:6:val6 $/ {15:32:delegate {1:1:def /}}`:testdata_fmt(`{15:32 {=path %[3]s {1:1:word builtins} {1:1:word trimprefix}}} %[1]s/builtins/trimprefix`),
	},
}
