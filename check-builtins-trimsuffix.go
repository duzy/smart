//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints__trimsuffix = map[string]map[string]any{
	"loader.go": map[string]any{
		`3:3:d $/ {3:13:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimsuffix {3:13 {=path %[3]s {1:1:word builtins} {1:1:word trimsuffix}}}`),
		`3:3:d $(dir3 $/) {3:6:delegate {3:8:builtin dir3} {=list {3:13:delegate {1:1:def /}}}}`:testdata_fmt(`{3:8:builtin dir3} %[1]s {3:6 {=path %[3]s}}`,trim_suffix{1,"/testdata"},trim_suffix{3," {3:8:word testdata}"},line_column_s("3:8")),

		`4:3:p $/ {4:6:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimsuffix {4:6 {=path %[3]s {1:1:word builtins} {1:1:word trimsuffix}}}`),

		`11:6:val1 $/ {11:40:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimsuffix {11:40 {=path %[3]s {1:1:word builtins} {1:1:word trimsuffix}}}`),
		`11:6:val1 $/ {11:44:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimsuffix {11:44 {=path %[3]s {1:1:word builtins} {1:1:word trimsuffix}}}`),
		`11:6:val1 $(base3 $/) {11:32:delegate {11:34:builtin base3} {=list {11:40:delegate {1:1:def /}}}}`:`{11:34:builtin base3} testdata/builtins/trimsuffix {11:32 {=path {11:40:word testdata} {11:40:word builtins} {11:40:word trimsuffix}}}`,
		`11:6:val1 $(trim-suffix $(base3 $/),$/) {11:9:delegate {11:11:builtin trim-suffix} {=list {11:32:delegate {11:34:builtin base3} {=list {11:40:delegate {1:1:def /}}}}} {=list {11:44:delegate {1:1:def /}}}}`:testdata_fmt(`{11:11:builtin trim-suffix} %[1]s {11:9 {=path %[3]s {11:11:punct TAIL}}}`,trim_suffix{1,"testdata"},trim_suffix{3," {11:11:word testdata}"},line_column_s("11:11")),

		`12:6:val2 $(pat0) {12:24:delegate {6:6:def pat0}}`:`{6:6:def pat0} testdata/** {12:24 {=path {6:9:word testdata} {=glob {6:18:meta **}}}}`,
		`12:6:val2 $/ {12:40:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimsuffix {12:40 {=path %[3]s {1:1:word builtins} {1:1:word trimsuffix}}}`),
		`12:6:val2 $/ {12:44:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimsuffix {12:44 {=path %[3]s {1:1:word builtins} {1:1:word trimsuffix}}}`),
		`12:6:val2 $(base2 $/) {12:32:delegate {12:34:builtin base2} {=list {12:40:delegate {1:1:def /}}}}`:`{12:34:builtin base2} builtins/trimsuffix {12:32 {=path {12:40:word builtins} {12:40:word trimsuffix}}}`,
		`12:6:val2 $(trim-suffix $(pat0) $(base2 $/),$/) {12:9:delegate {12:11:builtin trim-suffix} {=list {12:24:delegate {6:6:def pat0}} {12:32:delegate {12:34:builtin base2} {=list {12:40:delegate {1:1:def /}}}}} {=list {12:44:delegate {1:1:def /}}}}`:testdata_fmt(`{12:11:builtin trim-suffix} %[1]s {12:9 {=path %[3]s {12:11:punct TAIL}}}`,trim_suffix{1,"testdata"},trim_suffix{3," {12:11:word testdata}"},line_column_s("12:11")),

		`13:6:val3 $(pat0) {13:24:delegate {6:6:def pat0}}`:`{6:6:def pat0} testdata/** {13:24 {=path {6:9:word testdata} {=glob {6:18:meta **}}}}`,
		`13:6:val3 $/ {13:32:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimsuffix {13:32 {=path %[3]s {1:1:word builtins} {1:1:word trimsuffix}}}`),
		`13:6:val3 $(trim-suffix $(pat0),$/) {13:9:delegate {13:11:builtin trim-suffix} {=list {13:24:delegate {6:6:def pat0}}} {=list {13:32:delegate {1:1:def /}}}}`:testdata_fmt(`{13:11:builtin trim-suffix} %[1]s {13:9 {=path %[3]s {13:11:punct TAIL}}}`,trim_suffix{1,"testdata"},trim_suffix{3," {13:11:word testdata}"},line_column_s("13:11")),

		`14:6:val4 $(pat1) {14:24:delegate {7:6:def pat1}}`:`{7:6:def pat1} testdata/%% {14:24 {=path {7:9:word testdata} {=percpat {7:18} {=percpat {7:20} {7:20}}}}}`,
		`14:6:val4 $/ {14:32:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimsuffix {14:32 {=path %[3]s {1:1:word builtins} {1:1:word trimsuffix}}}`),
		`14:6:val4 $(trim-suffix /$(pat1),$/) {14:9:delegate {14:11:builtin trim-suffix} {=list {=path {14:23:punct ROOT} {14:24:delegate {7:6:def pat1}}}} {=list {14:32:delegate {1:1:def /}}}}`:testdata_fmt(`{14:11:builtin trim-suffix} %[1]s {14:9 {=path %[3]s}}`,trim_suffix{1,"/testdata"},trim_suffix{3," {14:11:word testdata}"},line_column_s("14:11")),

		`15:6:val5 $(pat2) {15:24:delegate {8:6:def pat2}}`:`{8:6:def pat2} testdata/**/ {15:24 {=path {8:9:word testdata} {=glob {8:18:meta **}} {8:21:punct TAIL}}}`,
		`15:6:val5 $/ {15:32:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimsuffix {15:32 {=path %[3]s {1:1:word builtins} {1:1:word trimsuffix}}}`),
		`15:6:val5 $(trim-suffix /$(pat2),$//) {15:9:delegate {15:11:builtin trim-suffix} {=list {=path {15:23:punct ROOT} {15:24:delegate {8:6:def pat2}}}} {=list {=path {15:32:delegate {1:1:def /}} {15:35:punct TAIL}}}}`:testdata_fmt(`{15:11:builtin trim-suffix} %[1]s {15:9 {=path %[3]s}}`,trim_suffix{1,"/testdata"},trim_suffix{3," {15:11:word testdata}"},line_column_s("15:11")),

		`16:6:val6 $(pat3) {16:24:delegate {9:6:def pat3}}`:`{9:6:def pat3} testdata/%%/ {16:24 {=path {9:9:word testdata} {=percpat {9:18} {=percpat {9:20} {9:20}}} {9:21:punct TAIL}}}`,
		`16:6:val6 $/ {16:32:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimsuffix {16:32 {=path %[3]s {1:1:word builtins} {1:1:word trimsuffix}}}`),
		`16:6:val6 $(trim-suffix /$(pat3),$//) {16:9:delegate {16:11:builtin trim-suffix} {=list {=path {16:23:punct ROOT} {16:24:delegate {9:6:def pat3}}}} {=list {=path {16:32:delegate {1:1:def /}} {16:35:punct TAIL}}}}`:testdata_fmt(`{16:11:builtin trim-suffix} %[1]s {16:9 {=path %[3]s}}`,trim_suffix{1,"/testdata"},trim_suffix{3," {16:11:word testdata}"},line_column_s("16:11")),

		`18:7:val7 $/ {18:40:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimsuffix {18:40 {=path %[3]s {1:1:word builtins} {1:1:word trimsuffix}}}`),
		`18:7:val7 $(trim-suffix **/testdata/**,$/) {18:10:delegate {18:12:builtin trim-suffix} {=list {=path {=glob {18:25:meta **}} {18:28:word testdata} {=glob {18:37:meta **}}}} {=list {18:40:delegate {1:1:def /}}}}`:`{18:12:builtin trim-suffix} {} {18:10 {18:40:null}}`,

		`19:7:val8 $/ {19:41:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimsuffix {19:41 {=path %[3]s {1:1:word builtins} {1:1:word trimsuffix}}}`),
		`19:7:val8 $(trim-suffix **/testdata/**/,$//) {19:10:delegate {19:12:builtin trim-suffix} {=list {=path {=glob {19:25:meta **}} {19:28:word testdata} {=glob {19:37:meta **}} {19:40:punct TAIL}}} {=list {=path {19:41:delegate {1:1:def /}} {19:44:punct TAIL}}}}`:`{19:12:builtin trim-suffix} {} {19:10 {19:41:null}}`,

		`20:7:val9 $/ {20:40:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimsuffix {20:40 {=path %[3]s {1:1:word builtins} {1:1:word trimsuffix}}}`),
		`20:7:val9 $(trim-suffix /**/testdata/**,$/) {20:10:delegate {20:12:builtin trim-suffix} {=list {=path {20:24:punct ROOT} {=glob {20:25:meta **}} {20:28:word testdata} {=glob {20:37:meta **}}}} {=list {20:40:delegate {1:1:def /}}}}`:`{20:12:builtin trim-suffix} {} {20:10 {20:40:null}}`,

		`21:7:val10 $/ {21:41:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimsuffix {21:41 {=path %[3]s {1:1:word builtins} {1:1:word trimsuffix}}}`),
		`21:7:val10 $(trim-suffix /**/testdata/**/,$//) {21:10:delegate {21:12:builtin trim-suffix} {=list {=path {21:24:punct ROOT} {=glob {21:25:meta **}} {21:28:word testdata} {=glob {21:37:meta **}} {21:40:punct TAIL}}} {=list {=path {21:41:delegate {1:1:def /}} {21:44:punct TAIL}}}}`:`{21:12:builtin trim-suffix} {} {21:10 {21:41:null}}`,

		`22:7:val11 $/ {22:40:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimsuffix {22:40 {=path %[3]s {1:1:word builtins} {1:1:word trimsuffix}}}`),
		`22:7:val11 $(trim-suffix testdata/**,$/) {22:10:delegate {22:12:builtin trim-suffix} {=list {=path {22:28:word testdata} {=glob {22:37:meta **}}}} {=list {22:40:delegate {1:1:def /}}}}`:testdata_fmt(`{22:12:builtin trim-suffix} %[1]s {22:10 {=path %[3]s {22:12:punct TAIL}}}`,trim_suffix{1,"testdata"},trim_suffix{3," {22:12:word testdata}"},line_column_s("22:12")),

		`23:7:val12 $/ {23:40:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimsuffix {23:40 {=path %[3]s {1:1:word builtins} {1:1:word trimsuffix}}}`),
		`23:7:val12 $(trim-suffix /testdata/**,$/) {23:10:delegate {23:12:builtin trim-suffix} {=list {=path {23:27:punct ROOT} {23:28:word testdata} {=glob {23:37:meta **}}}} {=list {23:40:delegate {1:1:def /}}}}`:testdata_fmt(`{23:12:builtin trim-suffix} %[1]s {23:10 {=path %[3]s}}`,trim_suffix{1,"/testdata"},trim_suffix{3," {23:12:word testdata}"},line_column_s("23:12")),

		`24:7:val13 $/ {24:41:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimsuffix {24:41 {=path %[3]s {1:1:word builtins} {1:1:word trimsuffix}}}`),
		`24:7:val13 $(trim-suffix testdata/**/,$//) {24:10:delegate {24:12:builtin trim-suffix} {=list {=path {24:28:word testdata} {=glob {24:37:meta **}} {24:40:punct TAIL}}} {=list {=path {24:41:delegate {1:1:def /}} {24:44:punct TAIL}}}}`:testdata_fmt(`{24:12:builtin trim-suffix} %[1]s {24:10 {=path %[3]s {24:12:punct TAIL}}}`,trim_suffix{1,"testdata"},trim_suffix{3," {24:12:word testdata}"},line_column_s("24:12")),

		`25:7:val14 $/ {25:41:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimsuffix {25:41 {=path %[3]s {1:1:word builtins} {1:1:word trimsuffix}}}`),
		`25:7:val14 $(trim-suffix /testdata/**/,$//) {25:10:delegate {25:12:builtin trim-suffix} {=list {=path {25:27:punct ROOT} {25:28:word testdata} {=glob {25:37:meta **}} {25:40:punct TAIL}}} {=list {=path {25:41:delegate {1:1:def /}} {25:44:punct TAIL}}}}`:testdata_fmt(`{25:12:builtin trim-suffix} %[1]s {25:10 {=path %[3]s}}`,trim_suffix{1,"/testdata"},trim_suffix{3," {25:12:word testdata}"},line_column_s("25:12")),

		`26:7:val15 $/ {26:41:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimsuffix {26:41 {=path %[3]s {1:1:word builtins} {1:1:word trimsuffix}}}`),
		`26:7:val15 $(trim-suffix /**/,$//) {26:10:delegate {26:12:builtin trim-suffix} {=list {=path {26:36:punct ROOT} {=glob {26:37:meta **}} {26:40:punct TAIL}}} {=list {=path {26:41:delegate {1:1:def /}} {26:44:punct TAIL}}}}`:`{26:12:builtin trim-suffix} {} {26:10 {26:41:null}}`,

		`28:7:val16 $/ {28:40:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimsuffix {28:40 {=path %[3]s {1:1:word builtins} {1:1:word trimsuffix}}}`),
		`28:7:val16 $(trim-suffix *?/testdata/*?,$/) {28:10:delegate {28:12:builtin trim-suffix} {=list {=path {=glob {28:25:meta *?}} {28:28:word testdata} {=glob {28:37:meta *?}}}} {=list {28:40:delegate {1:1:def /}}}}`:`{28:12:builtin trim-suffix} {} {28:10 {28:40:null}}`,

		`29:7:val17 $/ {29:41:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimsuffix {29:41 {=path %[3]s {1:1:word builtins} {1:1:word trimsuffix}}}`),
		`29:7:val17 $(trim-suffix *?/testdata/*?/,$//) {29:10:delegate {29:12:builtin trim-suffix} {=list {=path {=glob {29:25:meta *?}} {29:28:word testdata} {=glob {29:37:meta *?}} {29:40:punct TAIL}}} {=list {=path {29:41:delegate {1:1:def /}} {29:44:punct TAIL}}}}`:`{29:12:builtin trim-suffix} {} {29:10 {29:41:null}}`,

		`30:7:val18 $/ {30:40:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimsuffix {30:40 {=path %[3]s {1:1:word builtins} {1:1:word trimsuffix}}}`),
		`30:7:val18 $(trim-suffix /*?/testdata/*?,$/) {30:10:delegate {30:12:builtin trim-suffix} {=list {=path {30:24:punct ROOT} {=glob {30:25:meta *?}} {30:28:word testdata} {=glob {30:37:meta *?}}}} {=list {30:40:delegate {1:1:def /}}}}`:`{30:12:builtin trim-suffix} {} {30:10 {30:40:null}}`,

		`31:7:val19 $/ {31:41:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimsuffix {31:41 {=path %[3]s {1:1:word builtins} {1:1:word trimsuffix}}}`),
		`31:7:val19 $(trim-suffix /*?/testdata/*?/,$//) {31:10:delegate {31:12:builtin trim-suffix} {=list {=path {31:24:punct ROOT} {=glob {31:25:meta *?}} {31:28:word testdata} {=glob {31:37:meta *?}} {31:40:punct TAIL}}} {=list {=path {31:41:delegate {1:1:def /}} {31:44:punct TAIL}}}}`:`{31:12:builtin trim-suffix} {} {31:10 {31:41:null}}`,

		`32:7:val20 $/ {32:38:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimsuffix {32:38 {=path %[3]s {1:1:word builtins} {1:1:word trimsuffix}}}`),
		`32:7:val20 $(trim-suffix /*?/,$//) {32:10:delegate {32:12:builtin trim-suffix} {=list {=path {32:33:punct ROOT} {=glob {32:34:meta *?}} {32:37:punct TAIL}}} {=list {=path {32:38:delegate {1:1:def /}} {32:41:punct TAIL}}}}`:`{32:12:builtin trim-suffix} {} {32:10 {32:38:null}}`,

		`33:7:val21 $/ {33:38:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimsuffix {33:38 {=path %[3]s {1:1:word builtins} {1:1:word trimsuffix}}}`),
		`33:7:val21 $(trim-suffix /*?/test*/*?/,$//) {33:10:delegate {33:12:builtin trim-suffix} {=list {=path {33:24:punct ROOT} {=glob {33:25:meta *?}} {=glob {33:28:word test} {33:32:meta *}} {=glob {33:34:meta *?}} {33:37:punct TAIL}}} {=list {=path {33:38:delegate {1:1:def /}} {33:41:punct TAIL}}}}`:`{33:12:builtin trim-suffix} {} {33:10 {33:38:null}}`,

		`34:7:val22 $/ {34:38:delegate {1:1:def /}}`:testdata_fmt(`{1:1:def /} %[1]s/builtins/trimsuffix {34:38 {=path %[3]s {1:1:word builtins} {1:1:word trimsuffix}}}`),
		`34:7:val22 $(trim-suffix /*?/test*/**/,$//) {34:10:delegate {34:12:builtin trim-suffix} {=list {=path {34:24:punct ROOT} {=glob {34:25:meta *?}} {=glob {34:28:word test} {34:32:meta *}} {=glob {34:34:meta **}} {34:37:punct TAIL}}} {=list {=path {34:38:delegate {1:1:def /}} {34:41:punct TAIL}}}}`:`{34:12:builtin trim-suffix} {} {34:10 {34:38:null}}`,

		`35:7:val23 $/ {35:38:delegate {1:1:def /}}`:`{1:1:def /} /Volumes/workspace/go/src/extbit.io/smart/testdata/builtins/trimsuffix {35:38 {=path {1:1:punct ROOT} {1:1:word Volumes} {1:1:word workspace} {1:1:word go} {1:1:word src} {1:1:word extbit.io} {1:1:word smart} {1:1:word testdata} {1:1:word builtins} {1:1:word trimsuffix}}}`,
		`35:7:val23 $(trim-suffix /*?/*data/*?/,$//) {35:10:delegate {35:12:builtin trim-suffix} {=list {=path {35:24:punct ROOT} {=glob {35:25:meta *?}} {=glob {35:28:meta *} {35:29:word data}} {=glob {35:34:meta *?}} {35:37:punct TAIL}}} {=list {=path {35:38:delegate {1:1:def /}} {35:41:punct TAIL}}}}`:`{35:12:builtin trim-suffix} {} {35:10 {35:38:null}}`,

		`36:7:val24 $/ {36:38:delegate {1:1:def /}}`:`{1:1:def /} /Volumes/workspace/go/src/extbit.io/smart/testdata/builtins/trimsuffix {36:38 {=path {1:1:punct ROOT} {1:1:word Volumes} {1:1:word workspace} {1:1:word go} {1:1:word src} {1:1:word extbit.io} {1:1:word smart} {1:1:word testdata} {1:1:word builtins} {1:1:word trimsuffix}}}`,
		`36:7:val24 $(trim-suffix /**/*data/*?/,$//) {36:10:delegate {36:12:builtin trim-suffix} {=list {=path {36:24:punct ROOT} {=glob {36:25:meta **}} {=glob {36:28:meta *} {36:29:word data}} {=glob {36:34:meta *?}} {36:37:punct TAIL}}} {=list {=path {36:38:delegate {1:1:def /}} {36:41:punct TAIL}}}}`:`{36:12:builtin trim-suffix} {} {36:10 {36:38:null}}`,

		`37:7:val25 $/ {37:36:delegate {1:1:def /}}`:`{1:1:def /} /Volumes/workspace/go/src/extbit.io/smart/testdata/builtins/trimsuffix {37:36 {=path {1:1:punct ROOT} {1:1:word Volumes} {1:1:word workspace} {1:1:word go} {1:1:word src} {1:1:word extbit.io} {1:1:word smart} {1:1:word testdata} {1:1:word builtins} {1:1:word trimsuffix}}}`,
		`37:7:val25 $(trim-suffix /*?/t*a/*?/,$//) {37:10:delegate {37:12:builtin trim-suffix} {=list {=path {37:24:punct ROOT} {=glob {37:25:meta *?}} {=glob {37:28:word t} {37:29:meta *} {37:30:word a}} {=glob {37:32:meta *?}} {37:35:punct TAIL}}} {=list {=path {37:36:delegate {1:1:def /}} {37:39:punct TAIL}}}}`:`{37:12:builtin trim-suffix} {} {37:10 {37:36:null}}`,
	},
}

var checkstrs__trimsuffix = map[string]map[string]any{
	"loader.go": map[string]any{
		`3:3:d $/ {3:13:delegate {1:1:def /}}`:testdata_fmt(`{3:13 {=path %[3]s {1:1:word builtins} {1:1:word trimsuffix}}} %[1]s/builtins/trimsuffix`),
		`11:6:val1 $/ {11:40:delegate {1:1:def /}}`:testdata_fmt(`{11:40 {=path %[3]s {1:1:word builtins} {1:1:word trimsuffix}}} %[1]s/builtins/trimsuffix`),
		`12:6:val2 $/ {12:40:delegate {1:1:def /}}`:testdata_fmt(`{12:40 {=path %[3]s {1:1:word builtins} {1:1:word trimsuffix}}} %[1]s/builtins/trimsuffix`),
	},
}
