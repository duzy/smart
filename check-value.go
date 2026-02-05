//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints_value = map[string]map[string]any{
	"loader.go": map[string]any{
		`configure.types {=compound {52:1:word configure} {52:10:punct .} {52:11:word types}}`:`configure.types {=compound {52:1:word configure} {52:10:punct .} {52:11:word types}}`,
		`configure.types."atomic.h" {=compound {49:1:word configure} {49:10:punct .} {49:11:word types} {49:16:punct .} {=strcomp {49:18:raw atomic.h}}}`:`configure.types."atomic.h" {=compound {49:1:word configure} {49:10:punct .} {49:11:word types} {49:16:punct .} {=strcomp {49:18:raw atomic.h}}}`,
		`configure.types.<atomic.h> {=compound {50:1:word configure} {50:10:punct .} {50:11:word types} {50:16:punct .} {50:17:punct <} {50:18:word atomic} {50:24:punct .} {50:25:word h} {50:26:punct >}}`:`configure.types.<atomic.h> {=compound {50:1:word configure} {50:10:punct .} {50:11:word types} {50:16:punct .} {50:17:punct <} {50:18:word atomic} {50:24:punct .} {50:25:word h} {50:26:punct >}}`,

		`13:8:cond11 x?y {=compound {=cond {13:11:word x}} {13:13:word y}}`:`xy? {=cond {=compound {13:11:word x} {13:13:word y}}}`,
		`14:8:cond12 x???y {=compound {=cond {=cond {=cond {14:11:word x}}}} {14:15:word y}}`:`xy??? {=cond {=cond {=cond {=compound {14:11:word x} {14:15:word y}}}}}`,

		`15:8:cond13 &(something) {15:12:closure {15:14:word something}}`:`&(something) {15:12:closure {15:14:word something}}`,
		`15:8:cond13 x&(something) {=compound {15:11:word x} {15:12:closure {15:14:word something}}}`:`x&(something) {=compound {15:11:word x} {15:12:closure {15:14:word something}}}`,
		`15:8:cond13 x&(something)?y {=compound {=cond {=compound {15:11:word x} {15:12:closure {15:14:word something}}}} {15:25:word y}}`:`x&(something)y? {=cond {=compound {15:11:word x} {15:12:closure {15:14:word something}} {15:25:word y}}}`,

		`22:15:disjunction1 x{a b c}y{1 2 3}z {=compound {22:18:word x} {22:19:disjunction {=list {22:20:word a} {22:22:word b} {22:24:word c}}} {22:26:word y} {22:27:disjunction {=list {22:28:decimal 1} {22:30:decimal 2} {22:32:decimal 3}}} {22:34:word z}}`:`xay1z xay2z xay3z xby1z xby2z xby3z xcy1z xcy2z xcy3z {=list {=compound {22:18:word x} {22:20:word a} {22:26:word y} {22:28:decimal 1} {22:34:word z}} {=compound {22:18:word x} {22:20:word a} {22:26:word y} {22:30:decimal 2} {22:34:word z}} {=compound {22:18:word x} {22:20:word a} {22:26:word y} {22:32:decimal 3} {22:34:word z}} {=compound {22:18:word x} {22:22:word b} {22:26:word y} {22:28:decimal 1} {22:34:word z}} {=compound {22:18:word x} {22:22:word b} {22:26:word y} {22:30:decimal 2} {22:34:word z}} {=compound {22:18:word x} {22:22:word b} {22:26:word y} {22:32:decimal 3} {22:34:word z}} {=compound {22:18:word x} {22:24:word c} {22:26:word y} {22:28:decimal 1} {22:34:word z}} {=compound {22:18:word x} {22:24:word c} {22:26:word y} {22:30:decimal 2} {22:34:word z}} {=compound {22:18:word x} {22:24:word c} {22:26:word y} {22:32:decimal 3} {22:34:word z}}}`,
		`23:14:disjunction2 x{a b c}y{1 2 3}z {=compound {23:18:word x} {23:19:disjunction {=list {23:20:word a} {23:22:word b} {23:24:word c}}} {23:26:word y} {23:27:disjunction {=list {23:28:decimal 1} {23:30:decimal 2} {23:32:decimal 3}}} {23:34:word z}}`:`xay1z xay2z xay3z xby1z xby2z xby3z xcy1z xcy2z xcy3z {=list {=compound {23:18:word x} {23:20:word a} {23:26:word y} {23:28:decimal 1} {23:34:word z}} {=compound {23:18:word x} {23:20:word a} {23:26:word y} {23:30:decimal 2} {23:34:word z}} {=compound {23:18:word x} {23:20:word a} {23:26:word y} {23:32:decimal 3} {23:34:word z}} {=compound {23:18:word x} {23:22:word b} {23:26:word y} {23:28:decimal 1} {23:34:word z}} {=compound {23:18:word x} {23:22:word b} {23:26:word y} {23:30:decimal 2} {23:34:word z}} {=compound {23:18:word x} {23:22:word b} {23:26:word y} {23:32:decimal 3} {23:34:word z}} {=compound {23:18:word x} {23:24:word c} {23:26:word y} {23:28:decimal 1} {23:34:word z}} {=compound {23:18:word x} {23:24:word c} {23:26:word y} {23:30:decimal 2} {23:34:word z}} {=compound {23:18:word x} {23:24:word c} {23:26:word y} {23:32:decimal 3} {23:34:word z}}}`,

		`41:7:val4 a\,b\,c,x\,y\,z {=compound {41:18:word a} {41:19:escaped \,} {41:21:word b} {41:22:escaped \,} {41:24:word c} {41:25:punct ,} {41:26:word x} {41:27:escaped \,} {41:29:word y} {41:30:escaped \,} {41:32:word z}}`:`a\,b\,c,x\,y\,z {=compound {41:18:word a} {41:19:escaped \,} {41:21:word b} {41:22:escaped \,} {41:24:word c} {41:25:punct ,} {41:26:word x} {41:27:escaped \,} {41:29:word y} {41:30:escaped \,} {41:32:word z}}`,
		`42:7:val5 $(quote a\,b\,c,x\,y\,z) {42:10:delegate {42:12:builtin quote} {=list {=compound {42:18:word a} {42:19:escaped \,} {42:21:word b} {42:22:escaped \,} {42:24:word c}}} {=list {=compound {42:26:word x} {42:27:escaped \,} {42:29:word y} {42:30:escaped \,} {42:32:word z}}}}`:`{=quote a\,b\,c x\,y\,z} {42:10 {=quote {=list {=compound {42:18:word a} {42:19:escaped \,} {42:21:word b} {42:22:escaped \,} {42:24:word c}}} {=list {=compound {42:26:word x} {42:27:escaped \,} {42:29:word y} {42:30:escaped \,} {42:32:word z}}}}}`,

		`43:7:val6 extbit.io {=compound {43:17:word extbit} {43:23:punct .} {43:24:word io}}`:`extbit.io {=compound {43:17:word extbit} {43:23:punct .} {43:24:word io}}`,
		`44:7:val7 extbit.com {=compound {44:18:word extbit} {44:24:punct .} {44:25:word com}}`:`extbit.com {=compound {44:18:word extbit} {44:24:punct .} {44:25:word com}}`,
		`45:7:val8 extbit.com {=compound {45:18:word extbit} {45:24:punct .} {45:25:word com}}`:`extbit.com {=compound {45:18:word extbit} {45:24:punct .} {45:25:word com}}`,
		`46:7:val9 extbit.com {=compound {46:18:word extbit} {46:24:punct .} {46:25:word com}}`:`extbit.com {=compound {46:18:word extbit} {46:24:punct .} {46:25:word com}}`,
		`47:7:val10 ext.pub {=compound {47:18:word ext} {47:21:punct .} {47:22:word pub}}`:`ext.pub {=compound {47:18:word ext} {47:21:punct .} {47:22:word pub}}`,

		`47:7:val10 x%20y%20z {=compound {47:40:word x} {47:41:punct %} {47:42:decimal 20} {47:44:word y} {47:45:punct %} {47:46:decimal 20} {47:48:word z}}`:`x%20y%20z {=compound {47:40:word x} {47:41:punct %} {47:42:decimal 20} {47:44:word y} {47:45:punct %} {47:46:decimal 20} {47:48:word z}}`,

		`56:11:conf1 test.txt {=compound {56:51:word test} {56:55:punct .} {56:56:word txt}}`:`test.txt {=compound {56:51:word test} {56:55:punct .} {56:56:word txt}}`,
		`57:11:conf2 test.txt {=compound {57:100:word test} {57:104:punct .} {57:105:word txt}}`:`test.txt {=compound {57:100:word test} {57:104:punct .} {57:105:word txt}}`,

		`55:11:conf0 $@ {55:19:delegate {55:11:def @}}`:`conf0 {55:19 {55:11:word conf0}}`,

		`56:11:conf1 $/ {56:48:delegate {1:1:def /}}`:testdata_f(`%[1]s/value {56:48 {=path %[3]s {1:1:raw value}}}`),
		`57:11:conf2 $/ {57:97:delegate {1:1:def /}}`:testdata_f(`%[1]s/value {57:97 {=path %[3]s {1:1:raw value}}}`),

		`56:11:conf1 $0 {56:45:delegate {56:46:auto 0}}`:testdata_fs(
			`foo.o {56:45 {%[1]s/value/test.txt:1:1:raw foo.o}}`,
			`foo-x.o {56:45 {%[1]s/value/test.txt:2:1:raw foo-x.o}}`,
			`foo-x-y.o {56:45 {%[1]s/value/test.txt:3:1:raw foo-x-y.o}}`,
			`foo-x-y-z.o {56:45 {%[1]s/value/test.txt:4:1:raw foo-x-y-z.o}}`,
			`foobar.o {56:45 {%[1]s/value/test.txt:5:1:raw foobar.o}}`,
		),

		`56:11:conf1 $(grep {=regex ^.+?\.o$$},$0,$//test.txt) {56:19:delegate {56:21:builtin grep} {=list {56:34:regex ^.+?\.o$}} {=list {56:45:delegate {56:46:auto 0}}} {=list {=path {56:48:delegate {1:1:def /}} {=compound {56:51:word test} {56:55:punct .} {56:56:word txt}}}}}`:testdata_f(`foo.o foo-x.o foo-x-y.o foo-x-y-z.o foobar.o {=list `+
				`{56:19 {56:45 {%[1]s/value/test.txt:1:1:raw foo.o}}} `+
				`{56:19 {56:45 {%[1]s/value/test.txt:2:1:raw foo-x.o}}} `+
				`{56:19 {56:45 {%[1]s/value/test.txt:3:1:raw foo-x-y.o}}} `+
				`{56:19 {56:45 {%[1]s/value/test.txt:4:1:raw foo-x-y-z.o}}} `+
				`{56:19 {56:45 {%[1]s/value/test.txt:5:1:raw foobar.o}}}}`),
		`57:11:conf2 $(grep {=regex ^(.+?)((-(?P<i>.+?))*)(\.)(?P<x>o)$$},$0 $1 $2 $3 $(i) $5 $(x),$//test.txt) {57:19:delegate {57:21:builtin grep} {=list {57:34:regex ^(.+?)((-(?P<i>.+?))*)(\.)(?P<x>o)$}} {=list {57:72:delegate {56:46:auto 0}} {57:75:delegate {19:21:auto 1}} {57:78:delegate {57:79:auto 2}} {57:81:delegate {57:82:auto 3}} {57:84:delegate {57:86:auto i}} {57:89:delegate {57:90:auto 5}} {57:92:delegate {57:94:auto x}}} {=list {=path {57:97:delegate {1:1:def /}} {=compound {57:100:word test} {57:104:punct .} {57:105:word txt}}}}}`:testdata_f(`foo.o foo . o foo-x.o foo -x -x x . o foo-x-y.o foo -x-y -y y . o foo-x-y-z.o foo -x-y-z -z z . o foobar.o foobar . o {=list `+
				`{57:19 {57:72 {%[1]s/value/test.txt:1:1:raw foo.o}}} `+
				`{57:19 {57:75 {%[1]s/value/test.txt:1:1:raw foo}}} `+
				`{57:19 {57:78 {%[1]s/value/test.txt:1:4:raw}}} `+
				`{57:19 {57:81 {%[1]s/value/test.txt:1:4:raw}}} `+
				`{57:19 {57:84 {%[1]s/value/test.txt:1:4:raw}}} `+
				`{57:19 {57:89 {%[1]s/value/test.txt:1:4:raw .}}} `+
				`{57:19 {57:92 {%[1]s/value/test.txt:1:5:raw o}}} `+
				`{57:19 {57:72 {%[1]s/value/test.txt:2:1:raw foo-x.o}}} `+
				`{57:19 {57:75 {%[1]s/value/test.txt:2:1:raw foo}}} `+
				`{57:19 {57:78 {%[1]s/value/test.txt:2:4:raw -x}}} `+
				`{57:19 {57:81 {%[1]s/value/test.txt:2:4:raw -x}}} `+
				`{57:19 {57:84 {%[1]s/value/test.txt:2:5:raw x}}} `+
				`{57:19 {57:89 {%[1]s/value/test.txt:2:6:raw .}}} `+
				`{57:19 {57:92 {%[1]s/value/test.txt:2:7:raw o}}} `+
				`{57:19 {57:72 {%[1]s/value/test.txt:3:1:raw foo-x-y.o}}} `+
				`{57:19 {57:75 {%[1]s/value/test.txt:3:1:raw foo}}} `+
				`{57:19 {57:78 {%[1]s/value/test.txt:3:4:raw -x-y}}} `+
				`{57:19 {57:81 {%[1]s/value/test.txt:3:6:raw -y}}} `+
				`{57:19 {57:84 {%[1]s/value/test.txt:3:7:raw y}}} `+
				`{57:19 {57:89 {%[1]s/value/test.txt:3:8:raw .}}} `+
				`{57:19 {57:92 {%[1]s/value/test.txt:3:9:raw o}}} `+
				`{57:19 {57:72 {%[1]s/value/test.txt:4:1:raw foo-x-y-z.o}}} `+
				`{57:19 {57:75 {%[1]s/value/test.txt:4:1:raw foo}}} `+
				`{57:19 {57:78 {%[1]s/value/test.txt:4:4:raw -x-y-z}}} `+
				`{57:19 {57:81 {%[1]s/value/test.txt:4:8:raw -z}}} `+
				`{57:19 {57:84 {%[1]s/value/test.txt:4:9:raw z}}} `+
				`{57:19 {57:89 {%[1]s/value/test.txt:4:10:raw .}}} `+
				`{57:19 {57:92 {%[1]s/value/test.txt:4:11:raw o}}} `+
				`{57:19 {57:72 {%[1]s/value/test.txt:5:1:raw foobar.o}}} `+
				`{57:19 {57:75 {%[1]s/value/test.txt:5:1:raw foobar}}} `+
				`{57:19 {57:78 {%[1]s/value/test.txt:5:7:raw}}} `+
				`{57:19 {57:81 {%[1]s/value/test.txt:5:7:raw}}} `+
				`{57:19 {57:84 {%[1]s/value/test.txt:5:7:raw}}} `+
				`{57:19 {57:89 {%[1]s/value/test.txt:5:7:raw .}}} `+
				`{57:19 {57:92 {%[1]s/value/test.txt:5:8:raw o}}}}`),

		`57:11:conf2 $0 {57:72:delegate {56:46:auto 0}}`:testdata_fs(
			`foo.o {57:72 {%[1]s/value/test.txt:1:1:raw foo.o}}`,
			`foo-x.o {57:72 {%[1]s/value/test.txt:2:1:raw foo-x.o}}`,
			`foo-x-y.o {57:72 {%[1]s/value/test.txt:3:1:raw foo-x-y.o}}`,
			`foo-x-y-z.o {57:72 {%[1]s/value/test.txt:4:1:raw foo-x-y-z.o}}`,
			`foobar.o {57:72 {%[1]s/value/test.txt:5:1:raw foobar.o}}`,
		),
		`57:11:conf2 $1 {57:75:delegate {19:21:auto 1}}`:testdata_fs(
			`foo {57:75 {%[1]s/value/test.txt:1:1:raw foo}}`,
			`foo {57:75 {%[1]s/value/test.txt:2:1:raw foo}}`,
			`foo {57:75 {%[1]s/value/test.txt:3:1:raw foo}}`,
			`foo {57:75 {%[1]s/value/test.txt:4:1:raw foo}}`,
			`foobar {57:75 {%[1]s/value/test.txt:5:1:raw foobar}}`,
		),
		`57:11:conf2 $2 {57:78:delegate {57:79:auto 2}}`:testdata_fs(
			` {57:78 {%[1]s/value/test.txt:1:4:raw}}`,
			` {57:78 {%[1]s/value/test.txt:5:7:raw}}`,
			`-x {57:78 {%[1]s/value/test.txt:2:4:raw -x}}`,
			`-x-y {57:78 {%[1]s/value/test.txt:3:4:raw -x-y}}`,
			`-x-y-z {57:78 {%[1]s/value/test.txt:4:4:raw -x-y-z}}`,
		),
		`57:11:conf2 $3 {57:81:delegate {57:82:auto 3}}`:testdata_fs(
			` {57:81 {%[1]s/value/test.txt:1:4:raw}}`,
			` {57:81 {%[1]s/value/test.txt:5:7:raw}}`,
			`-x {57:81 {%[1]s/value/test.txt:2:4:raw -x}}`,
			`-y {57:81 {%[1]s/value/test.txt:3:6:raw -y}}`,
			`-z {57:81 {%[1]s/value/test.txt:4:8:raw -z}}`,
		),
		`57:11:conf2 $(i) {57:84:delegate {57:86:auto i}}`:testdata_fs(
			` {57:84 {%[1]s/value/test.txt:1:4:raw}}`,
			` {57:84 {%[1]s/value/test.txt:5:7:raw}}`,
			`x {57:84 {%[1]s/value/test.txt:2:5:raw x}}`,
			`y {57:84 {%[1]s/value/test.txt:3:7:raw y}}`,
			`z {57:84 {%[1]s/value/test.txt:4:9:raw z}}`,
		),
		`57:11:conf2 $5 {57:89:delegate {57:90:auto 5}}`:testdata_fs(
			`. {57:89 {%[1]s/value/test.txt:1:4:raw .}}`,
			`. {57:89 {%[1]s/value/test.txt:2:6:raw .}}`,
			`. {57:89 {%[1]s/value/test.txt:3:8:raw .}}`,
			`. {57:89 {%[1]s/value/test.txt:4:10:raw .}}`,
			`. {57:89 {%[1]s/value/test.txt:5:7:raw .}}`,
		),
		`57:11:conf2 $(x) {57:92:delegate {57:94:auto x}}`:testdata_fs(
			`o {57:92 {%[1]s/value/test.txt:1:5:raw o}}`,
			`o {57:92 {%[1]s/value/test.txt:2:7:raw o}}`,
			`o {57:92 {%[1]s/value/test.txt:3:9:raw o}}`,
			`o {57:92 {%[1]s/value/test.txt:4:11:raw o}}`,
			`o {57:92 {%[1]s/value/test.txt:5:8:raw o}}`,
		),

		`59:11:conf3 $@, {=compound {59:29:delegate {59:11:def @}} {59:31:punct ,}}`:`conf3, {=compound {59:29 {59:11:word conf3}} {59:31:punct ,}}`,
		`59:11:conf3 $>, {=compound {59:36:delegate {1:1:def >}} {59:38:punct ,}}`:`bar, {=compound {59:36 {59:23:word bar}} {59:38:punct ,}}`,
		`59:11:conf3 $@ {59:29:delegate {59:11:def @}}`:`conf3 {59:29 {59:11:word conf3}}`,
		`59:11:conf3 $< {59:33:delegate {1:1:def <}}`:`foo {59:33 {59:19:word foo}}`,
		`59:11:conf3 $> {59:36:delegate {1:1:def >}}`:`bar {59:36 {59:23:word bar}}`,
		`59:11:conf3 $^ {59:40:delegate {1:1:def ^}}`:`foo bar {=list {59:40 {59:19:word foo}} {59:40 {59:23:word bar}}}`,
	},
	"check-value_test.go": map[string]any{
		`62 9:9:cond01 x?y {=compound {=cond {9:11:word x}} {9:13:word y}}`:`xy? {=cond {=compound {9:11:word x} {9:13:word y}}}`,
		`62 9:9:cond01 xy {=compound {9:11:word x} {9:13:word y}}`:`xy {=compound {9:11:word x} {9:13:word y}}`,

		`72 10:9:cond02 x???y {=compound {=cond {=cond {=cond {10:11:word x}}}} {10:15:word y}}`:`xy??? {=cond {=cond {=cond {=compound {10:11:word x} {10:15:word y}}}}}`,
		`72 10:9:cond02 xy {=compound {10:11:word x} {10:15:word y}}`:`xy {=compound {10:11:word x} {10:15:word y}}`,

		`82 11:9:cond03 &(something) {11:12:closure {11:14:word something}}`:[]string{`{} {11:12:null}`,`&(something) {11:12:closure {11:14:word something}}`},
		`82 11:9:cond03 x&(something) {=compound {11:11:word x} {11:12:closure {11:14:word something}}}`:`x&(something) {=compound {11:11:word x} {11:12:closure {11:14:word something}}}`,
		`82 11:9:cond03 x&(something)?y {=compound {=cond {=compound {11:11:word x} {11:12:closure {11:14:word something}}}} {11:25:word y}}`:`x{}y? {=cond {=compound {11:11:word x} {11:12:null} {11:25:word y}}}`,
		`82 11:9:cond03 x&(something)y {=compound {11:11:word x} {11:12:closure {11:14:word something}} {11:25:word y}}`:`x{}y {=compound {11:11:word x} {11:12:null} {11:25:word y}}`,
		`82 11:9:cond03 x{}y {=compound {11:11:word x} {11:12:null} {11:25:word y}}`:`x{}y {=compound {11:11:word x} {11:12:null} {11:25:word y}}`,

		`93 13:8:cond11 xy {=compound {13:11:word x} {13:13:word y}}`:`xy {=compound {13:11:word x} {13:13:word y}}`,
		`103 14:8:cond12 xy {=compound {14:11:word x} {14:15:word y}}`:`xy {=compound {14:11:word x} {14:15:word y}}`,

		`113 15:8:cond13 &(something) {15:12:closure {15:14:word something}}`:[]string{`{} {15:12:null}`,`{15:14:word something} &(something) {15:12:closure {15:14:word something}}`},
		`113 15:8:cond13 x&(something)y {=compound {15:11:word x} {15:12:closure {15:14:word something}} {15:25:word y}}`:`x{}y {=compound {15:11:word x} {15:12:null} {15:25:word y}}`,
		`113 15:8:cond13 x&(something) {=compound {15:11:word x} {15:12:closure {15:14:word something}}}`:`x{} {=compound {15:11:word x} {15:12:null}}`,
		`113 15:8:cond13 x{}y {=compound {15:11:word x} {15:12:null} {15:25:word y}}`:`x{}y {=compound {15:11:word x} {15:12:null} {15:25:word y}}`,

		`133 18:16:disjunction00 x{} {=compound {18:18:word x} {18:20:null}}`:`x{} {=compound {18:18:word x} {18:20:null}}`,

		`143 19:16:disjunction01 $1 {19:20:delegate {19:21:auto 1}}`:`{} {19:20 {19:21:null}}`,

		`145 19:16:disjunction01 $1 {19:20:delegate {19:21:auto 1}}`:`a b c {=list {19:20 {19:18:word a}} {19:20 {19:18:word b}} {19:20 {19:18:word c}}}`,
		`145 19:16:disjunction01 x{$1} {=compound {19:18:word x} {19:19:disjunction {19:20:delegate {19:21:auto 1}}}}`:`xa xb xc {=list {=compound {19:18:word x} {19:20 {19:18:word a}}} {=compound {19:18:word x} {19:20 {19:18:word b}}} {=compound {19:18:word x} {19:20 {19:18:word c}}}}`,

		`149 19:16:disjunction01 xa {=compound {19:18:word x} {19:20 {19:18:word a}}}`:`xa {=compound {19:18:word x} {19:20 {19:18:word a}}}`,
		`149 19:16:disjunction01 xb {=compound {19:18:word x} {19:20 {19:18:word b}}}`:`xb {=compound {19:18:word x} {19:20 {19:18:word b}}}`,
		`149 19:16:disjunction01 xc {=compound {19:18:word x} {19:20 {19:18:word c}}}`:`xc {=compound {19:18:word x} {19:20 {19:18:word c}}}`,

		`159 20:16:disjunction02 &(something) {20:20:closure {20:22:word something}}`:[]string{`{} {20:20:null}`,`{20:22:word something} &(something) {20:20:closure {20:22:word something}}`},

		`183 22:15:disjunction1 xay1z {=compound {22:18:word x} {22:20:word a} {22:26:word y} {22:28:decimal 1} {22:34:word z}}`:`xay1z {=compound {22:18:word x} {22:20:word a} {22:26:word y} {22:28:decimal 1} {22:34:word z}}`,
		`183 22:15:disjunction1 xay2z {=compound {22:18:word x} {22:20:word a} {22:26:word y} {22:30:decimal 2} {22:34:word z}}`:`xay2z {=compound {22:18:word x} {22:20:word a} {22:26:word y} {22:30:decimal 2} {22:34:word z}}`,
		`183 22:15:disjunction1 xay3z {=compound {22:18:word x} {22:20:word a} {22:26:word y} {22:32:decimal 3} {22:34:word z}}`:`xay3z {=compound {22:18:word x} {22:20:word a} {22:26:word y} {22:32:decimal 3} {22:34:word z}}`,
		`183 22:15:disjunction1 xby1z {=compound {22:18:word x} {22:22:word b} {22:26:word y} {22:28:decimal 1} {22:34:word z}}`:`xby1z {=compound {22:18:word x} {22:22:word b} {22:26:word y} {22:28:decimal 1} {22:34:word z}}`,
		`183 22:15:disjunction1 xby2z {=compound {22:18:word x} {22:22:word b} {22:26:word y} {22:30:decimal 2} {22:34:word z}}`:`xby2z {=compound {22:18:word x} {22:22:word b} {22:26:word y} {22:30:decimal 2} {22:34:word z}}`,
		`183 22:15:disjunction1 xby3z {=compound {22:18:word x} {22:22:word b} {22:26:word y} {22:32:decimal 3} {22:34:word z}}`:`xby3z {=compound {22:18:word x} {22:22:word b} {22:26:word y} {22:32:decimal 3} {22:34:word z}}`,
		`183 22:15:disjunction1 xcy1z {=compound {22:18:word x} {22:24:word c} {22:26:word y} {22:28:decimal 1} {22:34:word z}}`:`xcy1z {=compound {22:18:word x} {22:24:word c} {22:26:word y} {22:28:decimal 1} {22:34:word z}}`,
		`183 22:15:disjunction1 xcy2z {=compound {22:18:word x} {22:24:word c} {22:26:word y} {22:30:decimal 2} {22:34:word z}}`:`xcy2z {=compound {22:18:word x} {22:24:word c} {22:26:word y} {22:30:decimal 2} {22:34:word z}}`,
		`183 22:15:disjunction1 xcy3z {=compound {22:18:word x} {22:24:word c} {22:26:word y} {22:32:decimal 3} {22:34:word z}}`:`xcy3z {=compound {22:18:word x} {22:24:word c} {22:26:word y} {22:32:decimal 3} {22:34:word z}}`,

		`197 23:14:disjunction2 xay1z {=compound {23:18:word x} {23:20:word a} {23:26:word y} {23:28:decimal 1} {23:34:word z}}`:`xay1z {=compound {23:18:word x} {23:20:word a} {23:26:word y} {23:28:decimal 1} {23:34:word z}}`,
		`197 23:14:disjunction2 xay2z {=compound {23:18:word x} {23:20:word a} {23:26:word y} {23:30:decimal 2} {23:34:word z}}`:`xay2z {=compound {23:18:word x} {23:20:word a} {23:26:word y} {23:30:decimal 2} {23:34:word z}}`,
		`197 23:14:disjunction2 xay3z {=compound {23:18:word x} {23:20:word a} {23:26:word y} {23:32:decimal 3} {23:34:word z}}`:`xay3z {=compound {23:18:word x} {23:20:word a} {23:26:word y} {23:32:decimal 3} {23:34:word z}}`,
		`197 23:14:disjunction2 xby1z {=compound {23:18:word x} {23:22:word b} {23:26:word y} {23:28:decimal 1} {23:34:word z}}`:`xby1z {=compound {23:18:word x} {23:22:word b} {23:26:word y} {23:28:decimal 1} {23:34:word z}}`,
		`197 23:14:disjunction2 xby2z {=compound {23:18:word x} {23:22:word b} {23:26:word y} {23:30:decimal 2} {23:34:word z}}`:`xby2z {=compound {23:18:word x} {23:22:word b} {23:26:word y} {23:30:decimal 2} {23:34:word z}}`,
		`197 23:14:disjunction2 xby3z {=compound {23:18:word x} {23:22:word b} {23:26:word y} {23:32:decimal 3} {23:34:word z}}`:`xby3z {=compound {23:18:word x} {23:22:word b} {23:26:word y} {23:32:decimal 3} {23:34:word z}}`,
		`197 23:14:disjunction2 xcy1z {=compound {23:18:word x} {23:24:word c} {23:26:word y} {23:28:decimal 1} {23:34:word z}}`:`xcy1z {=compound {23:18:word x} {23:24:word c} {23:26:word y} {23:28:decimal 1} {23:34:word z}}`,
		`197 23:14:disjunction2 xcy2z {=compound {23:18:word x} {23:24:word c} {23:26:word y} {23:30:decimal 2} {23:34:word z}}`:`xcy2z {=compound {23:18:word x} {23:24:word c} {23:26:word y} {23:30:decimal 2} {23:34:word z}}`,
		`197 23:14:disjunction2 xcy3z {=compound {23:18:word x} {23:24:word c} {23:26:word y} {23:32:decimal 3} {23:34:word z}}`:`xcy3z {=compound {23:18:word x} {23:24:word c} {23:26:word y} {23:32:decimal 3} {23:34:word z}}`,

		`275 38:8:val1 foo.c {=compound {38:10:word foo} {38:13:punct .} {38:14:word c}}`:`foo.c {=compound {38:10:word foo} {38:13:punct .} {38:14:word c}}`,
		`277 38:8:val1 foo.c {=compound {38:10:word foo} {38:13:punct .} {38:14:word c}}`:`foo.c {=compound {38:10:word foo} {38:13:punct .} {38:14:word c}}`,
		`279 38:8:val1 foo.c {=compound {38:10:word foo} {38:13:punct .} {38:14:word c}}`:`foo.c {=compound {38:10:word foo} {38:13:punct .} {38:14:word c}}`,
		`281 38:8:val1 foo.c {=compound {38:10:word foo} {38:13:punct .} {38:14:word c}}`:`foo.c {=compound {38:10:word foo} {38:13:punct .} {38:14:word c}}`,
		`291 39:8:val2 bar.c {=compound {39:14:word bar} {39:17:punct .} {39:18:word c}}`:`bar.c {=compound {39:14:word bar} {39:17:punct .} {39:18:word c}}`,
		`297 39:8:val2 bar.c {=compound {39:14:word bar} {39:17:punct .} {39:18:word c}}`:`bar.c {=compound {39:14:word bar} {39:17:punct .} {39:18:word c}}`,
		`299 39:8:val2 bar.c {=compound {39:14:word bar} {39:17:punct .} {39:18:word c}}`:`bar.c {=compound {39:14:word bar} {39:17:punct .} {39:18:word c}}`,
		`301 39:8:val2 bar.c {=compound {39:14:word bar} {39:17:punct .} {39:18:word c}}`:`bar.c {=compound {39:14:word bar} {39:17:punct .} {39:18:word c}}`,
		`305 39:8:val2 bar.c {=compound {39:14:word bar} {39:17:punct .} {39:18:word c}}`:`bar.c {=compound {39:14:word bar} {39:17:punct .} {39:18:word c}}`,
		`315 40:8:val3 bar.c {=compound {40:23:word bar} {40:26:punct .} {40:27:word c}}`:`bar.c {=compound {40:23:word bar} {40:26:punct .} {40:27:word c}}`,
		`321 40:8:val3 bar.c {=compound {40:23:word bar} {40:26:punct .} {40:27:word c}}`:`bar.c {=compound {40:23:word bar} {40:26:punct .} {40:27:word c}}`,
		`323 40:8:val3 bar.c {=compound {40:23:word bar} {40:26:punct .} {40:27:word c}}`:`bar.c {=compound {40:23:word bar} {40:26:punct .} {40:27:word c}}`,
		`325 40:8:val3 bar.c {=compound {40:23:word bar} {40:26:punct .} {40:27:word c}}`:`bar.c {=compound {40:23:word bar} {40:26:punct .} {40:27:word c}}`,
		`329 40:8:val3 bar.c {=compound {40:23:word bar} {40:26:punct .} {40:27:word c}}`:`bar.c {=compound {40:23:word bar} {40:26:punct .} {40:27:word c}}`,

		`345 41:7:val4 a\,b\,c,x\,y\,z {=compound {41:18:word a} {41:19:escaped \,} {41:21:word b} {41:22:escaped \,} {41:24:word c} {41:25:punct ,} {41:26:word x} {41:27:escaped \,} {41:29:word y} {41:30:escaped \,} {41:32:word z}}`:`a\,b\,c,x\,y\,z {=compound {41:18:word a} {41:19:escaped \,} {41:21:word b} {41:22:escaped \,} {41:24:word c} {41:25:punct ,} {41:26:word x} {41:27:escaped \,} {41:29:word y} {41:30:escaped \,} {41:32:word z}}`,
		`355 42:7:val5 a\,b\,c {=compound {42:18:word a} {42:19:escaped \,} {42:21:word b} {42:22:escaped \,} {42:24:word c}}`:`a\,b\,c {=compound {42:18:word a} {42:19:escaped \,} {42:21:word b} {42:22:escaped \,} {42:24:word c}}`,
		`355 42:7:val5 x\,y\,z {=compound {42:26:word x} {42:27:escaped \,} {42:29:word y} {42:30:escaped \,} {42:32:word z}}`:`x\,y\,z {=compound {42:26:word x} {42:27:escaped \,} {42:29:word y} {42:30:escaped \,} {42:32:word z}}`,

		`421 52:17:configure.types , {=compound {52:29 {51:24:raw}} {52:30:punct ,}}`:`, {=compound {52:29 {51:24:raw}} {52:30:punct ,}}`,
		`421 52:17:configure.types . {=compound {52:33 {51:24:raw}} {52:34:punct .}}`:`. {=compound {52:33 {51:24:raw}} {52:34:punct .}}`,
		`421 52:17:configure.types atomic.h, {=compound {52:29 {51:24:raw atomic.h}} {52:30:punct ,}}`:`atomic.h, {=compound {52:29 {51:24:raw atomic.h}} {52:30:punct ,}}`,
		`421 52:17:configure.types atomic.h. {=compound {52:33 {51:24:raw atomic.h}} {52:34:punct .}}`:`atomic.h. {=compound {52:33 {51:24:raw atomic.h}} {52:34:punct .}}`,

		`476 59:11:conf3 conf3, {=compound {59:29 {59:11:word conf3}} {59:31:punct ,}}`:`conf3, {=compound {59:29 {59:11:word conf3}} {59:31:punct ,}}`,
		`476 59:11:conf3 bar, {=compound {59:36 {59:23:word bar}} {59:38:punct ,}}`:`bar, {=compound {59:36 {59:23:word bar}} {59:38:punct ,}}`,
	},
}

var checkstrs_value = map[string]map[string]any{
	"loader.go": map[string]any{
		`56:11:conf1 $/ {56:48:delegate {1:1:def /}}`:testdata_f(`{56:48 {=path %[3]s {1:1:raw value}}} %[1]s/value`),
		`57:11:conf2 $/ {57:97:delegate {1:1:def /}}`:testdata_f(`{57:97 {=path %[3]s {1:1:raw value}}} %[1]s/value`),
	},
	"check-value_test.go": map[string]any{
		`82 11:9:cond03 &(something) {11:12:closure {11:14:word something}}`:`{11:12:null} `,
		`113 15:8:cond13 &(something) {15:12:closure {15:14:word something}}`:`{15:12:null} `,
		`143 19:16:disjunction01 $1 {19:20:delegate {19:21:auto 1}}`:`{19:20 {19:21:null}} `,
		`159 20:16:disjunction02 &(something) {20:20:closure {20:22:word something}}`:`{20:20:null} `,
	},
}
