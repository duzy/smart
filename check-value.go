//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints_value = map[string]map[string]any{
	"loader.go": map[string]any{
		`42:7:val5 $(quote a\,b\,c,x\,y\,z) {42:10:delegate {42:12:builtin quote} {=list {=compound {42:18:word a} {42:19:escaped \,} {42:21:word b} {42:22:escaped \,} {42:24:word c}}} {=list {=compound {42:26:word x} {42:27:escaped \,} {42:29:word y} {42:30:escaped \,} {42:32:word z}}}}`:`{42:12:builtin quote} {=quote a\,b\,c x\,y\,z} {42:10 {=quote {=list {=compound {42:18:word a} {42:19:escaped \,} {42:21:word b} {42:22:escaped \,} {42:24:word c}}} {=list {=compound {42:26:word x} {42:27:escaped \,} {42:29:word y} {42:30:escaped \,} {42:32:word z}}}}}`,
		`15:8:cond13 &(something) {15:12:closure {15:14:word something}}`:`{15:14:word something} &(something) {15:12:closure {15:14:word something}}`,
		`55:11:conf0 $@ {55:19:delegate {55:11:def @}}`:`{55:11:def @} conf0 {55:19 {55:11:word conf0}}`,
		`59:11:conf3 $@ {59:29:delegate {59:11:def @}}`:`{59:11:def @} conf3 {59:29 {59:11:word conf3}}`,
		`59:11:conf3 $< {59:33:delegate {1:1:def <}}`:`{1:1:def <} foo {59:33 {59:19:word foo}}`,
		`59:11:conf3 $> {59:36:delegate {1:1:def >}}`:`{1:1:def >} bar {59:36 {59:23:word bar}}`,
		`59:11:conf3 $^ {59:40:delegate {1:1:def ^}}`:`{1:1:def ^} foo bar {=list {59:40 {59:19:word foo}} {59:40 {59:23:word bar}}}`,
		`56:11:conf1 $/ {56:48:delegate {1:1:def /}}`:sfmt(`{1:1:def /} %[1]s/value {56:48 {=path %[2]s {1:1:word value}}}`,testdata_s,testdata_t),
		`57:11:conf2 $/ {57:97:delegate {1:1:def /}}`:sfmt(`{1:1:def /} %[1]s/value {57:97 {=path %[2]s {1:1:word value}}}`,testdata_s,testdata_t),
		`56:11:conf1 $0 {56:45:delegate {56:46:auto 0}}`:ssfmt([]string{
			`{%[1]s/value/test.txt:1:1:def 0} foo.o {56:45 {%[1]s/value/test.txt:1:1:raw foo.o}}`,
			`{%[1]s/value/test.txt:2:1:def 0} foo-x.o {56:45 {%[1]s/value/test.txt:2:1:raw foo-x.o}}`,
			`{%[1]s/value/test.txt:3:1:def 0} foo-x-y.o {56:45 {%[1]s/value/test.txt:3:1:raw foo-x-y.o}}`,
			`{%[1]s/value/test.txt:4:1:def 0} foo-x-y-z.o {56:45 {%[1]s/value/test.txt:4:1:raw foo-x-y-z.o}}`,
			`{%[1]s/value/test.txt:5:1:def 0} foobar.o {56:45 {%[1]s/value/test.txt:5:1:raw foobar.o}}`,
		},testdata_s),
		`56:11:conf1 $(grep {=regex ^.+?\.o$$},$0,$//test.txt) {56:19:delegate {56:21:builtin grep} {=list {56:34:regex ^.+?\.o$}} {=list {56:45:delegate {56:46:auto 0}}} {=list {=path {56:48:delegate {1:1:def /}} {=compound {56:51:word test} {56:55:punct .} {56:56:word txt}}}}}`:`{56:21:builtin grep}`+
			sfmt(` foo.o foo-x.o foo-x-y.o foo-x-y-z.o foobar.o {=list `+
				`{56:19 {56:45 {%[1]s/value/test.txt:1:1:raw foo.o}}} `+
				`{56:19 {56:45 {%[1]s/value/test.txt:2:1:raw foo-x.o}}} `+
				`{56:19 {56:45 {%[1]s/value/test.txt:3:1:raw foo-x-y.o}}} `+
				`{56:19 {56:45 {%[1]s/value/test.txt:4:1:raw foo-x-y-z.o}}} `+
				`{56:19 {56:45 {%[1]s/value/test.txt:5:1:raw foobar.o}}}}`,testdata_s),
		`57:11:conf2 $(grep {=regex ^(.+?)((-(?P<i>.+?))*)(\.)(?P<x>o)$$},$0 $1 $2 $3 $(i) $5 $(x),$//test.txt) {57:19:delegate {57:21:builtin grep} {=list {57:34:regex ^(.+?)((-(?P<i>.+?))*)(\.)(?P<x>o)$}} {=list {57:72:delegate {56:46:auto 0}} {57:75:delegate {19:21:auto 1}} {57:78:delegate {57:79:auto 2}} {57:81:delegate {57:82:auto 3}} {57:84:delegate {57:86:auto i}} {57:89:delegate {57:90:auto 5}} {57:92:delegate {57:94:auto x}}} {=list {=path {57:97:delegate {1:1:def /}} {=compound {57:100:word test} {57:104:punct .} {57:105:word txt}}}}}`:`{57:21:builtin grep}`+
			sfmt(` foo.o foo . o foo-x.o foo -x -x x . o foo-x-y.o foo -x-y -y y . o foo-x-y-z.o foo -x-y-z -z z . o foobar.o foobar . o {=list `+
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
				`{57:19 {57:92 {%[1]s/value/test.txt:5:8:raw o}}}}`,testdata_s),
		`57:11:conf2 $0 {57:72:delegate {56:46:auto 0}}`:ssfmt([]string{
			`{%[1]s/value/test.txt:1:1:def 0} foo.o {57:72 {%[1]s/value/test.txt:1:1:raw foo.o}}`,
			`{%[1]s/value/test.txt:2:1:def 0} foo-x.o {57:72 {%[1]s/value/test.txt:2:1:raw foo-x.o}}`,
			`{%[1]s/value/test.txt:3:1:def 0} foo-x-y.o {57:72 {%[1]s/value/test.txt:3:1:raw foo-x-y.o}}`,
			`{%[1]s/value/test.txt:4:1:def 0} foo-x-y-z.o {57:72 {%[1]s/value/test.txt:4:1:raw foo-x-y-z.o}}`,
			`{%[1]s/value/test.txt:5:1:def 0} foobar.o {57:72 {%[1]s/value/test.txt:5:1:raw foobar.o}}`,
		},testdata_s),
		`57:11:conf2 $1 {57:75:delegate {19:21:auto 1}}`:ssfmt([]string{
			`{%[1]s/value/test.txt:1:1:def 1} foo {57:75 {%[1]s/value/test.txt:1:1:raw foo}}`,
			`{%[1]s/value/test.txt:2:1:def 1} foo {57:75 {%[1]s/value/test.txt:2:1:raw foo}}`,
			`{%[1]s/value/test.txt:3:1:def 1} foo {57:75 {%[1]s/value/test.txt:3:1:raw foo}}`,
			`{%[1]s/value/test.txt:4:1:def 1} foo {57:75 {%[1]s/value/test.txt:4:1:raw foo}}`,
			`{%[1]s/value/test.txt:5:1:def 1} foobar {57:75 {%[1]s/value/test.txt:5:1:raw foobar}}`,
		},testdata_s),
		`57:11:conf2 $2 {57:78:delegate {57:79:auto 2}}`:ssfmt([]string{
			`{%[1]s/value/test.txt:1:4:def 2}  {57:78 {%[1]s/value/test.txt:1:4:raw}}`,
			`{%[1]s/value/test.txt:2:4:def 2} -x {57:78 {%[1]s/value/test.txt:2:4:raw -x}}`,
			`{%[1]s/value/test.txt:3:4:def 2} -x-y {57:78 {%[1]s/value/test.txt:3:4:raw -x-y}}`,
			`{%[1]s/value/test.txt:4:4:def 2} -x-y-z {57:78 {%[1]s/value/test.txt:4:4:raw -x-y-z}}`,
			`{%[1]s/value/test.txt:5:7:def 2}  {57:78 {%[1]s/value/test.txt:5:7:raw}}`,
		},testdata_s),
		`57:11:conf2 $3 {57:81:delegate {57:82:auto 3}}`:ssfmt([]string{
			`{%[1]s/value/test.txt:1:4:def 3}  {57:81 {%[1]s/value/test.txt:1:4:raw}}`,
			`{%[1]s/value/test.txt:2:4:def 3} -x {57:81 {%[1]s/value/test.txt:2:4:raw -x}}`,
			`{%[1]s/value/test.txt:3:6:def 3} -y {57:81 {%[1]s/value/test.txt:3:6:raw -y}}`,
			`{%[1]s/value/test.txt:4:8:def 3} -z {57:81 {%[1]s/value/test.txt:4:8:raw -z}}`,
			`{%[1]s/value/test.txt:5:7:def 3}  {57:81 {%[1]s/value/test.txt:5:7:raw}}`,
		},testdata_s),
		`57:11:conf2 $(i) {57:84:delegate {57:86:auto i}}`:ssfmt([]string{
			`{%[1]s/value/test.txt:1:4:def i}  {57:84 {%[1]s/value/test.txt:1:4:raw}}`,
			`{%[1]s/value/test.txt:2:5:def i} x {57:84 {%[1]s/value/test.txt:2:5:raw x}}`,
			`{%[1]s/value/test.txt:3:7:def i} y {57:84 {%[1]s/value/test.txt:3:7:raw y}}`,
			`{%[1]s/value/test.txt:4:9:def i} z {57:84 {%[1]s/value/test.txt:4:9:raw z}}`,
			`{%[1]s/value/test.txt:5:7:def i}  {57:84 {%[1]s/value/test.txt:5:7:raw}}`,
		},testdata_s),
		`57:11:conf2 $5 {57:89:delegate {57:90:auto 5}}`:ssfmt([]string{
			`{%[1]s/value/test.txt:1:4:def 5} . {57:89 {%[1]s/value/test.txt:1:4:raw .}}`,
			`{%[1]s/value/test.txt:2:6:def 5} . {57:89 {%[1]s/value/test.txt:2:6:raw .}}`,
			`{%[1]s/value/test.txt:3:8:def 5} . {57:89 {%[1]s/value/test.txt:3:8:raw .}}`,
			`{%[1]s/value/test.txt:4:10:def 5} . {57:89 {%[1]s/value/test.txt:4:10:raw .}}`,
			`{%[1]s/value/test.txt:5:7:def 5} . {57:89 {%[1]s/value/test.txt:5:7:raw .}}`,
		},testdata_s),
		`57:11:conf2 $(x) {57:92:delegate {57:94:auto x}}`:ssfmt([]string{
			`{%[1]s/value/test.txt:1:5:def x} o {57:92 {%[1]s/value/test.txt:1:5:raw o}}`,
			`{%[1]s/value/test.txt:2:7:def x} o {57:92 {%[1]s/value/test.txt:2:7:raw o}}`,
			`{%[1]s/value/test.txt:3:9:def x} o {57:92 {%[1]s/value/test.txt:3:9:raw o}}`,
			`{%[1]s/value/test.txt:4:11:def x} o {57:92 {%[1]s/value/test.txt:4:11:raw o}}`,
			`{%[1]s/value/test.txt:5:8:def x} o {57:92 {%[1]s/value/test.txt:5:8:raw o}}`,
		},testdata_s),
	},
	"check-value_test.go": map[string]any{
		`82 11:9:cond03 &(something) {11:12:closure {11:14:word something}}`:`{11:14:word something} {} {11:12:null}`,
		`113 15:8:cond13 &(something) {15:12:closure {15:14:word something}}`:`{15:14:word something} {} {15:12:null}`,
		`143 19:16:disjunction01 $1 {19:20:delegate {19:21:auto 1}}`:`{} {} {19:20:null}`,
		`145 19:16:disjunction01 $1 {19:20:delegate {19:21:auto 1}}`:`{19:16:def 1} a b c {=list {19:20 {19:18:word a}} {19:20 {19:18:word b}} {19:20 {19:18:word c}}}`,
		`159 20:16:disjunction02 &(something) {20:20:closure {20:22:word something}}`:`{20:22:word something} {} {20:20:null}`,
	},
}

var checkstrs_value = map[string]map[string]any{
	"loader.go": map[string]any{
		`56:11:conf1 $/ {56:48:delegate {1:1:def /}}`:sfmt(`{56:48 {=path %[2]s {1:1:word value}}} %[1]s/value`,testdata_s,testdata_t),
		`57:11:conf2 $/ {57:97:delegate {1:1:def /}}`:sfmt(`{57:97 {=path %[2]s {1:1:word value}}} %[1]s/value`,testdata_s,testdata_t),
	},
	"check-value_test.go": map[string]any{
	},
}
